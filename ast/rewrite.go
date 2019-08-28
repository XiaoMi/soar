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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pingcap/parser/ast"
	"reflect"
	"regexp"
	"strings"

	"github.com/XiaoMi/soar/common"

	"github.com/kr/pretty"
	"vitess.io/vitess/go/vt/sqlparser"
)

// Rule SQL重写规则
type Rule struct {
	Name        string                  `json:"Name"`
	Description string                  `json:"Description"`
	Original    string                  `json:"Original"` // 错误示范。为空或"暂不支持"不会出现在list-rewrite-rules中
	Suggest     string                  `json:"Suggest"`  // 正确示范。
	Func        func(*Rewrite) *Rewrite `json:"-"`        // 如果不定义 Func 需要多条 SQL 联动改写
}

// RewriteRules SQL重写规则，注意这个规则是有序的，先后顺序不能乱
var RewriteRules []Rule

func init() {
	RewriteRules = []Rule{
		{
			Name:        "dml2select",
			Description: "将数据库更新请求转换为只读查询请求，便于执行EXPLAIN",
			Original:    "DELETE FROM film WHERE length > 100",
			Suggest:     "select * from film where length > 100",
			Func:        (*Rewrite).RewriteDML2Select,
		},
		{
			Name:        "star2columns",
			Description: "为SELECT *补全表的列信息",
			Original:    "SELECT * FROM film",
			Suggest:     "select film.film_id, film.title from film",
			Func:        (*Rewrite).RewriteStar2Columns,
		},
		{
			Name:        "insertcolumns",
			Description: "为INSERT补全表的列信息",
			Original:    "insert into film values(1,2,3,4,5)",
			Suggest:     "insert into film(film_id, title, description, release_year, language_id) values (1, 2, 3, 4, 5)",
			Func:        (*Rewrite).RewriteInsertColumns,
		},
		{
			Name:        "having",
			Description: "将查询的 HAVING 子句改写为 WHERE 中的查询条件",
			Original:    "SELECT state, COUNT(*) FROM Drivers GROUP BY state HAVING state IN ('GA', 'TX') ORDER BY state",
			Suggest:     "select state, COUNT(*) from Drivers where state in ('GA', 'TX') group by state order by state asc",
			Func:        (*Rewrite).RewriteHaving,
		},
		{
			Name:        "orderbynull",
			Description: "如果 GROUP BY 语句不指定 ORDER BY 条件会导致无谓的排序产生，如果不需要排序建议添加 ORDER BY NULL",
			Original:    "SELECT sum(col1) FROM tbl GROUP BY col",
			Suggest:     "select sum(col1) from tbl group by col order by null",
			Func:        (*Rewrite).RewriteAddOrderByNull,
		},
		{
			Name:        "unionall",
			Description: "可以接受重复的时间，使用 UNION ALL 替代 UNION 以提高查询效率",
			Original:    "select country_id from city union select country_id from country",
			Suggest:     "select country_id from city union all select country_id from country",
			Func:        (*Rewrite).RewriteUnionAll,
		},
		{
			Name:        "or2in",
			Description: "将同一列不同条件的 OR 查询转写为 IN 查询",
			Original:    "select country_id from city where col1 = 1 or (col2 = 1 or col2 = 2 ) or col1 = 3;",
			Suggest:     "select country_id from city where (col2 in (1, 2)) or col1 in (1, 3);",
			Func:        (*Rewrite).RewriteOr2In,
		},
		{
			Name:        "innull",
			Description: "如果 IN 条件中可能有 NULL 值而又想匹配 NULL 值时，建议添加OR col IS NULL",
			Original:    "暂不支持",
			Suggest:     "暂不支持",
			Func:        (*Rewrite).RewriteInNull,
		},
		// 把所有跟 or 相关的重写完之后才进行 or 转 union 的重写
		{
			Name:        "or2union",
			Description: "将不同列的 OR 查询转为 UNION 查询，建议结合 unionall 重写策略一起使用",
			Original:    "暂不支持",
			Suggest:     "暂不支持",
			Func:        (*Rewrite).RewriteOr2Union,
		},
		{
			Name:        "dmlorderby",
			Description: "删除 DML 更新操作中无意义的 ORDER BY",
			Original:    "DELETE FROM tbl WHERE col1=1 ORDER BY col",
			Suggest:     "delete from tbl where col1 = 1",
			Func:        (*Rewrite).RewriteRemoveDMLOrderBy,
		},
		/*
			{
				Name:        "groupbyconst",
				Description: "删除无意义的GROUP BY常量",
				Original:    "SELECT sum(col1) FROM tbl GROUP BY 1;",
				Suggest:     "select sum(col1) from tbl",
				Func:        (*Rewrite).RewriteGroupByConst,
			},
		*/
		{
			Name:        "sub2join",
			Description: "将子查询转换为JOIN查询",
			Original:    "暂不支持",
			Suggest:     "暂不支持",
			Func:        (*Rewrite).RewriteSubQuery2Join,
		},
		{
			Name:        "join2sub",
			Description: "将JOIN查询转换为子查询",
			Original:    "暂不支持",
			Suggest:     "暂不支持",
			Func:        (*Rewrite).RewriteJoin2SubQuery,
		},
		{
			Name:        "distinctstar",
			Description: "DISTINCT *对有主键的表没有意义，可以将DISTINCT删掉",
			Original:    "SELECT DISTINCT * FROM film;",
			Suggest:     "SELECT * FROM film",
			Func:        (*Rewrite).RewriteDistinctStar,
		},
		{
			Name:        "standard",
			Description: "SQL标准化，如：关键字转换为小写",
			Original:    "SELECT sum(col1) FROM tbl GROUP BY 1;",
			Suggest:     "select sum(col1) from tbl group by 1",
			Func:        (*Rewrite).RewriteStandard,
		},
		{
			Name:        "mergealter",
			Description: "合并同一张表的多条ALTER语句",
			Original:    "ALTER TABLE t2 DROP COLUMN c;ALTER TABLE t2 DROP COLUMN d;",
			Suggest:     "ALTER TABLE t2 DROP COLUMN c, DROP COLUMN d;",
		},
		{
			Name:        "alwaystrue",
			Description: "删除无用的恒真判断条件",
			Original:    "SELECT count(col) FROM tbl where 'a'= 'a' or ('b' = 'b' and a = 'b');",
			Suggest:     "select count(col) from tbl where (a = 'b');",
			Func:        (*Rewrite).RewriteAlwaysTrue,
		},
		{
			Name:        "countstar",
			Description: "不建议使用COUNT(col)或COUNT(常量)，建议改写为COUNT(*)",
			Original:    "SELECT count(col) FROM tbl GROUP BY 1;",
			Suggest:     "SELECT count(*) FROM tbl GROUP BY 1;",
			Func:        (*Rewrite).RewriteCountStar,
		},
		{
			Name:        "innodb",
			Description: "建表时建议使用InnoDB引擎，非 InnoDB 引擎表自动转 InnoDB",
			Original:    "CREATE TABLE t1(id bigint(20) NOT NULL AUTO_INCREMENT);",
			Suggest:     "create table t1 (\n\tid bigint(20) not null auto_increment\n) ENGINE=InnoDB;",
			Func:        (*Rewrite).RewriteInnoDB,
		},
		{
			Name:        "autoincrement",
			Description: "将autoincrement初始化为1",
			Original:    "CREATE TABLE t1(id bigint(20) NOT NULL AUTO_INCREMENT) ENGINE=InnoDB AUTO_INCREMENT=123802;",
			Suggest:     "create table t1(id bigint(20) not null auto_increment) ENGINE=InnoDB auto_increment=1;",
			Func:        (*Rewrite).RewriteAutoIncrement,
		},
		{
			Name:        "intwidth",
			Description: "整型数据类型修改默认显示宽度",
			Original:    "create table t1 (id int(20) not null auto_increment) ENGINE=InnoDB;",
			Suggest:     "create table t1 (id int(10) not null auto_increment) ENGINE=InnoDB;",
			Func:        (*Rewrite).RewriteIntWidth,
		},
		{
			Name:        "truncate",
			Description: "不带 WHERE 条件的 DELETE 操作建议修改为 TRUNCATE",
			Original:    "DELETE FROM tbl",
			Suggest:     "truncate table tbl",
			Func:        (*Rewrite).RewriteTruncate,
		},
		{
			Name:        "rmparenthesis",
			Description: "去除没有意义的括号",
			Original:    "select col from table where (col = 1);",
			Suggest:     "select col from table where col = 1;",
			Func:        (*Rewrite).RewriteRmParenthesis,
		},
		// delimiter要放在最后，不然补不上
		{
			Name:        "delimiter",
			Description: "补全DELIMITER",
			Original:    "use sakila",
			Suggest:     "use sakila;",
			Func:        (*Rewrite).RewriteDelimiter,
		},
		// TODO in to exists
		// TODO exists to in
	}
}

