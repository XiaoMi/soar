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
	"testing"

	"github.com/XiaoMi/soar/common"
	"github.com/kr/pretty"
)

// TODO: go test -race不通过待解决
func TestQuery(t *testing.T) {
	common.Config.QueryTimeOut = 1
	_, err := connTest.Query("select sleep(2)")
	if err == nil {
		t.Error("connTest.Query not timeout")
	}
}

func TestColumnCardinality(_ *testing.T) {
	connTest.Database = "information_schema"
	a := connTest.ColumnCardinality("TABLES", "TABLE_SCHEMA")
	fmt.Println("TABLES.TABLE_SCHEMA:", a)
}

func TestDangerousSQL(t *testing.T) {
	testCase := map[string]bool{
		"select * from tb;delete from tb;": true,
		"show database;":                   false,
		"select * from t;":                 false,
		"explain delete from t;":           false,
	}

	db := Connector{}
	for sql, want := range testCase {
		got := db.dangerousQuery(sql)
		if got != want {
			t.Errorf("SQL:%s got:%v want:%v", sql, got, want)
		}
	}
}

func TestWarningsAndQueryCost(t *testing.T) {
	common.Config.ShowWarnings = true
	common.Config.ShowLastQueryCost = true
	res, err := connTest.Query("explain select * from sakila.film")
	if err != nil {
		t.Error("Query Error: ", err)
	} else {
		for _, w := range res.Warning {
			pretty.Println(w.Str(2))
		}
		fmt.Println(res.QueryCost)
		pretty.Println(err)
	}
}

func TestVersion(t *testing.T) {
	version, err := connTest.Version()
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(version)
}

func TestSource(t *testing.T) {
	res, err := connTest.Source("testdata/" + t.Name() + ".sql")
	if err != nil {
		t.Error("Query Error: ", err)
	}
	if res[0].Rows[0].Int(0) != 1 || res[1].Rows[0].Int(0) != 1 {
		t.Error("Source result not match, expect 1, 1")
	}
}
