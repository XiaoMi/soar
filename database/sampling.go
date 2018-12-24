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
	"database/sql"
	"fmt"

	"github.com/XiaoMi/soar/common"
	"strings"
)

/*--------------------
* The following choice of minrows is based on the paper
* "Random sampling for histogram construction: how much is enough?"
* by Surajit Chaudhuri, Rajeev Motwani and Vivek Narasayya, in
* Proceedings of ACM SIGMOD International Conference on Management
* of Data, 1998, Pages 436-447.  Their Corollary 1 to Theorem 5
* says that for table size n, histogram size k, maximum relative
* error in bin size f, and error probability gamma, the minimum
* random sample size is
*      r = 4 * k * ln(2*n/gamma) / f^2
* Taking f = 0.5, gamma = 0.01, n = 10^6 rows, we obtain
*      r = 305.82 * k
* Note that because of the log function, the dependence on n is
* quite weak; even at n = 10^12, a 300*k sample gives <= 0.66
* bin size error with probability 0.99.  So there's no real need to
* scale for n, which is a good thing because we don't necessarily
* know it at this point.
*--------------------
 */

// SamplingData 将数据从Remote拉取到 db 中
func (db *Connector) SamplingData(remote *Connector, tables ...string) error {
	// 计算需要泵取的数据量
	wantRowsCount := 300 * common.Config.SamplingStatisticTarget

	// 设置数据采样单条 SQL 中 value 的数量
	// 该数值越大，在内存中缓存的data就越多，但相对的，插入时速度就越快
	maxValCount := 200

	for _, table := range tables {
		// 表类型检查
		if remote.IsView(table) {
			return nil
		}

		tableStatus, err := remote.ShowTableStatus(table)
		if err != nil {
			return err
		}

		if len(tableStatus.Rows) == 0 {
			common.Log.Info("SamplingData, Table %s with no data, stop sampling", table)
			return nil
		}

		tableRows := tableStatus.Rows[0].Rows
		if tableRows == 0 {
			common.Log.Info("SamplingData, Table %s with no data, stop sampling", table)
			return nil
		}

		factor := float64(wantRowsCount) / float64(tableRows)
		common.Log.Debug("SamplingData, tableRows: %d, wantRowsCount: %d, factor: %f", tableRows, wantRowsCount, factor)

		err = startSampling(remote.Conn, db.Conn, db.Database, table, factor, wantRowsCount, maxValCount)
		if err != nil {
			common.Log.Error("(db *Connector) SamplingData Error : %v", err)
		}
	}
	return nil
}

// startSampling sampling data from OnlineDSN to TestDSN
// 因为涉及到的数据量问题，所以泵取与插入时同时进行的
// TODO: 加 ref link
func startSampling(conn, localConn *sql.DB, database, table string, factor float64, wants, maxValCount int) error {
	// generate where condition
	where := fmt.Sprintf("WHERE RAND() <= %f", factor)
	if factor >= 1 {
		where = ""
	}

	res, err := conn.Query(fmt.Sprintf("SELECT * FROM `%s`.`%s` %s LIMIT %d;", database, table, where, wants))
	if err != nil {
		return err
	}

	// column info
	columns, err := res.Columns()
	if err != nil {
		return err
	}
	row := make(map[string][]byte, len(columns))
	tableFields := make([]interface{}, 0)
	for _, col := range columns {
		if _, ok := row[col]; ok {
			tableFields = append(tableFields, row[col])
		}
	}

	// sampling data
	var valuesStr string
	var values []string
	columnsStr := "`" + strings.Join(columns, "`,`") + "`"
	for res.Next() {
		res.Scan(tableFields...)
		for _, val := range row {
			values = append(values, fmt.Sprintf(`unhex("%s")`, fmt.Sprintf("%x", val)))
		}
		valuesStr = fmt.Sprintf(`(%s)`, strings.Join(values, `,`))
		doSampling(localConn, database, table, columnsStr, valuesStr)
	}
	res.Close()
	return nil
}

// 将泵取的数据转换成Insert语句并在数据库中执行
func doSampling(conn *sql.DB, dbName, table, colDef, values string) {
	query := fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES %s;", dbName, table,
		colDef, values)

	_, err := conn.Exec(query)
	if err != nil {
		common.Log.Error("doSampling Error from %s.%s: %v", dbName, table, err)
	}
}
