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

	"github.com/kr/pretty"
	"vitess.io/vitess/go/vt/sqlparser"
)

var update = flag.Bool("update", false, "update .golden files")

func TestMain(m *testing.M) {
	// 初始化 init
	common.BaseDir = common.DevPath
	err := common.ParseConfig("")
	common.LogIfError(err, "init ParseConfig")
	common.Log.Debug("ast_test init")

	// 分割线
	flag.Parse()
	m.Run()

	// 环境清理
	//
}

func TestGetTableFromExprs(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestGetParseTableWithStmt(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindCondition(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	for _, sql := range common.TestSQLs {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		// pretty.Println(stmt)
		if err != nil {
			panic(err)
		}
		eq := FindEQColsInWhere(stmt)
		inEq := FindINEQColsInWhere(stmt)
		fmt.Println("WhereEQ:")
		pretty.Println(eq)
		fmt.Println("WhereINEQ:")
		pretty.Println(inEq)
		fmt.Println()
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindGroupBy(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindOrderBy(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindSubquery(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindJoinTable(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
		// pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		joinMeta := FindJoinTable(stmt, nil)
		pretty.Println(joinMeta)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindJoinCols(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
		// pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindJoinCols(stmt)
		pretty.Println(columns)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindJoinColBeWhereEQ(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
		// pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindEQColsInJoinCond(stmt)
		pretty.Println(columns)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindJoinColBeWhereINEQ(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
		// pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindINEQColsInJoinCond(stmt)
		pretty.Println(columns)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindAllCondition(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
		// pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindAllCondition(stmt)
		pretty.Println(columns)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindColumn(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqlList := []string{
		"select col, col2, sum(col1) from tb group by col",
		"select col from tb group by col,sum(col1)",
		"select col, sum(col1) from tb group by col",
	}
	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		// pretty.Println(stmt)
		if err != nil {
			panic(err)
		}

		columns := FindColumn(stmt)
		pretty.Println(columns)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindAllCols(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqlList := []string{
		"select * from tb where a = '1' order by c",
		"select * from tb where a = '1' group by c",
		"select * from tb where c = '1' group by a",
		"select * from tb join tb2 on c = c where c = '1' group by a",
	}

	targets := []Expression{
		OrderByExpression,
		GroupByExpression,
		WhereExpression,
		JoinExpression,
	}

	for i, sql := range sqlList {
		stmt, err := sqlparser.Parse(sql)
		if err != nil {
			t.Error(err)
			return
		}

		columns := FindAllCols(stmt, targets[i])
		if columns[0].Name != "c" {
			fmt.Println(sql)
			t.Error(fmt.Errorf("want 'c' got %v", columns))
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestGetSubqueryDepth(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
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
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestAppendTable(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqlList := []string{
		"select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;",
	}

	meta := make(map[string]*common.DB)
	for _, sql := range sqlList {
		fmt.Println(sql)
		stmt, err := sqlparser.Parse(sql)
		if err != nil {
			t.Error("syntax check error.")
		}

		err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch expr := node.(type) {
			case *sqlparser.AliasedTableExpr:
				switch table := expr.Expr.(type) {
				case sqlparser.TableName:
					appendTable(table, expr.As.String(), meta)
				default:
					if meta == nil {
						meta = make(map[string]*common.DB)
					}
					if meta[""] == nil {
						meta[""] = common.NewDB("")
					}
					meta[""].Table[""] = common.NewTable("")
					meta[""].Table[""].TableAliases = append(meta[""].Table[""].TableAliases, expr.As.String())
				}
			}
			return true, nil
		}, stmt)

		if err != nil {
			t.Error(err)
		}
	}

	// 仅对第一条测试SQL进行测试，验证别名正确性
	if meta[""].Table["customer_list"].TableAliases[0] != "l" || meta[""].Table["city"].TableAliases[0] != "c" {
		t.Error("alias filed\n", pretty.Sprint(meta))
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