// ListRewriteRules 打印SQL重写规则
func ListRewriteRules(rules []Rule) {
	switch common.Config.ReportType {
	case "json":
		js, err := json.MarshalIndent(rules, "", "  ")
		if err == nil {
			fmt.Println(string(js))
		}
	default:

		fmt.Print("# 重写规则\n\n[toc]\n\n")
		for _, r := range rules {
			if !common.Config.Verbose && (r.Original == "" || r.Original == "暂不支持") {
				continue
			}

			fmt.Print("## ", common.MarkdownEscape(r.Name),
				"\n* **Description**:", r.Description+"\n",
				"\n* **Original**:\n\n```sql\n", r.Original, "\n```\n",
				"\n* **Suggest**:\n\n```sql\n", r.Suggest, "\n```\n")

		}
	}
}

// Rewrite 用于重写SQL
type Rewrite struct {
	SQL     string
	NewSQL  string
	Stmt    sqlparser.Statement
	Columns common.TableColumns
}

// NewRewrite 返回一个*Rewrite对象，如果SQL无法被正常解析，将错误输出到日志中，返回一个nil
func NewRewrite(sql string) *Rewrite {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		common.Log.Error(err.Error(), sql)
		return nil
	}

	return &Rewrite{
		SQL:  sql,
		Stmt: stmt,
	}
}

// Rewrite 入口函数
func (rw *Rewrite) Rewrite() *Rewrite {
	defer func() {
		if err := recover(); err != nil {
			common.Log.Error("Query rewrite Error: %s, maybe hit a bug.\nQuery: %s \nAST: %s",
				err, rw.SQL, pretty.Sprint(rw.Stmt))
			return
		}
	}()

	for _, rule := range RewriteRules {
		if RewriteRuleMatch(rule.Name) && rule.Func != nil {
			rule.Func(rw)
			common.Log.Debug("Rewrite Rule:%s Output NewSQL: %s", rule.Name, rw.NewSQL)
		}
	}
	if rw.NewSQL == "" {
		rw.NewSQL = rw.SQL
	}
	rw.Stmt, _ = sqlparser.Parse(rw.NewSQL)

	// TODO: 重新前后返回结果一致性对比

	// TODO: 前后SQL性能对比
	return rw
}

// RewriteDelimiter delimiter: 补分号，可以指定不同的DELIMITER
func (rw *Rewrite) RewriteDelimiter() *Rewrite {
	if rw.NewSQL != "" {
		rw.NewSQL = strings.TrimSuffix(rw.NewSQL, common.Config.Delimiter) + common.Config.Delimiter
	} else {
		rw.NewSQL = strings.TrimSuffix(rw.SQL, common.Config.Delimiter) + common.Config.Delimiter
	}
	return rw
}

// RewriteStandard standard: 使用 vitess 提供的 String 功能将抽象语法树转写回 SQL，注意：这可能转写失败。
func (rw *Rewrite) RewriteStandard() *Rewrite {
	if _, err := sqlparser.Parse(rw.SQL); err == nil {
		rw.NewSQL = sqlparser.String(rw.Stmt)
	}
	return rw
}

