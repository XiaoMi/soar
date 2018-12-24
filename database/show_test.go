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

func TestShowTableStatus(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgDatabase := connTest.Database
	connTest.Database = "sakila"
	ts, err := connTest.ShowTableStatus("film")
	if err != nil {
		t.Error("ShowTableStatus Error: ", err)
	}
	if string(ts.Rows[0].Engine) != "InnoDB" {
		t.Error("film table should be InnoDB engine")
	}
	pretty.Println(ts)

	connTest.Database = "sakila"
	ts, err = connTest.ShowTableStatus("actor_info")
	if err != nil {
		t.Error("ShowTableStatus Error: ", err)
	}
	if string(ts.Rows[0].Comment) != "VIEW" {
		t.Error("actor_info should be VIEW")
	}
	pretty.Println(ts)
	connTest.Database = orgDatabase
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestShowTables(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgDatabase := connTest.Database
	connTest.Database = "sakila"
	ts, err := connTest.ShowTables()
	if err != nil {
		t.Error("ShowTableStatus Error: ", err)
	}

	err = common.GoldenDiff(func() {
		for _, table := range ts {
			fmt.Println(table)
		}
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	connTest.Database = orgDatabase
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestShowCreateDatabase(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	err := common.GoldenDiff(func() {
		fmt.Println(connTest.ShowCreateDatabase("sakila"))
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestShowCreateTable(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgDatabase := connTest.Database
	connTest.Database = "sakila"
	tables := []string{
		"film",
		"category",
		"customer_list",
		"inventory",
	}
	err := common.GoldenDiff(func() {
		for _, table := range tables {
			ts, err := connTest.ShowCreateTable(table)
			if err != nil {
				t.Error("ShowCreateTable Error: ", err)
			}
			fmt.Println(ts)
		}
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	connTest.Database = orgDatabase
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestShowIndex(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgDatabase := connTest.Database
	connTest.Database = "sakila"
	ti, err := connTest.ShowIndex("film")
	if err != nil {
		t.Error("ShowIndex Error: ", err)
	}

	err = common.GoldenDiff(func() {
		pretty.Println(ti)
		pretty.Println(ti.FindIndex(IndexKeyName, "idx_title"))
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	connTest.Database = orgDatabase
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestShowColumns(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgDatabase := connTest.Database
	connTest.Database = "sakila"
	ti, err := connTest.ShowColumns("actor_info")
	if err != nil {
		t.Error("ShowColumns Error: ", err)
	}

	err = common.GoldenDiff(func() {
		pretty.Println(ti)
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}

	connTest.Database = orgDatabase
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindColumn(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	ti, err := connTest.FindColumn("film_id", "sakila", "film")
	if err != nil {
		t.Error("FindColumn Error: ", err)
	}
	err = common.GoldenDiff(func() {
		pretty.Println(ti)
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestIsFKey(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	if !connTest.IsForeignKey("sakila", "film", "language_id") {
		t.Error("want True. got false")
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestShowReference(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	rv, err := connTest.ShowReference("sakila", "film")
	if err != nil {
		t.Error("ShowReference Error: ", err)
	}

	err = common.GoldenDiff(func() {
		pretty.Println(rv)
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
