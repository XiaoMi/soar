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
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/XiaoMi/soar/ast"
	"github.com/XiaoMi/soar/common"
	"github.com/XiaoMi/soar/database"

	"github.com/gedex/inflector"
	"github.com/percona/go-mysql/query"
	tidb "github.com/pingcap/tidb/ast"
	"github.com/pingcap/tidb/mysql"
	"github.com/pingcap/tidb/types"
	"vitess.io/vitess/go/vt/sqlparser"
)

// RuleOK OK
func (q *Query4Audit) RuleOK() Rule {
	return HeuristicRules["OK"]
}

// RuleImplicitAlias ALI.001
func (q *Query4Audit) RuleImplicitAlias() Rule {
	var rule = q.RuleOK()
	tkns := ast.Tokenizer(q.Query)
	if len(tkns) == 0 {
		return rule
	}
	if tkns[0].Type != sqlparser.SELECT {
		return rule
	}
	for i, tkn := range tkns {
		if tkn.Type == sqlparser.ID && i+1 < len(tkns) && tkn.Type == tkns[i+1].Type {
			rule = HeuristicRules["ALI.001"]
			break
		}
	}
	return rule
}

// RuleStarAlias ALI.002
func (q *Query4Audit) RuleStarAlias() Rule {
	var rule = q.RuleOK()
	re := regexp.MustCompile(`(?i)(\*\s+as\b)`)
	if re.FindString(q.Query) != "" {
		rule = HeuristicRules["ALI.002"]
	}
	return rule
}

// RuleSameAlias ALI.003
func (q *Query4Audit) RuleSameAlias() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case *sqlparser.AliasedExpr:
			switch n := expr.Expr.(type) {
			case *sqlparser.ColName:
				if n.Name.String() == expr.As.String() {
					rule = HeuristicRules["ALI.003"]
					return false, nil
				}
			}
		case *sqlparser.AliasedTableExpr:
			switch n := expr.Expr.(type) {
			case sqlparser.TableName:
				if n.Name.String() == expr.As.String() {
					rule = HeuristicRules["ALI.003"]
					return false, nil
				}
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RulePrefixLike ARG.001
func (q *Query4Audit) RulePrefixLike() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case *sqlparser.ComparisonExpr:
			if expr.Operator == "like" {
				switch sqlval := expr.Right.(type) {
				case *sqlparser.SQLVal:
					// prefix like with '%', '_'
					if sqlval.Type == 0 && (sqlval.Val[0] == 0x25 || sqlval.Val[0] == 0x5f) {
						rule = HeuristicRules["ARG.001"]
						return false, nil
					}
				}
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleEqualLike ARG.002
func (q *Query4Audit) RuleEqualLike() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case *sqlparser.ComparisonExpr:
			if expr.Operator == "like" {
				switch sqlval := expr.Right.(type) {
				case *sqlparser.SQLVal:
					// not start with '%', '_' && not end with '%', '_'
					if sqlval.Type == 0 {
						if sqlval.Val[0] != 0x25 &&
							sqlval.Val[0] != 0x5f &&
							sqlval.Val[len(sqlval.Val)-1] != 0x5f &&
							sqlval.Val[len(sqlval.Val)-1] != 0x25 {
							rule = HeuristicRules["ARG.002"]
							return false, nil
						}
					} else {
						rule = HeuristicRules["ARG.002"]
						return false, nil
					}
				}
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleImplicitConversion ARG.003
// 隐式类型转换检查：该项检查一定是在开启测试环境或线上环境情境下下进行的
func (idxAdv *IndexAdvisor) RuleImplicitConversion() Rule {
	/*
	* 两个参数至少有一个是 NULL 时，比较的结果也是 NULL，例外是使用 <=> 对两个 NULL 做比较时会返回 1，这两种情况都不需要做类型转换
	* 两个参数都是字符串，会按照字符串来比较，不做类型转换
	* 两个参数都是整数，按照整数来比较，不做类型转换
	* 十六进制的值和非数字做比较时，会被当做二进制串
	* 有一个参数是 TIMESTAMP 或 DATETIME，并且另外一个参数是常量，常量会被转换为 timestamp
	* 有一个参数是 decimal 类型，如果另外一个参数是 decimal 或者整数，会将整数转换为 decimal 后进行比较，如果另外一个参数是浮点数，则会把 decimal 转换为浮点数进行比较
	* 所有其他情况下，两个参数都会被转换为浮点数再进行比较
	 */
	rule := HeuristicRules["OK"]
	// 未开启测试环境不进行检查
	if common.Config.TestDSN.Disable {
		return rule
	}

	var content string
	conditions := ast.FindAllCondition(idxAdv.Ast)
	for _, cond := range conditions {
		var colList []*common.Column
		var values []*sqlparser.SQLVal

		// condition 左右两侧有且只有如下几种可能：
		// 1. 左列 & 右列
		// 2. 左列 & 右值(含函数)          (或相反)
		// 3. 左值(含函数) & 右值(含函数)	 (无需关注)
		switch node := cond.(type) {
		case *sqlparser.ComparisonExpr:
			// 获取condition左侧的信息
			switch nLeft := node.Left.(type) {
			case *sqlparser.SQLVal, *sqlparser.ValTuple:
				err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
					switch val := node.(type) {
					case *sqlparser.SQLVal:
						values = append(values, val)
					}
					return true, nil
				}, nLeft)
				common.LogIfError(err, "")

			case *sqlparser.ColName:
				left := &common.Column{Name: nLeft.Name.String()}
				if !nLeft.Qualifier.Name.IsEmpty() {
					left.Table = nLeft.Qualifier.Name.String()
				}
				colList = append(colList, left)
			}

			// 获取condition右侧的信息
			switch nRight := node.Right.(type) {
			case *sqlparser.SQLVal, *sqlparser.ValTuple:
				err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
					switch val := node.(type) {
					case *sqlparser.SQLVal:
						values = append(values, val)
					}
					return true, nil
				}, nRight)
				common.LogIfError(err, "")

			case *sqlparser.ColName:
				right := &common.Column{Name: nRight.Name.String()}
				if !nRight.Qualifier.Name.IsEmpty() {
					right.Table = nRight.Qualifier.Name.String()
				}
				colList = append(colList, right)
			}

			if len(colList) == 0 {
				continue
			}

			// 补全列信息
			colList = CompleteColumnsInfo(idxAdv.Ast, colList, idxAdv.vEnv)

			// 列与列比较
			if len(colList) == 2 {
				// 列信息补全后如果依然没有表信息，说明在该数据库中不存在该列
				// 如果列信息获取异常，可能会存在无法获取到数据类型的情况，对于这种情况将不会给予建议。
				needBreak := false
				for _, col := range colList {
					if col.Table == "" {
						common.Log.Warning("Column %s not exists", col.Name)
						needBreak = true
					}

					if col.DataType == "" {
						common.Log.Warning("Can't get column %s data type", col.Name)
						needBreak = true
					}

				}

				if needBreak {
					break
				}

				// 检查数据类型不一致导致的隐式数据转换
				type1 := common.GetDataTypeBase(colList[0].DataType)
				type2 := common.GetDataTypeBase(colList[1].DataType)
				common.Log.Debug("DataType: `%s`.`%s` (%s) VS `%s`.`%s` (%s)",
					colList[0].Table, colList[0].Name, type1,
					colList[1].Table, colList[1].Name, type2)
				if strings.ToLower(type1) != strings.ToLower(type2) {
					content += fmt.Sprintf("`%s`.`%s` (%s) VS `%s`.`%s` (%s) datatype not match",
						colList[0].Table, colList[0].Name, type1,
						colList[1].Table, colList[1].Name, type2)
					continue
				}

				// 检查字符集不一致导致的隐式数据转换
				common.Log.Debug("Charset: `%s`.`%s` (%s) VS `%s`.`%s` (%s)",
					colList[0].Table, colList[0].Name, colList[0].Character,
					colList[1].Table, colList[1].Name, colList[1].Character)
				if colList[0].Character != colList[1].Character {
					content += fmt.Sprintf("`%s`.`%s` (%s) VS `%s`.`%s` (%s) charset not match",
						colList[0].Table, colList[0].Name, colList[0].Character,
						colList[1].Table, colList[1].Name, colList[1].Character)
					continue
				}

				// 检查排序排序不一致导致的隐式数据转换
				common.Log.Debug("Collation: `%s`.`%s` (%s) VS `%s`.`%s` (%s)",
					colList[0].Table, colList[0].Name, colList[0].Collation,
					colList[1].Table, colList[1].Name, colList[1].Collation)
				if colList[0].Collation != colList[1].Collation {
					content += fmt.Sprintf("`%s`.`%s` (%s) VS `%s`.`%s` (%s) collation not match",
						colList[0].Table, colList[0].Name, colList[0].Collation,
						colList[1].Table, colList[1].Name, colList[1].Collation)
					continue
				}
			}

			typMap := map[sqlparser.ValType][]string{
				// date, time, datetime, timestamp, year
				sqlparser.StrVal: {
					"char", "varchar", "tinytext", "text", "mediumtext", "longtext",
					"date", "time", "datetime", "timestamp", "year",
				},
				sqlparser.IntVal: {
					"tinyint", "smallint", "mediumint", "int", "integer", "bigint", "timestamp", "year",
				},
				sqlparser.FloatVal: {
					"float", "double", "real", "decimal",
				},
			}

			typNameMap := map[sqlparser.ValType]string{
				sqlparser.StrVal:   "string",
				sqlparser.IntVal:   "int",
				sqlparser.FloatVal: "float",
			}

			// 列与值比较
			for _, val := range values {
				if colList[0].DataType == "" {
					common.Log.Debug("Can't get %s datatype", colList[0].Name)
					break
				}

				isCovered := true
				if types, ok := typMap[val.Type]; ok {
					for _, t := range types {
						if strings.HasPrefix(colList[0].DataType, t) {
							isCovered = false
						}
					}
				}

				if isCovered {
					if colList[0].Table == "" {
						common.Log.Warning("Column %s not exists", colList[0].Name)
						continue
					}

					c := fmt.Sprintf("%s.%s definition is %s not %s",
						colList[0].Table, colList[0].Name, colList[0].DataType, typNameMap[val.Type])

					common.Log.Debug("Implicit data type conversion: %s", c)
					content += c
				}
			}

		case *sqlparser.RangeCond:
			// TODO
		case *sqlparser.IsExpr:
			// TODO
		}
	}
	if content != "" {
		rule = HeuristicRules["ARG.003"]
		rule.Content = content
	}
	return rule
}