// RewriteAlwaysTrue always true: 删除恒真条件
func (rw *Rewrite) RewriteAlwaysTrue() (reWriter *Rewrite) {
	array := NewNodeList(rw.Stmt)
	tNode := array.Head
	for {
		omitAlwaysTrue(tNode)
		tNode = tNode.Next
		if tNode == nil {
			break
		}
	}

	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// isAlwaysTrue 用于判断ComparisonExpr是否是恒真
func isAlwaysTrue(expr *sqlparser.ComparisonExpr) bool {
	if expr == nil {
		return true
	}

	var result bool
	switch expr.Operator {
	case "<>":
		expr.Operator = "!="
	case "<=>":
		expr.Operator = "="
	case ">=", "<=", "!=", "=", ">", "<":
	default:
		return false
	}

	var left []byte
	var right []byte

	// left
	switch l := expr.Left.(type) {
	case *sqlparser.SQLVal:
		left = l.Val
	default:
		return false
	}

	// right
	switch r := expr.Right.(type) {
	case *sqlparser.SQLVal:
		right = r.Val
	default:
		return false
	}

	switch expr.Operator {
	case "=":
		result = bytes.Equal(left, right)
	case "!=":
		result = !bytes.Equal(left, right)
	case ">":
		result = bytes.Compare(left, right) > 0
	case ">=":
		result = bytes.Compare(left, right) >= 0
	case "<":
		result = bytes.Compare(left, right) < 0
	case "<=":
		result = bytes.Compare(left, right) <= 0
	default:
		result = false
	}

	return result
}

// omitAlwaysTrue 移除AST中的恒真条件
func omitAlwaysTrue(node *NodeItem) {
	if node == nil {
		return
	}

	switch self := node.Self.(type) {
	case *sqlparser.Where:
		if self != nil {
			switch cond := self.Expr.(type) {
			case *sqlparser.ComparisonExpr:
				if isAlwaysTrue(cond) {
					self.Expr = nil
				}
			case *sqlparser.ParenExpr:
				if cond.Expr == nil {
					self.Expr = nil
				}
			}
		}
	case *sqlparser.ParenExpr:
		if self != nil {
			switch cond := self.Expr.(type) {
			case *sqlparser.ComparisonExpr:
				if isAlwaysTrue(cond) {
					self.Expr = nil
				}
			}
		}
	case *sqlparser.AndExpr:
		if self != nil {
			var tmp sqlparser.Expr
			isRightTrue := false
			isLeftTrue := false
			tmp = nil

			// 查看左树的情况
			switch l := self.Left.(type) {
			case *sqlparser.ComparisonExpr:
				if isAlwaysTrue(l) {
					self.Left = nil
					isLeftTrue = true
					tmp = self.Right
				}
			case *sqlparser.ParenExpr:
				if l.Expr == nil {
					self.Left = nil
					isLeftTrue = true
					tmp = self.Right
				}
			default:
				if l == nil {
					isLeftTrue = true
					tmp = self.Right
				}
			}

			// 查看右树的情况
			switch r := self.Right.(type) {
			case *sqlparser.ComparisonExpr:
				if isAlwaysTrue(r) {
					self.Right = nil
					isRightTrue = true
					tmp = self.Left
				}
			case *sqlparser.ParenExpr:
				if r.Expr == nil {
					self.Right = nil
					isRightTrue = true
					tmp = self.Left
				}
			default:
				if r == nil {
					isRightTrue = true
					tmp = self.Left
				}
			}

			if isRightTrue && isLeftTrue {
				tmp = nil
			} else if !isLeftTrue && !isRightTrue {
				return
			}

			// 根据类型开始替换节点
			switch l := node.Prev.Self.(type) {
			case *sqlparser.Where:
				l.Expr = tmp
			case *sqlparser.ParenExpr:
				l.Expr = tmp
			case *sqlparser.AndExpr:
				if l.Left == self {
					l.Left = tmp
				} else if l.Right == self {
					l.Right = tmp
				}
			case *sqlparser.OrExpr:
				if l.Left == self {
					l.Left = tmp
				} else if l.Right == self {
					l.Right = tmp
				}
			default:
				// 未匹配到对应数据类型则从链表中移除该节点
				err := node.Array.Remove(node.Prev)
				common.LogIfError(err, "")
			}

		}

	case *sqlparser.OrExpr:
		// 与AndExpr相同
		if self != nil {
			var tmp sqlparser.Expr
			isRightTrue := false
			isLeftTrue := false
			tmp = nil

			switch l := self.Left.(type) {
			case *sqlparser.ComparisonExpr:
				if isAlwaysTrue(l) {
					self.Left = nil
					isLeftTrue = true
					tmp = self.Right
				}
			case *sqlparser.ParenExpr:
				if l.Expr == nil {
					self.Left = nil
					isLeftTrue = true
					tmp = self.Right
				}
			default:
				if l == nil {
					isLeftTrue = true
					tmp = self.Right
				}
			}

			switch r := self.Right.(type) {
			case *sqlparser.ComparisonExpr:
				if isAlwaysTrue(r) {
					self.Right = nil
					isRightTrue = true
					tmp = self.Left
				}
			case *sqlparser.ParenExpr:
				if r.Expr == nil {
					self.Right = nil
					isRightTrue = true
					tmp = self.Left
				}
			default:
				if r == nil {
					isRightTrue = true
					tmp = self.Left
				}
			}

			if isRightTrue && isLeftTrue {
				tmp = nil
			} else if !isLeftTrue && !isRightTrue {
				return
			}

			switch l := node.Prev.Self.(type) {
			case *sqlparser.Where:
				l.Expr = tmp
			case *sqlparser.ParenExpr:
				l.Expr = tmp
			case *sqlparser.AndExpr:
				if l.Left == self {
					l.Left = tmp
				} else if l.Right == self {
					l.Right = tmp
				}
			case *sqlparser.OrExpr:
				if l.Left == self {
					l.Left = tmp
				} else if l.Right == self {
					l.Right = tmp
				}
			default:
				err := node.Array.Remove(node.Prev)
				common.LogIfError(err, "")
			}
		}
	}

	omitAlwaysTrue(node.Prev)
}

// RewriteCountStar countstar: 将COUNT(col)改写为COUNT(*)
// COUNT(DISTINCT col)不能替换为COUNT(*)
func (rw *Rewrite) RewriteCountStar() *Rewrite {
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch f := node.(type) {
		case *sqlparser.FuncExpr:
			if strings.ToLower(f.Name.String()) == "count" && len(f.Exprs) > 0 {
				switch colExpr := f.Exprs[0].(type) {
				case *sqlparser.AliasedExpr:
					switch col := colExpr.Expr.(type) {
					case *sqlparser.ColName:
						f.Exprs[0] = &sqlparser.StarExpr{TableName: col.Qualifier}
					}
				}
			}
		}
		return true, nil
	}, rw.Stmt)
	common.LogIfError(err, "")
	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// RewriteInnoDB InnoDB: 为未指定 Engine 的表默认添加 InnoDB 引擎，将其他存储引擎转为 InnoDB
