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
	"strings"
	"time"

	"github.com/XiaoMi/soar/common"
	"github.com/ziutek/mymysql/mysql"
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

// SamplingData 将数据从 onlineConn 拉取到 db 中
func (db *Connector) SamplingData(onlineConn *Connector, database string, tables ...string) error {
	var err error
	if database == db.Database {
		return fmt.Errorf("SamplingData the same database, From: %s/%s, To: %s/%s", onlineConn.Addr, database, db.Addr, db.Database)
	}

	// 计算需要泵取的数据量
	wantRowsCount := 300 * common.Config.SamplingStatisticTarget

	for _, table := range tables {
		// 表类型检查
		if onlineConn.IsView(table) {
			return nil
		}

		// generate where condition
		var where string
		if common.Config.SamplingCondition == "" {
			tableStatus, err := onlineConn.ShowTableStatus(table)
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
			where = fmt.Sprintf("WHERE RAND() <= %f LIMIT %d", factor, wantRowsCount)
			if factor >= 1 {
				where = ""
			}
		} else {
			where = common.Config.SamplingCondition
		}
		err = db.startSampling(onlineConn.Conn, database, table, where)
	}
	return err
}

// startSampling sampling data from OnlineDSN to TestDSN
func (db *Connector) startSampling(onlineConn *sql.DB, database, table string, where string) error {
	samplingQuery := fmt.Sprintf("SELECT * FROM `%s`.`%s` %s", database, table, where)
	common.Log.Debug("startSampling with Query: %s", samplingQuery)
	res, err := onlineConn.Query(samplingQuery)
	if err != nil {
		return err
	}

	// columns list
	columns, err := res.Columns()
	if err != nil {
		return err
	}
	row := make([][]byte, len(columns))
	tableFields := make([]interface{}, 0)
	for i := range columns {
		tableFields = append(tableFields, &row[i])
	}
	columnTypes, err := res.ColumnTypes()
	if err != nil {
		return err
	}

	// sampling data
	var valuesCount int
	var valuesStr []string
	maxValuesCount := 200 // one time insert values count, TODO: config able
	columnsStr := "`" + strings.Join(columns, "`,`") + "`"
	for res.Next() {
		var values []string
		res.Scan(tableFields...)
		for i, val := range row {
			if val == nil {
				values = append(values, "NULL")
			} else {
				switch columnTypes[i].DatabaseTypeName() {
				case "TIMESTAMP", "DATETIME":
					t, err := time.Parse(time.RFC3339, string(val))
					common.LogIfWarn(err, "")
					values = append(values, fmt.Sprintf(`"%s"`, mysql.TimeString(t)))
				default:
					values = append(values, fmt.Sprintf(`unhex("%s")`, fmt.Sprintf("%x", val)))
				}
			}
			valuesStr = append(valuesStr, "("+strings.Join(values, `,`)+")")
			valuesCount++
			if maxValuesCount <= valuesCount {
				err = db.doSampling(table, columnsStr, strings.Join(valuesStr, `,`))
				if err != nil {
					break
				}
				values = make([]string, 0)
				valuesStr = make([]string, 0)
				valuesCount = 0
			}
		}
	}
	res.Close()
	return err
}

// 将泵取的数据转换成 insert 语句并在 testConn 数据库中执行
func (db *Connector) doSampling(table, colDef, values string) error {
	// db.Database is hashed database name
	query := fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES %s;", db.Database, table, colDef, values)
	res, err := db.Query(query)
	if res.Rows != nil {
		res.Rows.Close()
	}
	return err
}
