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
	"regexp"
	"strings"

	"github.com/XiaoMi/soar/common"

	"vitess.io/vitess/go/vt/sqlparser"
)

// Trace 用于存放 Select * From Information_Schema.Optimizer_Trace;输出的结果
type Trace struct {
	Rows []TraceRow
}

// TraceRow 中含有trace的基本信息
type TraceRow struct {
	Query                        string
	Trace                        string
	MissingBytesBeyondMaxMemSize int
	InsufficientPrivileges       int
}

// Trace 执行SQL，并对其Trace
func (db *Connector) Trace(sql string, params ...interface{}) ([]TraceRow, error) {
	common.Log.Debug("Trace SQL: %s", sql)
	var rows []TraceRow
	if common.Config.TestDSN.Version < 50600 {
		return rows, errors.New("version < 5.6, not support trace")
	}

	// 过滤不需要 Trace 的 SQL
	switch sqlparser.Preview(sql) {
	case sqlparser.StmtSelect, sqlparser.StmtUpdate, sqlparser.StmtDelete:
		sql = "explain " + sql
	case sqlparser.EXPLAIN:
	default:
		return rows, errors.New("no need trace")
	}

	// 测试环境如果检查是关闭的，则SQL不会被执行
	if common.Config.TestDSN.Disable {
		return rows, errors.New("dsn is disable")
	}

	// 数据库安全性检查：如果 Connector 的 IP 端口与 TEST 环境不一致，则启用SQL白名单
	// 不在白名单中的SQL不允许执行
	// 执行环境与test环境不相同
	if db.Addr != common.Config.TestDSN.Addr && db.dangerousQuery(sql) {
		return rows, fmt.Errorf("query Execution Deny: Execute SQL with DSN(%s/%s) '%s'",
			db.Addr, db.Database, fmt.Sprintf(sql, params...))
	}

	common.Log.Debug("Execute SQL with DSN(%s/%s) : %s", db.Addr, db.Database, sql)
	// 开启Trace
	common.Log.Debug("SET SESSION OPTIMIZER_TRACE='enabled=on'")
	trx, err := db.Conn.Begin()
	if err != nil {
		return rows, err
	}
	defer func() {
		trxErr := trx.Rollback()
		if trxErr != nil {
			common.Log.Debug(trxErr.Error())
		}
	}()
	_, err = trx.Query("SET SESSION OPTIMIZER_TRACE='enabled=on'")
	common.LogIfError(err, "")

	// 执行SQL，抛弃返回结果
	tmpRes, err := trx.Query(sql, params...)
	if err != nil {
		return rows, err
	}
	for tmpRes.Next() {
		continue
	}

	// 返回Trace结果
	res, err := trx.Query("SELECT * FROM information_schema.OPTIMIZER_TRACE")
	if err != nil {
		trxErr := trx.Rollback()
		if trxErr != nil {
			common.Log.Debug(trxErr.Error())
		}
		return rows, err
	}
	for res.Next() {
		var traceRow TraceRow
		err = res.Scan(&traceRow.Query, &traceRow.Trace, &traceRow.MissingBytesBeyondMaxMemSize, &traceRow.InsufficientPrivileges)
		if err != nil {
			common.LogIfError(err, "")
			break
		}
		rows = append(rows, traceRow)
	}
	res.Close()

	// 关闭Trace
	common.Log.Debug("SET SESSION OPTIMIZER_TRACE='enabled=off'")
	_, err = trx.Query("SET SESSION OPTIMIZER_TRACE='enabled=off'")
	common.LogIfError(err, "")
	return rows, err
}

// FormatTrace 格式化输出Trace信息
func FormatTrace(rows []TraceRow) string {
	explainReg := regexp.MustCompile(`(?i)^explain\s+`)
	str := []string{""}
	for _, row := range rows {
		str = append(str, "```sql")
		sql := explainReg.ReplaceAllString(row.Query, "")
		str = append(str, sql)
		str = append(str, "```\n")
		str = append(str, "```json")
		str = append(str, row.Trace)
		str = append(str, "```\n")
	}
	return strings.Join(str, "\n")
}