func (rw *Rewrite) RewriteInnoDB() *Rewrite {
	switch create := rw.Stmt.(type) {
	case *sqlparser.DDL:
		if create.Action != "create" {
			return rw
		}

		if strings.Contains(strings.ToLower(create.TableSpec.Options), "engine=") {
			reg := regexp.MustCompile(`(?i)engine=[a-z]+`)
			create.TableSpec.Options = reg.ReplaceAllString(create.TableSpec.Options, "ENGINE=InnoDB ")
		} else {
			create.TableSpec.Options = " ENGINE=InnoDB " + create.TableSpec.Options
		}

	}

	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// RewriteAutoIncrement autoincrement: 将auto_increment设置为1
func (rw *Rewrite) RewriteAutoIncrement() *Rewrite {
	switch create := rw.Stmt.(type) {
	case *sqlparser.DDL:
		if create.Action != "create" || create.TableSpec == nil {
			return rw
		}
		if strings.Contains(strings.ToLower(create.TableSpec.Options), "auto_increment=") {
			reg := regexp.MustCompile(`(?i)auto_increment=[0-9]+`)
			create.TableSpec.Options = reg.ReplaceAllString(create.TableSpec.Options, "auto_increment=1 ")
		}
	}

	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// RewriteIntWidth intwidth: int 类型转为 int(10)，bigint 类型转为 bigint(20)
func (rw *Rewrite) RewriteIntWidth() *Rewrite {
	switch create := rw.Stmt.(type) {
	case *sqlparser.DDL:
		if create.Action != "create" || create.TableSpec == nil {
			return rw
		}
		for _, col := range create.TableSpec.Columns {
			switch col.Type.Type {
			case "int", "integer":
				if col.Type.Length != nil &&
					(string(col.Type.Length.Val) != "10" && string(col.Type.Length.Val) != "11") {
					col.Type.Length = sqlparser.NewIntVal([]byte("10"))
				}
			case "bigint":
				if col.Type.Length != nil && string(col.Type.Length.Val) != "20" || col.Type.Length == nil {
					col.Type.Length = sqlparser.NewIntVal([]byte("20"))
				}
			default:
			}
		}
	}

	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// RewriteStar2Columns star2columns: 对应COL.001，SELECT补全*指代的列名
func (rw *Rewrite) RewriteStar2Columns() *Rewrite {
	// 如果未配置mysql环境或从环境中获取失败，*不进行替换
	if common.Config.TestDSN.Disable || len(rw.Columns) == 0 {
		common.Log.Debug("(rw *Rewrite) RewriteStar2Columns(): Rewrite failed. TestDSN.Disable: %v, len(rw.Columns):%d",
			common.Config.TestDSN.Disable, len(rw.Columns))
		return rw
	}

	// 单张表 select * 不补全表名，避免SQL过长，多张表的 select tb1.*, tb2.* 需要补全表名
	var multiTable bool
	if len(rw.Columns) > 1 {
		multiTable = true
	} else {
		for db := range rw.Columns {
			if len(rw.Columns[db]) > 1 {
				multiTable = true
			}
		}
	}

	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Select:

			// select * 可能出现的情况：
			// 1. select * from tb;
			// 2. select * from tb1,tb2;
			// 3. select tb1.* from tb1;
			// 4. select tb1.*,tb2.col from tb1,tb2;
			// 5. select db.tb1.* from tb1;
			// 6. select db.tb1.*,db.tb2.col from db.tb1,db.tb2;

			newSelectExprs := make(sqlparser.SelectExprs, 0)
			for _, expr := range n.SelectExprs {
				switch e := expr.(type) {
				case *sqlparser.StarExpr:
					// 一般情况下最外层循环不会超过两层
					for _, tables := range rw.Columns {
						for _, cols := range tables {
							for _, col := range cols {
								var table string
								if multiTable {
									table = col.Table
								}
								newExpr := &sqlparser.AliasedExpr{
									Expr: &sqlparser.ColName{
										Metadata: nil,
										Name:     sqlparser.NewColIdent(col.Name),
										Qualifier: sqlparser.TableName{
											Name: sqlparser.NewTableIdent(table),
											// 因为不建议跨DB的查询，所以这里的db前缀将不进行补齐
											Qualifier: sqlparser.TableIdent{},
										},
									},
									As: sqlparser.ColIdent{},
								}

								if e.TableName.Name.IsEmpty() {
									// 情况1，2
									newSelectExprs = append(newSelectExprs, newExpr)
								} else {
									// 其他情况下只有在匹配表名的时候才会进行替换
									if e.TableName.Name.String() == col.Table {
										newSelectExprs = append(newSelectExprs, newExpr)
									}
								}
							}
						}
					}
				default:
					newSelectExprs = append(newSelectExprs, e)
				}
			}

			n.SelectExprs = newSelectExprs
		}
		return true, nil
	}, rw.Stmt)
	common.LogIfError(err, "")
	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// RewriteInsertColumns insertcolumns: 对应COL.002，INSERT补全列名
func (rw *Rewrite) RewriteInsertColumns() *Rewrite {

	switch insert := rw.Stmt.(type) {
	case *sqlparser.Insert:
		switch insert.Action {
		case "insert", "replace":
			if insert.Columns != nil {
				return rw
			}

			newColumns := make(sqlparser.Columns, 0)
			db := insert.Table.Qualifier.String()
			table := insert.Table.Name.String()
			// 支持INSERT/REPLACE INTO VALUES形式，支持INSERT/REPLACE INTO SELECT
			colCount := 0
			switch v := insert.Rows.(type) {
			case sqlparser.Values:
				if len(v) > 0 {
					colCount = len(v[0])
				}

			case *sqlparser.Select:
				if l := len(v.SelectExprs); l > 0 {
					colCount = l
				}
			}

			// 开始对ast进行替换，补全前N列
			counter := 0
			for dbName, tb := range rw.Columns {
				for tbName, cols := range tb {
					for _, col := range cols {
						// 只有全部列补全完成的时候才会替换ast
						if counter == colCount {
							insert.Columns = newColumns
							rw.NewSQL = sqlparser.String(rw.Stmt)
							return rw
						}

						if db != "" {
							// 指定了DB的时候，只能怼指定DB的列
							if db == dbName && table == tbName {
								newColumns = append(newColumns, sqlparser.NewColIdent(col.Name))
								counter++
							}
						} else {
							// 没有指定DB的时候，将column中的列按顺序往里怼
							if table == tbName {
								newColumns = append(newColumns, sqlparser.NewColIdent(col.Name))
								counter++
							}
						}
					}
				}
			}
		}
	}
	return rw
}

// RewriteHaving having: 对应CLA.013，使用 WHERE 过滤条件替代 HAVING
func (rw *Rewrite) RewriteHaving() *Rewrite {
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Select:
			if n.Having != nil {
				if n.Where == nil {
					// WHERE 条件为空直接用 HAVING 替代 WHERE 即可
					n.Where = n.Having
				} else {
					// WHERE 条件不为空，需要对已有的条件进行括号保护，然后再 AND+HAVING
					n.Where = &sqlparser.Where{
						Expr: &sqlparser.AndExpr{
							Left: &sqlparser.ParenExpr{
								Expr: n.Where.Expr,
							},
							Right: n.Having.Expr,
						},
					}
				}
				// 别忘了重置 HAVING 和 Where.Type
				n.Where.Type = "where"
				n.Having = nil
			}
		}
		return true, nil
	}, rw.Stmt)
	common.LogIfError(err, "")
	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// RewriteAddOrderByNull orderbynull: 对应 CLA.008，GROUP BY 无排序要求时添加 ORDER BY NULL
func (rw *Rewrite) RewriteAddOrderByNull() *Rewrite {
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Select:
			if n.GroupBy != nil && n.OrderBy == nil {
				n.OrderBy = sqlparser.OrderBy{
					&sqlparser.Order{
						Expr:      &sqlparser.NullVal{},
						Direction: "asc",
					},
				}
			}
		}
		return true, nil
	}, rw.Stmt)
	common.LogIfError(err, "")
	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// RewriteOr2Union or2union: 将 OR 查询转写为 UNION ALL TODO: 暂无对应 HeuristicRules
// TODO: https://sqlperformance.com/2014/09/sql-plan/rewriting-queries-improve-performance
func (rw *Rewrite) RewriteOr2Union() *Rewrite {
	return rw
}

// RewriteUnionAll unionall: 不介意重复数据的情况下使用 union all 替换 union
func (rw *Rewrite) RewriteUnionAll() *Rewrite {
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Union:
			n.Type = "union all"
		}
		return true, nil
	}, rw.Stmt)
	common.LogIfError(err, "")
	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// RewriteOr2In or2in: 同一列的 OR 过滤条件使用 IN() 替代，如果值有相等的会进行合并
