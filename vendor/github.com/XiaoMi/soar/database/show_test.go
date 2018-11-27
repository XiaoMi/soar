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

	"github.com/kr/pretty"
	"vitess.io/vitess/go/vt/sqlparser"
)

func TestShowTableStatus(t *testing.T) {
	connTest.Database = "information_schema"
	ts, err := connTest.ShowTableStatus("TABLES")
	if err != nil {
		t.Error("ShowTableStatus Error: ", err)
	}
	pretty.Println(ts)
}

func TestShowTables(t *testing.T) {
	connTest.Database = "information_schema"
	ts, err := connTest.ShowTables()
	if err != nil {
		t.Error("ShowTableStatus Error: ", err)
	}
	pretty.Println(ts)
}

func TestShowCreateTable(t *testing.T) {
	connTest.Database = "information_schema"
	ts, err := connTest.ShowCreateTable("TABLES")
	if err != nil {
		t.Error("ShowCreateTable Error: ", err)
	}
	fmt.Println(ts)
	stmt, err := sqlparser.Parse(ts)
	pretty.Println(stmt, err)
}

func TestShowIndex(t *testing.T) {
	connTest.Database = "information_schema"
	ti, err := connTest.ShowIndex("TABLES")
	if err != nil {
		t.Error("ShowIndex Error: ", err)
	}
	pretty.Println(ti.FindIndex(IndexKeyName, "idx_store_id_film_id"))
}

func TestShowColumns(t *testing.T) {
	connTest.Database = "information_schema"
	ti, err := connTest.ShowColumns("TABLES")
	if err != nil {
		t.Error("ShowColumns Error: ", err)
	}
	pretty.Println(ti)
}

func TestFindColumn(t *testing.T) {
	ti, err := connTest.FindColumn("id", "")
	if err != nil {
		t.Error("FindColumn Error: ", err)
	}
	pretty.Println(ti)
}

func TestShowReference(t *testing.T) {
	rv, err := connTest.ShowReference("test2", "homeImg")
	if err != nil {
		t.Error("ShowReference Error: ", err)
	}
	pretty.Println(rv)
}

func TestIsFKey(t *testing.T) {
	if !connTest.IsFKey("sakila", "film", "language_id") {
		t.Error("want True. got false")
	}
}