// RuleNoWhere CLA.001 & CLA.014 & CLA.015
func (q *Query4Audit) RuleNoWhere() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Select:
			if n.Where == nil && sqlparser.String(n.From) != "dual" {
				rule = HeuristicRules["CLA.001"]
				return false, nil
			}
		case *sqlparser.Delete:
			if n.Where == nil {
				rule = HeuristicRules["CLA.014"]
				return false, nil
			}
		case *sqlparser.Update:
			if n.Where == nil {
				rule = HeuristicRules["CLA.015"]
				return false, nil
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleOrderByRand CLA.002
func (q *Query4Audit) RuleOrderByRand() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case sqlparser.OrderBy:
			for _, order := range n {
				switch expr := order.Expr.(type) {
				case *sqlparser.FuncExpr:
					if expr.Name.String() == "rand" {
						rule = HeuristicRules["CLA.002"]
						return false, nil
					}
				}
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleOffsetLimit CLA.003
func (q *Query4Audit) RuleOffsetLimit() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Limit:
			if n != nil && n.Offset != nil {
				switch v := n.Offset.(type) {
				case *sqlparser.SQLVal:
					offset, err := strconv.Atoi(string(v.Val))
					// 检查一下Offset阈值，太小了给这个建议也没什么用，阈值写死了没加配置
					if err == nil && offset > 1000 {
						rule = HeuristicRules["CLA.003"]
						return false, nil
					}
				}
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleGroupByConst CLA.004
func (q *Query4Audit) RuleGroupByConst() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case sqlparser.GroupBy:
			for _, group := range n {
				switch group.(type) {
				case *sqlparser.SQLVal:
					rule = HeuristicRules["CLA.004"]
					return false, nil
				}
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleGroupByConst GRP.001
func (idxAdv *IndexAdvisor) RuleGroupByConst() Rule {
	rule := HeuristicRules["OK"]

	// 非GroupBy语句
	if len(idxAdv.groupBy) == 0 || len(idxAdv.whereEQ) == 0 {
		return rule
	}

	for _, groupByCols := range idxAdv.groupBy {
		for _, whereEQCols := range idxAdv.whereEQ {
			if (groupByCols.Name == whereEQCols.Name) &&
				(groupByCols.DB == whereEQCols.DB) &&
				(groupByCols.Table == whereEQCols.Table) {
				rule = HeuristicRules["GRP.001"]
				break
			}
		}
	}
	return rule
}

// RuleOrderByConst CLA.005
func (q *Query4Audit) RuleOrderByConst() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case sqlparser.OrderBy:
			for _, order := range n {
				switch order.Expr.(type) {
				case *sqlparser.SQLVal:
					rule = HeuristicRules["CLA.005"]
					return false, nil
				}
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleOrderByConst CLA.005
// TODO: SELECT col FROM tbl WHERE col IN('NEWS') ORDER BY col;
func (idxAdv *IndexAdvisor) RuleOrderByConst() Rule {
	rule := HeuristicRules["OK"]

	// 非GroupBy语句
	if len(idxAdv.orderBy) == 0 || len(idxAdv.whereEQ) == 0 {
		return rule
	}

	for _, groupbyCols := range idxAdv.orderBy {
		for _, whereEQCols := range idxAdv.whereEQ {
			if (groupbyCols.Name == whereEQCols.Name) &&
				(groupbyCols.DB == whereEQCols.DB) &&
				(groupbyCols.Table == whereEQCols.Table) {
				rule = HeuristicRules["CLA.005"]
				break
			}
		}
	}
	return rule
}

// RuleDiffGroupByOrderBy CLA.006
func (q *Query4Audit) RuleDiffGroupByOrderBy() Rule {
	var rule = q.RuleOK()
	var groupbyTbls []sqlparser.TableIdent
	var orderbyTbls []sqlparser.TableIdent
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case sqlparser.GroupBy:
			// 检查group by涉及到表的个数
			for _, group := range n {
				switch g := group.(type) {
				case *sqlparser.ColName:
					tblExist := false
					for _, t := range groupbyTbls {
						if t.String() == g.Qualifier.Name.String() {
							tblExist = true
						}
					}
					if !tblExist {
						groupbyTbls = append(groupbyTbls, g.Qualifier.Name)
						if len(groupbyTbls) > 1 {
							rule = HeuristicRules["CLA.006"]

							return false, nil
						}
					}
				}
			}
		case sqlparser.OrderBy:
			// 检查order by涉及到表的个数
			for _, order := range n {
				switch o := order.Expr.(type) {
				case *sqlparser.ColName:
					tblExist := false
					for _, t := range orderbyTbls {
						if t.String() == o.Qualifier.Name.String() {
							tblExist = true
						}
					}
					if !tblExist {
						orderbyTbls = append(orderbyTbls, o.Qualifier.Name)
						if len(orderbyTbls) > 1 {
							rule = HeuristicRules["CLA.006"]

							return false, nil
						}
					}
				}
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")

	if rule.Item == "OK" {
		// 检查group by, order by涉及到表的个数
		for _, g := range groupbyTbls {
			tblExist := false
			for _, o := range orderbyTbls {
				if g.String() == o.String() {
					tblExist = true
				}
			}
			if !tblExist && len(orderbyTbls) > 0 {
				rule = HeuristicRules["CLA.006"]

				return rule
			}
		}
	}

	return rule
}

// RuleMixOrderBy CLA.007
func (q *Query4Audit) RuleMixOrderBy() Rule {
	var rule = q.RuleOK()
	var direction string
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case sqlparser.OrderBy:
			for _, order := range n {
				// 比较相邻两个order by列的方向
				if direction != "" && order.Direction != direction {
					rule = HeuristicRules["CLA.007"]

					return false, nil
				}
				direction = order.Direction
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleExplicitOrderBy CLA.008
func (q *Query4Audit) RuleExplicitOrderBy() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Select:
			// 有group by，但没有order by
			if n.GroupBy != nil && n.OrderBy == nil {
				rule = HeuristicRules["CLA.008"]

				return false, nil
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleOrderByExpr CLA.009
func (q *Query4Audit) RuleOrderByExpr() Rule {
	var rule = q.RuleOK()
	var orderByCols []string
	var selectCols []string
	funcExp := regexp.MustCompile(`[a-z0-9]\(`)
	allowExp := regexp.MustCompile("[a-z0-9_,.` ()]")
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case sqlparser.OrderBy:
			orderBy := sqlparser.String(n)
			// 函数名方式，如：from_unixtime(col)
			if funcExp.MatchString(orderBy) {
				rule = HeuristicRules["CLA.009"]

				return false, nil
			}

			// 运算符方式，如：colA - colB
			trim := allowExp.ReplaceAllFunc([]byte(orderBy), func(s []byte) []byte {
				return []byte("")
			})
			if string(trim) != "" {
				rule = HeuristicRules["CLA.009"]

				return false, nil
			}

			for _, o := range strings.Split(strings.TrimPrefix(orderBy, " order by "), ",") {
				orderByCols = append(orderByCols, strings.TrimSpace(strings.Split(o, " ")[0]))
			}
		case *sqlparser.Select:
			for _, s := range n.SelectExprs {
				selectCols = append(selectCols, sqlparser.String(s))
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")

	// AS情况，如：SELECT colA-colB a FROM tbl ORDER BY a;
	for _, o := range orderByCols {
		if o == "" {
			continue
		}
		for _, s := range selectCols {
			if strings.HasSuffix(s, " as "+o) {
				buf := strings.TrimSuffix(s, " as "+o)
				// 运算符
				trim := allowExp.ReplaceAllFunc([]byte(buf), func(s []byte) []byte {
					return []byte("")
				})
				if string(trim) != "" {
					rule = HeuristicRules["CLA.009"]

				}
				// 函数
				if funcExp.MatchString(s) {
					rule = HeuristicRules["CLA.009"]

				}
			}
		}
	}
	return rule
}

// RuleGroupByExpr CLA.010
func (q *Query4Audit) RuleGroupByExpr() Rule {
	var rule = q.RuleOK()
	var groupByCols []string
	var selectCols []string
	funcExp := regexp.MustCompile(`(?i)[a-z0-9]\(`)
	allowExp := regexp.MustCompile("(?i)[a-z0-9_,.` ()]")
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case sqlparser.GroupBy:
			groupBy := sqlparser.String(n)
			// 函数名方式，如：from_unixtime(col)
			if funcExp.MatchString(groupBy) {
				rule = HeuristicRules["CLA.010"]

				return false, nil
			}

			// 运算符方式，如：colA - colB
			trim := allowExp.ReplaceAllFunc([]byte(groupBy), func(s []byte) []byte {
				return []byte("")
			})
			if string(trim) != "" {
				rule = HeuristicRules["CLA.010"]

				return false, nil
			}

			for _, o := range strings.Split(strings.TrimPrefix(groupBy, " group by "), ",") {
				groupByCols = append(groupByCols, strings.TrimSpace(strings.Split(o, " ")[0]))
			}
		case *sqlparser.Select:
			for _, s := range n.SelectExprs {
				selectCols = append(selectCols, sqlparser.String(s))
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")

	// AS情况，如：SELECT colA-colB a FROM tbl GROUP BY a;
	for _, g := range groupByCols {
		if g == "" {
			continue
		}
		for _, s := range selectCols {
			if strings.HasSuffix(s, " as "+g) {
				buf := strings.TrimSuffix(s, " as "+g)
				// 运算符
				trim := allowExp.ReplaceAllFunc([]byte(buf), func(s []byte) []byte {
					return []byte("")
				})
				if string(trim) != "" {
					rule = HeuristicRules["CLA.010"]

				}
				// 函数
				if funcExp.MatchString(s) {
					rule = HeuristicRules["CLA.010"]

				}
			}
		}
	}
	return rule
}

// RuleTblCommentCheck CLA.011
func (q *Query4Audit) RuleTblCommentCheck() Rule {
	var rule = q.RuleOK()
	switch node := q.Stmt.(type) {
	case *sqlparser.DDL:
		if node.Action != "create" {
			return rule
		}
		if node.TableSpec == nil {
			return rule
		}
		if options := node.TableSpec.Options; options == "" {
			rule = HeuristicRules["CLA.011"]

		} else {
			reg := regexp.MustCompile("(?i)comment")
			if !reg.MatchString(options) {
				rule = HeuristicRules["CLA.011"]
			}
		}
	}
	return rule
}

// RuleSelectStar COL.001
func (q *Query4Audit) RuleSelectStar() Rule {
	var rule = q.RuleOK()
	// 先把count(*)替换为count(1)
	re := regexp.MustCompile(`(?i)count\s*\(\s*\*\s*\)`)
	sql := re.ReplaceAllString(q.Query, "count(1)")
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		common.Log.Debug("RuleSelectStar sqlparser.Parse Error: %v", err)
		return rule
	}
	err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch node.(type) {
		case *sqlparser.StarExpr:
			rule = HeuristicRules["COL.001"]
			return false, nil
		}
		return true, nil
	}, stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleInsertColDef COL.002
func (q *Query4Audit) RuleInsertColDef() Rule {
	var rule = q.RuleOK()
	switch node := q.Stmt.(type) {
	case *sqlparser.Insert:
		if node.Columns == nil {
			rule = HeuristicRules["COL.002"]
			return rule
		}
	}
	return rule
}

// RuleAddDefaultValue COL.004
func (q *Query4Audit) RuleAddDefaultValue() Rule {
	var rule = q.RuleOK()
	for _, node := range q.TiStmt {
		switch n := node.(type) {
		case *tidb.CreateTableStmt:
			for _, c := range n.Cols {
				colDefault := false
				for _, o := range c.Options {
					// 忽略AutoIncrement类型的默认值检查
					if o.Tp == tidb.ColumnOptionDefaultValue || o.Tp == tidb.ColumnOptionAutoIncrement {
						colDefault = true
					}
				}
				if !colDefault {
					rule = HeuristicRules["COL.004"]
					break
				}
			}
		case *tidb.AlterTableStmt:
			for _, s := range n.Specs {
				switch s.Tp {
				case tidb.AlterTableAddColumns, tidb.AlterTableChangeColumn, tidb.AlterTableModifyColumn:
					for _, c := range s.NewColumns {
						colDefault := false
						for _, o := range c.Options {
							// 忽略AutoIncrement类型的默认值检查
							if o.Tp == tidb.ColumnOptionDefaultValue || o.Tp == tidb.ColumnOptionAutoIncrement {
								colDefault = true
							}
						}
						if !colDefault {
							rule = HeuristicRules["COL.004"]
							break
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleColCommentCheck COL.005
func (q *Query4Audit) RuleColCommentCheck() Rule {
	var rule = q.RuleOK()
	for _, node := range q.TiStmt {
		switch n := node.(type) {
		case *tidb.CreateTableStmt:
			for _, c := range n.Cols {
				colComment := false
				for _, o := range c.Options {
					if o.Tp == tidb.ColumnOptionComment {
						colComment = true
					}
				}
				if !colComment {
					rule = HeuristicRules["COL.005"]
					break
				}
			}
		case *tidb.AlterTableStmt:
			for _, s := range n.Specs {
				switch s.Tp {
				case tidb.AlterTableAddColumns, tidb.AlterTableChangeColumn, tidb.AlterTableModifyColumn:
					for _, c := range s.NewColumns {
						colComment := false
						for _, o := range c.Options {
							if o.Tp == tidb.ColumnOptionComment {
								colComment = true
							}
						}
						if !colComment {
							rule = HeuristicRules["COL.005"]
							break
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleIPString LIT.001
func (q *Query4Audit) RuleIPString() Rule {
	var rule = q.RuleOK()
	re := regexp.MustCompile(`['"]\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	if re.FindString(q.Query) != "" {
		rule = HeuristicRules["LIT.001"]
		if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
			rule.Position = position[0]
		}
	}
	return rule
}

// RuleDataNotQuote LIT.002
func (q *Query4Audit) RuleDataNotQuote() Rule {
	var rule = q.RuleOK()
	// 2010-01-01
	re := regexp.MustCompile(`.\d{4}\s*-\s*\d{1,2}\s*-\s*\d{1,2}\b`)
	sqls := re.FindAllString(q.Query, -1)
	for _, sql := range sqls {
		re = regexp.MustCompile(`^['"\w-].*`)
		if re.FindString(sql) == "" {
			rule = HeuristicRules["LIT.002"]
		}
	}

	// 10-01-01
	re = regexp.MustCompile(`.\d{2}\s*-\s*\d{1,2}\s*-\s*\d{1,2}\b`)
	sqls = re.FindAllString(q.Query, -1)
	for _, sql := range sqls {
		re = regexp.MustCompile(`^['"\w-].*`)
		if re.FindString(sql) == "" {
			rule = HeuristicRules["LIT.002"]
		}
	}

	if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
		rule.Position = position[0]
	}
	return rule
}

// RuleSQLCalcFoundRows KWR.001
func (q *Query4Audit) RuleSQLCalcFoundRows() Rule {
	var rule = q.RuleOK()
	tkns := ast.Tokenizer(q.Query)
	for _, tkn := range tkns {
		if tkn.Val == "sql_calc_found_rows" {
			rule = HeuristicRules["KWR.001"]
			break
		}
	}
	return rule
}

// RuleCommaAnsiJoin JOI.001
func (q *Query4Audit) RuleCommaAnsiJoin() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Select:
			ansiJoin := false
			commaJoin := false
			for _, f := range n.From {
				switch f.(type) {
				case *sqlparser.JoinTableExpr:
					ansiJoin = true
				case *sqlparser.AliasedTableExpr:
					commaJoin = true
				}
			}
			if ansiJoin && commaJoin {
				rule = HeuristicRules["JOI.001"]
				return false, nil
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleDupJoin JOI.002
func (q *Query4Audit) RuleDupJoin() Rule {
	var rule = q.RuleOK()
	var tables []string
	switch q.Stmt.(type) {
	// TODO: 这里未检查UNION SELECT
	case *sqlparser.Union:
		return rule
	default:
		err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch n := node.(type) {
			case *sqlparser.AliasedTableExpr:
				switch table := n.Expr.(type) {
				case sqlparser.TableName:
					for _, t := range tables {
						if t == table.Name.String() {
							rule = HeuristicRules["JOI.002"]
							return false, nil
						}
					}
					tables = append(tables, table.Name.String())
				}
			}
			return true, nil
		}, q.Stmt)
		common.LogIfError(err, "")
	}
	return rule
}

// RuleImpossibleOuterJoin JOI.003
// TODO: 未实现完
func (idxAdv *IndexAdvisor) RuleImpossibleOuterJoin() Rule {
	rule := HeuristicRules["OK"]

	var joinTables []string         // JOIN相关表名
	var whereEQTables []string      // WHERE等值判断条件表名
	var joinNotWhereTables []string // 是JOIN相关表，但未出现在WHERE等值判断条件中的表名

	// 非JOIN语句
	if len(idxAdv.joinCond) == 0 || len(idxAdv.whereEQ) == 0 {
		return rule
	}

	for _, l1 := range idxAdv.joinCond {
		for _, l2 := range l1 {
			if l2.Table != "" && l2.Table != "dual" {
				joinTables = append(joinTables, l2.Table)
			}
		}
	}

	for _, w := range idxAdv.whereEQ {
		whereEQTables = append(whereEQTables, w.Table)
	}

	for _, j := range joinTables {
		found := false
		for _, w := range whereEQTables {
			if j == w {
				found = true
			}
		}
		if !found {
			joinNotWhereTables = append(joinNotWhereTables, j)
		}
	}

	// TODO:
	fmt.Println(joinNotWhereTables)
	/*
		if len(joinNotWhereTables) == 0 {
			rule = HeuristicRules["JOI.003"]
		}
	*/
	rule = HeuristicRules["JOI.003"]
	return rule
}

// TODO: JOI.004

// RuleNoDeterministicGroupby RES.001
func (q *Query4Audit) RuleNoDeterministicGroupby() Rule {
	var rule = q.RuleOK()
	var groupbyCols []*common.Column
	var selectCols []*common.Column
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Select:
			// 过滤select列
			selectCols = ast.FindColumn(n.SelectExprs)
			// 过滤group by列
			groupbyCols = ast.FindColumn(n.GroupBy)
			// `select *`, but not `select count(*)`
			if strings.Contains(sqlparser.String(n), " * ") && len(groupbyCols) > 0 {
				rule = HeuristicRules["RES.001"]
				return false, nil
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")

	// TODO：暂时只检查了列名，未对库表名进行检查，也未处理AS
	for _, s := range selectCols {
		// 无group by退出
		if len(groupbyCols) == 0 {
			break
		}
		found := false
		for _, g := range groupbyCols {
			if g.Name == s.Name {
				found = true
			}
		}
		if !found {
			rule = HeuristicRules["RES.001"]
			break
		}
	}
	return rule
}

// RuleNoDeterministicLimit RES.002
func (q *Query4Audit) RuleNoDeterministicLimit() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.Select:
			if n.Limit != nil && n.OrderBy == nil {
				rule = HeuristicRules["RES.002"]
				return false, nil
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleUpdateDeleteWithLimit RES.003
func (q *Query4Audit) RuleUpdateDeleteWithLimit() Rule {
	var rule = q.RuleOK()
	switch s := q.Stmt.(type) {
	case *sqlparser.Update:
		if s.Limit != nil {
			rule = HeuristicRules["RES.003"]
		}
	}
	return rule
}

// RuleUpdateDeleteWithOrderby RES.004
func (q *Query4Audit) RuleUpdateDeleteWithOrderby() Rule {
	var rule = q.RuleOK()
	switch s := q.Stmt.(type) {
	case *sqlparser.Update:
		if s.OrderBy != nil {
			rule = HeuristicRules["RES.004"]
		}
	}
	return rule
}

// RuleUpdateSetAnd RES.005
func (q *Query4Audit) RuleUpdateSetAnd() Rule {
	var rule = q.RuleOK()
	switch s := q.Stmt.(type) {
	case *sqlparser.Update:
		if strings.Contains(sqlparser.String(s.Exprs), " and ") {
			rule = HeuristicRules["RES.005"]
		}
	}
	return rule
}

// RuleImpossibleWhere RES.006
func (q *Query4Audit) RuleImpossibleWhere() Rule {
	var rule = q.RuleOK()
	// BETWEEN 10 AND 5
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.RangeCond:
			if n.Operator == "between" {
				from := 0
				to := 0
				switch s := n.From.(type) {
				case *sqlparser.SQLVal:
					from, _ = strconv.Atoi(string(s.Val))
				}
				switch s := n.To.(type) {
				case *sqlparser.SQLVal:
					to, _ = strconv.Atoi(string(s.Val))
				}
				if from > to {
					rule = HeuristicRules["RES.006"]
					return false, nil
				}
			}
		case *sqlparser.ComparisonExpr:
			factor := false
			switch n.Operator {
			case "!=", "<>":
			case "=", "<=>":
				factor = true
			default:
				return true, nil
			}

			var left []byte
			var right []byte

			// left
			switch l := n.Left.(type) {
			case *sqlparser.SQLVal:
				left = l.Val
			default:
				return true, nil
			}

			// right
			switch r := n.Right.(type) {
			case *sqlparser.SQLVal:
				right = r.Val
			default:
				return true, nil
			}

			// compare
			if (!bytes.Equal(left, right) && factor) || (bytes.Equal(left, right) && !factor) {
				rule = HeuristicRules["RES.006"]
			}
			return false, nil
		}

		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleMeaninglessWhere RES.007
func (q *Query4Audit) RuleMeaninglessWhere() Rule {
	var rule = q.RuleOK()
	// 1=1, 0=0
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.ComparisonExpr:
			factor := false
			switch n.Operator {
			case "!=", "<>":
				factor = true
			case "=", "<=>":
			default:
				return true, nil
			}

			var left []byte
			var right []byte

			// left
			switch l := n.Left.(type) {
			case *sqlparser.SQLVal:
				left = l.Val
			default:
				return true, nil
			}

			// right
			switch r := n.Right.(type) {
			case *sqlparser.SQLVal:
				right = r.Val
			default:
				return true, nil
			}

			// compare
			if (bytes.Equal(left, right) && !factor) || (!bytes.Equal(left, right) && factor) {
				rule = HeuristicRules["RES.007"]
			}
			return false, nil
		}

		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleLoadFile RES.008
func (q *Query4Audit) RuleLoadFile() Rule {
	var rule = q.RuleOK()
	// 去除注释
	sql := string(database.RemoveSQLComments([]byte(q.Query)))
	// 去除多余的空格和回车
	sql = strings.Join(strings.Fields(sql), " ")
	tks := ast.Tokenize(sql)
	for i, tk := range tks {
		// 注意：每个关键字token的结尾是带空格的，这里偷懒没trimspace直接加空格比较
		// LOAD DATA...
		if strings.ToLower(tk.Val) == "load " && i+1 < len(tks) &&
			strings.ToLower(tks[i+1].Val) == "data " {
			rule = HeuristicRules["RES.008"]
			break
		}

		// SELECT ... INTO OUTFILE
		if strings.ToLower(tk.Val) == "into " && i+1 < len(tks) &&
			(strings.ToLower(tks[i+1].Val) == "outfile " || strings.ToLower(tks[i+1].Val) == "dumpfile ") {
			rule = HeuristicRules["RES.008"]
			break
		}
	}
	return rule
}

// RuleStandardINEQ STA.001
func (q *Query4Audit) RuleStandardINEQ() Rule {
	var rule = q.RuleOK()
	re := regexp.MustCompile(`(!=)`)
	if re.FindString(q.Query) != "" {
		rule = HeuristicRules["STA.001"]
		if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
			rule.Position = position[0]
		}
	}
	return rule
}

// RuleUseKeyWord KWR.002
func (q *Query4Audit) RuleUseKeyWord() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		if q.TiStmt == nil {
			common.Log.Error("TiStmt is nil, SQL: %s", q.Query)
			return rule
		}

		for _, tiStmtNode := range q.TiStmt {
			switch stmt := tiStmtNode.(type) {
			case *tidb.AlterTableStmt:
				// alter
				for _, spec := range stmt.Specs {
					for _, column := range spec.NewColumns {
						if ast.IsMysqlKeyword(column.Name.String()) {
							return HeuristicRules["KWR.002"]
						}
					}
				}

			case *tidb.CreateTableStmt:
				// create
				if ast.IsMysqlKeyword(stmt.Table.Name.String()) {
					return HeuristicRules["KWR.002"]
				}

				for _, col := range stmt.Cols {
					if ast.IsMysqlKeyword(col.Name.String()) {
						return HeuristicRules["KWR.002"]
					}
				}
			}

		}
	}

	return rule
}

// RulePluralWord KWR.003
// Reference: https://en.wikipedia.org/wiki/English_plurals
func (q *Query4Audit) RulePluralWord() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		if q.TiStmt == nil {
			common.Log.Error("TiStmt is nil, SQL: %s", q.Query)
			return rule
		}

		for _, tiStmtNode := range q.TiStmt {
			switch stmt := tiStmtNode.(type) {
			case *tidb.AlterTableStmt:
				// alter
				for _, spec := range stmt.Specs {
					for _, column := range spec.NewColumns {
						if inflector.Singularize(column.Name.String()) != column.Name.String() {
							return HeuristicRules["KWR.003"]
						}
					}
				}

			case *tidb.CreateTableStmt:
				// create
				if inflector.Singularize(stmt.Table.Name.String()) != stmt.Table.Name.String() {
					return HeuristicRules["KWR.003"]
				}

				for _, col := range stmt.Cols {
					if inflector.Singularize(col.Name.String()) != col.Name.String() {
						return HeuristicRules["KWR.003"]
					}
				}
			}

		}

	}
	return rule
}

// RuleInsertSelect LCK.001
func (q *Query4Audit) RuleInsertSelect() Rule {
	var rule = q.RuleOK()
	switch n := q.Stmt.(type) {
	case *sqlparser.Insert:
		switch n.Rows.(type) {
		case *sqlparser.Select:
			rule = HeuristicRules["LCK.001"]
		}
	}
	return rule
}

// RuleInsertOnDup LCK.002
func (q *Query4Audit) RuleInsertOnDup() Rule {
	var rule = q.RuleOK()
	switch n := q.Stmt.(type) {
	case *sqlparser.Insert:
		if n.OnDup != nil {
			rule = HeuristicRules["LCK.002"]
			return rule
		}
	}
	return rule
}

// RuleInSubquery SUB.001
func (q *Query4Audit) RuleInSubquery() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch node.(type) {
		case *sqlparser.Subquery:
			rule = HeuristicRules["SUB.001"]
			return false, nil
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleSubqueryDepth SUB.004
func (q *Query4Audit) RuleSubqueryDepth() Rule {
	var rule = q.RuleOK()
	if depth := ast.GetSubqueryDepth(q.Stmt); depth > common.Config.MaxSubqueryDepth {
		rule = HeuristicRules["SUB.004"]
	}
	return rule
}

// RuleSubQueryLimit SUB.005
// 只有IN的SUBQUERY限制了LIMIT，FROM子句中的SUBQUERY并未限制LIMIT
func (q *Query4Audit) RuleSubQueryLimit() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.ComparisonExpr:
			if n.Operator == "in" {
				switch r := n.Right.(type) {
				case *sqlparser.Subquery:
					switch s := r.Select.(type) {
					case *sqlparser.Select:
						if s.Limit != nil {
							rule = HeuristicRules["SUB.005"]
							return false, nil
						}
					}
				}
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleSubQueryFunctions SUB.006
func (q *Query4Audit) RuleSubQueryFunctions() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch node.(type) {
		case *sqlparser.Subquery:
			err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
				switch node.(type) {
				case *sqlparser.FuncExpr:
					rule = HeuristicRules["SUB.006"]
					return false, nil
				}
				return true, nil
			}, node)
			common.LogIfError(err, "")
		}

		if rule.Item == "OK" {
			return true, nil
		}
		return false, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleMultiValueAttribute LIT.003
func (q *Query4Audit) RuleMultiValueAttribute() Rule {
	var rule = q.RuleOK()
	re := regexp.MustCompile(`(?i)(id\s+varchar)|(id\s+text)|(id\s+regexp)`)
	if re.FindString(q.Query) != "" {
		rule = HeuristicRules["LIT.003"]
		if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
			rule.Position = position[0]
		}
	}
	return rule
}

// RuleAddDelimiter LIT.004
func (q *Query4Audit) RuleAddDelimiter() Rule {
	var rule = q.RuleOK()
	re := regexp.MustCompile(`(?i)(^use\s+[0-9a-z_-]*)|(^show\s+databases)`)
	if re.FindString(q.Query) != "" && !strings.HasSuffix(q.Query, common.Config.Delimiter) {
		rule = HeuristicRules["LIT.004"]
		if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
			rule.Position = position[0]
		}
	}
	return rule
}

// RuleRecursiveDependency KEY.003
func (q *Query4Audit) RuleRecursiveDependency() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				// create statement
				for _, ref := range node.Constraints {
					if ref != nil && ref.Tp == tidb.ConstraintForeignKey {
						rule = HeuristicRules["KEY.003"]
					}
				}

			case *tidb.AlterTableStmt:
				// alter table statement
				for _, spec := range node.Specs {
					if spec.Constraint != nil && spec.Constraint.Tp == tidb.ConstraintForeignKey {
						rule = HeuristicRules["KEY.003"]
					}
				}
			}
		}
	}

	if rule.Item == "KEY.003" {
		re := regexp.MustCompile(`(?i)(\s+references\s+)`)
		if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
			rule.Position = position[0]
		}
	}

	return rule
}

// RuleImpreciseDataType COL.009
func (q *Query4Audit) RuleImpreciseDataType() Rule {
	var rule = q.RuleOK()
	if q.TiStmt != nil {
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				// Create table statement
				for _, col := range node.Cols {
					switch col.Tp.Tp {
					case mysql.TypeFloat, mysql.TypeDouble, mysql.TypeDecimal, mysql.TypeNewDecimal:
						rule = HeuristicRules["COL.009"]
					}
				}

			case *tidb.AlterTableStmt:
				// Alter table statement
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableAddColumns, tidb.AlterTableChangeColumn, tidb.AlterTableModifyColumn:
						for _, col := range spec.NewColumns {
							switch col.Tp.Tp {
							case mysql.TypeFloat, mysql.TypeDouble,
								mysql.TypeDecimal, mysql.TypeNewDecimal:
								rule = HeuristicRules["COL.009"]
							}
						}
					}
				}

			case *tidb.InsertStmt:
				// Insert statement
				for _, values := range node.Lists {
					for _, value := range values {
						switch value.GetDatum().Kind() {
						case types.KindFloat32, types.KindFloat64, types.KindMysqlDecimal:
							rule = HeuristicRules["COL.009"]
						}
					}
				}

			case *tidb.SelectStmt:
				// Select statement
				switch where := node.Where.(type) {
				case *tidb.BinaryOperationExpr:
					switch where.R.GetDatum().Kind() {
					case types.KindFloat32, types.KindFloat64, types.KindMysqlDecimal:
						rule = HeuristicRules["COL.009"]
					}
				}
			}
		}
	}

	return rule
}

// RuleValuesInDefinition COL.010
func (q *Query4Audit) RuleValuesInDefinition() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, col := range node.Cols {
					switch col.Tp.Tp {
					case mysql.TypeSet, mysql.TypeEnum, mysql.TypeBit:
						rule = HeuristicRules["COL.010"]
					}
				}
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableAddColumns, tidb.AlterTableChangeColumn, tidb.AlterTableModifyColumn:
						for _, col := range spec.NewColumns {
							switch col.Tp.Tp {
							case mysql.TypeSet, mysql.TypeEnum, mysql.TypeBit:
								rule = HeuristicRules["COL.010"]
							}
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleIndexAttributeOrder KEY.004
func (q *Query4Audit) RuleIndexAttributeOrder() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateIndexStmt:
				if len(node.IndexColNames) > 1 {
					rule = HeuristicRules["KEY.004"]
					break
				}
			case *tidb.CreateTableStmt:
				for _, constraint := range node.Constraints {
					// 当一条索引中包含多个列的时候给予建议
					if len(constraint.Keys) > 1 {
						rule = HeuristicRules["KEY.004"]
						break
					}
				}
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					if spec.Tp == tidb.AlterTableAddConstraint && len(spec.Constraint.Keys) > 1 {
						rule = HeuristicRules["KEY.004"]
						break
					}
				}
			}
		}
	}
	return rule
}

// RuleNullUsage COL.011
func (q *Query4Audit) RuleNullUsage() Rule {
	var rule = q.RuleOK()
	re := regexp.MustCompile(`(?i)(\s+null\s+)`)
	if re.FindString(q.Query) != "" {
		rule = HeuristicRules["COL.011"]
		if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
			rule.Position = position[0]
		}
	}
	return rule
}

// RuleStringConcatenation FUN.003
func (q *Query4Audit) RuleStringConcatenation() Rule {
	var rule = q.RuleOK()
	re := regexp.MustCompile(`(?i)(\|\|)`)
	if re.FindString(q.Query) != "" {
		rule = HeuristicRules["FUN.003"]
		if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
			rule.Position = position[0]
		}
	}
	return rule
}

// RuleSysdate FUN.004
func (q *Query4Audit) RuleSysdate() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.FuncExpr:
			if n.Name.String() == "sysdate" {
				rule = HeuristicRules["FUN.004"]
				return false, nil
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleCountConst FUN.005
func (q *Query4Audit) RuleCountConst() Rule {
	var rule = q.RuleOK()
	fingerprint := query.Fingerprint(q.Query)
	countReg := regexp.MustCompile(`(?i)count\(\s*[0-9a-z?]*\s*\)`)
	if countReg.MatchString(fingerprint) {
		rule = HeuristicRules["FUN.005"]
		if position := countReg.FindIndex([]byte(q.Query)); len(position) > 0 {
			rule.Position = position[0]
		}
	}
	return rule
}

// RuleSumNPE FUN.006
func (q *Query4Audit) RuleSumNPE() Rule {
	var rule = q.RuleOK()
	fingerprint := query.Fingerprint(q.Query)
	sumReg := regexp.MustCompile(`(?i)sum\(\s*[0-9a-z?]*\s*\)`)
	isnullReg := regexp.MustCompile(`(?i)isnull\(sum\(\s*[0-9a-z?]*\s*\)\)`)
	if sumReg.MatchString(fingerprint) && !isnullReg.MatchString(fingerprint) {
		rule = HeuristicRules["FUN.006"]
		if position := isnullReg.FindIndex([]byte(q.Query)); len(position) > 0 {
			rule.Position = position[0]
		}
	}
	return rule
}

// RulePatternMatchingUsage ARG.007
func (q *Query4Audit) RulePatternMatchingUsage() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.Select:
		re := regexp.MustCompile(`(?i)(\bregexp\b)|(\bsimilar to\b)`)
		if re.FindString(q.Query) != "" {
			rule = HeuristicRules["ARG.007"]
		}
	}
	return rule
}

// RuleSpaghettiQueryAlert CLA.012
func (q *Query4Audit) RuleSpaghettiQueryAlert() Rule {
	var rule = q.RuleOK()
	if len(query.Fingerprint(q.Query)) > common.Config.SpaghettiQueryLength {
		rule = HeuristicRules["CLA.012"]
	}
	return rule
}

// RuleReduceNumberOfJoin JOI.005
func (q *Query4Audit) RuleReduceNumberOfJoin() Rule {
	var rule = q.RuleOK()
	var tables []string
	switch q.Stmt.(type) {
	// TODO: UNION有可能有多张表，这里未检查UNION SELECT
	case *sqlparser.Union:
		return rule
	default:
		err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch n := node.(type) {
			case *sqlparser.AliasedTableExpr:
				switch table := n.Expr.(type) {
				case sqlparser.TableName:
					exist := false
					for _, t := range tables {
						if t == table.Name.String() {
							exist = true
							break
						}
					}
					if !exist {
						tables = append(tables, table.Name.String())
					}
				}
			}
			return true, nil
		}, q.Stmt)
		common.LogIfError(err, "")
	}
	if len(tables) > common.Config.MaxJoinTableCount {
		rule = HeuristicRules["JOI.005"]
	}
	return rule
}

