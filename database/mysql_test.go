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

func init() {
	common.BaseDir = common.DevPath
	common.ParseConfig("")
	connTest = &Connector{
		Addr:     common.Config.OnlineDSN.Addr,
		User:     common.Config.OnlineDSN.User,
		Pass:     common.Config.OnlineDSN.Password,
		Database: common.Config.OnlineDSN.Schema,
		Charset:  common.Config.OnlineDSN.Charset,
	}
	if _, err := connTest.Version(); err != nil {
		common.Log.Critical("Test env Error: %v", err)
		os.Exit(0)
	}
}

func TestNewConnection(t *testing.T) {
	_, err := connTest.NewConnection()
	if err != nil {
		t.Errorf("TestNewConnection, Error: %s", err.Error())
	}
}

// TODO: go test -race不通过待解决
func TestQuery(t *testing.T) {
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
	// TODO: timeout test
}

func TestColumnCardinality(t *testing.T) {
	orgDatabase := connTest.Database
	connTest.Database = "sakila"
	a := connTest.ColumnCardinality("actor", "first_name")
	if a >= 1 || a <= 0 {
		t.Error("sakila.actor.first_name cardinality should in (0, 1), now it's", a)
	}
	connTest.Database = orgDatabase
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
		for res.Warning.Next() {
			var str string
			err = res.Warning.Scan(str)
			if err != nil {
				t.Error(err.Error())
			}
			pretty.Println(str)
		}
		fmt.Println(res.QueryCost, err)
	}
}

func TestVersion(t *testing.T) {
	version, err := connTest.Version()
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(version)
}

func TestRemoveSQLComments(t *testing.T) {
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
}

func TestSingleIntValue(t *testing.T) {
	val, err := connTest.SingleIntValue("read_only")
	if err != nil {
		t.Error(err)
	}
	if val < 0 {
		t.Error("SingleIntValue, return should large than zero")
	}
}

func TestIsView(t *testing.T) {
	originalDatabase := connTest.Database
	connTest.Database = "sakila"
	if !connTest.IsView("actor_info") {
		t.Error("actor_info should be a VIEW")
	}
	connTest.Database = originalDatabase
}
