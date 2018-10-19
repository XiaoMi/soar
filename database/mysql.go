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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoMi/soar/ast"
	"github.com/XiaoMi/soar/common"

	"github.com/ziutek/mymysql/mysql"
	// mymysql driver
	_ "github.com/ziutek/mymysql/native"
	"vitess.io/vitess/go/vt/sqlparser"
)

// Connector 数据库连接基本对象
type Connector struct {
	Addr     string
	User     string
	Pass     string
	Database string
	Charset  string
}

// QueryResult 数据库查询返回值
type QueryResult struct {
	Rows      []mysql.Row
	Result    mysql.Result
	Error     error
	Warning   []mysql.Row
	QueryCost float64
}

// NewConnection 创建新连接
func (db *Connector) NewConnection() mysql.Conn {
	return mysql.New("tcp", "", db.Addr, db.User, db.Pass, db.Database)
}

// Query 执行SQL
func (db *Connector) Query(sql string, params ...interface{}) (*QueryResult, error) {
	// 测试环境如果检查是关闭的，则SQL不会被执行
	if common.Config.TestDSN.Disable {
		return nil, errors.New("TestDsn Disable")
	}

	// 数据库安全性检查：如果Connector的IP端口与TEST环境不一致，则启用SQL白名单
	// 不在白名单中的SQL不允许执行
	// 执行环境与test环境不相同
	if db.Addr != common.Config.TestDSN.Addr && db.dangerousQuery(sql) {
		return nil, fmt.Errorf("query execution deny: execute SQL with DSN(%s/%s) '%s'",
			db.Addr, db.Database, fmt.Sprintf(sql, params...))
	}

	common.Log.Debug("Execute SQL with DSN(%s/%s) : %s", db.Addr, db.Database, fmt.Sprintf(sql, params...))
	conn := db.NewConnection()

	// 设置SQL连接超时时间
	conn.SetTimeout(time.Duration(common.Config.ConnTimeOut) * time.Second)
	defer conn.Close()
	err := conn.Connect()
	if err != nil {
		return nil, err
	}

	// 添加SQL执行超时限制
	ch := make(chan QueryResult, 1)
	go func() {
		res := QueryResult{}
		res.Rows, res.Result, res.Error = conn.Query(sql, params...)

		if common.Config.ShowWarnings {
			warning, _, err := conn.Query("SHOW WARNINGS")
			if err == nil {
				res.Warning = warning
			}
		}

		// SHOW WARNINGS并不会影响last_query_cost
		if common.Config.ShowLastQueryCost {
			cost, _, err := conn.Query("SHOW SESSION STATUS LIKE 'last_query_cost'")
			if err == nil {
				if len(cost) > 0 {
					res.QueryCost = cost[0].Float(1)
				}
			}
		}

		ch <- res
	}()

	select {
	case res := <-ch:
		return &res, res.Error
	case <-time.After(time.Duration(common.Config.QueryTimeOut) * time.Second):
		return nil, errors.New("query execution timeout")
	}

}

// Version 获取MySQL数据库版本
func (db *Connector) Version() (int, error) {
	// 从数据库中获取版本信息
	res, err := db.Query("select @@version")
	if err != nil {
		common.Log.Warn("(db *Connector) Version() Error: %v", err)
		return -1, err
	}

	// 从MySQL版本中获取版本号
	var reg *regexp.Regexp
	var v int
	reg, err = regexp.Compile(`[^0-9]+`)
	if err != nil {
		// 如果获取不到version信息，则以最新版本为准
		v = 999
		return v, err
	}
	version := reg.ReplaceAllString(res.Rows[0].Str(0), "")[:3]
	v, err = strconv.Atoi(version)
	if err != nil {
		// 如果获取不到version信息，则以最新版本为准
		v = 999
	}
	return v, err
}

// Source execute sql from file
func (db *Connector) Source(file string) ([]*QueryResult, error) {
	var sqlCounter int // SQL 计数器
	var result []*QueryResult

	fd, err := os.Open(file)
	defer func() {
		err = fd.Close()
		if err != nil {
			common.Log.Error("(db *Connector) Source(%s) fd.Close failed: %s", file, err.Error())
		}
	}()
	if err != nil {
		common.Log.Warning("(db *Connector) Source(%s) os.Open failed: %s", file, err.Error())
		return nil, err
	}
	data, err := ioutil.ReadAll(fd)
	if err != nil {
		common.Log.Critical("ioutil.ReadAll Error: %s", err.Error())
		return nil, err
	}

	sql := strings.TrimSpace(string(data))
	buf := strings.TrimSpace(sql)
	for ; ; sqlCounter++ {
		if buf == "" {
			break
		}

		// 查询请求切分
		sql, bufBytes := ast.SplitStatement([]byte(buf), []byte(common.Config.Delimiter))
		buf = string(bufBytes)
		sql = strings.TrimSpace(sql)
		common.Log.Debug("Source Query SQL: %s", sql)

		res, e := db.Query(sql)
		if e != nil {
			common.Log.Error("(db *Connector) Source Filename: %s, SQLCounter.: %d", file, sqlCounter)
			return result, e
		}
		result = append(result, res)
	}
	return result, nil
}

// SingleIntValue 获取某个int型变量的值
func (db *Connector) SingleIntValue(option string) (int, error) {
	// 从数据库中获取信息
	res, err := db.Query("select @@%s", option)
	if err != nil {
		common.Log.Warn("(db *Connector) SingleIntValue() Error: %v", err)
		return -1, err
	}

	return res.Rows[0].Int(0), err
}

// ColumnCardinality 粒度计算
func (db *Connector) ColumnCardinality(tb, col string) float64 {
	// 获取该表上的已有的索引

	// show table status 获取总行数（近似）
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
	res, err := db.Query("select count(distinct `%s`) from `%s`.`%s`", col, db.Database, tb)
	if err != nil {
		common.Log.Warn("(db *Connector) ColumnCardinality() Query Error: %v", err)
		return 0
	}

	colNum := res.Rows[0].Float(0)

	// 散粒度区间：[0,1]
	return colNum / float64(rowTotal)
}

// IsView 判断表是否是视图
func (db *Connector) IsView(tbName string) bool {
	tbStatus, err := db.ShowTableStatus(tbName)
	if err != nil {
		common.Log.Error("(db *Connector) IsView Error: %v:", err)
		return false
	}

	if len(tbStatus.Rows) > 0 {
		if tbStatus.Rows[0].Comment == "VIEW" {
			return true
		}
	}

	return false

}

// RemoveSQLComments 去除SQL中的注释
func RemoveSQLComments(sql []byte) []byte {
	cmtReg := regexp.MustCompile(`("(""|[^"])*")|('(''|[^'])*')|(--[^\n\r]*)|(#.*)|(/\*([^*]|[\r\n]|(\*+([^*/]|[\r\n])))*\*+/)`)

	return cmtReg.ReplaceAllFunc(sql, func(s []byte) []byte {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') ||
			(string(s[:3]) == "/*!") {
			return s
		}
		return []byte("")
	})
}

// 为了防止在Online环境进行误操作，通过dangerousQuery来判断能否在Online执行
func (db *Connector) dangerousQuery(query string) bool {
	queries, err := sqlparser.SplitStatementToPieces(strings.TrimSpace(strings.ToLower(query)))
	if err != nil {
		return true
	}

	for _, sql := range queries {
		dangerous := true
		whiteList := []string{
			"select",
			"show",
			"explain",
			"describe",
		}

		for _, prefix := range whiteList {
			if strings.HasPrefix(sql, prefix) {
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
