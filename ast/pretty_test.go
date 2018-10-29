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
	"flag"
	"fmt"
	"testing"

	"github.com/XiaoMi/soar/common"

	"vitess.io/vitess/go/vt/sqlparser"
)

var update = flag.Bool("update", false, "update .golden files")

var TestSqlsPretty = []string{
	"select sourcetable, if(f.lastcontent = ?, f.lastupdate, f.lastcontent) as lastactivity, f.totalcount as activity, type.class as type, (f.nodeoptions & ?) as nounsubscribe from node as f inner join contenttype as type on type.contenttypeid = f.contenttypeid inner join subscribed as sd on sd.did = f.nodeid and sd.userid = ? union all select f.name as title, f.userid as keyval, ? as sourcetable, ifnull(f.lastpost, f.joindate) as lastactivity, f.posts as activity, ? as type, ? as nounsubscribe from user as f inner join userlist as ul on ul.relationid = f.userid and ul.userid = ? where ul.type = ? and ul.aq = ? order by title limit ?",
	"administrator command: Init DB",
	"CALL foo(1, 2, 3)",
	"### Channels ###\n\u0009\u0009\u0009\u0009\u0009SELECT sourcetable, IF(f.lastcontent = 0, f.lastupdate, f.lastcontent) AS lastactivity,\n\u0009\u0009\u0009\u0009\u0009f.totalcount AS activity, type.class AS type,\n\u0009\u0009\u0009\u0009\u0009(f.nodeoptions \u0026 512) AS noUnsubscribe\n\u0009\u0009\u0009\u0009\u0009FROM node AS f\n\u0009\u0009\u0009\u0009\u0009INNER JOIN contenttype AS type ON type.contenttypeid = f.contenttypeid \n\n\u0009\u0009\u0009\u0009\u0009INNER JOIN subscribed AS sd ON sd.did = f.nodeid AND sd.userid = 15965\n UNION  ALL \n\n\u0009\u0009\u0009\u0009\u0009### Users ###\n\u0009\u0009\u0009\u0009\u0009SELECT f.name AS title, f.userid AS keyval, 'user' AS sourcetable, IFNULL(f.lastpost, f.joindate) AS lastactivity,\n\u0009\u0009\u0009\u0009\u0009f.posts as activity, 'Member' AS type,\n\u0009\u0009\u0009\u0009\u00090 AS noUnsubscribe\n\u0009\u0009\u0009\u0009\u0009FROM user AS f\n\u0009\u0009\u0009\u0009\u0009INNER JOIN userlist AS ul ON ul.relationid = f.userid AND ul.userid = 15965\n\u0009\u0009\u0009\u0009\u0009WHERE ul.type = 'f' AND ul.aq = 'yes'\n ORDER BY title ASC LIMIT 100",
	"CREATE DATABASE org235_percona345 COLLATE 'utf8_general_ci'",
	"insert into abtemp.coxed select foo.bar from foo",
	"insert into foo(a, b, c) value(2, 4, 5)",
	"insert into foo(a, b, c) values(2, 4, 5)",
	"insert into foo(a, b, c) values(2, 4, 5) , (2,4,5)",
	"insert into foo values (1, '(2)', 'This is a trick: ). More values.', 4)",
	"insert into tb values (1)",
	"INSERT INTO t (ts) VALUES ('()', '\\(', '\\)')",
	"INSERT INTO t (ts) VALUES (NOW())",
	"INSERT INTO t () VALUES ()",
	"insert into t values (1), (2), (3)\n\n\ton duplicate key update query_count=1",
	"insert into t values (1) on duplicate key update query_count=COALESCE(query_count, 0) + VALUES(query_count)",
	"LOAD DATA INFILE '/tmp/foo.txt' INTO db.tbl",
	"select 0e0, +6e-30, -6.00 from foo where a = 5.5 or b=0.5 or c=.5",
	"select 0x0, x'123', 0b1010, b'10101' from foo",
	"select 123_foo from 123_foo",
	"select 123foo from 123foo",
	`SELECT 	1 AS one FROM calls USE INDEX(index_name)`,
	"SELECT /*!40001 SQL_NO_CACHE */ * FROM `film`",
	"SELECT 'a' 'b' 'c' 'd' FROM kamil",
	"SELECT BENCHMARK(100000000, pow(rand(), rand())), 1 FROM `-hj-7d6-shdj5-7jd-kf-g988h-`.`-aaahj-7d6-shdj5-7&^%$jd-kf-g988h-9+4-5*6ab-`",
	"SELECT c FROM org235.t WHERE id=0xdeadbeaf",
	"select c from t where i=1 order by c asc",
	"SELECT c FROM t WHERE id=0xdeadbeaf",
	"SELECT c FROM t WHERE id=1",
	"select `col` from `table-1` where `id` = 5",
	"SELECT `db`.*, (CASE WHEN (`date_start` <=  '2014-09-10 09:17:59' AND `date_end` >=  '2014-09-10 09:17:59') THEN 'open' WHEN (`date_start` >  '2014-09-10 09:17:59' AND `date_end` >  '2014-09-10 09:17:59') THEN 'tbd' ELSE 'none' END) AS `status` FROM `foo` AS `db` WHERE (a_b in ('1', '10101'))",
	"select field from `-master-db-1`.`-table-1-` order by id, ?;",
	"select   foo",
	"select foo_1 from foo_2_3",
	"select foo -- bar\n",
	"select foo-- bar\n,foo",
	"select '\\\\' from foo",
	"select * from foo limit 5",
	"select * from foo limit 5, 10",
	"select * from foo limit 5 offset 10",
	"SELECT * from foo where a = 5",
	"select * from foo where a in (5) and b in (5, 8,9 ,9 , 10)",
	"SELECT '' '' '' FROM kamil",
	" select  * from\nfoo where a = 5",
	"SELECT * FROM prices.rt_5min where id=1",
	"SELECT * FROM table WHERE field = 'value' /*arbitrary/31*/ ",
	"SELECT * FROM table WHERE field = 'value' /*arbitrary31*/ ",
	"SELECT *    FROM t WHERE 1=1 AND id=1",
	"select * from t where (base.nid IN  ('1412', '1410', '1411'))",
	`select * from t where i=1      order            by
             a,  b          ASC, d    DESC,

                                    e asc`,
	"select * from t where i=1 order by a, b ASC, d DESC, e asc",
	"select 'hello'\n",
	"select 'hello', '\nhello\n', \"hello\", '\\'' from foo",
	"SELECT ID, name, parent, type FROM posts WHERE _name IN ('perf','caching') AND (type = 'page' OR type = 'attachment')",
	"SELECT name, value FROM variable",
	"select \n-- bar\n foo",
	"select null, 5.001, 5001. from foo",
	"select sleep(2) from test.n",
	"SELECT t FROM field WHERE  (entity_type = 'node') AND (entity_id IN  ('609')) AND (language IN  ('und')) AND (deleted = '0') ORDER BY delta ASC",
	"select  t.table_schema,t.table_name,engine  from information_schema.tables t  inner join information_schema.columns c  on t.table_schema=c.table_schema and t.table_name=c.table_name group by t.table_schema,t.table_name having  sum(if(column_key in ('PRI','UNI'),1,0))=0",
	"/* -- S++ SU ABORTABLE -- spd_user: rspadim */SELECT SQL_SMALL_RESULT SQL_CACHE DISTINCT centro_atividade FROM est_dia WHERE unidade_id=1001 AND item_id=67 AND item_id_red=573",
	`UPDATE groups_search SET  charter = '   -------3\'\' XXXXXXXXX.\n    \n    -----------------------------------------------------', show_in_list = 'Y' WHERE group_id='aaaaaaaa'`,
	"use `foo`",
	"select sourcetable, if(f.lastcontent = ?, f.lastupdate, f.lastcontent) as lastactivity, f.totalcount as activity, type.class as type, (f.nodeoptions & ?) as nounsubscribe from node as f inner join contenttype as type on type.contenttypeid = f.contenttypeid inner join subscribed as sd on sd.did = f.nodeid and sd.userid = ? union all select f.name as title, f.userid as keyval, ? as sourcetable, ifnull(f.lastpost, f.joindate) as lastactivity, f.posts as activity, ? as type, ? as nounsubscribe from user as f inner join userlist as ul on ul.relationid = f.userid and ul.userid = ? where ul.type = ? and ul.aq = ? order by title limit ?",
	"CREATE INDEX part_of_name ON customer (name(10));",
	"alter table `sakila`.`t1` add index `idx_col`(`col`)",
	"alter table `sakila`.`t1` add UNIQUE index `idx_col`(`col`)",
	"alter table `sakila`.`t1` add index `idx_ID`(`ID`)",

	// ADD|DROP COLUMN
	"ALTER TABLE t2 DROP COLUMN c, DROP COLUMN d;",
	"ALTER TABLE T2 ADD COLUMN C int;",
	"ALTER TABLE T2 ADD COLUMN D int FIRST;",
	"ALTER TABLE T2 ADD COLUMN E int AFTER D;",

	// RENMAE COLUMN
	"ALTER TABLE t1 RENAME COLUMN a TO b",

	// RENAME INDEX
	"ALTER TABLE t1 RENAME INDEX idx_a TO idx_b",
	"ALTER TABLE t1 RENAME KEY idx_a TO idx_b",

	// RENAME TABLE
	"ALTER TABLE db.old_table RENAME new_table;",
	"ALTER TABLE old_table RENAME TO new_table;",
	"ALTER TABLE old_table RENAME AS new_table;",

	// MODIFY & CHANGE
	"ALTER TABLE t1 MODIFY col1 BIGINT UNSIGNED DEFAULT 1 COMMENT 'my column';",
	"ALTER TABLE t1 CHANGE b a INT NOT NULL;",

	// COMMENT
	"/*!40000 select 1*/;",
}

