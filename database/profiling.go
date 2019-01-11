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
	"strings"

	"github.com/XiaoMi/soar/common"

	"vitess.io/vitess/go/vt/sqlparser"
)

// Profiling show profile 输出的结果
type Profiling struct {
	Rows []ProfilingRow
}

// ProfilingRow show profile每一行信息
type ProfilingRow struct {
	Status   string
	Duration float64
	// TODO: 支持show profile all, 不过目前看所有的信息过多有点眼花缭乱
}

// Profiling 执行SQL，并对其 Profile
func (db *Connector) Profiling(sql string, params ...interface{}) ([]ProfilingRow, error) {
	var rows []ProfilingRow
	// 过滤不需要 profiling 的 SQL
	switch sqlparser.Preview(sql) {
	case sqlparser.StmtSelect, sqlparser.StmtUpdate, sqlparser.StmtDelete:
	default:
		return rows, errors.New("no need profiling")
	}

	// 测试环境如果检查是关闭的，则 SQL 不会被执行
	if common.Config.TestDSN.Disable {
		return rows, errors.New("dsn is disable")
	}

	// 数据库安全性检查：如果 Connector 的 IP 端口与 TEST 环境不一致，则启用 SQL 白名单
	// 不在白名单中的 SQL 不允许执行
	// 执行环境与 test 环境不相同
	if db.Addr != common.Config.TestDSN.Addr && db.dangerousQuery(sql) {
		return rows, fmt.Errorf("query execution deny: Execute SQL with DSN(%s/%s) '%s'",
			db.Addr, db.Database, fmt.Sprintf(sql, params...))
	}

	common.Log.Debug("Execute SQL with DSN(%s/%s) : %s", db.Addr, db.Database, sql)
	// Keep connection
	// https://github.com/go-sql-driver/mysql/issues/208
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

	// 开启 Profiling
	_, err = trx.Query("set @@profiling=1")
	common.LogIfError(err, "")

	// 执行 SQL，抛弃返回结果
	tmpRes, err := trx.Query(sql, params...)
	if err != nil {
		return rows, err
	}
	for tmpRes.Next() {
		continue
	}

	// 返回 Profiling 结果
	res, err := trx.Query("show profile")
	if err != nil {
		trxErr := trx.Rollback()
		if trxErr != nil {
			common.Log.Debug(trxErr.Error())
		}
		return rows, err
	}
	var profileRow ProfilingRow
	for res.Next() {
		err = res.Scan(&profileRow.Status, &profileRow.Duration)
		if err != nil {
			common.LogIfError(err, "")
			break
		}
		rows = append(rows, profileRow)
	}
	res.Close()

	// 关闭 Profiling
	_, err = trx.Query("set @@profiling=0")
	common.LogIfError(err, "")
	return rows, err
}

// FormatProfiling 格式化输出 Profiling 信息
func FormatProfiling(rows []ProfilingRow) string {
	str := []string{"| Status | Duration |"}
	str = append(str, "| --- | --- |")
	for _, row := range rows {
		str = append(str, fmt.Sprintf("| %s | %f |", row.Status, row.Duration))
	}
	return strings.Join(str, "\n")
}