func (rw *Rewrite) RewriteOr2In() *Rewrite {
	// 通过 AST 生成 node 的双向链表，链表顺序为书写顺序
	nodeList := NewNodeList(rw.Stmt)
	tNode := nodeList.First()

	for {
		tNode.or2in()
		if tNode.Next == nil {
			break
		}
		tNode = tNode.Next
	}

	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// or2in 用于将 or 转换成 in
func (node *NodeItem) or2in() {
	if node == nil || node.Self == nil {
		return
	}

	switch selfNode := node.Self.(type) {
	case *sqlparser.OrExpr:
		newExpr := mergeExprs(selfNode.Left, selfNode.Right)
		if newExpr != nil {
			// or 自身两个节点可以合并的情况下，将父节点中的 expr 替换成新的
			switch pre := node.Prev.Self.(type) {
			case *sqlparser.OrExpr:
				if pre.Left == node.Self {
					node.Self = newExpr
					pre.Left = newExpr
				} else if pre.Right == node.Self {
					node.Self = newExpr
					pre.Right = newExpr
				}
			case *sqlparser.AndExpr:
				if pre.Left == node.Self {
					node.Self = newExpr
					pre.Left = newExpr
				} else if pre.Right == node.Self {
					node.Self = newExpr
					pre.Right = newExpr
				}
			case *sqlparser.Where:
				node.Self = newExpr
				pre.Expr = newExpr
			case *sqlparser.ParenExpr:
				// 如果 SQL 书写中带了括号，暂不会进行跨括号的合并
				// TODO: 无意义括号打平，加个 rewrite rule
				node.Self = newExpr
				pre.Expr = newExpr
			}
		} else {
			// or 自身两个节点如不可以合并，则检测是否可以与父节点合并
			// 与父节点的合并不能跨越and、括号等，可能会改变语义
			// 检查自身左右节点是否能与上层节点中合并，or 只能与 or 合并
			switch pre := node.Prev.Self.(type) {
			case *sqlparser.OrExpr:
				// AST 中如果出现复合条件，则一定在左树，所以只需要判断左边就可以
				if pre.Left == selfNode {
					switch n := pre.Right.(type) {
					case *sqlparser.ComparisonExpr:
						newLeftExpr := mergeExprs(selfNode.Left, n)
						newRightExpr := mergeExprs(selfNode.Right, n)

						// newLeftExpr 与 newRightExpr 一定有一个是 nil，
						// 否则说明该 orExpr 下的两个节点可合并，可以通过最后的向前递归合并 pre 节点中的 expr
						if newLeftExpr == nil || newRightExpr == nil {
							if newLeftExpr != nil {
								pre.Right = newLeftExpr
								pre.Left = selfNode.Right
								err := node.Array.Remove(node)
								common.LogIfError(err, "")
							}

							if newRightExpr != nil {
								pre.Right = newRightExpr
								pre.Left = selfNode.Left
								err := node.Array.Remove(node)
								common.LogIfError(err, "")
							}
						}
					}
				}
			}
		}
	}

	// 逆向合并由更改 AST 后产生的新的可合并节点
	node.Prev.or2in()
}

// mergeExprs 将两个属于同一个列的 ComparisonExpr 合并成一个，如果不能合并则返回 nil
func mergeExprs(left, right sqlparser.Expr) *sqlparser.ComparisonExpr {
	// 用于对比两个列是否相同
	colInLeft := ""
	colInRight := ""
	lOperator := ""
	rOperator := ""

	// 用于存放 expr 左右子树中的值
	var values []sqlparser.SQLNode

	// SQL 中使用到的列
	var colName *sqlparser.ColName

	// 左子树
	switch l := left.(type) {
	case *sqlparser.ComparisonExpr:
		// 获取列名
		colName, colInLeft = getColumnName(l.Left)
		// 获取值
		if colInLeft != "" {
			switch v := l.Right.(type) {
			case *sqlparser.SQLVal, sqlparser.ValTuple, *sqlparser.BoolVal, *sqlparser.NullVal:
				values = append(values, v)
			}
		}
		// 获取 operator
		lOperator = l.Operator
	default:
		return nil
	}

	// 右子树
	switch r := right.(type) {
	case *sqlparser.ComparisonExpr:
		// 获取列名
		if colName.Name.String() != "" {
			common.Log.Warn("colName shouldn't has value, but now it's %s", colName.Name.String())
		}
		colName, colInRight = getColumnName(r.Left)
		// 获取值
		if colInRight != "" {
			switch v := r.Right.(type) {
			case *sqlparser.SQLVal, sqlparser.ValTuple, *sqlparser.BoolVal, *sqlparser.NullVal:
				values = append(values, v)
			}
		}
		// 获取 operator
		rOperator = r.Operator
	default:
		return nil
	}

	// operator 替换，用于在之后判断是否可以合并
	switch lOperator {
	case "in", "=":
		lOperator = "="
	default:
		return nil
	}

	switch rOperator {
	case "in", "=":
		rOperator = "="
	default:
		return nil
	}

	// 不匹配则返回
	if colInLeft == "" || colInLeft != colInRight ||
		lOperator == "" || lOperator != rOperator {
		return nil
	}

	// 合并左右子树的值
	newValTuple := make(sqlparser.ValTuple, 0)
	for _, v := range values {
		switch v := v.(type) {
		case *sqlparser.SQLVal:
			newValTuple = append(newValTuple, v)
		case *sqlparser.BoolVal:
			newValTuple = append(newValTuple, v)
		case *sqlparser.NullVal:
			newValTuple = append(newValTuple, v)
		case sqlparser.ValTuple:
			newValTuple = append(newValTuple, v...)
		}
	}

	// 去 expr 中除重复的 value,
	newValTuple = removeDup(newValTuple...)
	newExpr := &sqlparser.ComparisonExpr{
		Operator: "in",
		Left:     colName,
		Right:    newValTuple,
	}
	// 如果只有一个值则是一个等式，没有必要转写成 in
	if len(newValTuple) == 1 {
		newExpr = &sqlparser.ComparisonExpr{
			Operator: lOperator,
			Left:     colName,
			Right:    newValTuple[0],
		}
	}

	return newExpr
}

// removeDup 清除 sqlparser.ValTuple 中重复的值
func removeDup(vt ...sqlparser.Expr) sqlparser.ValTuple {
	uni := make(sqlparser.ValTuple, 0)
	m := make(map[string]sqlparser.SQLNode)

	for _, value := range vt {
		switch v := value.(type) {
		case *sqlparser.SQLVal:
			// Type:Val, 冒号用于分隔 Type 和 Val，防止两种不同类型拼接后出现同一个值
			if _, ok := m[string(v.Type)+":"+sqlparser.String(v)]; !ok {
				uni = append(uni, v)
				m[string(v.Type)+":"+sqlparser.String(v)] = v
			}
		case *sqlparser.BoolVal:
			if _, ok := m[sqlparser.String(v)]; !ok {
				uni = append(uni, v)
				m[sqlparser.String(v)] = v
			}
		case *sqlparser.NullVal:
			if _, ok := m[sqlparser.String(v)]; !ok {
				uni = append(uni, v)
				m[sqlparser.String(v)] = v
			}
		case sqlparser.ValTuple:
			for _, val := range removeDup(v...) {
				switch v := val.(type) {
				case *sqlparser.SQLVal:
					if _, ok := m[string(v.Type)+":"+sqlparser.String(v)]; !ok {
						uni = append(uni, v)
						m[string(v.Type)+":"+sqlparser.String(v)] = v
					}
				case *sqlparser.BoolVal:
					if _, ok := m[sqlparser.String(v)]; !ok {
						uni = append(uni, v)
						m[sqlparser.String(v)] = v
					}
				case *sqlparser.NullVal:
					if _, ok := m[sqlparser.String(v)]; !ok {
						uni = append(uni, v)
						m[sqlparser.String(v)] = v
					}
				}
			}
		}
	}

	return uni
}

// RewriteInNull innull: TODO: 对应ARG.004
func (rw *Rewrite) RewriteInNull() *Rewrite {
	return rw
}

// RewriteRmParenthesis rmparenthesis: 去除无意义的括号
func (rw *Rewrite) RewriteRmParenthesis() *Rewrite {
	rw.rmParenthesis()
	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// rmParenthesis 用于语出无用的括号
func (rw *Rewrite) rmParenthesis() {
	continueFlag := false
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch node := node.(type) {
		case *sqlparser.Where:
			if node == nil {
				return true, nil
			}
			switch paren := node.Expr.(type) {
			case *sqlparser.ParenExpr:
				switch paren.Expr.(type) {
				case *sqlparser.ComparisonExpr:
					node.Expr = paren.Expr
					continueFlag = true
				}
			}

		case *sqlparser.ParenExpr:
			switch paren := node.Expr.(type) {
			case *sqlparser.ParenExpr:
				switch paren.Expr.(type) {
				case *sqlparser.ComparisonExpr:
					node.Expr = paren.Expr
					continueFlag = true
				}
			}

		case *sqlparser.AndExpr:
			switch left := node.Left.(type) {
			case *sqlparser.ParenExpr:
				switch inner := left.Expr.(type) {
				case *sqlparser.ComparisonExpr:
					node.Left = inner
					continueFlag = true
				}
			}

			switch right := node.Right.(type) {
			case *sqlparser.ParenExpr:
				switch inner := right.Expr.(type) {
				case *sqlparser.ComparisonExpr:
					node.Right = inner
					continueFlag = true
				}
			}

		case *sqlparser.OrExpr:
			switch left := node.Left.(type) {
			case *sqlparser.ParenExpr:
				switch inner := left.Expr.(type) {
				case *sqlparser.ComparisonExpr:
					node.Left = inner
					continueFlag = true
				}
			}

			switch right := node.Right.(type) {
			case *sqlparser.ParenExpr:
				switch inner := right.Expr.(type) {
				case *sqlparser.ComparisonExpr:
					node.Right = inner
					continueFlag = true
				}
			}
		}
		return true, nil
	}, rw.Stmt)
	common.LogIfError(err, "")
	// 本层的修改可能使得原本不符合条件的括号变为无意义括号
	// 每次修改都需要再过滤一遍语法树
	if continueFlag {
		rw.rmParenthesis()
	} else {
		return
	}
}

