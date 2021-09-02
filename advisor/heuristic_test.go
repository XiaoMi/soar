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

	"github.com/XiaoMi/soar/env"
	"github.com/kr/pretty"
	"vitess.io/vitess/go/vt/sqlparser"
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
			"select 1",
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
	sqls := [][]string{
		{
			"select tbl.* AS c1,c2,c3 from tbl where id < 1000",
			"SELECT * as",
		},
		{
			`SELECT c1, c2, c3, FROM tb WHERE id < 1000 AND content="mytest* as test"`,
			`select *`,
		},
	}
	for _, sql := range sqls[0] {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleStarAlias()
		if rule.Item != "ALI.002" {
			t.Error("Rule not match:", rule.Item, "Expect : ALI.002")
		}
	}
	for _, sql := range sqls[1] {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleStarAlias()
		if rule.Item != "OK" {
			t.Error("Rule not match:", rule.Item, "Expect : OK")
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
	sqls := [][]string{
		{
			"select col from tbl where id like 'abc'",
			"select col from tbl where id like 1",
		},
		{
			"select col from tbl where id like 'abc%'",
			"select col from tbl where id like '%abc'",
			"select col from tbl where id like 'a%c'", // issue #273
		},
	}
	for _, sql := range sqls[0] {
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

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleEqualLike()
			if rule.Item == "ARG.002" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.003
// TODO:

func TestTimeFormatError(t *testing.T) {
	rightTimes := []string{
		`2020-01-01`,
		`2020-01-01 23:59:59`,
		`2020-01-01 23:59:59.0`,   // 0ms
		`2020-01-01 23:59:59.123`, // 123ms
	}
	for _, rt := range rightTimes {
		if !timeFormatCheck(rt) {
			t.Error("wrong time format")
		}
	}

	wrongTimes := []string{
		``,                    // 空时间
		`2020-01-01 abc`,      // 含英文字符
		`2020–02-15 23:59:59`, // 2020 后面的不是减号，是个 连接符
	}
	for _, wt := range wrongTimes {
		if timeFormatCheck(wt) {
			t.Error("wrong time format")
		}
	}
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
	sqls := [][]string{
		{
			"SELECT col FROM tbl order by cola - cl;",                // order by 列运算
			"SELECT cola - cl col FROM tbl order by col;",            // 别名为列运算
			"SELECT cola FROM tbl order by from_unixtime(col);",      // order by 函数运算
			"SELECT from_unixtime(col) cola FROM tbl order by cola;", // 别名为函数运算
		},
		{
			`SELECT tbl.col FROM tbl ORDER BY col`,
			"SELECT sum(col) AS col FROM tbl ORDER BY dt",
			"SELECT tbl.col FROM tb, tbl WHERE tbl.tag_id = tb.id ORDER BY tbl.col",
			"SELECT col FROM tbl order by `timestamp`;",           // 列名为关键字
			"select col from tb where cl = 1 order by APPLY_TIME", // issue #104 case sensitive
		},
	}
	for _, sql := range sqls[0] {
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
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleOrderByExpr()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
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
		"SELECT col FROM tbl GROUP by cola - col;",
		"SELECT cola - col col FROM tbl GROUP by col;",
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
			"ENGINE = InnoDB DEFAULT CHARSET=utf8",
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
	sqls := [][]string{
		{
			"insert into tbl values(1,'name')",
			"replace into tbl values(1,'name')",
		},
		{
			"insert into tb (col) values ('hello world')",
		},
	}
	for _, sql := range sqls[0] {
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

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleInsertColDef()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
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
			`create table tb (a text)`,
			`alter table tb add a text`,
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
	sqls := [][]string{
		{
			"insert into tbl (IP,name) values('10.20.306.122','test')",
		},
		{
			`CREATE USER IF NOT EXISTS 'test'@'1.1.1.1';`,
			"ALTER USER 'test'@'1.1.1.1' IDENTIFIED WITH 'mysql_native_password' AS '*xxxxx' REQUIRE NONE PASSWORD EXPIRE DEFAULT ACCOUNT UNLOCK;",
			"GRANT SELECT ON `test`.* TO 'test'@'1.1.1.1';",
			`GRANT USAGE ON *.* TO 'test'@'1.1.1.1';`,
		},
	}
	for _, sql := range sqls[0] {
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

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleIPString()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
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
	sqls := [][]string{
		{
			"select col1,col2 from tbl where time < 2018-01-10",
			"select col1,col2 from tbl where time < 18-01-10",
			"INSERT INTO tb1 SELECT * FROM tb2 WHERE time < 2020-01-10",
		},
		{
			"select col1,col2 from tbl where time < '2018-01-10'",
			"INSERT INTO `tb` (`col`) VALUES ('timestamp=2019-12-16')",
			"insert into tb (col) values (' 2020-09-15 ')",
			"replace into tb (col) values (' 2020-09-15 ')",
			"INSERT INTO tb1 SELECT * FROM tb2 WHERE time < '2020-01-10'",
		},
	}
	for _, sql := range sqls[0] {
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

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleDataNotQuote()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
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
		"select col1,col2 from tbl where name='tony' limit 10",
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
			"update tbl set col = 1 AND cl = 2 where col=3;",
			"update table1 set a = ( select a from table2 where b=1 and c=2) and b=1 where d=2",
		},
		{
			"update tbl set col = 1 ,cl = 2 where col=3;",
			// https://github.com/XiaoMi/soar/issues/226
			"update table1 set a = ( select a from table2 where b=1 and c=2), b=1, c=2 where d=2",
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
			"select * from tbl where 'a';",
			"select * from tbl where 'a' limit 1;",
			"select * from tbl where 1;",
			"select * from tbl where 1 limit 1;",
			"select * from tbl where id = 1 or 2;",
			"select * from tbl where true;",
			"select * from tbl where 'true';",
		},
		{
			"select * from tbl where false;",
			"select * from tbl where 'false';",
			"select * from tbl where 0;",
			"select * from tbl where '0';",
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

// RES.009
func TestRuleMultiCompare(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"SELECT * FROM tbl WHERE col = col = 'abc'",
			"SELECT * FROM tbl WHERE col = 'def' and col = col = 'abc'",
			"SELECT * FROM tbl WHERE col = 'def' or col = col = 'abc'",
			"SELECT * FROM tbl WHERE col = col = 'abc' and col = 'def'",
			"UPDATE tbl set col = 1 WHERE col = col = 'abc'",
			"DELETE FROM tbl WHERE col = col = 'abc'",
		},
		{
			"SELECT * FROM tbl WHERE col = 'abc'",
			// https://github.com/XiaoMi/soar/issues/169
			"SELECT * FROM tbl WHERE col = 'abc' and c = 1",
			"update tb set c = 1 where a = 2 and b = 3",
			"delete from tb where a = 2 and b = 3",
		},
	}

	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMultiCompare()
			if rule.Item != "RES.009" {
				t.Error("Rule not match:", rule.Item, "Expect : RES.009, SQL: ", sql)
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMultiCompare()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK, SQL: ", sql)
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// RES.010
func TestRuleCreateOnUpdate(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`CREATE TABLE category (
  category_id TINYINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(25) NOT NULL,
  last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY  (category_id)
)`,
		},
		{
			`CREATE TABLE category (
  category_id TINYINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(25) NOT NULL,
  last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY  (category_id)
)`,
		},
	}

	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCreateOnUpdate()
			if rule.Item != "RES.010" {
				t.Error("Rule not match:", rule.Item, "Expect : RES.010, SQL: ", sql)
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCreateOnUpdate()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK, SQL: ", sql)
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// RES.011
func TestRuleUpdateOnUpdate(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`UPDATE category SET name='ActioN' WHERE category_id=1`,
		},
		{
			`select * from film limit 1`,
			"UPDATE category SET name='ActioN', last_update=last_update WHERE category_id=1",
		},
	}

	for _, sql := range sqls[0] {
		vEnv.BuildVirtualEnv(rEnv, sql)
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			t.Error(syntaxErr)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}
		idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
		if err != nil {
			t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
		}

		if idxAdvisor != nil {
			rule := idxAdvisor.RuleUpdateOnUpdate()
			if rule.Item != "RES.011" {
				t.Error("Rule not match:", rule.Item, "Expect : RES.011, SQL:", sql)
			}
		}
	}

	for _, sql := range sqls[1] {
		vEnv.BuildVirtualEnv(rEnv, sql)
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			t.Error(syntaxErr)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}
		idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
		if err != nil {
			t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
		}

		if idxAdvisor != nil {
			rule := idxAdvisor.RuleUpdateOnUpdate()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK, SQL:", sql)
			}
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

// KWR.004
func TestRuleMultiBytesWord(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"select col as 列 from tb",
			"select col as `列` from tb",
		},
		{
			"select col as c from tb",
			"select '列'",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMultiBytesWord()
			if rule.Item != "KWR.004" {
				t.Error("Rule not match:", rule.Item, "Expect : KWR.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMultiBytesWord()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// KWR.005
func TestRuleInvisibleUnicode(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	// 不可见的 unicode 可以通过 https://unicode-table.com 复制得到
	sqls := [][]string{
		{
			`select 1`,   // SQL 中包含 non-broken-space
			`select​ 1;`, // SQL 中包含 zero-width space
		},
		{
			"select 1",    // 正常 SQL
			`select "1 "`, // 值中包含 non-broken-space
			`select "1​"`, // 值中包含 zero-width space
		},
	}
	for _, sql := range sqls[0] {
		q, _ := NewQuery4Audit(sql)
		// 含有特殊 unicode 字符的 SQL 语法肯定是不通过的
		rule := q.RuleInvisibleUnicode()
		if rule.Item != "KWR.005" {
			t.Error("Rule not match:", rule.Item, "Expect : KWR.005")
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleInvisibleUnicode()
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
			`ALTER TABLE tbl2 ADD KEY p_id (p_id);`,
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
			"REPLACE INTO `storage` (`hostname`,`storagehost`, `filename`, `starttime`, `binlogstarttime`, `uploadname`, `binlogsize`, `filesize`, `md5`, `status`) VALUES (1, 1, 1, 1, 1, 1, ?, ?);",
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
		`create index idx1 on tab(last_name,first_name);`,
		`alter table tab add index idx1 (last_name,first_name);`,
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
		`select c1,c2,c3 from tab where c4 is null or c4 <> 1;`,
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
		`select c1 || coalesce(' ' || c2 || ' ', ' ') || c3 as c from tab;`,
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
		`select Sysdate();`,
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
		`SELECT DISTINCT c.c_id,count(DISTINCT c.c_name),count(DISTINCT c.c_e),count(DISTINCT c.c_n),count(DISTINCT c.c_me),c.c_d FROM (select distinct id, name from B) as e WHERE e.country_id = c.country_id;`,
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

// FUN.007
func TestRuleForbiddenTrigger(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`CREATE TRIGGER t1 AFTER INSERT ON work FOR EACH ROW INSERT INTO time VALUES(NOW());`,
	}
	for _, sql := range sqls {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleForbiddenTrigger()
		if rule.Item != "FUN.007" {
			t.Error("Rule not match:", rule.Item, "Expect : FUN.007")
		}

	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// FUN.008
func TestRuleForbiddenProcedure(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`CREATE PROCEDURE simpleproc (OUT param1 INT)`,
	}
	for _, sql := range sqls {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleForbiddenProcedure()
		if rule.Item != "FUN.008" {
			t.Error("Rule not match:", rule.Item, "Expect : FUN.008")
		}

	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// FUN.009
func TestRuleForbiddenFunction(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`CREATE FUNCTION hello (s CHAR(20));`,
	}
	for _, sql := range sqls {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleForbiddenFunction()
		if rule.Item != "FUN.009" {
			t.Error("Rule not match:", rule.Item, "Expect : FUN.009")
		}

	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// TBL.006
func TestRuleForbiddenView(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`create view v_today (today) AS SELECT CURRENT_DATE;`,
		`CREATE VIEW v (col) AS SELECT 'abc';`,
	}
	for _, sql := range sqls {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleForbiddenView()
		if rule.Item != "TBL.006" {
			t.Error("Rule not match:", rule.Item, "Expect : TBL.006")
		}

	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// TBL.007
func TestRuleForbiddenTempTable(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"CREATE TEMPORARY TABLE `work` (`time` time DEFAULT NULL) ENGINE=InnoDB;",
	}
	for _, sql := range sqls {
		q, _ := NewQuery4Audit(sql)
		rule := q.RuleForbiddenTempTable()
		if rule.Item != "TBL.007" {
			t.Error("Rule not match:", rule.Item, "Expect : TBL.007")
		}

	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// JOI.006
func TestRuleNestedSubQueries(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		`SELECT s,p,d FROM tab WHERE p.p_id = (SELECT s.p_id FROM tab WHERE s.c_id = 100996 AND s.q = 1 );`,
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
	sqls := [][]string{
		{
			`SELECT c1,c2,c3 FROM tab WHERE c1 = 14 OR c1 = 14;`,
		},
		{
			`SELECT c1,c2,c3 FROM tab WHERE c1 = 14 OR c2 = 17;`,
			`SELECT c1,c2,c3 FROM tab WHERE c1 = 14 OR c1 IS NULL;`,
		},
	}
	for _, sql := range sqls[0] {
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
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleORUsage()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
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
			`create table tb ( a varchar(10) default ' ');`,
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
func TestRuleNot(t *testing.T) {
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

// ARG.012
func TestRuleInsertValues(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`INSERT INTO tb VALUES (1), (2)`,
			`REPLACE INTO tb VALUES (1), (2)`,
		},
		{
			`INSERT INTO tb VALUES (1)`,
		},
	}
	oldMaxValueCount := common.Config.MaxValueCount
	common.Config.MaxValueCount = 1
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleInsertValues()
			if rule.Item != "ARG.012" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.012")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleInsertValues()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Config.MaxValueCount = oldMaxValueCount
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// ARG.013
func TestRuleFullWidthQuote(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`CREATE TABLE tb (a varchar(10) default '“”')`,
			`CREATE TABLE tb (a varchar(10) default '‘’')`,
			`ALTER TABLE tb ADD COLUMN a VARCHAR(10) DEFAULT "“”"`,
		},
		{
			`CREATE TABLE tb (a varchar(10) default '""')`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleFullWidthQuote()
			if rule.Item != "ARG.013" {
				t.Error("Rule not match:", rule.Item, "Expect : ARG.013")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleFullWidthQuote()
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

// SUB.007
func TestRuleUNIONLimit(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`(SELECT * FROM tb1 ORDER BY name) UNION ALL (SELECT * FROM tb2 ORDER BY name) LIMIT 20;`,
			`(SELECT * FROM tb1 ORDER BY name LIMIT 20) UNION ALL (SELECT * FROM tb2 ORDER BY name) LIMIT 20;`,
			`(SELECT * FROM tb1 ORDER BY name) UNION ALL (SELECT * FROM tb2 ORDER BY name LIMIT 20) LIMIT 20;`,
		},
		{
			`(SELECT * FROM tb1 ORDER BY name LIMIT 20) UNION ALL (SELECT * FROM tb2 ORDER BY name LIMIT 20) LIMIT 20;`,
			`SELECT * FROM tb1 ORDER BY name LIMIT 20`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUNIONLimit()
			if rule.Item != "SUB.007" {
				t.Error("Rule not match:", rule.Item, "Expect : SUB.007", sql)
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleUNIONLimit()
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

// SEC.004
func TestRuleInjection(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`select benchmark(10, rand())`,
			`select sleep(1)`,
			`select Sleep(1)`,
			`select get_lock('lock_name', 1)`,
			`select release_lock('lock_name')`,
		},
		{
			"select * from `sleep`",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleInjection()
			if rule.Item != "SEC.004" {
				t.Error("Rule not match:", rule.Item, "Expect : SEC.004")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleInjection()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
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
		{
			`select id from t where substring(name,1,3)='abc';`,
			`SELECT * FROM tbl WHERE UNIX_TIMESTAMP(loginTime) BETWEEN UNIX_TIMESTAMP('2018-11-16 09:46:00 +0800 CST') AND UNIX_TIMESTAMP('2018-11-22 00:00:00 +0800 CST')`,
			`select id from t where num/2 = 100`,
			`select id from t where num/2 < 100`,
			// 时间 builtin 函数
			`SELECT * FROM tb WHERE DATE '2020-01-01'`,
			`DELETE FROM tb WHERE DATE '2020-01-01'`,
			`UPDATE tb SET col = 1 WHERE DATE '2020-01-01'`,
			`SELECT * FROM tb WHERE TIME '10:01:01'`,
			`SELECT * FROM tb WHERE TIMESTAMP '1587181360'`,
			`select * from mysql.user where user  = "root" and date '2020-02-01'`,
			// 右侧使用函数比较
			`select id from t where 'abc'=substring(name,1,3);`,
		},
		// 正常 SQL
		{
			`select id from t where col = (select 1)`,
			`select id from t where col = 1`,
		},
	}
	for i, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleCompareWithFunction()
			if rule.Item != "FUN.001" {
				t.Errorf("SQL: %d,  Rule not match: %s Expect : FUN.001", i, rule.Item)
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
		`truncate TABLE tbl_name;`,
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
func TestRuleIsNullIsNotNull(t *testing.T) {
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
		"create table `dual`(id int, primary key (id));",
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
			`ALTER TABLE t1 CHANGE a b BIGINT NOT NULL, character set utf8`,
			`ALTER TABLE t1 CHANGE a b BIGINT NOT NULL,character set utf8`,
			`alter table t1 default collate = utf8_unicode_ci;`,
			`ALTER TABLE tbl_name CHARACTER SET 'utf8';`,
			// `ALTER TABLE tbl_name CHARACTER SET charset_name;`, // FIXME: unknown CHARACTER SET
			// `alter table t1 convert to character set utf8 collate utf8_unicode_ci;`, // FIXME: syntax not compatible
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
	sqls := [][]string{
		{
			"CREATE TABLE `tb`(`c` longblob NOT NULL);",
		},
		{
			"CREATE TABLE `tbl` (`c` longblob);",
			"alter TABLE `tbl` add column `c` longblob;",
			"alter TABLE `tbl` add column `c` text;",
			"alter TABLE `tbl` add column `c` blob;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleBLOBNotNull()
			if rule.Item != "COL.012" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.012")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleBLOBNotNull()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
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
		"CREATE TABLE `tb` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` longblob NOT NULL DEFAULT '', PRIMARY KEY (`id`));",
		"alter TABLE `tb` add index idx_idx (`id`);",
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
			"CREATE TABLE `tb` ( `id` Bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'auto id', Primary  key (`id`) ) ENGINE = InnoDB COMMENT 'comment'",
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

// KEY.010
func TestRuleFulltextIndex(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			`ALTER TABLE tb ADD FULLTEXT INDEX ip (ip);`,
			// `CREATE FULLTEXT INDEX ft_ip ON tb (ip);`, // TODO: tidb not support yet
			`CREATE TABLE tb ( id int(10) unsigned NOT NULL AUTO_INCREMENT, ip varchar(255) NOT NULL DEFAULT '', PRIMARY KEY (id), FULLTEXT KEY ip (ip) ) ENGINE=InnoDB;`,
		},
		{
			`ALTER TABLE tbl add INDEX idx_col (col);`,
			`CREATE INDEX part_of_name ON customer (name(10));`,
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleFulltextIndex()
			if rule.Item != "KEY.010" {
				t.Error("Rule not match:", rule.Item, "Expect : KEY.010")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleFulltextIndex()
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
			`ALTER TABLE t1 ADD c_time timestamp NOT NULL default "0000-00-00"`,
			`ALTER TABLE t1 ADD c_time timestamp NOT NULL default '0'`,
			`ALTER TABLE t1 ADD c_time timestamp NOT NULL default 0`,
			`ALTER TABLE t1 ADD c_time datetime NOT NULL default 0`,
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
				t.Error("Rule not match:", rule.Item, "Expect : COL.013", sql)
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
			"CREATE TABLE `tb` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT,  `pad` char(60) NOT NULL DEFAULT '', PRIMARY KEY (`id`)) ENGINE=InnoDB AUTO_INCREMENT=13",
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
			"CREATE TABLE tb (a nvarchar(10))",
			"CREATE TABLE tb (a nchar(10))",
		},
		// 反面的例子
		{
			"CREATE TABLE `tb` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` char(120) NOT NULL DEFAULT '', PRIMARY KEY (`id`))",
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
			//"CREATE DATABASE sbtest /*!40100 DEFAULT CHARACTER SET latin1 */;", // FIXME:
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

// TBL.008
func TestRuleTableCollateCheck(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE DATABASE sbtest /*!40100 DEFAULT COLLATE latin1_bin */;",
			"create table tbl (a int) DEFAULT COLLATE=latin1_bin;",
		},
		{
			"create table tlb (a int);",
			"ALTER TABLE `tbl` add column a int, add column b int ;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTableCollateCheck()
			if rule.Item != "TBL.008" {
				t.Error("Rule not match:", rule.Item, "Expect : TBL.008")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTableCollateCheck()
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
			"CREATE TABLE `tb` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` blob NOT NULL DEFAULT '', PRIMARY KEY (`id`));",
			"CREATE TABLE `tb` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` json NOT NULL DEFAULT '', PRIMARY KEY (`id`));",
			"alter table `tb` add column `c` blob NOT NULL DEFAULT '';",
			"alter table `tb` add column `c` json NOT NULL DEFAULT '';",
		},
		{
			"CREATE TABLE `tb` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` blob NOT NULL, PRIMARY KEY (`id`));",
			"CREATE TABLE `tb` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` json NOT NULL, PRIMARY KEY (`id`));",
			"CREATE TABLE `tb` (`col` text NOT NULL);",
			"alter table `tb` add column `c` blob NOT NULL;",
			"alter table `tb` add column `c` json NOT NULL;",
			"ALTER TABLE tb ADD COLUMN a BLOB DEFAULT NULL",
			"ALTER TABLE tb ADD COLUMN a JSON DEFAULT NULL",
			"CREATE TABLE tb ( a BLOB DEFAULT NULL)",
			"CREATE TABLE tb ( a JSON DEFAULT NULL)",
			"alter TABLE `tbl` add column `c` longblob;",
			"alter TABLE `tbl` add column `c` text;",
			"alter TABLE `tbl` add column `c` blob;",
			"alter TABLE `tbl` add column `c` json;",
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
			"CREATE TABLE `tb` ( `id` int(1) );",
			"CREATE TABLE `tb` ( `id` bigint(1) );",
			"alter TABLE `tb` add column `id` bigint(1);",
			"alter TABLE `tb` add column `id` int(1);",
		},
		{
			"CREATE TABLE `tb` ( `id` int(10));",
			"CREATE TABLE `tb` ( `id` bigint(20));",
			"alter TABLE `tb` add column `id` bigint(20);",
			"alter TABLE `tb` add column `id` int(10);",
			"CREATE TABLE `tb` ( `id` int);",
			"alter TABLE `tb` add column `id` bigint;",
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
			"CREATE TABLE `tb` ( `id` varchar(4000) );",
			"CREATE TABLE `tb` ( `id` varchar(3500) );",
			"alter TABLE `tb` add column `id` varchar(3500);",
		},
		{
			"CREATE TABLE `tb` ( `id` varchar(1024));",
			"CREATE TABLE `tb` ( `id` varchar(20));",
			"alter TABLE `tb` add column `id` varchar(35);",
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

// COL.018
func TestRuleColumnNotAllowType(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())

	sqls := [][]string{
		{
			"CREATE TABLE tab (a BOOLEAN);",
			"CREATE TABLE tab (a BOOLEAN );",
			"ALTER TABLE `tb` add column `a` BOOLEAN;",
		},
		{
			"CREATE TABLE `tb` ( `id` varchar(1024));",
			"ALTER TABLE `tb` add column `id` varchar(35);",
		},
	}

	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleColumnNotAllowType()
			if rule.Item != "COL.018" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.018")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleColumnNotAllowType()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule.Item, "Expect : OK")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.019
func TestRuleTimePrecision(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		// 正面的例子
		{
			"CREATE TABLE t1 (t TIME(3), dt DATETIME(6));",
			"ALTER TABLE t1 add t TIME(3);",
		},
		// 反面的例子
		{
			"CREATE TABLE t1 (t TIME, dt DATETIME);",
			"ALTER TABLE t1 add t TIME;",
		},
	}
	for _, sql := range sqls[0] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTimePrecision()
			if rule.Item != "COL.019" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.019")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}

	for _, sql := range sqls[1] {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleTimePrecision()
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

// COL.007
func TestRuleMaxTextColsCount(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"create table tbl (a int, b text, c blob, d text);",
	}

	common.Config.MaxColCount = 0
	for _, sql := range sqls {
		q, err := NewQuery4Audit(sql)
		if err == nil {
			rule := q.RuleMaxTextColsCount()
			if rule.Item != "COL.007" {
				t.Error("Rule not match:", rule.Item, "Expect : COL.007")
			}
		} else {
			t.Error("sqlparser.Parse Error:", err)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// COL.007
func TestRuleMaxTextColsCountWithEnv(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgMaxTextColsCount := common.Config.MaxTextColsCount
	common.Config.MaxTextColsCount = 1

	vEnv, rEnv := env.BuildEnv()
	defer vEnv.CleanUp()
	initSQLs := []string{
		`CREATE TABLE t1 (id int, title text);`,
		`CREATE TABLE t2 (id int, title text);`,
	}

	for _, sql := range initSQLs {
		vEnv.BuildVirtualEnv(rEnv, sql)
	}

	sqls := [][]string{
		{
			"alter table t1 add column other text;",
		},
		{
			"alter table t2 add column col varchar(10);",
		},
	}

	for _, sql := range sqls[0] {
		vEnv.BuildVirtualEnv(rEnv, sql)
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			t.Error(syntaxErr)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}
		idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
		if err != nil {
			t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
		}

		if idxAdvisor != nil {
			rule := idxAdvisor.RuleMaxTextColsCount()
			if rule.Item != "COL.007" {
				t.Error("Rule not match:", rule, "Expect : COL.007, SQL:", sql)
			}
		}
	}

	for _, sql := range sqls[1] {
		vEnv.BuildVirtualEnv(rEnv, sql)
		stmt, syntaxErr := sqlparser.Parse(sql)
		if syntaxErr != nil {
			t.Error(syntaxErr)
		}

		q := &Query4Audit{Query: sql, Stmt: stmt}
		idxAdvisor, err := NewAdvisor(vEnv, *rEnv, *q)
		if err != nil {
			t.Error("NewAdvisor Error: ", err, "SQL: ", sql)
		}

		if idxAdvisor != nil {
			rule := idxAdvisor.RuleMaxTextColsCount()
			if rule.Item != "OK" {
				t.Error("Rule not match:", rule, "Expect : OK, SQL:", sql)
			}
		}
	}

	common.Config.MaxTextColsCount = orgMaxTextColsCount
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// TBL.002
func TestRuleAllowEngine(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := [][]string{
		{
			"CREATE TABLE tbl (a int) engine=MyISAM;",
			"ALTER TABLE tbl engine=MyISAM;",
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
		"CREATE TABLE `tb` ( `id` int(10) NOT NULL AUTO_INCREMENT, `c` longblob, PRIMARY KEY (`id`));",
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
			"SELECT `dataType` FROM tb;",
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
	err := errors.New(`received #1146 error from MySQL server: "can't xxxx"`)
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
