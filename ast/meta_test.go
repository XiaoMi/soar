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
	"vitess.io/vitess/go/vt/sqlparser"
)

func TestGetTableFromExprs(t *testing.T) {
	tbExprs := sqlparser.TableExprs{
		&sqlparser.AliasedTableExpr{
			Expr: sqlparser.TableName{
				Name:      sqlparser.NewTableIdent("table"),
				Qualifier: sqlparser.NewTableIdent("db"),
			},
			As: sqlparser.NewTableIdent("as"),
		},
	}
	meta := GetTableFromExprs(tbExprs)
	if tb, ok := meta["db"]; !ok {
		t.Errorf("no table qualifier, meta: %s", pretty.Sprint(tb))
	}
}

func TestGetParseTableWithStmt(t *testing.T) {
	for _, sql := range common.TestSQLs {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		if err != nil {
			t.Errorf("SQL Parsed error: %v", err)
		}
		meta := GetMeta(stmt, nil)
		pretty.Println(meta)
		fmt.Println()
	}
}

func TestFindCondition(t *testing.T) {
	for _, sql := range common.TestSQLs {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		//pretty.Println(stmt)
		if err != nil {
			panic(err)
		}
		eq := FindEQColsInWhere(stmt)
		inEq := FindINEQColsInWhere(stmt)
		fmt.Println("WherEQ:")
		pretty.Println(eq)
		fmt.Println("WherINEQ:")
		pretty.Println(inEq)
		fmt.Println()
	}
}

func TestFindGroupBy(t *testing.T) {
	sqlList := []string{
		"select a from t group by c",
	}

	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		if err != nil {
			panic(err)
		}
		res := FindGroupByCols(stmt)
		pretty.Println(res)
		fmt.Println()
	}
}

func TestFindOrderBy(t *testing.T) {
	sqlList := []string{
		"select a from t group by c order by d, c desc",
		"select a from t group by c order by d desc",
	}

	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		if err != nil {
			panic(err)
		}
		res := FindOrderByCols(stmt)
		pretty.Println(res)
		fmt.Println()
	}
}

func TestFindSubquery(t *testing.T) {
	sqlList := []string{
		"SELECT * FROM t1 WHERE column1 = (SELECT column1 FROM (SELECT column1 FROM t2) a);",
		"select column1 from t2",
		"SELECT * FROM t1 WHERE column1 = (SELECT column1 FROM t2);",
		"select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;",
	}

	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		if err != nil {
			panic(err)
		}

		subquery := FindSubquery(0, stmt)
		fmt.Println(len(subquery))
		pretty.Println(subquery)
	}

}

func TestFindJoinTable(t *testing.T) {
	sqlList := []string{
		"SELECT * FROM t1 LEFT JOIN (t2 CROSS JOIN t3 CROSS JOIN t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;",
		"SELECT * FROM t1 LEFT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT * FROM t1 RIGHT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT left_tbl.* FROM left_tbl LEFT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
		"SELECT left_tbl.* FROM left_tbl RIGHT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
	}

	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		//pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		joinMeta := FindJoinTable(stmt, nil)
		pretty.Println(joinMeta)
	}
}

func TestFindJoinCols(t *testing.T) {
	sqlList := []string{
		"SELECT * FROM t1 LEFT JOIN (t2 CROSS JOIN t3 CROSS JOIN t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"select t from a LEFT JOIN b USING (c1, c2, c3)",
		"select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;",
		"SELECT * FROM t1 LEFT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT * FROM t1 RIGHT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT left_tbl.* FROM left_tbl LEFT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
		"SELECT left_tbl.* FROM left_tbl RIGHT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
	}

	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		//pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindJoinCols(stmt)
		pretty.Println(columns)
	}
}

func TestFindJoinColBeWhereEQ(t *testing.T) {
	sqlList := []string{
		"select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;",
		"SELECT * FROM t1 LEFT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT * FROM t1 RIGHT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT left_tbl.* FROM left_tbl LEFT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
		"SELECT left_tbl.* FROM left_tbl RIGHT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
	}

	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		//pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindEQColsInJoinCond(stmt)
		pretty.Println(columns)
	}
}

