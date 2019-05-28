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

	"github.com/kr/pretty"
)

func TestTokenize(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	err := common.GoldenDiff(func() {
		for _, sql := range common.TestSQLs {
			fmt.Println(sql)
			fmt.Println(Tokenize(sql))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestTokenizer(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select c1,c2,c3 from t1,t2 join t3 on t1.c1=t2.c1 and t1.c3=t3.c1 where id>1000",
		"select sourcetable, if(f.lastcontent = ?, f.lastupdate, f.lastcontent) as lastactivity, f.totalcount as activity, type.class as type, (f.nodeoptions & ?) as nounsubscribe from node as f inner join contenttype as type on type.contenttypeid = f.contenttypeid inner join subscribed as sd on sd.did = f.nodeid and sd.userid = ? union all select f.name as title, f.userid as keyval, ? as sourcetable, ifnull(f.lastpost, f.joindate) as lastactivity, f.posts as activity, ? as type, ? as nounsubscribe from user as f inner join userlist as ul on ul.relationid = f.userid and ul.userid = ? where ul.type = ? and ul.aq = ? order by title limit ?",
		"select c1 from t1 where id>=1000", // test ">="
		"select SQL_CALC_FOUND_ROWS col from tbl where id>1000",
		"SELECT * FROM tb WHERE id=?;",
		"SELECT * FROM tb WHERE id is null;",
		"SELECT * FROM tb WHERE id is not null;",
		"SELECT * FROM tb WHERE id between 1 and 3;",
		"alter table inventory add index idx_store_film` (`store_id`,`film_id`);",
		`UPDATE xxx SET c1=' LOGGER.error(""); }' WHERE id = 2 ;`,
	}
	err := common.GoldenDiff(func() {
		for _, sql := range sqls {
			pretty.Println(Tokenizer(sql))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestGetQuotedString(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	var str = []string{
		`"hello world"`,
		"`hello world`",
		`'hello world'`,
		"hello world",
		`'hello \'world'`,
		`"hello \"wor\"ld"`,
		`"hello \"world"`,
		`""`,
		`''`,
		"``",
		`'hello 'world'`,
		`"hello "world"`,
		`' LOGGER.error(""); }'`,
	}
	err := common.GoldenDiff(func() {
		for _, s := range str {
			fmt.Printf("orignal: %s\nquoted: %s\n", s, getQuotedString(s))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestCompress(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	err := common.GoldenDiff(func() {
		for _, sql := range common.TestSQLs {
			fmt.Println(sql)
			fmt.Println(Compress(sql))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFormat(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	err := common.GoldenDiff(func() {
		for _, sql := range common.TestSQLs {
			fmt.Println(sql)
			fmt.Println(format(sql))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestSplitStatement(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	bufs := [][]byte{
		[]byte("select * from test;hello"),              // 0
		[]byte("select 'asd;fas', col from test;hello"), // 1
		[]byte("-- select * from test;hello"),           // 2
		[]byte("#select * from test;hello"),             // 3
		[]byte("select * /*comment*/from test;hello"),   // 4
		[]byte("select * /*comment;*/from test;hello"),  // 5
		[]byte(`select * /*comment
		;*/
		from test;hello`), // 6
		[]byte(`select * from test`), // 7
		// https://github.com/XiaoMi/soar/issues/66
		[]byte(`/*comment*/`),  // 8
		[]byte(`/*comment*/;`), // 9
		[]byte(`--`),           // 10
		[]byte(`-- comment`),   // 11
		[]byte(`# comment`),    // 12
		// https://github.com/XiaoMi/soar/issues/116
		[]byte(`select
*
-- comment
from tb
where col = 1`), // 13
		[]byte(`select
* --
from tb
where col = 1`), // 14
		[]byte(`select
* #
from tb
where col = 1`), // 15
		[]byte(`select
*
--
from tb
where col = 1`), // 16
		[]byte(`select * from
-- comment
tb;
select col from tb where col = 1;`), // 17
		// https://github.com/XiaoMi/soar/issues/120
		[]byte(`
-- comment
select col from tb;
select col from tb;
`), // 18
		[]byte(`INSERT /*+ SET_VAR(foreign_key_checks=OFF) */ INTO t2 VALUES(2);`), // 19
		[]byte(`select /*!50000 1,*/ 1;`),                                          // 20
		[]byte(`UPDATE xxx SET c1=' LOGGER.error(""); }' WHERE id = 2 ;`),          // 21
		[]byte("UPDATE `xxx` SET aaa='a;' WHERE `id` = 15;"),                       // 22
	}
	// \G 分隔符
	buf2s := [][]byte{
		[]byte("select * from test\\Ghello"),                    // 0
		[]byte("select 'hello\\Gworld', col from test\\Ghello"), // 1
		[]byte("-- select * from test\\Ghello"),                 // 2
		[]byte("#select * from test\\Ghello"),                   // 3
		[]byte("select * /*comment*/from test\\Ghello"),         // 4
		[]byte("select * /*comment;*/from test\\Ghello"),        // 5
		[]byte(`select * /*comment
        \\G*/
        from test\\Ghello`), // 6
	}
	// single sql test
	// SplitStatement(bufs[22], []byte(";"))
	// return
	err := common.GoldenDiff(func() {
		for i, buf := range bufs {
			sql, _, _ := SplitStatement(buf, []byte(";"))
			fmt.Println(i, sql)
		}

		for i, buf := range buf2s {
			sql, _, _ := SplitStatement(buf, []byte("\\G"))
			fmt.Println(i, sql)
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestLeftNewLines(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	bufs := [][]byte{
		[]byte(`
		select * from test;hello`),
		[]byte(`select * /*comment
        ;*/
        from test;hello`),
		[]byte(`select * from test`),
	}
	err := common.GoldenDiff(func() {
		for _, buf := range bufs {
			fmt.Println(LeftNewLines(buf))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestNewLines(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	bufs := [][]byte{
		[]byte(`
		select * from test;hello`),
		[]byte(`select * /*comment
        ;*/
        from test;hello`),
		[]byte(`select * from test`),
	}
	err := common.GoldenDiff(func() {
		for _, buf := range bufs {
			fmt.Println(NewLines(buf))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestQueryType(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	var testSQLs = []string{
		` select 1`,
		`/*comment*/ select 1`,
		`(select 1)`,
		`grant select on *.* to user@'localhost'`,
		`REVOKE INSERT ON *.* FROM 'jeffrey'@'localhost';`,
	}
	// fmt.Println(testSQLs[len(testSQLs)-1])
	// fmt.Println(QueryType(testSQLs[len(testSQLs)-1]))
	// return
	err := common.GoldenDiff(func() {
		for _, buf := range append(testSQLs, common.TestSQLs...) {
			fmt.Println(QueryType(buf))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
