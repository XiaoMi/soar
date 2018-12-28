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
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/XiaoMi/soar/common"

	"github.com/kr/pretty"
)

var connTest *Connector
var update = flag.Bool("update", false, "update .golden files")

func TestMain(m *testing.M) {
	// 初始化 init
	common.BaseDir = common.DevPath
	err := common.ParseConfig("")
	common.LogIfError(err, "init ParseConfig")
	common.Log.Debug("mysql_test init")
	connTest, err = NewConnector(common.Config.TestDSN)
	if err != nil {
		common.Log.Critical("Test env Error: %v", err)
		os.Exit(0)
	}

	if _, err := connTest.Version(); err != nil {
		common.Log.Critical("Test env Error: %v", err)
		os.Exit(0)
	}

	// 分割线
	flag.Parse()
	m.Run()

	// 环境清理
	//
}

func TestQuery(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	res, err := connTest.Query("select 0")
	if err != nil {
		t.Error(err.Error())
	}
	for res.Rows.Next() {
		var val int
		err = res.Rows.Scan(&val)
		if err != nil {
			t.Error(err.Error())
		}
		if val != 0 {
			t.Error("should return 0")
		}
	}
	res.Rows.Close()
	// TODO: timeout test
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestColumnCardinality(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgDatabase := connTest.Database
	connTest.Database = "sakila"
	a := connTest.ColumnCardinality("actor", "first_name")
	if a > 1 || a <= 0 {
		t.Error("sakila.actor.first_name cardinality should in [0, 1], now it's", a)
	}
	connTest.Database = orgDatabase
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestDangerousSQL(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestWarningsAndQueryCost(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	common.Config.ShowWarnings = true
	common.Config.ShowLastQueryCost = true
	res, err := connTest.Query("explain select * from sakila.film")
	if err != nil {
		t.Error("Query Error: ", err)
	} else {
		for res.Warning.Next() {
			var str string
			err = res.Warning.Scan(str)
			if err != nil {
				t.Error(err.Error())
			}
			pretty.Println(str)
		}
		res.Warning.Close()
		fmt.Println(res.QueryCost, err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestVersion(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	version, err := connTest.Version()
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(version)
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRemoveSQLComments(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	SQLs := []string{
		`-- comment`,
		`--`,
		`# comment`,
		`/* multi-line
comment*/`,
		`--
-- comment`,
	}

	err := common.GoldenDiff(func() {
		for _, sql := range SQLs {
			fmt.Println(RemoveSQLComments(sql))
		}
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestSingleIntValue(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	val, err := connTest.SingleIntValue("read_only")
	if err != nil {
		t.Error(err)
	}
	if val < 0 {
		t.Error("SingleIntValue, return should large than zero")
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestIsView(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	originalDatabase := connTest.Database
	connTest.Database = "sakila"
	if !connTest.IsView("actor_info") {
		t.Error("actor_info should be a VIEW")
	}
	connTest.Database = originalDatabase
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestNullString(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	cases := [][]byte{
		nil,
		[]byte("NULL"),
	}
	for _, buf := range cases {
		if NullString(buf) != "NULL" {
			t.Errorf("%s want NULL", string(buf))
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestEscape(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	cases := []string{
		"",
		"hello world",
		"hello' world",
		`hello" world`,
		"hello\000world",
		`hello\ world`,
		"hello\032world",
		"hello\rworld",
		"hello\nworld",
	}
	err := common.GoldenDiff(func() {
		for _, str := range cases {
			fmt.Println(Escape(str, false))
			fmt.Println(Escape(str, true))
		}
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