// RewriteRemoveDMLOrderBy dmlorderby: 对应 RES.004，删除无 LIMIT 条件时 UPDATE, DELETE 中包含的 ORDER BY
func (rw *Rewrite) RewriteRemoveDMLOrderBy() *Rewrite {
	switch st := rw.Stmt.(type) {
	case *sqlparser.Update:
		err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch n := node.(type) {
			case *sqlparser.Select:
				if n.OrderBy != nil && n.Limit == nil {
					n.OrderBy = nil
				}
				return false, nil
			}
			return true, nil
		}, rw.Stmt)
		common.LogIfError(err, "")
		if st.OrderBy != nil && st.Limit == nil {
			st.OrderBy = nil
		}
	case *sqlparser.Delete:
		err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch n := node.(type) {
			case *sqlparser.Select:
				if n.OrderBy != nil && n.Limit == nil {
					n.OrderBy = nil
				}
				return false, nil
			}
			return true, nil
		}, rw.Stmt)
		common.LogIfError(err, "")
		if st.OrderBy != nil && st.Limit == nil {
			st.OrderBy = nil
		}
	}
	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// RewriteGroupByConst 对应CLA.004，将GROUP BY CONST替换为列名
// TODO:
func (rw *Rewrite) RewriteGroupByConst() *Rewrite {
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Select:
			groupByCol := false
			if n.GroupBy != nil {
				for _, group := range n.GroupBy {
					switch group.(type) {
					case *sqlparser.SQLVal:
					default:
						groupByCol = true
					}
				}
				if !groupByCol {
					// TODO: 这里只是去掉了GROUP BY并没解决问题
					n.GroupBy = nil
				}
			}
		}
		return true, nil
	}, rw.Stmt)
	common.LogIfError(err, "")
	rw.NewSQL = sqlparser.String(rw.Stmt)
	return rw
}

// RewriteSubQuery2Join 将 subquery 转写成 join
func (rw *Rewrite) RewriteSubQuery2Join() *Rewrite {
	var err error
	// 如果未配置 mysql 环境或从环境中获取失败
	if common.Config.TestDSN.Disable || len(rw.Columns) == 0 {
		common.Log.Debug("(rw *Rewrite) RewriteSubQuery2Join(): Rewrite failed. TestDSN.Disable: %v, len(rw.Columns):%d",
			common.Config.TestDSN.Disable, len(rw.Columns))
		return rw
	}

	if rw.NewSQL == "" {
		rw.NewSQL = sqlparser.String(rw.Stmt)
	}

	// query backup
	backup := rw.NewSQL
	var subQueryList []string
	err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch sub := node.(type) {
		case sqlparser.SelectStatement:
			subStr := sqlparser.String(sub)
			if strings.HasPrefix(subStr, "(") {
				subStr = subStr[1 : len(subStr)-1]
			}
			subQueryList = append(subQueryList, subStr)
		}
		return true, nil
	}, rw.Stmt)
	common.LogIfError(err, "")
	if length := len(subQueryList); length > 1 {
		lastResult := ""
		for i := length - 1; i > 0; i-- {
			if lastResult == "" {
				lastResult, err = rw.sub2Join(subQueryList[i-1], subQueryList[i])
			} else {
				// 将subquery的部分替换成上次合并的结果
				subQueryList[i-1] = strings.Replace(subQueryList[i-1], subQueryList[i], lastResult, -1)
				lastResult, err = rw.sub2Join(subQueryList[i-1], lastResult)
			}

			if err != nil {
				common.Log.Error("RewriteSubQuery2Join Error: %v", err)
				return rw
			}
		}
		rw.NewSQL = lastResult
	} else if length == 1 {
		var newSQL string
		newSQL, err = rw.sub2Join(rw.NewSQL, subQueryList[0])
		if err == nil {
			rw.NewSQL = newSQL
		}
	}

	// 因为这个修改不会直接修改rw.stmt，所以需要将rw.stmt也更新一下
	newStmt, err := sqlparser.Parse(rw.NewSQL)
	if err != nil {
		rw.NewSQL = backup
		rw.Stmt, _ = sqlparser.Parse(backup)
	} else {
		rw.Stmt = newStmt
	}

	return rw
}

