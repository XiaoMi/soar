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

package ast

import (
	"fmt"
	"testing"

	"github.com/XiaoMi/soar/common"
)

func TestPrintPrettyStmtNode(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select 1`,
		`select * f`, // syntax error case
	}
	err := common.GoldenDiff(func() {
		for _, sql := range sqls {
			PrintPrettyStmtNode(sql, "", "")
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestStmtNode2JSON(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select 1`,
		`select * f`, // syntax error case
	}
	err := common.GoldenDiff(func() {
		for _, sql := range sqls {
			fmt.Println(StmtNode2JSON(sql, "", ""))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestSchemaMetaInfo(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"use world_x;",
		"select 1;",
		"syntax error case",
		"select * from ta join tb using (id)",
		"select * from ta, tb limit 1",
		"drop table tb",
		"drop table db.tb",
		"drop database db",
		"create database db",
		"create index idx_col on tbl (col)",
		"DROP INDEX idx_col on tbl",
	}
	// fmt.Println(sqls[len(sqls)-1])
	// fmt.Println(SchemaMetaInfo(sqls[len(sqls)-1], "sakila"))
	// return
	err := common.GoldenDiff(func() {
		for _, sql := range append(sqls, common.TestSQLs...) {
			fmt.Println(sql)
			fmt.Println(SchemaMetaInfo(sql, "sakila"))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRemoveIncompatibleWords(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`CREATE TEMPORARY TABLE IF NOT EXISTS t_film AS (SELECT * FROM film)`,
			`CREATE CONSTRAINT col_fk FOREIGN KEY (col) REFERENCES tb (id) ON UPDATE CASCADE`,
			"CREATE FULLTEXT KEY col_fk (col) /*!50100 WITH PARSER `ngram` */",
			`CREATE /*!50100 PARTITION BY LIST (col)`,
			`CREATE col varchar(10) CHARACTER SET gbk DEFAULT NULL`,
		},
		{
			`CREATE TABLE IF NOT EXISTS t_film AS (SELECT * FROM film)`,
			`CREATE CONSTRAINT col_fk FOREIGN KEY (col) REFERENCES tb (id)`,
			"CREATE FULLTEXT KEY col_fk (col) /* 50100 WITH PARSER `ngram` */",
			`CREATE /* 50100 PARTITION BY LIST (col)`,
			`CREATE col varchar(10) DEFAULT NULL`,
		},
	}
	for k, sql := range sqls[0] {
		sql = removeIncompatibleWords(sql)
		if sqls[1][k] != sql {
			fmt.Println(sql)
			t.Fatal(sql)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
