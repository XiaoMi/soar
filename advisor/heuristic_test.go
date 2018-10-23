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

package advisor

import (
	"errors"
	"sort"
	"testing"

	"github.com/XiaoMi/soar/common"

	"github.com/kr/pretty"
)

// ALI.001
func TestRuleImplicitAlias(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"select col c from tbl where id < 1000",
			"select col from tbl tb where id < 1000",
		},
		{
			"do 1",
		},
	}
	for _, sql := range sqls[0] {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleImplicitAlias()
		if rule.Item != "ALI.001" {
			t.Error("Rule not match:", rule.Item, "Expect : ALI.001")
		}
	}
	for _, sql := range sqls[1] {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleImplicitAlias()
		if rule.Item != "OK" {
			t.Error("Rule not match:", rule.Item, "Expect : OK")
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ALI.002
func TestRuleStarAlias(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select tbl.* as c1,c2,c3 from tbl where id < 1000",
	}
	for _, sql := range sqls {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleStarAlias()
		if rule.Item != "ALI.002" {
			t.Error("Rule not match:", rule.Item, "Expect : ALI.002")
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ALI.003
func TestRuleSameAlias(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select col as col from tbl where id < 1000",
		"select col from tbl as tbl where id < 1000",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSameAlias()
			if rule.Item != "ALI.003" {
				t.Error("Rule not match:", rule.Item, "Expect : ALI.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.001
func TestRulePrefixLike(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select col from tbl where id like '%abc'",
		"select col from tbl where id like '_abc'",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RulePrefixLike()
			if rule.Item != "ARG.001" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.002
func TestRuleEqualLike(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select col from tbl where id like 'abc'",
		"select col from tbl where id like 1",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleEqualLike()
			if rule.Item != "ARG.002" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.001
func TestRuleNoWhere(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{"select col from tbl",
			"delete from tbl",
			"update tbl set col=1",
			"insert into city (country_id) select country_id from country",
		},
		{
			`select 1;`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNoWhere()
			if rule.Item != "CLA.001" && rule.Item != "CLA.014" && rule.Item != "CLA.015" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.001/CLA.014/CLA.015")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNoWhere()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.002
func TestRuleOrderByRand(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select col from tbl where id = 1 order by rand()",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleOrderByRand()
			if rule.Item != "CLA.002" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.003
func TestRuleOffsetLimit(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select c1,c2 from tbl where name=xx order by number limit 1 offset 2000",
		"select c1,c2 from tbl where name=xx order by number limit 2000,1",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleOffsetLimit()
			if rule.Item != "CLA.003" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.004
func TestRuleGroupByConst(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select col1,col2 from tbl where col1='abc' group by 1",
		"select col1,col2 from tbl group by 1",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleGroupByConst()
			if rule.Item != "CLA.004" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.005
func TestRuleOrderByConst(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		// "select id from test where id=1 order by id",
		"select id from test where id=1 order by 1",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleOrderByConst()
			if rule.Item != "CLA.005" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.005")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.006
func TestRuleDiffGroupByOrderBy(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select tb1.col, tb2.col from tb1, tb2 where id=1 group by tb1.col, tb2.col",
		"select tb1.col, tb2.col from tb1, tb2 where id=1 order by tb1.col, tb2.col",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleDiffGroupByOrderBy()
			if rule.Item != "CLA.006" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.006")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.007
func TestRuleMixOrderBy(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select c1,c2,c3 from t1 where c1='foo' order by c2 desc, c3 asc",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMixOrderBy()
			if rule.Item != "CLA.007" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.007")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.008
func TestRuleExplicitOrderBy(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select c1,c2,c3 from t1 where c1='foo' group by c2",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleExplicitOrderBy()
			if rule.Item != "CLA.008" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.008")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.009
func TestRuleOrderByExpr(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"SELECT col FROM tbl order by cola - colb;",              // order by 列运算
		"SELECT cola - colb col FROM tbl order by col;",          // 别名为列运算
		"SELECT cola FROM tbl order by from_unixtime(col);",      // order by 函数运算
		"SELECT from_unixtime(col) cola FROM tbl order by cola;", // 别名为函数运算

		// 反面例子
		// `SELECT tbl.col FROM tbl ORDER BY col`,
		// "SELECT sum(col) AS col FROM tbl ORDER BY dt",
		// "SELECT tbl.col FROM tb, tbl WHERE tbl.tag_id = tb.id ORDER BY tbl.col",
		// "SELECT col FROM tbl order by `timestamp`;", // 列名为关键字
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleOrderByExpr()
			if rule.Item != "CLA.009" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.009")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.010
func TestRuleGroupByExpr(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"SELECT col FROM tbl GROUP by cola - colb;",
		"SELECT cola - colb col FROM tbl GROUP by col;",
		"SELECT cola FROM tbl GROUP by from_unixtime(col);",
		"SELECT from_unixtime(col) cola FROM tbl GROUP by cola;",

		// 反面例子
		// `SELECT tbl.col FROM tbl GROUP BY col`,
		// "SELECT dt, sum(col) AS col FROM tbl GROUP BY dt",
		// "SELECT tbl.col FROM tb, tbl WHERE tbl.tag_id = tb.id GROUP BY tbl.col",
		// "SELECT col FROM tbl GROUP by `timestamp`;", // 列名为关键字
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleGroupByExpr()
			if rule.Item != "CLA.010" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.010")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.011
func TestRuleTblCommentCheck(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"CREATE TABLE `test1`( `ID` bigint(20) NOT NULL AUTO_INCREMENT," +
			" `c1` varchar(128) DEFAULT NULL, `c2` varchar(300) DEFAULT NULL," +
			" `c3` varchar(32) DEFAULT NULL, `c4` int(11) NOT NULL, `c5` double NOT NULL," +
			" `c6` text NOT NULL, PRIMARY KEY (`ID`), KEY `idx_c3_c2_c4_c5_c6` " +
			"(`c3`,`c2`(255),`c4`,`c5`,`c6`(255)), KEY `idx_c3_c2_c4` (`c3`,`c2`,`c4`)) " +
			"ENGINE=InnoDB DEFAULT CHARSET=utf8",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTblCommentCheck()
			if rule.Item != "CLA.011" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.011")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.001
func TestRuleSelectStar(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select * from tbl where id=1",
		"select col, * from tbl where id=1",
		// 反面例子
		// "select count(*) from film where id=1",
		// `select count(* ) from film where id=1`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSelectStar()
			if rule.Item != "COL.001" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.002
func TestRuleInsertColDef(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"insert into tbl values(1,'name')",
		"replace into tbl values(1,'name')",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleInsertColDef()
			if rule.Item != "COL.002" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.004
func TestRuleAddDefaultValue(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"create table test(id int)",
			`ALTER TABLE test change id id varchar(10);`,
			`ALTER TABLE test modify id varchar(10);`,
		},
		{
			`ALTER TABLE test modify id varchar(10) DEFAULT '';`,
			`ALTER TABLE test CHANGE id id varchar(10) DEFAULT '';`,
			"create table test(id int not null default 0 comment '用户id')",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAddDefaultValue()
			if rule.Item != "COL.004" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAddDefaultValue()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.005
func TestRuleColCommentCheck(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"create table test(id int not null default 0)",
			`alter table test add column a int`,
			`ALTER TABLE t1 CHANGE b b INT NOT NULL;`,
		},
		{
			"create table test(id int not null default 0 comment '用户id')",
			`alter table test add column a int comment 'test'`,
			`ALTER TABLE t1 AUTO_INCREMENT = 13;`,
			`ALTER TABLE t1 CHANGE b b INT NOT NULL COMMENT 'test';`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleColCommentCheck()
			if rule.Item != "COL.005" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.005")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleColCommentCheck()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// LIT.001
func TestRuleIPString(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"insert into tbl (IP,name) values('10.20.306.122','test')",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleIPString()
			if rule.Item != "LIT.001" {
				t.Error("Rule not match:", rule.Item, "Expect : LIT.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// LIT.002
func TestRuleDataNotQuote(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select col1,col2 from tbl where time < 2018-01-10",
		"select col1,col2 from tbl where time < 18-01-10",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleDataNotQuote()
			if rule.Item != "LIT.002" {
				t.Error("Rule not match:", rule.Item, "Expect : LIT.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KWR.001
func TestRuleSQLCalcFoundRows(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select SQL_CALC_FOUND_ROWS col from tbl where id>1000",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSQLCalcFoundRows()
			if rule.Item != "KWR.001" {
				t.Error("Rule not match:", rule.Item, "Expect : KWR.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// JOI.001
func TestRuleCommaAnsiJoin(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select c1,c2,c3 from t1,t2 join t3 on t1.c1=t2.c1 and t1.c3=t3.c1 where id>1000;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCommaAnsiJoin()
			if rule.Item != "JOI.001" {
				t.Error("Rule not match:", rule.Item, "Expect : JOI.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// JOI.002
func TestRuleDupJoin(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select tb1.col from (tb1, tb2) join tb2 on tb1.id=tb.id where tb1.id=1;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleDupJoin()
			if rule.Item != "JOI.002" {
				t.Error("Rule not match:", rule.Item, "Expect : JOI.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// RES.001
func TestRuleNoDeterministicGroupby(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		// 正面CASE
		{
			"select c1,c2,c3 from t1 where c2='foo' group by c2",
			"select col, col2, sum(col1) from tb group by col",
			"select col, col1 from tb group by col,sum(col1)",
			"select * from tb group by col",
		},

		// 反面CASE
		{
			"select id from film",
			"select col, sum(col1) from tb group by col",
			"select * from file",
			"SELECT COUNT(*) AS cnt, language_id FROM film GROUP BY language_id;",
			"SELECT COUNT(*) AS cnt FROM film GROUP BY language_id;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNoDeterministicGroupby()
			if rule.Item != "RES.001" {
				t.Error("Rule not match:", rule.Item, "Expect : RES.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNoDeterministicGroupby()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// RES.002
func TestRuleNoDeterministicLimit(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select col1,col2 from tbl where name='zhangsan' limit 10",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNoDeterministicLimit()
			if rule.Item != "RES.002" {
				t.Error("Rule not match:", rule.Item, "Expect : RES.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// RES.003
func TestRuleUpdateDeleteWithLimit(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"UPDATE film SET length = 120 WHERE title = 'abc' LIMIT 1;",
		},
		{
			"UPDATE film SET length = 120 WHERE title = 'abc';",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUpdateDeleteWithLimit()
			if rule.Item != "RES.003" {
				t.Error("Rule not match:", rule.Item, "Expect : RES.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUpdateDeleteWithLimit()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// RES.004
func TestRuleUpdateDeleteWithOrderby(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"UPDATE film SET length = 120 WHERE title = 'abc' ORDER BY title;",
		},
		{
			"UPDATE film SET length = 120 WHERE title = 'abc';",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUpdateDeleteWithOrderby()
			if rule.Item != "RES.004" {
				t.Error("Rule not match:", rule.Item, "Expect : RES.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUpdateDeleteWithOrderby()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// RES.005
func TestRuleUpdateSetAnd(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"update tbl set col = 1 and cl = 2 where col=3;",
		},
		{
			"update tbl set col = 1 ,cl = 2 where col=3;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUpdateSetAnd()
			if rule.Item != "RES.005" {
				t.Error("Rule not match:", rule.Item, "Expect : RES.005")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUpdateSetAnd()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// RES.006
func TestRuleImpossibleWhere(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"select * from tbl where 1 != 1;",
			"select * from tbl where 'a' != 'a';",
			"select * from tbl where col between 10 AND 5;",
		},
		{
			"select * from tbl where 1 = 1;",
			"select * from tbl where 'a' != 1;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleImpossibleWhere()
			if rule.Item != "RES.006" {
				t.Error("Rule not match:", rule.Item, "Expect : RES.006, SQL: ", sql)
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleImpossibleWhere()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK, SQL: ", sql)
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// RES.007
func TestRuleMeaninglessWhere(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"select * from tbl where 1 = 1;",
			"select * from tbl where 'a' = 'a';",
			"select * from tbl where 'a' != 1;",
		},
		{
			"select * from tbl where 2 = 1;",
			"select * from tbl where 'b' = 'a';",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMeaninglessWhere()
			if rule.Item != "RES.007" {
				t.Error("Rule not match:", rule.Item, "Expect : RES.007, SQL: ", sql)
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMeaninglessWhere()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK, SQL: ", sql)
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// RES.008
func TestRuleLoadFile(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"LOAD DATA INFILE 'data.txt' INTO TABLE db2.my_table;",
			"LOAD    DATA INFILE 'data.txt' INTO TABLE db2.my_table;",
			"LOAD /*COMMENT*/DATA INFILE 'data.txt' INTO TABLE db2.my_table;",
			`SELECT a,b,a+b INTO OUTFILE '/tmp/result.txt' FIELDS TERMINATED BY ',' OPTIONALLY ENCLOSED BY '"' LINES TERMINATED BY '\n' FROM test_table;`,
		},
		{
			"SELECT id, data INTO @x, @y FROM test.t1 LIMIT 1;",
		},
	}
	for _, sql := range sqls[0] {
		q := &Query4Audit{Query: sql}
		rule := q.RuleLoadFile()
		if rule.Item != "RES.008" {
			t.Error("Rule not match:", rule.Item, "Expect : RES.008, SQL: ", sql)
		}
	}

	for _, sql := range sqls[1] {
		q := &Query4Audit{Query: sql}
		rule := q.RuleLoadFile()
		if rule.Item != "OK" {
			t.Error("Rule not match:", rule.Item, "Expect : OK, SQL: ", sql)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// STA.001
func TestRuleStandardINEQ(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select col1,col2 from tbl where type!=0",
		// "select col1,col2 from tbl where type<>0",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleStandardINEQ()
			if rule.Item != "STA.001" {
				t.Error("Rule not match:", rule.Item, "Expect : STA.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KWR.002
func TestRuleUseKeyWord(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE TABLE tbl (`select` int)",
			"CREATE TABLE `select` (a int)",
			"ALTER TABLE tbl ADD COLUMN `select` varchar(10)",
		},
		{
			"CREATE TABLE tbl (a int)",
			"ALTER TABLE tbl ADD COLUMN col varchar(10)",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUseKeyWord()
			if rule.Item != "KWR.002" {
				t.Error("Rule not match:", rule.Item, "Expect : KWR.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUseKeyWord()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KWR.003
func TestRulePluralWord(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE TABLE tbl (`people` int)",
			"CREATE TABLE people (a int)",
			"ALTER TABLE tbl ADD COLUMN people varchar(10)",
		},
		{
			"CREATE TABLE tbl (`person` int)",
			"ALTER TABLE tbl ADD COLUMN person varchar(10)",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RulePluralWord()
			if rule.Item != "KWR.003" {
				t.Error("Rule not match:", rule.Item, "Expect : KWR.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RulePluralWord()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// LCK.001
func TestRuleInsertSelect(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`INSERT INTO tbl SELECT * FROM tbl2;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleInsertSelect()
			if rule.Item != "LCK.001" {
				t.Error("Rule not match:", rule.Item, "Expect : LCK.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// LCK.002
func TestRuleInsertOnDup(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`INSERT INTO t1(a,b,c) VALUES (1,2,3) ON DUPLICATE KEY UPDATE c=c+1;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleInsertOnDup()
			if rule.Item != "LCK.002" {
				t.Error("Rule not match:", rule.Item, "Expect : LCK.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// SUB.001
func TestRuleInSubquery(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select col1,col2,col3 from table1 where col2 in(select col from table2)",
		"SELECT col1,col2,col3 from table1 where col2 =(SELECT col2 FROM `table1` limit 1)",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleInSubquery()
			if rule.Item != "SUB.001" {
				t.Error("Rule not match:", rule.Item, "Expect : SUB.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// LIT.003
func TestRuleMultiValueAttribute(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select c1,c2,c3,c4 from tab1 where col_id REGEXP '[[:<:]]12[[:>:]]'",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMultiValueAttribute()
			if rule.Item != "LIT.003" {
				t.Error("Rule not match:", rule.Item, "Expect : LIT.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// LIT.003
func TestRuleAddDelimiter(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`use sakila
		select * from film`,
			`use sakila`,
			`show databases`,
		},
		{
			`use sakila;`,
		},
	}
	for _, sql := range sqls[0] {
		q, _ := NewQuery4Audit(sql)

		rule := q.RuleAddDelimiter()
		if rule.Item != "LIT.004" {
			t.Error("Rule not match:", rule.Item, "Expect : LIT.004")
		}
	}
	for _, sql := range sqls[1] {
		q, _ := NewQuery4Audit(sql)

		rule := q.RuleAddDelimiter()
		if rule.Item != "OK" {
			t.Error("Rule not match:", rule.Item, "Expect : OK")
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KEY.003
func TestRuleRecursiveDependency(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`CREATE TABLE tab2 (
          p_id BIGINT UNSIGNED NOT NULL,
          a_id BIGINT UNSIGNED NOT NULL,
          PRIMARY KEY (p_id, a_id),
          FOREIGN KEY (p_id) REFERENCES tab1(p_id),
          FOREIGN KEY (a_id) REFERENCES tab3(a_id)
         );`,
			`ALTER TABLE tbl2 add FOREIGN KEY (p_id) REFERENCES tab1(p_id);`,
		},
		{
			`ALTER TABLE tbl2 ADD KEY (p_id) p_id;`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleRecursiveDependency()
			if rule.Item != "KEY.003" {
				t.Error("Rule not match:", rule.Item, "Expect : KEY.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleRecursiveDependency()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.009
func TestRuleImpreciseDataType(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`CREATE TABLE tab2 (
          p_id BIGINT UNSIGNED NOT NULL,
          a_id BIGINT UNSIGNED NOT NULL,
          hours float NOT null,
          PRIMARY KEY (p_id, a_id)
         );`,
			`alter table tbl add column c float not null;`,
			`insert into tb (col) values (0.00001);`,
			`select * from tb where col = 0.00001;`,
		},
		{
			"REPLACE INTO `binks3` (`hostname`,`storagehost`, `filename`, `starttime`, `binlogstarttime`, `uploadname`, `binlogsize`, `filesize`, `md5`, `status`) VALUES (1, 1, 1, 1, 1, 1, ?, ?);",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleImpreciseDataType()
			if rule.Item != "COL.009" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.009")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleImpreciseDataType()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.010
func TestRuleValuesInDefinition(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`create table tab1(status ENUM('new', 'in progress', 'fixed'))`,
		`alter table tab1 add column status ENUM('new', 'in progress', 'fixed')`,
	}

	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleValuesInDefinition()
			if rule.Item != "COL.010" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.010")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KEY.004
func TestRuleIndexAttributeOrder(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`create index idx1 on tabl(last_name,first_name);`,
		`alter table tabl add index idx1 (last_name,first_name);`,
		`CREATE TABLE test (id int,blob_col BLOB, INDEX(blob_col(10),id));`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleIndexAttributeOrder()
			if rule.Item != "KEY.004" {
				t.Error("Rule not match:", rule.Item, "Expect : KEY.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.011
func TestRuleNullUsage(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select c1,c2,c3 from tabl where c4 is null or c4 <> 1;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNullUsage()
			if rule.Item != "COL.011" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.011")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// FUN.003
func TestRuleStringConcatenation(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select c1 || coalesce(' ' || c2 || ' ', ' ') || c3 as c from tabl;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleStringConcatenation()
			if rule.Item != "FUN.003" {
				t.Error("Rule not match:", rule.Item, "Expect : FUN.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// FUN.004
func TestRuleSysdate(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select sysdate();`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSysdate()
			if rule.Item != "FUN.004" {
				t.Error("Rule not match:", rule.Item, "Expect : FUN.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// FUN.005
func TestRuleCountConst(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`select count(1) from tbl;`,
			`select count(col) from tbl;`,
		},
		{
			`select count(*) from tbl`,
			`select count(DISTINCT col) from tbl`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCountConst()
			if rule.Item != "FUN.005" {
				t.Error("Rule not match:", rule.Item, "Expect : FUN.005")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCountConst()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// FUN.006
func TestRuleSumNPE(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`select sum(1) from tbl;`,
			`select sum(col) from tbl;`,
		},
		{
			`SELECT IF(ISNULL(SUM(COL)), 0, SUM(COL)) FROM tbl`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSumNPE()
			if rule.Item != "FUN.006" {
				t.Error("Rule not match:", rule.Item, "Expect : FUN.006")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSumNPE()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.007
func TestRulePatternMatchingUsage(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select c1,c2,c3,c4 from tab1 where col_id REGEXP '[[:<:]]12[[:>:]]';`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RulePatternMatchingUsage()
			if rule.Item != "ARG.007" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.007")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.012
func TestRuleSpaghettiQueryAlert(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select 1`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			common.Config.SpaghettiQueryLength = 1
			rule := q.RuleSpaghettiQueryAlert()
			if rule.Item != "CLA.012" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.012")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// JOI.005
func TestRuleReduceNumberOfJoin(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select bp1.p_id, b1.d_d as l, b1.b_id from b1 join bp1 on (b1.b_id = bp1.b_id) left outer join (b1 as b2 join bp2 on (b2.b_id = bp2.b_id)) on (bp1.p_id = bp2.p_id ) join bp21 on (b1.b_id = bp1.b_id) join bp31 on (b1.b_id = bp1.b_id) join bp41 on (b1.b_id = bp1.b_id) where b2.b_id = 0; `,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleReduceNumberOfJoin()
			if rule.Item != "JOI.005" {
				t.Error("Rule not match:", rule.Item, "Expect : JOI.005")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// DIS.001
func TestRuleDistinctUsage(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`SELECT DISTINCT c.c_id,count(DISTINCT c.c_name),count(DISTINCT c.c_e),count(DISTINCT c.c_n),count(DISTINCT c.c_me),c.c_d FROM (select distinct xing, name from B) as e WHERE e.country_id = c.country_id;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleDistinctUsage()
			if rule.Item != "DIS.001" {
				t.Error("Rule not match:", rule.Item, "Expect : DIS.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// DIS.002
func TestRuleCountDistinctMultiCol(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"SELECT COUNT(DISTINCT col, col2) FROM tbl;",
		},
		{
			"SELECT COUNT(DISTINCT col) FROM tbl;",
			`SELECT JSON_OBJECT( "key", p.id, "title", p.name, "manufacturer", p.manufacturer, "price", p.price, "specifications", JSON_OBJECTAGG(a.name, v.value)) as product FROM product as p JOIN value as v ON p.id = v.prod_id JOIN attribute as a ON a.id = v.attribute_id GROUP BY v.prod_id`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCountDistinctMultiCol()
			if rule.Item != "DIS.002" {
				t.Error("Rule not match:", rule.Item, "Expect : DIS.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCountDistinctMultiCol()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// DIS.003
// RuleDistinctStar
func TestRuleDistinctStar(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"SELECT DISTINCT * FROM film;",
			"SELECT DISTINCT film.* FROM film;",
		},
		{
			"SELECT DISTINCT col FROM film;",
			"SELECT DISTINCT film.* FROM film, tbl;",
			"SELECT DISTINCT * FROM film, tbl;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleDistinctStar()
			if rule.Item != "DIS.003" {
				t.Error("Rule not match:", rule.Item, "Expect : DIS.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleDistinctStar()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.013
func TestRuleHavingClause(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`SELECT s.c_id,count(s.c_id) FROM s where c = test GROUP BY s.c_id HAVING s.c_id <> '1660' AND s.c_id <> '2' order by s.c_id;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleHavingClause()
			if rule.Item != "CLA.013" {
				t.Error("Rule not match:", rule.Item, "Expect : CLA.013")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// CLA.017
func TestRuleForbiddenSyntax(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`create view v_today (today) AS SELECT CURRENT_DATE;`,
		`CREATE VIEW v (mycol) AS SELECT 'abc';`,
		`CREATE FUNCTION hello (s CHAR(20));`,
		`CREATE PROCEDURE simpleproc (OUT param1 INT)`,
	}
	for _, sql := range sqls {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleForbiddenSyntax()
		if rule.Item != "CLA.017" {
			t.Error("Rule not match:", rule.Item, "Expect : CLA.017")
		}

	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// JOI.006
func TestRuleNestedSubQueries(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`SELECT s,p,d FROM tabl WHERE p.p_id = (SELECT s.p_id FROM tabl WHERE s.c_id = 100996 AND s.q = 1 );`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNestedSubQueries()
			if rule.Item != "JOI.006" {
				t.Error("Rule not match:", rule.Item, "Expect : JOI.006")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// JOI.007
func TestRuleMultiDeleteUpdate(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`DELETE u FROM users u LEFT JOIN hobby tna ON u.id = tna.uid WHERE tna.hobby = 'piano'; `,
		`UPDATE users u LEFT JOIN hobby h ON u.id = h.uid SET u.name = 'pianoboy' WHERE h.hobby = 'piano';`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMultiDeleteUpdate()
			if rule.Item != "JOI.007" {
				t.Error("Rule not match:", rule.Item, "Expect : JOI.007")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// JOI.008
func TestRuleMultiDBJoin(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`SELECT s,p,d FROM db1.tb1 join db2.tb2 on db1.tb1.a = db2.tb2.a where db1.tb1.a > 10;`,
		`SELECT s,p,d FROM db1.tb1 join tb2 on db1.tb1.a = tb2.a where db1.tb1.a > 10;`,
		// `SELECT s,p,d FROM db1.tb1 join db1.tb2 on db1.tb1.a = db1.tb2.a where db1.tb1.a > 10;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMultiDBJoin()
			if rule.Item != "JOI.008" {
				t.Error("Rule not match:", rule.Item, "Expect : JOI.008")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.008
func TestRuleORUsage(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`SELECT c1,c2,c3 FROM tabl WHERE c1 = 14 OR c2 = 17;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleORUsage()
			if rule.Item != "ARG.008" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.008")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.009
func TestRuleSpaceWithQuote(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`SELECT 'a ';`,
			`SELECT ' a';`,
			`SELECT "a ";`,
			`SELECT " a";`,
		},
		{
			`select ''`,
			`select 'a'`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSpaceWithQuote()
			if rule.Item != "ARG.009" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.009")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSpaceWithQuote()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.010
func TestRuleHint(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`SELECT * FROM t1 USE INDEX (i1) ORDER BY a;`,
			`SELECT * FROM t1 IGNORE INDEX (i1) ORDER BY (i2);`,
			// TODO: vitess syntax not support now
			// `SELECT * FROM t1 USE INDEX (i1,i2) IGNORE INDEX (i2);`,
			// `SELECT * FROM t1 USE INDEX (i1) IGNORE INDEX (i2) USE INDEX (i2);`,
		},
		{
			`select ''`,
			`select 'a'`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleHint()
			if rule.Item != "ARG.010" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.010")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleHint()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.011
func TestNot(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`select id from t where num not in(1,2,3);`,
			`select id from t where num not like "a%"`,
		},
		{
			`select id from t where num in(1,2,3);`,
			`select id from t where num like "a%"`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNot()
			if rule.Item != "ARG.011" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.011")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNot()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// SUB.002
func TestRuleUNIONUsage(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select teacher_id as id,people_name as name from t1,t2 where t1.teacher_id=t2.people_id union select student_id as id,people_name as name from t1,t2 where t1.student_id=t2.people_id;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUNIONUsage()
			if rule.Item != "SUB.002" {
				t.Error("Rule not match:", rule.Item, "Expect : SUB.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// SUB.003
func TestRuleDistinctJoinUsage(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`SELECT DISTINCT c.c_id, c.c_name FROM c,e WHERE e.c_id = c.c_id;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleDistinctJoinUsage()
			if rule.Item != "SUB.003" {
				t.Error("Rule not match:", rule.Item, "Expect : SUB.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// SUB.005
func TestRuleSubQueryLimit(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`SELECT * FROM staff WHERE name IN (SELECT NAME FROM customer ORDER BY name LIMIT 1)`,
		},
		{
			`select * from (select id from tbl limit 3) as foo`,
			`select * from tbl where id in (select t.id from (select * from tbl limit 3)as t)`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSubQueryLimit()
			if rule.Item != "SUB.005" {
				t.Error("Rule not match:", rule.Item, "Expect : SUB.005")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSubQueryLimit()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// SUB.006
func TestRuleSubQueryFunctions(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`SELECT * FROM staff WHERE name IN (SELECT max(NAME) FROM customer)`,
		},
		{
			`select * from (select id from tbl limit 3) as foo`,
			`select * from tbl where id in (select t.id from (select * from tbl limit 3)as t)`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSubQueryFunctions()
			if rule.Item != "SUB.006" {
				t.Error("Rule not match:", rule.Item, "Expect : SUB.006")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSubQueryFunctions()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// SEC.002
func TestRuleReadablePasswords(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`create table test(id int,name varchar(20) not null,password varchar(200)not null);`,
		`alter table test add column password varchar(200) not null;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleReadablePasswords()
			if rule.Item != "SEC.002" {
				t.Error("Rule not match:", rule.Item, "Expect : SEC.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// SEC.003
func TestRuleDataDrop(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`delete from tb where a = b;`,
		`truncate table tb;`,
		`drop table tb;`,
		`drop database db;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleDataDrop()
			if rule.Item != "SEC.003" {
				t.Error("Rule not match:", rule.Item, "Expect : SEC.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// FUN.001
func TestCompareWithFunction(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{`select id from t where substring(name,1,3)='abc';`},
		// TODO: 右侧使用函数比较
		{`select id from t where 'abc'=substring(name,1,3);`},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCompareWithFunction()
			if rule.Item != "FUN.001" {
				t.Error("Rule not match:", rule.Item, "Expect : FUN.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCompareWithFunction()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// FUN.002
func TestRuleCountStar(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`SELECT c3, COUNT(*) AS accounts FROM tab where c2 < 10000 GROUP BY c3 ORDER BY num;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCountStar()
			if rule.Item != "FUN.002" {
				t.Error("Rule not match:", rule.Item, "Expect : FUN.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// SEC.001
func TestRuleTruncateTable(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`TRUNCATE TABLE tbl_name;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTruncateTable()
			if rule.Item != "SEC.001" {
				t.Error("Rule not match:", rule.Item, "Expect : SEC.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.005
func TestRuleIn(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select id from t where num in(1,2,3);`,
		`SELECT * FROM tbl WHERE col IN (NULL)`,
		`SELECT * FROM tbl WHERE col NOT IN (NULL)`,
	}
	common.Config.MaxInCount = 0
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleIn()
			if rule.Item != "ARG.005" && rule.Item != "ARG.004" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.005 OR ARG.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.006
func TestRuleisNullIsNotNull(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`select id from t where num is null;`,
		`select id from t where num is not null;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleIsNullIsNotNull()
			if rule.Item != "ARG.006" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.006")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.008
func TestRuleVarcharVSChar(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`create table t1(id int,name char(20),last_time date);`,
		`create table t1(id int,name binary(20),last_time date);`,
		`alter table t1 add column id int, add column name binary(20), add column last_time date;`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleVarcharVSChar()
			if rule.Item != "COL.008" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.008")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// TBL.003
func TestRuleCreateDualTable(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`create table dual(id int, primary key (id));`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCreateDualTable()
			if rule.Item != "TBL.003" {
				t.Error("Rule not match:", rule.Item, "Expect : TBL.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ALT.001
func TestRuleAlterCharset(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`alter table tbl default character set 'utf8';`,
			`alter table tbl default character set='utf8';`,
			`ALTER TABLE t1 CHANGE a b BIGINT NOT NULL, default character set utf8`,
			`ALTER TABLE t1 CHANGE a b BIGINT NOT NULL,default character set utf8`,
			`ALTER TABLE tbl_name CHARACTER SET charset_name;`,
			`ALTER TABLE t1 CHANGE a b BIGINT NOT NULL, character set utf8`,
			`ALTER TABLE t1 CHANGE a b BIGINT NOT NULL,character set utf8`,
			`alter table t1 convert to character set utf8 collate utf8_unicode_ci;`,
			`alter table t1 default collate = utf8_unicode_ci;`,
		},
		{
			// 反面的例子
			`ALTER TABLE t MODIFY latin1_text_col TEXT CHARACTER SET utf8`,
			`ALTER TABLE t1 CHANGE c1 c1 TEXT CHARACTER SET utf8;`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAlterCharset()
			if rule.Item != "ALT.001" {
				t.Error(sql, " Rule not match:", rule.Item, "Expect : ALT.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAlterCharset()
			if rule.Item != "OK" {
				t.Error(sql, " Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ALT.003
func TestRuleAlterDropColumn(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`alter table film drop column title;`,
		},
		{
			// 反面的例子
			`ALTER TABLE t1 CHANGE c1 c1 TEXT CHARACTER SET utf8;`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAlterDropColumn()
			if rule.Item != "ALT.003" {
				t.Error(sql, " Rule not match:", rule.Item, "Expect : ALT.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAlterDropColumn()
			if rule.Item != "OK" {
				t.Error(sql, " Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ALT.004
func TestRuleAlterDropKey(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`alter table film drop primary key`,
			`alter table film drop foreign key fk_film_language`,
		},
		{
			// 反面的例子
			`ALTER TABLE t1 CHANGE c1 c1 TEXT CHARACTER SET utf8;`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAlterDropKey()
			if rule.Item != "ALT.004" {
				t.Error(sql, " Rule not match:", rule.Item, "Expect : ALT.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAlterDropKey()
			if rule.Item != "OK" {
				t.Error(sql, " Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.012
func TestRuleCantBeNull(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"CREATE TABLE `sbtest` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` longblob, PRIMARY KEY (`id`));",
		"alter TABLE `sbtest` add column `c` longblob;",
		"alter TABLE `sbtest` add column `c` text;",
		"alter TABLE `sbtest` add column `c` blob;",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCantBeNull()
			if rule.Item != "COL.012" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.012")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KEY.006
func TestRuleTooManyKeyParts(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"CREATE TABLE `sbtest` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` longblob NOT NULL DEFAULT '', PRIMARY KEY (`id`));",
		"alter TABLE `sbtest` add index idx_idx (`id`);",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			common.Config.MaxIdxColsCount = 0
			rule := q.RuleTooManyKeyParts()
			if rule.Item != "KEY.006" {
				t.Error("Rule not match:", rule.Item, "Expect : KEY.006")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KEY.005
func TestRuleTooManyKeys(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"create table tbl ( a char(10), b int, primary key (`a`)) engine=InnoDB;",
		"create table tbl ( a varchar(64) not null, b int, PRIMARY KEY (`a`), key `idx_a_b` (`a`,`b`)) engine=InnoDB",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			common.Config.MaxIdxCount = 0
			rule := q.RuleTooManyKeys()
			if rule.Item != "KEY.005" {
				t.Error("Rule not match:", rule.Item, "Expect : KEY.005")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KEY.007
func TestRulePKNotInt(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"create table tbl ( a char(10), b int, primary key (`a`)) engine=InnoDB;",
			"create table tbl ( a int, b int, primary key (`a`)) engine=InnoDB;",
			"create table tbl ( a bigint, b int, primary key (`a`)) engine=InnoDB;",
			"create table tbl ( a int unsigned, b int, primary key (`a`)) engine=InnoDB;",
			"create table tbl ( a bigint unsigned, b int, primary key (`a`)) engine=InnoDB;",
		},
		{
			"CREATE TABLE tbl (a int unsigned auto_increment, b int, primary key(`a`)) engine=InnoDB;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RulePKNotInt()
			if rule.Item != "KEY.007" && rule.Item != "KEY.001" {
				t.Error("Rule not match:", rule.Item, "Expect : KEY.007 OR KEY.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RulePKNotInt()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KEY.008
func TestRuleOrderByMultiDirection(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`SELECT col FROM tbl order by col desc, col2 asc`,
		},
		{
			`SELECT col FROM tbl order by col, col2`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleOrderByMultiDirection()
			if rule.Item != "KEY.008" {
				t.Error("Rule not match:", rule.Item, "Expect : KEY.008")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleOrderByMultiDirection()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KEY.009
func TestRuleUniqueKeyDup(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`ALTER TABLE customer ADD UNIQUE INDEX part_of_name (name(10));`,
			`CREATE UNIQUE INDEX part_of_name ON customer (name(10));`,
		},
		{
			`ALTER TABLE tbl add INDEX idx_col (col);`,
			`CREATE INDEX part_of_name ON customer (name(10));`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUniqueKeyDup()
			if rule.Item != "KEY.009" {
				t.Error("Rule not match:", rule.Item, "Expect : KEY.009")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUniqueKeyDup()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.013
func TestRuleTimestampDefault(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE TABLE tbl( `id` bigint not null, `create_time` timestamp) ENGINE=InnoDB DEFAULT CHARSET=utf8;",
			"ALTER TABLE t1 MODIFY b timestamp NOT NULL;",
		},
		{
			"CREATE TABLE tbl (`id` bigint not null, `update_time` timestamp default current_timestamp)",
			"ALTER TABLE t1 MODIFY b timestamp NOT NULL default current_timestamp;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTimestampDefault()
			if rule.Item != "COL.013" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.013")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTimestampDefault()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// TBL.004
func TestRuleAutoIncrementInitNotZero(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		// 正面的例子
		{
			"CREATE TABLE `sbtest` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT,  `pad` char(60) NOT NULL DEFAULT '', PRIMARY KEY (`id`)) ENGINE=InnoDB AUTO_INCREMENT=13",
		},
		// 反面的例子
		{
			"CREATE TABLE `test1` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `pad` char(60) NOT NULL DEFAULT '', PRIMARY KEY (`id`))",
			"CREATE TABLE `test1` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `pad` char(60) NOT NULL DEFAULT '', PRIMARY KEY (`id`)) auto_increment = 1",
			"CREATE TABLE `test1` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `pad` char(60) NOT NULL DEFAULT '', PRIMARY KEY (`id`)) auto_increment = 1 DEFAULT CHARSET=latin1",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAutoIncrementInitNotZero()
			if rule.Item != "TBL.004" {
				t.Error("Rule not match:", rule.Item, "Expect : TBL.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAutoIncrementInitNotZero()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.014
func TestRuleColumnWithCharset(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		// 正面的例子
		{
			"CREATE TABLE `tb2` ( `id` int(11) DEFAULT NULL, `col` char(10) CHARACTER SET utf8 DEFAULT NULL)",
			"alter table tb2 change col col char(10) CHARACTER SET utf8 DEFAULT NULL;",
		},
		// 反面的例子
		{
			"CREATE TABLE `sbtest` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` char(120) NOT NULL DEFAULT '', PRIMARY KEY (`id`))",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleColumnWithCharset()
			if rule.Item != "COL.014" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.014")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleColumnWithCharset()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// TBL.005
func TestRuleTableCharsetCheck(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"create table tbl (a int) DEFAULT CHARSET=latin1;",
			"ALTER TABLE tbl CONVERT TO CHARACTER SET latin1;",
		},
		{
			"create table tlb (a int);",
			"ALTER TABLE `tbl` add column a int, add column b int ;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTableCharsetCheck()
			if rule.Item != "TBL.005" {
				t.Error("Rule not match:", rule.Item, "Expect : TBL.005")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTableCharsetCheck()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.015
func TestRuleBlobDefaultValue(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE TABLE `sbtest` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` blob NOT NULL DEFAULT '', PRIMARY KEY (`id`));",
			"alter table `sbtest` add column `c` blob NOT NULL DEFAULT '';",
		},
		{
			"CREATE TABLE `sbtest` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` blob NOT NULL, PRIMARY KEY (`id`));",
			"alter table `sbtest` add column `c` blob NOT NULL DEFAULT NULL;",
		},
	}

	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleBlobDefaultValue()
			if rule.Item != "COL.015" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.015")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleBlobDefaultValue()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.016
func TestRuleIntPrecision(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE TABLE `sbtest` ( `id` int(1) );",
			"CREATE TABLE `sbtest` ( `id` bigint(1) );",
			"alter TABLE `sbtest` add column `id` bigint(1);",
			"alter TABLE `sbtest` add column `id` int(1);",
		},
		{
			"CREATE TABLE `sbtest` ( `id` int(10));",
			"CREATE TABLE `sbtest` ( `id` bigint(20));",
			"alter TABLE `sbtest` add column `id` bigint(20);",
			"alter TABLE `sbtest` add column `id` int(10);",
			"CREATE TABLE `sbtest` ( `id` int);",
			"alter TABLE `sbtest` add column `id` bigint;",
		},
	}

	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleIntPrecision()
			if rule.Item != "COL.016" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.016")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleIntPrecision()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.017
func TestRuleVarcharLength(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE TABLE `sbtest` ( `id` varchar(4000) );",
			"CREATE TABLE `sbtest` ( `id` varchar(3500) );",
			"alter TABLE `sbtest` add column `id` varchar(3500);",
		},
		{
			"CREATE TABLE `sbtest` ( `id` varchar(1024));",
			"CREATE TABLE `sbtest` ( `id` varchar(20));",
			"alter TABLE `sbtest` add column `id` varchar(35);",
		},
	}

	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleVarcharLength()
			if rule.Item != "COL.017" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.017")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleVarcharLength()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KEY.002
func TestRuleNoOSCKey(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		// 正面的例子
		{
			"CREATE TABLE tbl (a int, b int)",
		},
		// 反面的例子
		{
			"CREATE TABLE tbl (a int, primary key(`a`))",
			"CREATE TABLE tbl (a int, unique key(`a`))",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNoOSCKey()
			if rule.Item != "KEY.002" {
				t.Error("Rule not match:", rule.Item, "Expect : KEY.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleNoOSCKey()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.006
func TestRuleTooManyFields(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"create table tbl (a int);",
	}

	common.Config.MaxColCount = 0
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTooManyFields()
			if rule.Item != "COL.006" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.006")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// TBL.002
func TestRuleAllowEngine(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE TABLE tbl (a int) engine=myisam;",
			"ALTER TABLE tbl engine=myisam;",
			"CREATE TABLE tbl (a int);",
		},
		{
			"CREATE TABLE tbl (a int) engine = InnoDB;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAllowEngine()
			if rule.Item != "TBL.002" {
				t.Error("Rule not match:", rule.Item, "Expect : TBL.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAllowEngine()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// TBL.001
func TestRulePartitionNotAllowed(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`CREATE TABLE trb3 (id INT, name VARCHAR(50), purchased DATE) PARTITION BY RANGE( YEAR(purchased) )
	(
        PARTITION p0 VALUES LESS THAN (1990),
        PARTITION p1 VALUES LESS THAN (1995),
        PARTITION p2 VALUES LESS THAN (2000),
        PARTITION p3 VALUES LESS THAN (2005)
    );`,
		`ALTER TABLE t1 ADD PARTITION (PARTITION p3 VALUES LESS THAN (2002));`,
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RulePartitionNotAllowed()
			if rule.Item != "TBL.001" {
				t.Error("Rule not match:", rule.Item, "Expect : TBL.001")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.003
func TestRuleAutoIncUnsigned(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"CREATE TABLE `sbtest` ( `id` int(10) NOT NULL AUTO_INCREMENT, `c` longblob, PRIMARY KEY (`id`));",
		"ALTER TABLE `tbl` ADD COLUMN `id` int(10) NOT NULL AUTO_INCREMENT;",
	}
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleAutoIncUnsigned()
			if rule.Item != "COL.003" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// STA.003
func TestRuleIdxPrefix(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE TABLE tbl (a int, unique key `xx_a` (`a`));",
			"CREATE TABLE tbl (a int, key `xx_a` (`a`));",
			`ALTER TABLE tbl ADD INDEX xx_a (a)`,
			`ALTER TABLE tbl ADD UNIQUE INDEX xx_a (a)`,
		},
		{
			`ALTER TABLE tbl ADD INDEX idx_a (a)`,
			`ALTER TABLE tbl ADD UNIQUE INDEX uk_a (a)`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleIdxPrefix()
			if rule.Item != "STA.003" {
				t.Error("Rule not match:", rule.Item, "Expect : STA.003")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleIdxPrefix()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// STA.004
func TestRuleStandardName(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE TABLE `tbl-name` (a int);",
			"CREATE TABLE `tbl `(a int)",
			"CREATE TABLE t__bl (a int);",
		},
		{
			"CREATE TABLE tbl (a int)",
			"CREATE TABLE `tbl`(a int)",
			"CREATE TABLE `tbl` (a int) ENGINE=InnoDB DEFAULT CHARSET=utf8",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleStandardName()
			if rule.Item != "STA.004" {
				t.Error("Rule not match:", rule.Item, "Expect : STA.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleStandardName()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// STA.002
func TestRuleSpaceAfterDot(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"SELECT * FROM sakila. film",
			"SELECT film. length FROM film",
		},
		{
			"SELECT * FROM sakila.film",
			"SELECT film.length FROM film",
			"SELECT * FROM t1, t2 WHERE t1.title = t2.title",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSpaceAfterDot()
			if rule.Item != "STA.002" {
				t.Error("Rule not match:", rule.Item, "Expect : STA.002")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleSpaceAfterDot()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRuleMySQLError(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	err := errors.New(`Received #1146 error from MySQL server: "can't xxxx"`)
	if RuleMySQLError("ERR.002", err).Content != "" {
		t.Error("Want: '', Bug get: ", err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestMergeConflictHeuristicRules(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	tmpRules := make(map[string]Rule)
	for item, val := range HeuristicRules {
		tmpRules[item] = val
	}
	err := common.GoldenDiff(func() {
		suggest := MergeConflictHeuristicRules(tmpRules)
		var sortedSuggest []string
		for item := range suggest {
			sortedSuggest = append(sortedSuggest, item)
		}
		sort.Strings(sortedSuggest)
		for _, item := range sortedSuggest {
			pretty.Println(suggest[item])
		}
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