// sub2Join 将 subquery 转写成 join
func (rw *Rewrite) sub2Join(parent, sub string) (string, error) {
	// 只处理SelectStatement
	if sqlparser.Preview(parent) != sqlparser.StmtSelect || sqlparser.Preview(sub) != sqlparser.StmtSelect {
		return "", nil
	}

	// 如果子查询不属于parent,则不处理
	if !strings.Contains(parent, sub) {
		return "", nil
	}

	// 解析外层SQL语法树
	stmt, err := sqlparser.Parse(parent)
	if err != nil {
		common.Log.Warn("(rw *Rewrite) RewriteSubQuery2Join() sub2Join sql `%s` parsed error: %v", parent, err)
		return "", err
	}

	switch stmt.(type) {
	case sqlparser.SelectStatement:
	default:
		common.Log.Debug("Query `%s` not select statement.", parent)
		return "", nil
	}

	// 解析子查询语法树
	subStmt, err := sqlparser.Parse(sub)
	if err != nil {
		common.Log.Warn("(rw *Rewrite) RewriteSubQuery2Join() sub2Join sql `%s` parsed error: %v", sub, err)
		return "", err
	}

	// 获取外部SQL用到的表
	stmtMeta := GetTableFromExprs(stmt.(*sqlparser.Select).From)
	// 获取内部SQL用到的表
	subMeta := GetTableFromExprs(subStmt.(*sqlparser.Select).From)

	// 处理关联条件
	err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch p := node.(type) {
		case *sqlparser.ComparisonExpr:
			// a in (select * from tb)
			switch subquery := p.Right.(type) {
			case *sqlparser.Subquery:

				// 获取左边的列
				var leftColumn *sqlparser.ColName

				switch l := p.Left.(type) {
				case *sqlparser.ColName:
					leftColumn = l
				default:
					return false, nil
				}

				// 用于存放获取的subquery中的列，有且只有一个
				var rightColumn sqlparser.SQLNode

				// 对subquery中的列进行替换
				switch subSelectStmt := subquery.Select.(type) {
				case *sqlparser.Select:
					cachingOperator := p.Operator

					rightColumn = subSelectStmt.SelectExprs[0]

					rightCol, _ := getColumnName(rightColumn.(*sqlparser.AliasedExpr).Expr)
					if rightCol != nil {
						// 将subquery替换为等值条件
						p.Operator = "="

						// selectExpr 信息补齐
						var newExprs []sqlparser.SelectExpr
						err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
							switch col := node.(type) {
							case *sqlparser.StarExpr:
								if col.TableName.Name.IsEmpty() {
									for dbName, db := range stmtMeta {
										for tbName := range db.Table {

											col.TableName.Name = sqlparser.NewTableIdent(tbName)
											if dbName != "" {
												col.TableName.Qualifier = sqlparser.NewTableIdent(dbName)
											}

											newExprs = append(newExprs, col)
										}
									}
								}
							case *sqlparser.AliasedExpr:
								switch n := col.Expr.(type) {
								case *sqlparser.ColName:
									col.Expr = columnFromWhere(n, stmtMeta, rw.Columns)
								}
							}
							return true, nil
						}, stmt.(*sqlparser.Select).SelectExprs)
						common.LogIfError(err, "")

						// 原节点列信息补齐
						p.Left = columnFromWhere(leftColumn, stmtMeta, rw.Columns)

						// 将子查询中的节点上提，补充前缀信息
						p.Right = columnFromWhere(rightCol, subMeta, rw.Columns)

						// subquery Where条件中的列信息补齐
						subWhereExpr := subStmt.(*sqlparser.Select).Where
						err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
							switch n := node.(type) {
							case *sqlparser.ComparisonExpr:
								switch left := n.Left.(type) {
								case *sqlparser.ColName:
									n.Left = columnFromWhere(left, subMeta, rw.Columns)
								}

								switch right := n.Right.(type) {
								case *sqlparser.ColName:
									n.Right = columnFromWhere(right, subMeta, rw.Columns)
								}
							}
							return true, nil
						}, subWhereExpr)
						common.LogIfError(err, "")
						// 如果 subquery 中存在 Where 条件，怼在 parent 的 where 中后面
						if subWhereExpr != nil {
							if stmt.(*sqlparser.Select).Where != nil {
								stmt.(*sqlparser.Select).Where.Expr = &sqlparser.AndExpr{
									Left:  stmt.(*sqlparser.Select).Where.Expr,
									Right: subWhereExpr.Expr,
								}
							} else {
								stmt.(*sqlparser.Select).Where = subWhereExpr
							}
						}

						switch cachingOperator {
						case "in":
							// 将表以 inner join 的形式追加到 parent 的 from 中
							var newTables []sqlparser.TableExpr
							for _, subExpr := range subStmt.(*sqlparser.Select).From {
								has := false
								for _, expr := range stmt.(*sqlparser.Select).From {
									if reflect.DeepEqual(expr, subExpr) {
										has = true
									}
								}
								if !has {
									newTables = append(newTables, subExpr)
								}
							}
							stmt.(*sqlparser.Select).From = append(stmt.(*sqlparser.Select).From, newTables...)
						case "not in":
							// 将表以left join 的形式 追加到 parent 的 from 中
							// TODO:
						}
					}

				}
			}
		}
		return true, nil
	}, stmt)
	common.LogIfError(err, "")
	newSQL := sqlparser.String(stmt)
	return newSQL, nil
}

// columnFromWhere 获取列是来自哪个表，并补充前缀
func columnFromWhere(col *sqlparser.ColName, meta common.Meta, columns common.TableColumns) *sqlparser.ColName {

	for dbName, db := range meta {
		for tbName := range db.Table {
			for _, tables := range columns {
				for _, columns := range tables {
					for _, column := range columns {
						if strings.EqualFold(col.Name.String(), column.Name) {
							if col.Qualifier.Name.IsEmpty() && tbName == column.Table {
								col.Qualifier.Name = sqlparser.NewTableIdent(column.Table)
								return col
							}
							if (dbName == "" && tbName == column.Table) || (tbName == column.Table && dbName == column.DB) {
								col.Qualifier.Name = sqlparser.NewTableIdent(column.Table)
								if dbName != "" {
									col.Qualifier.Qualifier = sqlparser.NewTableIdent(column.DB)
								}
								return col
							}
						}
					}
				}
			}

		}
	}
	return col
}

// RewriteJoin2SubQuery join2sub: TODO:
// https://mariadb.com/kb/en/library/subqueries-and-joins/
func (rw *Rewrite) RewriteJoin2SubQuery() *Rewrite {
	return rw
}

// RewriteDistinctStar distinctstar: 对应DIS.003，将多余的`DISTINCT *`删除
func (rw *Rewrite) RewriteDistinctStar() *Rewrite {
	// 注意：这里并未对表是否有主键做检查，按照我们的SQL编程规范，一张表必须有主键
	switch rw.Stmt.(type) {
	case *sqlparser.Select:
		meta := GetMeta(rw.Stmt, nil)
		for _, m := range meta {
			if len(m.Table) == 1 {
				// distinct tbl.*, distinct *, count(distinct *)
				re := regexp.MustCompile(`(?i)((distinct\s*\*)|(distinct\s+[0-9a-z_` + "`" + `]*\.\*))`)
				if re.MatchString(rw.SQL) {
					rw.NewSQL = re.ReplaceAllString(rw.SQL, "*")
				}
			}
			break
		}
	}
	if rw.NewSQL == "" {
		rw.NewSQL = rw.SQL
	}
	rw.Stmt, _ = sqlparser.Parse(rw.NewSQL)
	return rw
}

// RewriteTruncate truncate: DELETE 全表修改为 TRUNCATE TABLE
func (rw *Rewrite) RewriteTruncate() *Rewrite {
	switch n := rw.Stmt.(type) {
	case *sqlparser.Delete:
		meta := GetMeta(rw.Stmt, nil)
		if len(meta) == 1 && n.Where == nil {
			for _, db := range meta {
				for _, tbl := range db.Table {
					rw.NewSQL = "truncate table " + tbl.TableName
				}
			}
		}
	}
	return rw
}