func TestPretty(t *testing.T) {
	err := common.GoldenDiff(func() {
		for _, sql := range append(TestSqlsPretty, common.TestSQLs...) {
			fmt.Println(sql)
			fmt.Println(Pretty(sql, "builtin"))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
}

func TestIsKeyword(t *testing.T) {
	tks := map[string]bool{
		"AGAINST":        true,
		"AUTO_INCREMENT": true,
		"ADD":            true,
		"BETWEEN":        true,
		".":              false,
		"actions":        false,
		`"`:              false,
		":":              false,
	}
	for tk, v := range tks {
		if IsMysqlKeyword(tk) != v {
			t.Error("isKeyword:", tk)
		}
	}
}

func TestRemoveComments(t *testing.T) {
	for _, sql := range TestSqlsPretty {
		stmt, _ := sqlparser.Parse(sql)
		newSQL := sqlparser.String(stmt)
		if newSQL != sql {
			fmt.Print(newSQL)
		}
	}
}

func TestMysqlEscapeString(t *testing.T) {
	var strs = []map[string]string{
		{
			"input":  "abc",
			"output": "abc",
		},
		{
			"input":  "'abc",
			"output": "\\'abc",
		},
		{
			"input": `
abc`,
			"output": `\
abc`,
		},
		{
			"input":  "\"abc",
			"output": "\\\"abc",
		},
	}
	for _, str := range strs {
		output, err := MysqlEscapeString(str["input"])
		if err != nil {
			t.Error("TestMysqlEscapeString", err)
		} else {
			if output != str["output"] {
				t.Error("TestMysqlEscapeString", output, str["output"])
			}
		}
	}
}
