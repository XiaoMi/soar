/*
 * Copyright 2018 Xiaomi, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package database

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoMi/soar/common"

	// for database/sql
	_ "github.com/go-sql-driver/mysql"
	"vitess.io/vitess/go/vt/sqlparser"
)

// Connector 数据库连接基本对象
type Connector struct {
	Addr     string
	User     string
	Pass     string
	Database string
	Charset  string
	Conn     *sql.DB
}

// QueryResult 数据库查询返回值
type QueryResult struct {
	Rows      *sql.Rows
	Error     error
	Warning   *sql.Rows
	QueryCost float64
}

// NewConnector 创建新连接
func NewConnector(dsn *common.Dsn) (*Connector, error) {
	conn, err := sql.Open("mysql", common.FormatDSN(dsn))
	if err != nil {
		return nil, err
	}
	connector := &Connector{
		Addr:     dsn.Addr,
		User:     dsn.User,
		Pass:     dsn.Password,
		Database: dsn.Schema,
		Charset:  dsn.Charset,
		Conn:     conn,
	}
	return connector, err
}

// Query 执行SQL
func (db *Connector) Query(sql string, params ...interface{}) (QueryResult, error) {
	var res QueryResult
	var err error
	// 测试环境如果检查是关闭的，则SQL不会被执行
	if common.Config.TestDSN.Disable {
		return res, errors.New("dsn is disable")
	}
	// 数据库安全性检查：如果 Connector 的 IP 端口与 TEST 环境不一致，则启用SQL白名单
	// 不在白名单中的SQL不允许执行
	// 执行环境与test环境不相同
	if db.Addr != common.Config.TestDSN.Addr && db.dangerousQuery(sql) {
		return res, fmt.Errorf("query execution deny: execute SQL with DSN(%s/%s) '%s'",
			db.Addr, db.Database, fmt.Sprintf(sql, params...))
	}

	if db.Database == "" {
		db.Database = "information_schema"
	}

	common.Log.Debug("Execute SQL with DSN(%s/%s) : %s", db.Addr, db.Database, fmt.Sprintf(sql, params...))
	_, err = db.Conn.Exec("USE " + db.Database)
	if err != nil {
		common.Log.Error(err.Error())
		return res, err
	}
	res.Rows, res.Error = db.Conn.Query(sql, params...)

	if common.Config.ShowWarnings {
		res.Warning, err = db.Conn.Query("SHOW WARNINGS")
		common.LogIfError(err, "")
	}

	// SHOW WARNINGS 并不会影响 last_query_cost
	if common.Config.ShowLastQueryCost {
		cost, err := db.Conn.Query("SHOW SESSION STATUS LIKE 'last_query_cost'")
		if err == nil {
			var varName string
			if cost.Next() {
				err = cost.Scan(&varName, &res.QueryCost)
				common.LogIfError(err, "")
			}
			if err := cost.Close(); err != nil {
				common.Log.Error(err.Error())
			}
		}
	}

	if res.Error != nil && err == nil {
		err = res.Error
	}
	return res, err
}

// Version 获取MySQL数据库版本
func (db *Connector) Version() (int, error) {
	version := 99999
	// 从数据库中获取版本信息
	res, err := db.Query("select @@version")
	if err != nil {
		common.Log.Warn("(db *Connector) Version() Error: %v", err)
		return version, err
	}

	// MariaDB https://mariadb.com/kb/en/library/comment-syntax/
	// MySQL https://dev.mysql.com/doc/refman/8.0/en/comments.html
	var versionStr string
	var versionSeg []string
	if res.Rows.Next() {
		err = res.Rows.Scan(&versionStr)
	}
	if err := res.Rows.Close(); err != nil {
		common.Log.Error(err.Error())
	}
	versionStr = strings.Split(versionStr, "-")[0]
	versionSeg = strings.Split(versionStr, ".")
	if len(versionSeg) == 3 {
		versionStr = fmt.Sprintf("%s%02s%02s", versionSeg[0], versionSeg[1], versionSeg[2])
		version, err = strconv.Atoi(versionStr)
	}
	return version, err
}

// SingleIntValue 获取某个int型变量的值
func (db *Connector) SingleIntValue(option string) (int, error) {
	// 从数据库中获取信息
	res, err := db.Query("select @@" + option)
	if err != nil {
		common.Log.Warn("(db *Connector) SingleIntValue() Error: %v", err)
		return -1, err
	}

	var intVal int
	if res.Rows.Next() {
		err = res.Rows.Scan(&intVal)
	}
	if err := res.Rows.Close(); err != nil {
		common.Log.Error(err.Error())
	}
	return intVal, err
}

// ColumnCardinality 粒度计算
func (db *Connector) ColumnCardinality(tb, col string) float64 {
	// 获取该表上的已有的索引

	// show table status 获取总行数（近似）
	common.Log.Debug("ColumnCardinality, ShowTableStatus check `%s` status Rows", tb)
	tbStatus, err := db.ShowTableStatus(tb)
	if err != nil {
		common.Log.Warn("(db *Connector) ColumnCardinality() ShowTableStatus Error: %v", err)
		return 0
	}

	// 如果是视图或表中无数据，rowTotal 都为0
	// 视图不需要加索引，无数据相当于散粒度为1
	if len(tbStatus.Rows) == 0 {
		common.Log.Debug("(db *Connector) ColumnCardinality() No table status: %s", tb)
		return 1
	}
	rowTotal := tbStatus.Rows[0].Rows
	if rowTotal == 0 {
		if common.Config.Sampling {
			common.Log.Debug("ColumnCardinality, %s rowTotal == 0", tb)
		}
		return 1
	}

	// rowTotal > xxx 时保护数据库，不对该值计算散粒度，xxx可以在配置中设置
	if rowTotal > common.Config.MaxTotalRows {
		return 0.5
	}

	// 计算该列散粒度
	db.Conn.Stats()
	res, err := db.Query(fmt.Sprintf("select count(distinct `%s`) from `%s`.`%s`",
		Escape(col, false),
		Escape(db.Database, false),
		Escape(tb, false)))
	if err != nil {
		common.Log.Warn("(db *Connector) ColumnCardinality() Query Error: %v", err)
		return 0
	}

	var colNum float64
	if res.Rows.Next() {
		err = res.Rows.Scan(&colNum)
		if err != nil {
			common.Log.Warn("(db *Connector) ColumnCardinality() Query Error: %v", err)
			return 0
		}
	}
	res.Rows.Close()

	// 当table status元数据不准确时 rowTotal 可能远小于count(*)，导致散粒度大于1
	if colNum > float64(rowTotal) {
		return 1
	}
	// 散粒度区间：[0,1]
	return colNum / float64(rowTotal)
}

// IsView 判断表是否是视图
func (db *Connector) IsView(tbName string) bool {
	common.Log.Debug("IsView, ShowTableStatus check if `%s` is view", tbName)
	tbStatus, err := db.ShowTableStatus(tbName)
	if err != nil {
		common.Log.Error("(db *Connector) IsView Error: %v:", err)
		return false
	}

	if len(tbStatus.Rows) > 0 {
		if string(tbStatus.Rows[0].Comment) == "VIEW" {
			return true
		}
	}

	return false
}

// RemoveSQLComments 去除SQL中的注释
func RemoveSQLComments(sql string) string {
	buf := []byte(sql)
	cmtReg := regexp.MustCompile(`("(""|[^"])*")|('(''|[^'])*')|(--[^\n\r]*)|(#.*)|(/\*([^*]|[\r\n]|(\*+([^*/]|[\r\n])))*\*+/)`)

	res := cmtReg.ReplaceAllFunc(buf, func(s []byte) []byte {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') ||
			(string(s[:3]) == "/*!") {
			return s
		}
		return []byte("")
	})
	return strings.TrimSpace(string(res))
}

// 为了防止在 Online 环境进行误操作，通过 dangerousQuery 来判断能否在 Online 执行
func (db *Connector) dangerousQuery(query string) bool {
	queries, err := sqlparser.SplitStatementToPieces(strings.TrimSpace(strings.ToLower(query)))
	if err != nil {
		return true
	}

	for _, query := range queries {
		dangerous := true
		whiteList := []string{
			"select",
			"show",
			"explain",
			"describe",
			"desc",
		}

		for _, prefix := range whiteList {
			if strings.HasPrefix(query, prefix) {
				dangerous = false
				break
			}
		}

		if dangerous {
			return true
		}
	}

	return false
}

// TimeFormat standard MySQL datetime format
const TimeFormat = "2006-01-02 15:04:05.000000000"

// TimeString returns t as string in MySQL format Converts time.Time zero to MySQL zero.
func TimeString(t time.Time) string {
	if t.IsZero() {
		return "0000-00-00 00:00:00"
	}
	if t.Nanosecond() == 0 {
		return t.Format(TimeFormat[:19])
	}
	return t.Format(TimeFormat)
}

// NullString null able string
func NullString(buf []byte) string {
	if buf == nil {
		return "NULL"
	}
	return string(buf)
}

// NullFloat null able float
func NullFloat(buf []byte) float64 {
	if buf == nil {
		return 0
	}
	f, _ := strconv.ParseFloat(string(buf), 64)
	return f
}

// NullInt null able int
func NullInt(buf []byte) int64 {
	if buf == nil {
		return 0
	}
	i, _ := strconv.ParseInt(string(buf), 10, 64)
	return i
}

// quoteEscape sql_mode=no_backslash_escapes
func quoteEscape(source string) string {
	var buf bytes.Buffer
	last := 0
	for ii, bb := range source {
		if bb == '\'' {
			_, err := io.WriteString(&buf, source[last:ii])
			common.LogIfWarn(err, "")
			_, err = io.WriteString(&buf, `''`)
			common.LogIfWarn(err, "")
			last = ii + 1
		}
	}
	_, err := io.WriteString(&buf, source[last:])
	common.LogIfWarn(err, "")
	return buf.String()
}

// stringEscape mysql_escape_string
// https://github.com/liule/golang_escape
func stringEscape(source string) string {
	var j int
	if source == "" {
		return source
	}
	tempStr := source[:]
	desc := make([]byte, len(tempStr)*2)
	for i, b := range tempStr {
		flag := false
		var escape byte
		switch b {
		case '\000':
			flag = true
			escape = '\000'
		case '\r':
			flag = true
			escape = '\r'
		case '\n':
			flag = true
			escape = '\n'
		case '\\':
			flag = true
			escape = '\\'
		case '\'':
			flag = true
			escape = '\''
		case '"':
			flag = true
			escape = '"'
		case '\032':
			flag = true
			escape = 'Z'
		default:
		}
		if flag {
			desc[j] = '\\'
			desc[j+1] = escape
			j = j + 2
		} else {
			desc[j] = tempStr[i]
			j = j + 1
		}
	}
	return string(desc[0:j])
}

// Escape like C API mysql_escape_string()
func Escape(source string, NoBackslashEscapes bool) string {
	// NoBackslashEscapes https://dev.mysql.com/doc/refman/8.0/en/sql-mode.html#sqlmode_no_backslash_escapes
	// TODO: NoBackslashEscapes always false
	if NoBackslashEscapes {
		return quoteEscape(source)
	}
	return stringEscape(source)
}