// RewriteDML2Select dml2select: DML 转成 SELECT，兼容低版本的 EXPLAIN
func (rw *Rewrite) RewriteDML2Select() *Rewrite {
	if rw.Stmt == nil {
		return rw
	}

	switch stmt := rw.Stmt.(type) {
	case *sqlparser.Select:
		rw.NewSQL = rw.SQL
	case *sqlparser.Delete: // Multi DELETE not support yet.
		rw.NewSQL = delete2Select(stmt)
	case *sqlparser.Insert:
		rw.NewSQL = insert2Select(stmt)
	case *sqlparser.Update: // Multi UPDATE not support yet.
		rw.NewSQL = update2Select(stmt)
	}
	rw.Stmt, _ = sqlparser.Parse(rw.NewSQL)
	return rw
}

// delete2Select 将 Delete 语句改写成 Select
func delete2Select(stmt *sqlparser.Delete) string {
	newSQL := &sqlparser.Select{
		SelectExprs: []sqlparser.SelectExpr{
			new(sqlparser.StarExpr),
		},
		From:    stmt.TableExprs,
		Where:   stmt.Where,
		OrderBy: stmt.OrderBy,
	}
	return sqlparser.String(newSQL)
}

// update2Select 将 Update 语句改写成 Select
func update2Select(stmt *sqlparser.Update) string {
	newSQL := &sqlparser.Select{
		SelectExprs: []sqlparser.SelectExpr{
			new(sqlparser.StarExpr),
		},
		From:    stmt.TableExprs,
		Where:   stmt.Where,
		OrderBy: stmt.OrderBy,
		Limit:   stmt.Limit,
	}
	return sqlparser.String(newSQL)
}

// insert2Select 将 Insert 语句改写成 Select
func insert2Select(stmt *sqlparser.Insert) string {
	switch row := stmt.Rows.(type) {
	// 如果insert包含子查询，只需要explain该子树
	case *sqlparser.Select, *sqlparser.Union, *sqlparser.ParenSelect:
		return sqlparser.String(row)
	}

	return "select 1 from DUAL"
}

// AlterAffectTable 获取ALTER影响的库表名，返回：`db`.`table`
func AlterAffectTable(stmt sqlparser.Statement) string {
	switch n := stmt.(type) {
	case *sqlparser.DDL:
		tableName := strings.ToLower(n.Table.Name.String())
		dbName := strings.ToLower(n.Table.Qualifier.String())
		if tableName != "" && tableName != "dual" {
			if dbName == "" {
				return "`" + tableName + "`"
			}

			return "`" + dbName + "`.`" + tableName + "`"
		}
	}
	return ""
}

// MergeAlterTables mergealter: 将同一张表的多条 ALTER 语句合成一条 ALTER 语句
// @input: sql, alter string
// @output: [[db.]table]sql, 如果找不到 DB，key 为表名；如果找得到 DB，key 为 db.table
func MergeAlterTables(sqls ...string) map[string]string {
	alterSQLs := make(map[string][]string)
	mergedAlterStr := make(map[string]string)

	// table/column/index name can be quoted in back ticks
	backTicks := "(`[^\\s]*`)"

	alterExp := regexp.MustCompile(`(?i)alter\s*table\s*(` + backTicks + `|([^\s]*))\s*`)   // ALTER TABLE
	renameExp := regexp.MustCompile(`(?i)rename\s*table\s*(` + backTicks + `|([^\s]*))\s*`) // RENAME TABLE
	// CREATE [UNIQUE|FULLTEXT|SPATIAL|PRIMARY] [KEY|INDEX] idx_name ON tbl_name
	createIndexExp := regexp.MustCompile(`(?i)create((unique)|(fulltext)|(spatial)|(primary)|(\s*)\s*)((index)|(key))\s*`)
	indexNameExp := regexp.MustCompile(`(?i)(` + backTicks + `|([^\s]*))\s*`)

	for _, sql := range sqls {
		sql = strings.Trim(sql, common.Config.Delimiter)
		stmts, err := TiParse(sql, "", "")
		if err != nil {
			common.Log.Warn(err.Error())
			continue
		}
		// stmt, _ := sqlparser.Parse(sql)
		alterSQL := ""
		dbName := ""
		tableName := ""
		for _, stmt := range stmts {
			switch n := stmt.(type) {
			case *ast.AlterTableStmt:
				// 注意: 表名和库名不区分大小写
				tableName = n.Table.Name.L
				dbName = n.Table.Schema.L
				if alterExp.MatchString(sql) {
					common.Log.Debug("rename alterExp: ALTER %v %v", tableName, alterExp.ReplaceAllString(sql, ""))
					alterSQL = fmt.Sprint(alterExp.ReplaceAllString(sql, ""))
				}
			case *ast.CreateIndexStmt:
				tableName = n.Table.Name.L
				dbName = n.Table.Schema.L

				buf := createIndexExp.ReplaceAllString(sql, "")
				idxName := strings.TrimSpace(indexNameExp.FindString(buf))
				buf = string([]byte(buf)[strings.Index(buf, "("):])
				common.Log.Error(buf)
				common.Log.Debug("alter createIndexExp: ALTER %v ADD INDEX %v %v", tableName, "ADD INDEX", idxName, buf)
				alterSQL = fmt.Sprint("ADD INDEX", " "+idxName+" ", buf)
			case *ast.RenameTableStmt:
				// 注意: 表名和库名不区分大小写
				tableName = n.OldTable.Name.L
				dbName = n.OldTable.Schema.L
				if alterExp.MatchString(sql) {
					common.Log.Debug("rename alterExp: ALTER %v %v", tableName, alterExp.ReplaceAllString(sql, ""))
					alterSQL = fmt.Sprint(alterExp.ReplaceAllString(sql, ""))
				} else if renameExp.MatchString(sql) {
					common.Log.Debug("rename renameExp: ALTER %v %v", tableName, alterExp.ReplaceAllString(sql, ""))
					alterSQL = fmt.Sprint(alterExp.ReplaceAllString(sql, ""))
				} else {
					common.Log.Warn("rename not match: ALTER %v %v", tableName, sql)
				}
			default:
			}
		}

		if alterSQL != "" && tableName != "" && tableName != "dual" {
			if dbName == "" {
				alterSQLs["`"+tableName+"`"] = append(alterSQLs["`"+tableName+"`"], alterSQL)
			} else {
				alterSQLs["`"+dbName+"`.`"+tableName+"`"] = append(alterSQLs["`"+dbName+"`.`"+tableName+"`"], alterSQL)
			}
		}
	}
	for k, v := range alterSQLs {
		mergedAlterStr[k] = fmt.Sprintln("ALTER TABLE", k, strings.Join(v, ", "), common.Config.Delimiter)
	}
	return mergedAlterStr
}

// RewriteRuleMatch 检查重写规则是否生效
func RewriteRuleMatch(name string) bool {
	for _, r := range common.Config.RewriteRules {
		if r == name {
			return true
		}
	}
	return false
}