// RuleDistinctUsage DIS.001
func (q *Query4Audit) RuleDistinctUsage() Rule {
	// Distinct
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.Select:
		re := regexp.MustCompile(`(?i)(\bdistinct\b)`)
		if len(re.FindAllString(q.Query, -1)) > common.Config.MaxDistinctCount {
			rule = HeuristicRules["DIS.001"]
		}
	}
	return rule
}

// RuleCountDistinctMultiCol DIS.002
func (q *Query4Audit) RuleCountDistinctMultiCol() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.FuncExpr:
			str := strings.ToLower(sqlparser.String(n))
			if strings.HasPrefix(str, "count") && strings.Contains(str, ",") {
				rule = HeuristicRules["DIS.002"]
				return false, nil
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleDistinctStar DIS.003
func (q *Query4Audit) RuleDistinctStar() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.Select:
		meta := ast.GetMeta(q.Stmt, nil)
		for _, m := range meta {
			if len(m.Table) == 1 {
				// distinct tbl.* from tbl和 distinct *
				re := regexp.MustCompile(`(?i)((\s+distinct\s*\*)|(\s+distinct\s+[0-9a-z_` + "`" + `]*\.\*))`)
				if re.MatchString(q.Query) {
					rule = HeuristicRules["DIS.003"]
				}
			}
			break
		}
	}
	return rule
}