func TestFindJoinColBeWhereINEQ(t *testing.T) {
	sqlList := []string{
		"select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;",
		"SELECT * FROM t1 LEFT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT * FROM t1 RIGHT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b > 'b' AND t4.c = t1.c)",
		"SELECT left_tbl.* FROM left_tbl LEFT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
		"SELECT left_tbl.* FROM left_tbl RIGHT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
	}

	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		//pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindINEQColsInJoinCond(stmt)
		pretty.Println(columns)
	}
}

func TestFindAllCondition(t *testing.T) {
	sqlList := []string{
		"SELECT * FROM t1 LEFT JOIN (t2 CROSS JOIN t3 CROSS JOIN t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"select t from a LEFT JOIN b USING (c1, c2, c3)",
		"select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;",
		"SELECT * FROM t1 LEFT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT * FROM t1 RIGHT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT left_tbl.* FROM left_tbl LEFT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
		"SELECT left_tbl.* FROM left_tbl RIGHT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
		"SELECT * FROM t1 where a in ('a','b')",
		"SELECT * FROM t1 where a BETWEEN 'bar' AND 'foo'",
		"SELECT * FROM t1 where a = sum(a,b)",
		"SELECT distinct a FROM t1 where a = '2001-01-01 01:01:01'",
	}

	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		//pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindAllCondition(stmt)
		pretty.Println(columns)
	}
}

func TestFindColumn(t *testing.T) {
	sqlList := []string{
		"select col, col2, sum(col1) from tb group by col",
		"select col from tb group by col,sum(col1)",
		"select col, sum(col1) from tb group by col",
	}
	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		//pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindColumn(stmt)
		pretty.Println(columns)
	}
}

func TestFindAllCols(t *testing.T) {
	sqlList := []string{
		"SELECT * FROM t1 LEFT JOIN (t2 CROSS JOIN t3 CROSS JOIN t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"select t from a LEFT JOIN b USING (c1, c2, c3)",
		"select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;",
		"SELECT * FROM t1 LEFT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT * FROM t1 RIGHT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT left_tbl.* FROM left_tbl LEFT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
		"SELECT left_tbl.* FROM left_tbl RIGHT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
		"SELECT * FROM t1 where a in ('a','b')",
		"SELECT * FROM t1 where a BETWEEN 'bar' AND 'foo'",
		"SELECT * FROM t1 where a = sum(a,b)",
		"SELECT distinct a FROM t1 where a = '2001-01-01 01:01:01'",
	}

	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		//pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindAllCols(stmt, "order by")
		pretty.Println(columns)
	}
}

func TestGetSubqueryDepth(t *testing.T) {
	sqlList := []string{
		"SELECT * FROM t1 LEFT JOIN (t2 CROSS JOIN t3 CROSS JOIN t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"select t from a LEFT JOIN b USING (c1, c2, c3)",
		"select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;",
		"SELECT * FROM t1 LEFT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT * FROM t1 RIGHT JOIN (t2, t3, t4) ON (t2.a = t1.a AND t3.b = t1.b AND t4.c = t1.c)",
		"SELECT left_tbl.* FROM left_tbl LEFT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
		"SELECT left_tbl.* FROM left_tbl RIGHT JOIN right_tbl ON left_tbl.id = right_tbl.id WHERE right_tbl.id IS NULL;",
		"SELECT * FROM t1 where a in ('a','b')",
		"SELECT * FROM t1 where a BETWEEN 'bar' AND 'foo'",
		"SELECT * FROM t1 where a = sum(a,b)",
		"SELECT distinct a FROM t1 where a = '2001-01-01 01:01:01'",
	}

	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		if err != nil {
			t.Error("syntax check error.")
		}

		dep := GetSubqueryDepth(stmt)
		fmt.Println(dep)
	}
}
