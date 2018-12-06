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
	"io"
	"regexp"
	"strings"
	"time"

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
func (db *Connector) Trace(sql string, params ...interface{}) (*QueryResult, error) {
	common.Log.Debug("Trace SQL: %s", sql)
	if common.Config.TestDSN.Version < 50600 {
		return nil, errors.New("version < 5.6, not support trace")
	}

	// 过滤不需要 Trace 的 SQL
	switch sqlparser.Preview(sql) {
	case sqlparser.StmtSelect, sqlparser.StmtUpdate, sqlparser.StmtDelete:
		sql = "explain " + sql
	case sqlparser.EXPLAIN:
	default:
		return nil, errors.New("no need trace")
	}

	// 测试环境如果检查是关闭的，则SQL不会被执行
	if common.Config.TestDSN.Disable {
		return nil, errors.New("Dsn Disable")
	}

	// 数据库安全性检查：如果 Connector 的 IP 端口与 TEST 环境不一致，则启用SQL白名单
	// 不在白名单中的SQL不允许执行
	// 执行环境与test环境不相同
	if db.Addr != common.Config.TestDSN.Addr && db.dangerousQuery(sql) {
		return nil, fmt.Errorf("query Execution Deny: Execute SQL with DSN(%s/%s) '%s'",
			db.Addr, db.Database, fmt.Sprintf(sql, params...))
	}

	common.Log.Debug("Execute SQL with DSN(%s/%s) : %s", db.Addr, db.Database, sql)
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
		// 开启Trace
		common.Log.Debug("SET SESSION OPTIMIZER_TRACE='enabled=on'")
		_, _, err = conn.Query("SET SESSION OPTIMIZER_TRACE='enabled=on'")
		common.LogIfError(err, "")

		// 执行SQL，抛弃返回结果
		result, err := conn.Start(sql, params...)
		if err != nil {
			ch <- QueryResult{
				Error: err,
			}
			return
		}
		row := result.MakeRow()
		for {
			err = result.ScanRow(row)
			if err == io.EOF {
				break
			}
		}

		// 返回Trace结果
		res := QueryResult{}
		res.Rows, res.Result, res.Error = conn.Query("SELECT * FROM information_schema.OPTIMIZER_TRACE")

		// 关闭Trace
		common.Log.Debug("SET SESSION OPTIMIZER_TRACE='enabled=off'")
		_, _, err = conn.Query("SET SESSION OPTIMIZER_TRACE='enabled=off'")
		if err != nil {
			fmt.Println(err.Error())
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

// getTrace 获取trace信息
func getTrace(res *QueryResult) Trace {
	var rows []TraceRow
	for _, row := range res.Rows {
		rows = append(rows, TraceRow{
			Query:                        row.Str(0),
			Trace:                        row.Str(1),
			MissingBytesBeyondMaxMemSize: row.Int(2),
			InsufficientPrivileges:       row.Int(3),
		})
	}
	return Trace{Rows: rows}
}

// FormatTrace 格式化输出Trace信息
func FormatTrace(res *QueryResult) string {
	explainReg := regexp.MustCompile(`(?i)^explain\s+`)
	trace := getTrace(res)
	str := []string{""}
	for _, row := range trace.Rows {
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