// RuleHavingClause CLA.013
func (q *Query4Audit) RuleHavingClause() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case *sqlparser.Select:
			if expr.Having != nil {
				rule = HeuristicRules["CLA.013"]
				return false, nil
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleUpdatePrimaryKey CLA.016
func (idxAdv *IndexAdvisor) RuleUpdatePrimaryKey() Rule {
	rule := HeuristicRules["OK"]
	switch node := idxAdv.Ast.(type) {
	case *sqlparser.Update:
		var setColumns []*common.Column

		err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch node.(type) {
			case *sqlparser.UpdateExpr:
				// 获取set操作的全部column
				setColumns = append(setColumns, ast.FindAllCols(node)...)
			}
			return true, nil
		}, node)
		common.LogIfError(err, "")
		setColumns = idxAdv.calcCardinality(CompleteColumnsInfo(idxAdv.Ast, setColumns, idxAdv.vEnv))
		for _, col := range setColumns {
			idxMeta := idxAdv.IndexMeta[idxAdv.vEnv.DBHash(col.DB)][col.Table]
			if idxMeta == nil {
				return rule
			}
			for _, idx := range idxMeta.IdxRows {
				if idx.KeyName == "PRIMARY" {
					if col.Name == idx.ColumnName {
						rule = HeuristicRules["CLA.016"]
						return rule
					}
					continue
				}
			}
		}
	}

	return rule
}

