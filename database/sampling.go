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
	"fmt"
	"io"
	"strconv"
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

// SamplingData 将数据从Remote拉取到 db 中
func (db *Connector) SamplingData(remote Connector, tables ...string) error {
	// 计算需要泵取的数据量
	wantRowsCount := 300 * common.Config.SamplingStatisticTarget

	// 设置数据采样单条SQL中value的数量
	// 该数值越大，在内存中缓存的data就越多，但相对的，插入时速度就越快
	maxValCount := 200

	// 获取数据库连接对象
	conn := remote.NewConnection()
	localConn := db.NewConnection()

	// 连接数据库
	err := conn.Connect()
	defer conn.Close()
	if err != nil {
		return err
	}

	err = localConn.Connect()
	defer localConn.Close()
	if err != nil {
		return err
	}

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

		err = startSampling(conn, localConn, db.Database, table, factor, wantRowsCount, maxValCount)
		if err != nil {
			common.Log.Error("(db *Connector) SamplingData Error : %v", err)
		}
	}
	return nil
}

// 开始从环境中泵取数据
// 因为涉及到的数据量问题，所以泵取与插入时同时进行的
// TODO 加 ref link
func startSampling(conn, localConn mysql.Conn, database, table string, factor float64, wants, maxValCount int) error {
	// 从线上数据库获取所需dump的表中所有列的数据类型，备用
	// 由于测试库中的库表为刚建立的，所以在information_schema中很可能没有这个表的信息
	var dataTypes []string
	q := fmt.Sprintf("select DATA_TYPE from information_schema.COLUMNS where TABLE_SCHEMA='%s' and TABLE_NAME = '%s'",
		database, table)
	common.Log.Debug("Sampling data execute: %s", q)
	rs, _, err := localConn.Query(q)
	if err != nil {
		common.Log.Debug("Sampling data got data type Err: %v", err)
	} else {
		for _, r := range rs {
			dataTypes = append(dataTypes, r.Str(0))
		}
	}

	// 生成where条件
	where := fmt.Sprintf("where RAND()<=%f", factor)
	if factor >= 1 {
		where = ""
	}

	sql := fmt.Sprintf("select * from `%s` %s limit %d;", table, where, wants)
	res, err := conn.Start(sql)
	if err != nil {
		return err
	}

	// GetRow method allocates a new chunk of memory for every received row.
	row := res.MakeRow()
	rowCount := 0
	valCount := 0

	// 获取所有的列名
	columns := make([]string, len(res.Fields()))
	for i, filed := range res.Fields() {
		columns[i] = filed.Name
	}
	colDef := strings.Join(columns, ",")

	// 开始填充数据
	var valList []string
	for {
		err := res.ScanRow(row)
		if err == io.EOF {
			// 扫描结束
			if len(valList) > 0 {
				// 如果缓存中还存在未插入的数据，则把缓存中的数据刷新到DB中
				doSampling(localConn, database, table, colDef, strings.Join(valList, ","))
			}
			break
		}

		if err != nil {
			return err
		}

		values := make([]string, len(columns))
		for i := range row {
			// TODO 不支持坐标类型的导出
			switch data := row[i].(type) {
			case nil:
				// str = ""
			case []byte:
				// 先尝试转成数字，如果报错则转换成string
				v, err := row.Int64Err(i)
				values[i] = strconv.FormatInt(v, 10)
				if err != nil {
					values[i] = string(data)
				}
			case time.Time:
				values[i] = mysql.TimeString(data)
			case time.Duration:
				values[i] = mysql.DurationString(data)
			default:
				values[i] = fmt.Sprint(data)
			}

			// 非text/varchar类的数据类型，如果dump出的数据为空，则说明该值为null值
			// 应转换其value为null，如果用空（''）进行替代，会导致出现语法错误。
			if len(dataTypes) == len(res.Fields()) && values[i] == "" &&
				(!strings.Contains(dataTypes[i], "char") ||
					!strings.Contains(dataTypes[i], "text")) {
				values[i] = "null"
			} else {
				values[i] = "'" + values[i] + "'"
			}
		}

		valuesStr := fmt.Sprintf(`(%s)`, strings.Join(values, `,`))
		valList = append(valList, valuesStr)

		rowCount++
		valCount++

		if rowCount%maxValCount == 0 {
			doSampling(localConn, database, table, colDef, strings.Join(valList, ","))
			valCount = 0
			valList = make([]string, 0)

		}
	}

	common.Log.Debug("%d rows sampling out", rowCount)
	return nil
}

// 将泵取的数据转换成Insert语句并在数据库中执行
func doSampling(conn mysql.Conn, dbName, table, colDef, values string) {
	sql := fmt.Sprintf("Insert into `%s`.`%s`(%s) values%s;", dbName, table,
		colDef, values)

	_, _, err := conn.Query(sql)

	if err != nil {
		common.Log.Error("doSampling Error from %s.%s: %v", dbName, table, err)
	}

}
