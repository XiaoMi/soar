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
	"strings"
	"time"

	"github.com/XiaoMi/soar/common"

	"vitess.io/vitess/go/vt/sqlparser"
)

// Profiling show profile输出的结果
type Profiling struct {
	Rows []ProfilingRow
}

// ProfilingRow show profile每一行信息
type ProfilingRow struct {
	Status   string
	Duration float64
	// TODO: 支持show profile all，不过目前看all的信息过多有点眼花缭乱
}

// Profiling 执行SQL，并对其Profiling
func (db *Connector) Profiling(sql string, params ...interface{}) (*QueryResult, error) {
	// 过滤不需要profiling的SQL
	switch sqlparser.Preview(sql) {
	case sqlparser.StmtSelect, sqlparser.StmtUpdate, sqlparser.StmtDelete:
	default:
		return nil, errors.New("no need profiling")
	}

	// 测试环境如果检查是关闭的，则SQL不会被执行
	if common.Config.TestDSN.Disable {
		return nil, errors.New("TestDsn Disable")
	}

	// 数据库安全性检查：如果Connector的IP端口与TEST环境不一致，则启用SQL白名单
	// 不在白名单中的SQL不允许执行
	// 执行环境与test环境不相同
	if db.Addr != common.Config.TestDSN.Addr && db.dangerousQuery(sql) {
		return nil, fmt.Errorf("query execution deny: Execute SQL with DSN(%s/%s) '%s'",
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
		// 开启Profiling
		_, _, err = conn.Query("set @@profiling=1")
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

		// 返回Profiling结果
		res := QueryResult{}
		res.Rows, res.Result, res.Error = conn.Query("show profile")
		_, _, err = conn.Query("set @@profiling=0")
		common.LogIfError(err, "")
		ch <- res
	}()

	select {
	case res := <-ch:
		return &res, res.Error
	case <-time.After(time.Duration(common.Config.QueryTimeOut) * time.Second):
		return nil, errors.New("query execution timeout")
	}
}

func getProfiling(res *QueryResult) Profiling {
	var rows []ProfilingRow
	for _, row := range res.Rows {
		rows = append(rows, ProfilingRow{
			Status:   row.Str(0),
			Duration: row.Float(1),
		})
	}
	return Profiling{Rows: rows}
}

// FormatProfiling 格式化输出Profiling信息
func FormatProfiling(res *QueryResult) string {
	profiling := getProfiling(res)
	str := []string{"| Status | Duration |"}
	str = append(str, "| --- | --- |")
	for _, row := range profiling.Rows {
		str = append(str, fmt.Sprintf("| %s | %f |", row.Status, row.Duration))
	}
	return strings.Join(str, "\n")
}