// RuleForbiddenSyntax CLA.017
func (q *Query4Audit) RuleForbiddenSyntax() Rule {
	var rule = q.RuleOK()

	// 由于vitess对某些语法的支持不完善，使得如创建临时表等语句无法通过语法检查
	// 所以这里使用正则对触发器、临时表、存储过程等进行匹配
	// 但是目前支持的也不是非常全面，有待完善匹配规则
	// TODO TiDB 目前还不支持触发器、存储过程、自定义函数、外键

	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`(?i)CREATE\s+TRIGGER\s+`),

		regexp.MustCompile(`(?i)CREATE\s+TEMPORARY\s+TABLE\s+`),

		regexp.MustCompile(`(?i)CREATE\s+VIEW\s+`),
		regexp.MustCompile(`(?i)REPLACE\s+VIEW\s+`),

		regexp.MustCompile(`(?i)CREATE\s+PROCEDURE\s+`),
		regexp.MustCompile(`(?i)CREATE\s+FUNCTION\s+`),
	}

	for _, reg := range forbidden {
		if reg.MatchString(q.Query) {
			rule = HeuristicRules["CLA.017"]
			if position := reg.FindIndex([]byte(q.Query)); len(position) > 0 {
				rule.Position = position[0]
			}
			break
		}
	}
	return rule
}

