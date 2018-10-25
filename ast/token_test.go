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

func TestTokenizer(_ *testing.T) {
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
	}
	for _, sql := range sqls {
		pretty.Println(Tokenizer(sql))
	}
}

func TestGetQuotedString(t *testing.T) {
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
	}
	for _, s := range str {
		fmt.Printf("orignal: %s\nquoted: %s\n", s, getQuotedString(s))
	}
}

func TestTokenizer2(t *testing.T) {
	for _, sql := range common.TestSQLs {
		fmt.Println(sql)
		fmt.Println(Tokenize(sql))
	}
}

func TestCompress(t *testing.T) {
	for _, sql := range common.TestSQLs {
		fmt.Println(sql)
		fmt.Println(Compress(sql))
	}
}

func TestFormat(t *testing.T) {
	err := common.GoldenDiff(func() {
		for _, sql := range common.TestSQLs {
			fmt.Println(sql)
			fmt.Println(format(sql))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
}

func TestSplitStatement(t *testing.T) {
	bufs := [][]byte{
		[]byte("select * from test;hello"),
		[]byte("select 'asd;fas', col from test;hello"),
		[]byte("-- select * from test;hello"),
		[]byte("#select * from test;hello"),
		[]byte("select * /*comment*/from test;hello"),
		[]byte("select * /*comment;*/from test;hello"),
		[]byte(`select * /*comment
        ;*/
        from test;hello`),
		[]byte(`select * from test`),
	}
	for _, buf := range bufs {
		fmt.Println(SplitStatement(buf, []byte(common.Config.Delimiter)))
	}
	buf2s := [][]byte{
		[]byte("select * from test\\Ghello"),
		[]byte("select 'asd\\Gfas', col from test\\Ghello"),
		[]byte("-- select * from test\\Ghello"),
		[]byte("#select * from test\\Ghello"),
		[]byte("select * /*comment*/from test\\Ghello"),
		[]byte("select * /*comment;*/from test\\Ghello"),
		[]byte(`select * /*comment
        \\G*/
        from test\\Ghello`),
	}
	for _, buf := range buf2s {
		fmt.Println(SplitStatement(buf, []byte("\\G")))
	}
}

func TestLeftNewLines(t *testing.T) {
	bufs := [][]byte{
		[]byte(`
		select * from test;hello`),
		[]byte(`select * /*comment
        ;*/
        from test;hello`),
		[]byte(`select * from test`),
	}
	for _, buf := range bufs {
		fmt.Println(LeftNewLines(buf))
	}
}

func TestNewLines(t *testing.T) {
	bufs := [][]byte{
		[]byte(`
		select * from test;hello`),
		[]byte(`select * /*comment
        ;*/
        from test;hello`),
		[]byte(`select * from test`),
	}
	for _, buf := range bufs {
		fmt.Println(NewLines(buf))
	}
}