// RuleNestedSubQueries JOI.006
func (q *Query4Audit) RuleNestedSubQueries() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch node.(type) {
		case *sqlparser.Subquery:
			rule = HeuristicRules["JOI.006"]
			return false, nil
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleMultiDeleteUpdate JOI.007
func (q *Query4Audit) RuleMultiDeleteUpdate() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.Delete, *sqlparser.Update:
		err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch node.(type) {
			case *sqlparser.JoinTableExpr:
				rule = HeuristicRules["JOI.007"]
				return false, nil
			}
			return true, nil
		}, q.Stmt)
		common.LogIfError(err, "")
	}
	return rule
}

// RuleMultiDBJoin JOI.008
func (q *Query4Audit) RuleMultiDBJoin() Rule {
	var rule = q.RuleOK()
	meta := ast.GetMeta(q.Stmt, nil)
	dbCount := 0
	for range meta {
		dbCount++
	}

	if dbCount > 1 {
		err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch node.(type) {
			case *sqlparser.JoinTableExpr:
				rule = HeuristicRules["JOI.008"]
				return false, nil
			}
			return true, nil
		}, q.Stmt)
		common.LogIfError(err, "")
	}
	return rule
}

// RuleORUsage ARG.008
func (q *Query4Audit) RuleORUsage() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.Select:
		err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch node.(type) {
			case *sqlparser.OrExpr:
				rule = HeuristicRules["ARG.008"]
				return false, nil
			}
			return true, nil
		}, q.Stmt)
		common.LogIfError(err, "")
	}
	return rule
}

// RuleSpaceWithQuote ARG.009
func (q *Query4Audit) RuleSpaceWithQuote() Rule {
	var rule = q.RuleOK()
	for _, tk := range ast.Tokenize(q.Query) {
		if tk.Type == ast.TokenTypeQuote {
			// 序列化的Val是带引号，所以要取第2个最倒数第二个，这样也就不用担心len<2了。
			switch tk.Val[1] {
			case ' ':
				rule = HeuristicRules["ARG.009"]
			}
			switch tk.Val[len(tk.Val)-2] {
			case ' ':
				rule = HeuristicRules["ARG.009"]
			}
		}
	}
	return rule
}

// RuleHint ARG.010
// TODO: sql_no_cache, straight join
func (q *Query4Audit) RuleHint() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.IndexHints:
			if n != nil {
				rule = HeuristicRules["ARG.010"]
			}
			return false, nil
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleNot ARG.011
func (q *Query4Audit) RuleNot() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.ComparisonExpr:
			if strings.HasPrefix(n.Operator, "not") {
				rule = HeuristicRules["ARG.011"]
				return false, nil
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleUNIONUsage SUB.002
func (q *Query4Audit) RuleUNIONUsage() Rule {
	var rule = q.RuleOK()
	switch s := q.Stmt.(type) {
	case *sqlparser.Union:
		if s.Type == "union" {
			rule = HeuristicRules["SUB.002"]
		}
	}
	return rule
}

// RuleDistinctJoinUsage SUB.003
func (q *Query4Audit) RuleDistinctJoinUsage() Rule {
	var rule = q.RuleOK()
	switch expr := q.Stmt.(type) {
	case *sqlparser.Select:
		if expr.Distinct != "" {
			if expr.From != nil {
				if len(expr.From) > 1 {
					rule = HeuristicRules["SUB.003"]
				}
			}
		}
	}
	return rule
}

// RuleReadablePasswords SEC.002
func (q *Query4Audit) RuleReadablePasswords() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		re := regexp.MustCompile(`(?i)(password)|(password)|(pwd)`)
		for _, tiStmt := range q.TiStmt {
			// create table stmt
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, col := range node.Cols {
					switch col.Tp.Tp {
					case mysql.TypeString, mysql.TypeVarchar, mysql.TypeVarString,
						mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob:
						if re.FindString(q.Query) != "" {
							return HeuristicRules["SEC.002"]
						}
					}
				}

			case *tidb.AlterTableStmt:
				// alter table stmt
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableModifyColumn, tidb.AlterTableChangeColumn, tidb.AlterTableAddColumns:
						for _, col := range spec.NewColumns {
							switch col.Tp.Tp {
							case mysql.TypeString, mysql.TypeVarchar, mysql.TypeVarString,
								mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob:
								if re.FindString(q.Query) != "" {
									return HeuristicRules["SEC.002"]
								}
							}
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleDataDrop SEC.003
func (q *Query4Audit) RuleDataDrop() Rule {
	var rule = q.RuleOK()
	switch s := q.Stmt.(type) {
	case *sqlparser.DBDDL:
		if s.Action == "drop" {
			rule = HeuristicRules["SEC.003"]
		}
	case *sqlparser.DDL:
		if s.Action == "drop" || s.Action == "truncate" {
			rule = HeuristicRules["SEC.003"]
		}
	case *sqlparser.Delete:
		rule = HeuristicRules["SEC.003"]
	}
	return rule
}

// RuleCompareWithFunction FUN.001
func (q *Query4Audit) RuleCompareWithFunction() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.ComparisonExpr:
			if strings.HasSuffix(sqlparser.String(n.Left), ")") {
				rule = HeuristicRules["FUN.001"]
				return false, nil
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleCountStar FUN.002
func (q *Query4Audit) RuleCountStar() Rule {
	var rule = q.RuleOK()
	switch n := q.Stmt.(type) {
	case *sqlparser.Select:
		// count(N), count(col), count(*)
		re := regexp.MustCompile(`(?i)(count\(\s*[*0-9a-z_` + "`" + `]*\s*\))`)
		if re.FindString(q.Query) != "" && n.Where != nil {
			rule = HeuristicRules["FUN.002"]
		}
	}
	return rule
}

// RuleTruncateTable SEC.001
func (q *Query4Audit) RuleTruncateTable() Rule {
	var rule = q.RuleOK()
	switch s := q.Stmt.(type) {
	case *sqlparser.DDL:
		if s.Action == "truncate" {
			rule = HeuristicRules["SEC.001"]
		}
	}
	return rule
}

// RuleIn ARG.005 && ARG.004
func (q *Query4Audit) RuleIn() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.ComparisonExpr:
			switch n.Operator {
			case "in":
				switch r := n.Right.(type) {
				case sqlparser.ValTuple:
					// IN (NULL)
					for _, v := range r {
						switch v.(type) {
						case *sqlparser.NullVal:
							rule = HeuristicRules["ARG.004"]
							return false, nil
						}
					}
					if len(r) > common.Config.MaxInCount {
						rule = HeuristicRules["ARG.005"]
						return false, nil
					}
				}
			case "not in":
				switch r := n.Right.(type) {
				case sqlparser.ValTuple:
					// NOT IN (NULL)
					for _, v := range r {
						switch v.(type) {
						case *sqlparser.NullVal:
							rule = HeuristicRules["ARG.004"]
							return false, nil
						}
					}
				}
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleIsNullIsNotNull ARG.006
func (q *Query4Audit) RuleIsNullIsNotNull() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.Select:
		re := regexp.MustCompile(`(?i)is\s*(not)?\s+null\b`)
		if re.FindString(q.Query) != "" {
			rule = HeuristicRules["ARG.006"]
		}
	}
	return rule
}

// RuleVarcharVSChar COL.008
func (q *Query4Audit) RuleVarcharVSChar() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, col := range node.Cols {
					switch col.Tp.Tp {
					// 在 TiDB 的 AST 中，char 和 binary 的 type 都是 mysql.TypeString
					// 只是 binary 数据类型的 character 和 collate 是 binary
					case mysql.TypeString:
						rule = HeuristicRules["COL.008"]
					}
				}

			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableAddColumns, tidb.AlterTableChangeColumn, tidb.AlterTableModifyColumn:
						for _, col := range spec.NewColumns {
							switch col.Tp.Tp {
							case mysql.TypeString:
								rule = HeuristicRules["COL.008"]
							}
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleCreateDualTable TBL.003
func (q *Query4Audit) RuleCreateDualTable() Rule {
	var rule = q.RuleOK()
	switch s := q.Stmt.(type) {
	case *sqlparser.DDL:
		if s.NewName.Name.String() == "dual" {
			rule = HeuristicRules["TBL.003"]

		}
	}
	return rule
}

// RuleAlterCharset ALT.001
func (q *Query4Audit) RuleAlterCharset() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableOption:
						for _, option := range spec.Options {
							if option.Tp == tidb.TableOptionCharset ||
								option.Tp == tidb.TableOptionCollate {
								rule = HeuristicRules["ALT.001"]
								break
							}
						}
					}

					if rule.Item == "ALT.001" {
						break
					}
				}
			}
		}
	}
	return rule
}

// RuleAlterDropColumn ALT.003
func (q *Query4Audit) RuleAlterDropColumn() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableDropColumn:
						rule = HeuristicRules["ALT.003"]
					}
				}
			}
		}

		if rule.Item == "ALT.003" {
			re := regexp.MustCompile(`(?i)(drop\s+column)`)
			if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
				rule.Position = position[0]
			}
		}
	}
	return rule
}

// RuleAlterDropKey ALT.004
func (q *Query4Audit) RuleAlterDropKey() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableDropPrimaryKey,
						tidb.AlterTableDropIndex,
						tidb.AlterTableDropForeignKey:
						rule = HeuristicRules["ALT.004"]
					}
				}
			}
		}
	}
	return rule
}

// RuleCantBeNull COL.012
func (q *Query4Audit) RuleCantBeNull() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, col := range node.Cols {
					switch col.Tp.Tp {
					case mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob:
						if !mysql.HasNotNullFlag(col.Tp.Flag) {
							rule = HeuristicRules["COL.012"]
						}
					}
				}

			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableAddColumns, tidb.AlterTableModifyColumn, tidb.AlterTableChangeColumn:
						for _, col := range spec.NewColumns {
							switch col.Tp.Tp {
							case mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob:
								if !mysql.HasNotNullFlag(col.Tp.Flag) {
									rule = HeuristicRules["COL.012"]
								}
							}
						}
					}
				}
			}
		}
	}

	return rule
}

// RuleTooManyKeys KEY.005
func (q *Query4Audit) RuleTooManyKeys() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				if len(node.Constraints) > common.Config.MaxIdxCount {
					rule = HeuristicRules["KEY.005"]
				}
			}
		}
	}
	return rule
}

// RuleTooManyKeyParts KEY.006
func (q *Query4Audit) RuleTooManyKeyParts() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, constraint := range node.Constraints {
					if len(constraint.Keys) > common.Config.MaxIdxColsCount {
						return HeuristicRules["KEY.006"]
					}

					if constraint.Refer != nil && len(constraint.Refer.IndexColNames) > common.Config.MaxIdxColsCount {
						return HeuristicRules["KEY.006"]
					}
				}

			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableAddConstraint:
						if spec.Constraint != nil {
							if len(spec.Constraint.Keys) > common.Config.MaxIdxColsCount {
								return HeuristicRules["KEY.006"]
							}

							if spec.Constraint.Refer != nil {
								if len(spec.Constraint.Refer.IndexColNames) > common.Config.MaxIdxColsCount {
									return HeuristicRules["KEY.006"]
								}
							}
						}
					}
				}
			}
		}
	}

	return rule
}

// RulePKNotInt KEY.007 && KEY.001
func (q *Query4Audit) RulePKNotInt() Rule {
	var rule = q.RuleOK()
	var pk sqlparser.ColIdent
	switch s := q.Stmt.(type) {
	case *sqlparser.DDL:
		if s.Action == "create" {
			if s.TableSpec == nil {
				return rule
			}
			for _, idx := range s.TableSpec.Indexes {
				if idx.Info.Type == "primary key" {
					if len(idx.Columns) == 1 {
						pk = idx.Columns[0].Column
						break
					}
				}
			}

			// 未指定主键
			if pk.String() == "" {
				rule = HeuristicRules["KEY.007"]
				return rule
			}

			// 主键非int, bigint类型
			for _, col := range s.TableSpec.Columns {
				if pk.String() == col.Name.String() {
					switch col.Type.Type {
					case "int", "bigint", "integer":
						if !col.Type.Unsigned {
							rule = HeuristicRules["KEY.007"]
						}
						if !col.Type.Autoincrement {
							rule = HeuristicRules["KEY.001"]
						}
					default:
						rule = HeuristicRules["KEY.007"]
					}
				}
			}
		}
	}
	return rule
}

// RuleOrderByMultiDirection KEY.008
func (q *Query4Audit) RuleOrderByMultiDirection() Rule {
	var rule = q.RuleOK()
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case sqlparser.OrderBy:
			order := ""
			for _, col := range strings.Split(sqlparser.String(n), ",") {
				orders := strings.Split(col, " ")
				if order != "" && order != orders[len(orders)-1] {
					rule = HeuristicRules["KEY.008"]
					return false, nil
				}
				order = orders[len(orders)-1]
			}
		}
		return true, nil
	}, q.Stmt)
	common.LogIfError(err, "")
	return rule
}

// RuleUniqueKeyDup KEY.009
// TODO: 目前只是给建议，期望能够实现自动检查
func (q *Query4Audit) RuleUniqueKeyDup() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateIndexStmt:
				// create index
				if node.Unique {
					re := regexp.MustCompile(`(?i)(create\s+(unique)\s)`)
					rule = HeuristicRules["KEY.009"]
					if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
						rule.Position = position[0]
					}
					return rule
				}

			case *tidb.AlterTableStmt:
				// alter table add constraint
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableAddConstraint:
						if spec.Constraint == nil {
							continue
						}
						switch spec.Constraint.Tp {
						case tidb.ConstraintPrimaryKey, tidb.ConstraintUniq, tidb.ConstraintUniqKey, tidb.ConstraintUniqIndex:
							re := regexp.MustCompile(`(?i)(add\s+(unique)\s)`)
							rule = HeuristicRules["KEY.009"]
							if position := re.FindIndex([]byte(q.Query)); len(position) > 0 {
								rule.Position = position[0]
							}
							return rule
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleTimestampDefault COL.013
func (q *Query4Audit) RuleTimestampDefault() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, col := range node.Cols {
					if col.Tp.Tp == mysql.TypeTimestamp {
						hasDefault := false
						for _, option := range col.Options {
							if option.Tp == tidb.ColumnOptionDefaultValue {
								hasDefault = true
							}
						}
						if !hasDefault {
							rule = HeuristicRules["COL.013"]
							break
						}
					}
				}
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableAddColumns,
						tidb.AlterTableModifyColumn,
						tidb.AlterTableChangeColumn,
						tidb.AlterTableAlterColumn:
						for _, col := range spec.NewColumns {
							if col.Tp.Tp == mysql.TypeTimestamp {
								hasDefault := false
								for _, option := range col.Options {
									if option.Tp == tidb.ColumnOptionDefaultValue {
										hasDefault = true
									}
								}
								if !hasDefault {
									rule = HeuristicRules["COL.013"]
									break
								}
							}
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleAutoIncrementInitNotZero TBL.004
func (q *Query4Audit) RuleAutoIncrementInitNotZero() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, opt := range node.Options {
					if opt.Tp == tidb.TableOptionAutoIncrement && opt.UintValue > 1 {
						rule = HeuristicRules["TBL.004"]
					}
				}

			}
		}
	}
	return rule
}

// RuleColumnWithCharset COL.014
func (q *Query4Audit) RuleColumnWithCharset() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, col := range node.Cols {
					if col.Tp.Charset != "" || col.Tp.Collate != "" {
						rule = HeuristicRules["COL.014"]
						break
					}
				}
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableAlterColumn, tidb.AlterTableChangeColumn,
						tidb.AlterTableModifyColumn, tidb.AlterTableAddColumns:
						for _, col := range spec.NewColumns {
							if col.Tp.Charset != "" || col.Tp.Collate != "" {
								rule = HeuristicRules["COL.014"]
								break
							}
						}
					}

				}
			}
		}
	}
	return rule
}

// RuleTableCharsetCheck TBL.005
func (q *Query4Audit) RuleTableCharsetCheck() Rule {
	var rule = q.RuleOK()

	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				var allow bool
				var hasCharset bool
				for _, opt := range node.Options {
					if opt.Tp == tidb.TableOptionCharset {
						hasCharset = true
						for _, ch := range common.Config.TableAllowCharsets {
							if strings.TrimSpace(strings.ToLower(ch)) == strings.TrimSpace(strings.ToLower(opt.StrValue)) {
								allow = true
								break
							}
						}
					}
				}

				// 未指定字符集使用MySQL默认配置字符集，我们认为MySQL的配置是被优化过的。
				if hasCharset && !allow {
					rule = HeuristicRules["TBL.005"]
					break
				}

			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					var allow bool
					var hasCharset bool
					switch spec.Tp {
					case tidb.AlterTableOption:
						for _, opt := range spec.Options {
							if opt.Tp == tidb.TableOptionCharset {
								hasCharset = true
								for _, ch := range common.Config.TableAllowCharsets {
									if strings.TrimSpace(strings.ToLower(ch)) == strings.TrimSpace(strings.ToLower(opt.StrValue)) {
										allow = true
										break
									}
								}
							}
						}
						// 未指定字符集使用MySQL默认配置字符集，我们认为MySQL的配置是被优化过的。
						if hasCharset && !allow {
							rule = HeuristicRules["TBL.005"]
							break
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleBlobDefaultValue COL.015
func (q *Query4Audit) RuleBlobDefaultValue() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, col := range node.Cols {
					switch col.Tp.Tp {
					case mysql.TypeBlob, mysql.TypeMediumBlob, mysql.TypeTinyBlob, mysql.TypeLongBlob:
						for _, opt := range col.Options {
							if opt.Tp == tidb.ColumnOptionDefaultValue && opt.Expr.GetType().Tp != mysql.TypeNull {
								rule = HeuristicRules["COL.015"]
								break
							}
						}
					}
				}
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableModifyColumn, tidb.AlterTableAlterColumn,
						tidb.AlterTableChangeColumn, tidb.AlterTableAddColumns:
						for _, col := range spec.NewColumns {
							switch col.Tp.Tp {
							case mysql.TypeBlob, mysql.TypeMediumBlob, mysql.TypeTinyBlob, mysql.TypeLongBlob:
								for _, opt := range col.Options {
									if opt.Tp == tidb.ColumnOptionDefaultValue && opt.Expr.GetType().Tp != mysql.TypeNull {
										rule = HeuristicRules["COL.015"]
										break
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleIntPrecision COL.016
func (q *Query4Audit) RuleIntPrecision() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, col := range node.Cols {
					switch col.Tp.Tp {
					case mysql.TypeLong:
						if (col.Tp.Flen < 10 || col.Tp.Flen > 11) && col.Tp.Flen > 0 {
							// 有些语言ORM框架会生成int(11)，有些语言的框架生成int(10)
							rule = HeuristicRules["COL.016"]
							break
						}
					case mysql.TypeLonglong:
						if (col.Tp.Flen != 20) && col.Tp.Flen > 0 {
							rule = HeuristicRules["COL.016"]
							break
						}
					}
				}
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableAddColumns, tidb.AlterTableChangeColumn,
						tidb.AlterTableAlterColumn, tidb.AlterTableModifyColumn:
						for _, col := range spec.NewColumns {
							switch col.Tp.Tp {
							case mysql.TypeLong:
								if (col.Tp.Flen < 10 || col.Tp.Flen > 11) && col.Tp.Flen > 0 {
									// 有些语言ORM框架会生成int(11)，有些语言的框架生成int(10)
									rule = HeuristicRules["COL.016"]
									break
								}
							case mysql.TypeLonglong:
								if col.Tp.Flen != 20 && col.Tp.Flen > 0 {
									rule = HeuristicRules["COL.016"]
									break
								}
							}
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleVarcharLength COL.017
func (q *Query4Audit) RuleVarcharLength() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, col := range node.Cols {
					switch col.Tp.Tp {
					case mysql.TypeVarchar, mysql.TypeVarString:
						if col.Tp.Flen > common.Config.MaxVarcharLength {
							rule = HeuristicRules["COL.017"]
							break
						}
					}
				}
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableAddColumns, tidb.AlterTableChangeColumn,
						tidb.AlterTableAlterColumn, tidb.AlterTableModifyColumn:
						for _, col := range spec.NewColumns {
							switch col.Tp.Tp {
							case mysql.TypeVarchar, mysql.TypeVarString:
								if col.Tp.Flen > common.Config.MaxVarcharLength {
									rule = HeuristicRules["COL.017"]
									break
								}
							}
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleNoOSCKey KEY.002
func (q *Query4Audit) RuleNoOSCKey() Rule {
	var rule = q.RuleOK()
	switch s := q.Stmt.(type) {
	case *sqlparser.DDL:
		if s.Action == "create" {
			pkReg := regexp.MustCompile(`(?i)(primary\s+key)`)
			if !pkReg.MatchString(q.Query) {
				ukReg := regexp.MustCompile(`(?i)(unique\s+((key)|(index)))`)
				if !ukReg.MatchString(q.Query) {
					rule = HeuristicRules["KEY.002"]
				}
			}
		}
	}
	return rule
}

// RuleTooManyFields COL.006
func (q *Query4Audit) RuleTooManyFields() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				if len(node.Cols) > common.Config.MaxColCount {
					rule = HeuristicRules["COL.006"]
				}
			}
		}
	}
	return rule
}

// RuleAllowEngine TBL.002
func (q *Query4Audit) RuleAllowEngine() Rule {
	var rule = q.RuleOK()
	var hasDefaultEngine bool
	var allowedEngine bool
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, opt := range node.Options {
					if opt.Tp == tidb.TableOptionEngine {
						hasDefaultEngine = true
						// 使用了非推荐的存储引擎
						for _, engine := range common.Config.TableAllowEngines {
							if strings.EqualFold(opt.StrValue, engine) {
								allowedEngine = true
							}
						}
						// common.Config.TableAllowEngines 为空时不给予建议
						if !allowedEngine && len(common.Config.TableAllowEngines) > 0 {
							rule = HeuristicRules["TBL.002"]
							break
						}
					}
				}
				// 建表语句未指定表的存储引擎
				if !hasDefaultEngine {
					rule = HeuristicRules["TBL.002"]
					break
				}
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableOption:
						for _, opt := range spec.Options {
							if opt.Tp == tidb.TableOptionEngine {
								// 使用了非推荐的存储引擎
								for _, engine := range common.Config.TableAllowEngines {
									if strings.EqualFold(opt.StrValue, engine) {
										allowedEngine = true
									}
								}
								// common.Config.TableAllowEngines 为空时不给予建议
								if !allowedEngine && len(common.Config.TableAllowEngines) > 0 {
									rule = HeuristicRules["TBL.002"]
									break
								}
							}
						}
					}
				}
			}
		}
	}
	return rule
}

// RulePartitionNotAllowed TBL.001
func (q *Query4Audit) RulePartitionNotAllowed() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				if node.Partition != nil {
					rule = HeuristicRules["TBL.001"]
					break
				}
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					if len(spec.PartDefinitions) > 0 {
						rule = HeuristicRules["TBL.001"]
						break
					}
				}
			}
		}
	}
	return rule
}

// RuleAutoIncUnsigned COL.003:
func (q *Query4Audit) RuleAutoIncUnsigned() Rule {
	var rule = q.RuleOK()
	switch q.Stmt.(type) {
	case *sqlparser.DDL:
		for _, tiStmt := range q.TiStmt {
			switch node := tiStmt.(type) {
			case *tidb.CreateTableStmt:
				for _, col := range node.Cols {
					for _, opt := range col.Options {
						if opt.Tp == tidb.ColumnOptionAutoIncrement {
							if !mysql.HasUnsignedFlag(col.Tp.Flag) {
								rule = HeuristicRules["COL.003"]
								break
							}
						}

						if rule.Item == "COL.003" {
							break
						}
					}
				}
			case *tidb.AlterTableStmt:
				for _, spec := range node.Specs {
					switch spec.Tp {
					case tidb.AlterTableChangeColumn, tidb.AlterTableAlterColumn,
						tidb.AlterTableModifyColumn, tidb.AlterTableAddColumns:
						for _, col := range spec.NewColumns {
							for _, opt := range col.Options {
								if opt.Tp == tidb.ColumnOptionAutoIncrement {
									if !mysql.HasUnsignedFlag(col.Tp.Flag) {
										rule = HeuristicRules["COL.003"]
										break
									}
								}

								if rule.Item == "COL.003" {
									break
								}
							}
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleSpaceAfterDot STA.002
func (q *Query4Audit) RuleSpaceAfterDot() Rule {
	var rule = q.RuleOK()
	tks := ast.Tokenize(q.Query)
	for i, tk := range tks {
		switch tk.Type {

		// SELECT * FROM db. tbl
		// SELECT tbl. col FROM tbl
		case ast.TokenTypeWord:
			if len(tks) > i+1 &&
				tks[i+1].Type == ast.TokenTypeWhitespace &&
				strings.HasSuffix(tk.Val, ".") {
				common.Log.Debug("RuleSpaceAfterDot: ", tk.Val, tks[i+1].Val)
				rule = HeuristicRules["STA.002"]
				return rule
			}
		default:
		}
	}
	return rule
}

// RuleIdxPrefix STA.003
func (q *Query4Audit) RuleIdxPrefix() Rule {
	var rule = q.RuleOK()
	for _, node := range q.TiStmt {
		switch n := node.(type) {
		case *tidb.CreateTableStmt:
			for _, c := range n.Constraints {
				switch c.Tp {
				case tidb.ConstraintIndex, tidb.ConstraintKey:
					if !strings.HasPrefix(c.Name, common.Config.IdxPrefix) {
						rule = HeuristicRules["STA.003"]
					}
				case tidb.ConstraintUniq, tidb.ConstraintUniqKey, tidb.ConstraintUniqIndex:
					if !strings.HasPrefix(c.Name, common.Config.UkPrefix) {
						rule = HeuristicRules["STA.003"]
					}
				}
			}
		case *tidb.AlterTableStmt:
			for _, s := range n.Specs {
				switch s.Tp {
				case tidb.AlterTableAddConstraint:
					switch s.Constraint.Tp {
					case tidb.ConstraintIndex, tidb.ConstraintKey:
						if !strings.HasPrefix(s.Constraint.Name, common.Config.IdxPrefix) {
							rule = HeuristicRules["STA.003"]
						}
					case tidb.ConstraintUniq, tidb.ConstraintUniqKey, tidb.ConstraintUniqIndex:
						if !strings.HasPrefix(s.Constraint.Name, common.Config.UkPrefix) {
							rule = HeuristicRules["STA.003"]
						}
					}
				}
			}
		}
	}
	return rule
}

// RuleStandardName STA.004
func (q *Query4Audit) RuleStandardName() Rule {
	var rule = q.RuleOK()
	allowReg := regexp.MustCompile(`(?i)[a-z0-9_` + "`" + `]`)
	for _, tk := range ast.Tokenize(q.Query) {
		if tk.Val == "``" {
			rule = HeuristicRules["STA.004"]
		}

		switch tk.Type {
		// 反引号中可能有乱七八糟的东西
		case ast.TokenTypeBacktickQuote:
			// 特殊字符，连续下划线
			if allowReg.ReplaceAllString(tk.Val, "") != "" || strings.Contains(tk.Val, "__") {
				rule = HeuristicRules["STA.004"]
			}
			// 统一大小写
			if !(strings.ToLower(tk.Val) == tk.Val || strings.ToUpper(tk.Val) == tk.Val) {
				rule = HeuristicRules["STA.004"]
			}
		case ast.TokenTypeWord:
			// TOKEN_TYPE_WORD中处理连续下划线的情况，其他情况容易误伤
			if strings.Contains(tk.Val, "__") {
				rule = HeuristicRules["STA.004"]
			}
		default:
		}
	}
	return rule
}

// MergeConflictHeuristicRules merge conflict rules
func MergeConflictHeuristicRules(rules map[string]Rule) map[string]Rule {
	// KWR.001 VS ERR.000
	// select sql_calc_found_rows * from film
	if _, ok := rules["KWR.001"]; ok {
		delete(rules, "ERR.000")
	}

	// SUB.001 VS OWN.004 VS JOI.006
	if _, ok := rules["SUB.001"]; ok {
		delete(rules, "ARG.005")
		delete(rules, "JOI.006")
	}

	// SUB.004 VS SUB.001
	if _, ok := rules["SUB.004"]; ok {
		delete(rules, "SUB.001")
	}

	// KEY.007 VS KEY.002
	if _, ok := rules["KEY.007"]; ok {
		delete(rules, "KEY.002")
	}

	// JOI.002 VS JOI.006
	if _, ok := rules["JOI.002"]; ok {
		delete(rules, "JOI.006")
	}

	// JOI.008 VS JOI.007
	if _, ok := rules["JOI.008"]; ok {
		delete(rules, "JOI.007")
	}
	return rules
}

// RuleMySQLError ERR.XXX
func RuleMySQLError(item string, err error) Rule {

	type MySQLError struct {
		ErrCode   string
		ErrString string
	}

	// vitess 语法检查出错返回的是ERR.000
	switch item {
	case "ERR.000":
		return Rule{
			Item:     item,
			Summary:  "MySQL执行出错 " + err.Error(),
			Severity: "L8",
			Content:  err.Error(),
		}
	}

	// Received #1146 error from MySQL server: "table xxx doesn't exist"
	errReg := regexp.MustCompile(`(?i)Received #([0-9]+) error from MySQL server: ['"](.*)['"]`)
	errStr := err.Error()
	msg := errReg.FindStringSubmatch(errStr)
	var mysqlError MySQLError

	if len(msg) == 3 {
		if msg[1] != "" && msg[2] != "" {
			mysqlError = MySQLError{
				ErrCode:   msg[1],
				ErrString: msg[2],
			}
		}
	} else {
		var errcode string
		if strings.HasPrefix(err.Error(), "syntax error at position") {
			errcode = "1064"
		}
		mysqlError = MySQLError{
			ErrCode:   errcode,
			ErrString: err.Error(),
		}
	}
	switch mysqlError.ErrCode {
	// 1146 ER_NO_SUCH_TABLE
	case "", "1146":
		return Rule{
			Item:     item,
			Summary:  "MySQL执行出错",
			Severity: "L0",
			Content:  "",
		}
	default:
		return Rule{
			Item:     item,
			Summary:  "MySQL执行出错 " + mysqlError.ErrString,
			Severity: "L8",
			Content:  mysqlError.ErrString,
		}
	}
}
