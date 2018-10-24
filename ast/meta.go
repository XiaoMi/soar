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
	"strings"

	"github.com/XiaoMi/soar/common"
	"vitess.io/vitess/go/vt/sqlparser"
)

// GetTableFromExprs 从sqlparser.Exprs中获取所有的库表
func GetTableFromExprs(exprs sqlparser.TableExprs, metas ...common.Meta) common.Meta {
	meta := make(map[string]*common.DB)
	if len(metas) >= 1 {
		meta = metas[0]
	}

	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case *sqlparser.AliasedTableExpr:

			switch table := expr.Expr.(type) {
			case sqlparser.TableName:
				db := table.Qualifier.String()
				tb := table.Name.String()

				if meta[db] == nil {
					meta[db] = common.NewDB(db)
				}

				meta[db].Table[tb] = common.NewTable(tb)

				// alias去重
				aliasExist := false
				for _, existedAlias := range meta[db].Table[tb].TableAliases {
					if existedAlias == expr.As.String() {
						aliasExist = true
					}
				}

				if !aliasExist {
					meta[db].Table[tb].TableAliases = append(meta[db].Table[tb].TableAliases, expr.As.String())
				}
			}
		}
		return true, nil
	}, exprs)
	common.LogIfWarn(err, "")
	return meta
}

// GetMeta 获取元数据信息，构建到db->table层级。
// 从 SQL 或 Statement 中获取表信息，并返回。当 meta 不为 nil 时，返回值会将新老 meta 合并去重
func GetMeta(stmt sqlparser.Statement, meta common.Meta) common.Meta {
	// 初始化meta
	if meta == nil {
		meta = make(map[string]*common.DB)
	}

	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case *sqlparser.DDL:
			// 如果SQL是一个DDL，则不需要继续遍历语法树了
			db1 := expr.Table.Qualifier.String()
			tb1 := expr.Table.Name.String()
			db2 := expr.NewName.Qualifier.String()
			tb2 := expr.NewName.Name.String()

			if tb1 != "" {
				if _, ok := meta[db1]; !ok {
					meta[db1] = common.NewDB(db1)
				}

				meta[db1].Table[tb1] = common.NewTable(tb1)
			}

			if tb2 != "" {
				if _, ok := meta[db2]; !ok {
					meta[db2] = common.NewDB(db2)
				}

				meta[db2].Table[tb2] = common.NewTable(tb2)
			}

			return false, nil
		case *sqlparser.AliasedTableExpr:
			// 非 DDL 情况下处理 TableExpr
			// 在 sqlparser 中存在三种 TableExpr: AliasedTableExpr，ParenTableExpr 以及 JoinTableExpr。
			// 其中 AliasedTableExpr 是其他两种 TableExpr 的基础组成，SQL中的 表信息（别名、前缀）在这个结构体中。

			switch table := expr.Expr.(type) {

			// 获取表名、别名与前缀名（数据库名）
			// 表名存放在 AST 中 TableName 里，包含表名与表前缀名。
			// 当与 As 相对应的 Expr 为 TableName 的时候，别名才是一张实体表的别名，否则为结果集的别名。
			case sqlparser.TableName:
				db := table.Qualifier.String()
				tb := table.Name.String()

				if meta[db] == nil {
					meta[db] = common.NewDB(db)
				}

				meta[db].Table[tb] = common.NewTable(tb)

				// alias去重
				aliasExist := false
				for _, existedAlias := range meta[db].Table[tb].TableAliases {
					if existedAlias == expr.As.String() {
						aliasExist = true
					}
				}
				if !aliasExist {
					meta[db].Table[tb].TableAliases = append(meta[db].Table[tb].TableAliases, expr.As.String())
				}

			default:
				// 如果 AliasedTableExpr 中的 Expr 不是 TableName 结构体，则表示该表为一个查询结果集（子查询或临时表）。
				// 在这里记录一下别名，但将列名制空，用来保证在其他环节中判断列前缀的时候不会有遗漏
				// 最终结果为所有的子查询别名都会归于 ""（空） 数据库 ""（空） 表下，对于空数据库，空表后续在索引优化时直接PASS
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
	common.LogIfWarn(err, "")
	return meta
}

// eqOperators 等值条件判断关键字
var eqOperators = map[string]string{
	"=":            "eq",
	"<=>":          "eq",
	"is true":      "eq",
	"is false":     "eq",
	"is not true":  "eq",
	"is not false": "eq",
	"is null":      "eq",
	"in":           "eq", // 单值的in属于等值条件
}

// inEqOperators 非等值条件判断关键字
var inEqOperators = map[string]string{
	"<":           "inEq",
	">":           "inEq",
	"<=":          "inEq",
	">=":          "inEq",
	"!=":          "inEq",
	"is not null": "inEq",
	"like":        "inEq",
	"not like":    "inEq",
	"->":          "inEq",
	"->>":         "inEq",
	"between":     "inEq",
	"not between": "inEq",
	"in":          "inEq", // 多值in属于非等值条件

	// 某些非等值条件无需添加索引，所以忽略即可
	// 比如"not in"，比如"exists"、 "not exists"等
}

// FindColumn 从传入的node中获取所有可能加索引的的column信息
func FindColumn(node sqlparser.SQLNode) []*common.Column {
	common.Log.Debug("Enter:  FindColumn, Caller: %s", common.Caller())
	var result []*common.Column
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch col := node.(type) {
		case *sqlparser.FuncExpr:
			// 忽略function
			return false, nil
		case *sqlparser.ColName:
			result = common.MergeColumn(result, &common.Column{
				Name:  col.Name.String(),
				Table: col.Qualifier.Name.String(),
				DB:    col.Qualifier.Qualifier.String(),
				Alias: make([]string, 0),
			})
		}

		return true, nil
	}, node)
	common.LogIfWarn(err, "")
	return result
}

// inEqIndexAble 判断非等值查询条件是否可以复用到索引
// Output: true 可以考虑添加索引， false 不需要添加索引
func inEqIndexAble(node sqlparser.SQLNode) bool {
	common.Log.Debug("Enter:  inEqIndexAble(), Caller: %s", common.Caller())
	var indexAble bool
	switch expr := node.(type) {
	case *sqlparser.ComparisonExpr:
		// like前百分号查询无法使用索引
		// TODO date类型的like属于隐式数据类型转换，会导致无法使用索引
		if expr.Operator == "like" || expr.Operator == "not like" {
			switch right := expr.Right.(type) {
			case *sqlparser.SQLVal:
				return !(strings.HasPrefix(string(right.Val), "%"))
			}
		}

		// 如果是in查询，则需要判断in查询是否是多值查询
		if expr.Operator == "in" {
			switch right := expr.Right.(type) {
			case sqlparser.ValTuple:
				// 若是单值查询则应该属于等值条件而非非等值条件
				return len(right) > 1
			}
		}

		_, indexAble = inEqOperators[expr.Operator]

	case *sqlparser.IsExpr:
		_, indexAble = inEqOperators[expr.Operator]

	case *sqlparser.RangeCond:
		_, indexAble = inEqOperators[expr.Operator]

	default:
		indexAble = false
	}
	return indexAble
}

// FindWhereEQ 找到Where中的等值条件
func FindWhereEQ(node sqlparser.SQLNode) []*common.Column {
	common.Log.Debug("Enter:  FindWhereEQ(), Caller: %s", common.Caller())
	return append(FindEQColsInWhere(node), FindEQColsInJoinCond(node)...)
}

// FindWhereINEQ 找到Where条件中的非等值条件
func FindWhereINEQ(node sqlparser.SQLNode) []*common.Column {
	common.Log.Debug("Enter:  FindWhereINEQ(), Caller: %s", common.Caller())
	return append(FindINEQColsInWhere(node), FindINEQColsInJoinCond(node)...)
}

// FindEQColsInWhere 获取等值条件信息
// 将所有值得加索引的condition条件信息进行过滤
func FindEQColsInWhere(node sqlparser.SQLNode) []*common.Column {
	common.Log.Debug("Enter:  FindEQColsInWhere(), Caller: %s", common.Caller())
	var columns []*common.Column
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch node := node.(type) {
		// 对AST中所有节点进行扫描
		case *sqlparser.Subquery, *sqlparser.JoinTableExpr, *sqlparser.BinaryExpr, *sqlparser.OrExpr:
			// 忽略子查询，join condition，数值计算，or condition
			return false, nil

		case *sqlparser.ComparisonExpr:
			var newCols []*common.Column
			// ComparisonExpr中可能含有等值查询列条件
			switch node.Operator {
			case "in":
				// 对in进行特别判断，只有单值的in条件才算做是等值查询
				switch right := node.Right.(type) {
				case sqlparser.ValTuple:
					if len(right) == 1 {
						newCols = FindColumn(node)
					}
				}

			default:
				if _, ok := eqOperators[node.Operator]; ok {
					newCols = FindColumn(node)
				}
			}

			// operator两边都为列的情况不提供索引建议
			// 如果该列位于function中则不予提供索引建议
			if len(newCols) == 1 {
				columns = common.MergeColumn(columns, newCols[0])
			}

		case *sqlparser.IsExpr:
			// IsExpr中可能含有等值查询列条件
			if _, ok := eqOperators[node.Operator]; ok {
				newCols := FindColumn(node)
				if len(newCols) == 1 {
					columns = common.MergeColumn(columns, newCols[0])
				}
			}
		}
		return true, nil

	}, node)
	common.LogIfWarn(err, "")
	return columns
}

// FindINEQColsInWhere 获取非等值条件中可能需要加索引的列
// 将所有值得加索引的condition条件信息进行过滤
// TODO: 将where条件中隐含的join条件合并到join condition中
func FindINEQColsInWhere(node sqlparser.SQLNode) []*common.Column {
	common.Log.Debug("Enter:  FindINEQColsInWhere(), Caller: %s", common.Caller())
	var columns []*common.Column
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch node := node.(type) {
		// 对AST中所有节点进行扫描
		case *sqlparser.Subquery, *sqlparser.JoinTableExpr, *sqlparser.BinaryExpr, *sqlparser.OrExpr:
			// 忽略子查询，join condition，数值计算，or condition
			return false, nil

		case *sqlparser.ComparisonExpr:
			// ComparisonExpr中可能含有非等值查询列条件
			if inEqIndexAble(node) {
				newCols := FindColumn(node)
				// operator两边都为列的情况不提供索引建议
				if len(newCols) == 1 {
					columns = common.MergeColumn(columns, newCols[0])
				}
			}
		case *sqlparser.IsExpr:
			// IsExpr中可能含有非等值查询列条件
			if inEqIndexAble(node) {
				newCols := FindColumn(node)
				if len(newCols) == 1 {
					columns = common.MergeColumn(columns, newCols[0])
				}
			}

		case *sqlparser.RangeCond:
			// RangeCond中只存在非等值条件查询
			if inEqIndexAble(node) {
				columns = common.MergeColumn(columns, FindColumn(node)...)
			}
		}

		return true, nil

	}, node)
	common.LogIfWarn(err, "")
	return columns
}

// FindGroupByCols 获取groupBy中可能需要加索引的列信息
func FindGroupByCols(node sqlparser.SQLNode) []*common.Column {
	common.Log.Debug("Enter:  FindGroupByCols(), Caller: %s", common.Caller())
	isIgnore := false
	var columns []*common.Column
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case sqlparser.GroupBy:
			err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
				switch node := node.(type) {
				case *sqlparser.BinaryExpr, *sqlparser.FuncExpr:
					// 如果group by中出现数值计算、函数等
					isIgnore = true
					return false, nil
				default:
					columns = common.MergeColumn(columns, FindColumn(node)...)
				}
				return true, nil
			}, expr)
			common.LogIfWarn(err, "")
		case *sqlparser.Subquery, *sqlparser.JoinTableExpr, *sqlparser.BinaryExpr:
			// 忽略子查询，join condition以及数值计算
			return false, nil
		}
		return true, nil
	}, node)
	common.LogIfWarn(err, "")
	if isIgnore {
		return []*common.Column{}
	}

	return columns
}

// FindOrderByCols 为索引优化获取orderBy中可能添加索引的列信息
func FindOrderByCols(node sqlparser.SQLNode) []*common.Column {
	common.Log.Debug("Enter:  FindOrderByCols(), Caller: %s", common.Caller())
	var columns []*common.Column
	lastDirection := ""
	directionNotEq := false
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case *sqlparser.Order:
			// MySQL对于排序顺序不同的查询无法使用索引（8.0后支持）
			if lastDirection != "" && expr.Direction != lastDirection {
				directionNotEq = true
				return false, nil
			}
			lastDirection = expr.Direction
			columns = common.MergeColumn(columns, FindColumn(expr)...)
		case *sqlparser.Subquery, *sqlparser.JoinTableExpr, *sqlparser.BinaryExpr:
			// 忽略子查询，join condition以及数值计算
			return false, nil
		}
		return true, nil
	}, node)
	common.LogIfWarn(err, "")
	if directionNotEq {
		// 当发现Order by中排序顺序不同时，即放弃Oder by条件中的字段
		return []*common.Column{}
	}

	return columns
}

// FindJoinTable 获取 Join 中需要添加索引的表
// join 优化添加索引分为三种类型：1. inner join, 2. left join, 3.right join
// 针对三种优化类型，需要三种不同的索引添加方案:
// 1. inner join 需要对 join 左右的表添加索引
// 2. left join 由于左表为全表扫描，需要对右表的关联列添加索引。
// 3. right join 与 left join 相反，需要对左表的关联列添加索引。
// 以上添加索引的策略前提为join的表为实体表而非临时表。
func FindJoinTable(node sqlparser.SQLNode, meta common.Meta) common.Meta {
	common.Log.Debug("Enter:  FindJoinTable(), Caller: %s", common.Caller())
	if meta == nil {
		meta = make(common.Meta)
	}
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case *sqlparser.JoinTableExpr:
			switch expr.Join {
			case "join", "natural join":
				// 两边表都需要
				findJoinTable(expr.LeftExpr, meta)
				findJoinTable(expr.RightExpr, meta)
			case "left join", "natural left join", "straight_join":
				// 只需要右表
				findJoinTable(expr.RightExpr, meta)

			case "right join", "natural right join":
				// 只需要左表
				findJoinTable(expr.LeftExpr, meta)
			}
		}
		return true, nil
	}, node)
	common.LogIfWarn(err, "")
	return meta
}

// findJoinTable 获取join table
func findJoinTable(expr sqlparser.TableExpr, meta common.Meta) {
	common.Log.Debug("Enter:  findJoinTable(), Caller: %s", common.Caller())
	if meta == nil {
		meta = make(common.Meta)
	}
	switch tableExpr := expr.(type) {
	case *sqlparser.AliasedTableExpr:
		switch table := tableExpr.Expr.(type) {
		// 获取表名、别名与前缀名（数据库名）
		// 表名存放在 AST 中 TableName 里，包含表名与表前缀名。
		// 当与 As 相对应的 Expr 为 TableName 的时候，别名才是一张实体表的别名，否则为结果集的别名。
		case sqlparser.TableName:
			db := table.Qualifier.String()
			tb := table.Name.String()

			if meta == nil {
				meta = make(map[string]*common.DB)
			}

			if meta[db] == nil {
				meta[db] = common.NewDB(db)
			}

			meta[db].Table[tb] = common.NewTable(tb)

			// alias去重
			aliasExist := false
			for _, existedAlias := range meta[db].Table[tb].TableAliases {
				if existedAlias == tableExpr.As.String() {
					aliasExist = true
				}
			}
			if !aliasExist {
				meta[db].Table[tb].TableAliases = append(meta[db].Table[tb].TableAliases, tableExpr.As.String())
			}
		}
	case *sqlparser.ParenTableExpr:
		// join 时可能会同时 join 多张表
		for _, tbExpr := range tableExpr.Exprs {
			findJoinTable(tbExpr, meta)
		}
	default:
		// 如果是如上两种类型都没有命中，说明join的表为临时表，递归调用 FindJoinTable 继续下探查找。
		// NOTE: 这里需要注意的是，如果不递归寻找，如果存在子查询结果集的join表，subquery也会把这个查询提取出。
		// 所以针对default这一段理论上可以忽略处理（待测试）
		FindJoinTable(tableExpr, meta)
	}
}

// FindJoinCols 获取 join condition 中使用到的列（必须是 `列 operator 列` 的情况。
// 如果列对应的值或是function，则应该移到where condition中）
// 某些where条件隐含在Join条件中（INNER JOIN）
func FindJoinCols(node sqlparser.SQLNode) [][]*common.Column {
	common.Log.Debug("Enter:  FindJoinCols(), Caller: %s", common.Caller())
	var columns [][]*common.Column
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case *sqlparser.JoinTableExpr:
			// on
			if on := expr.Condition.On; on != nil {
				cols := FindColumn(expr.Condition.On)
				if len(cols) > 1 {
					columns = append(columns, cols)
				}
			}

			// using
			if using := expr.Condition.Using; using != nil {
				left := ""
				right := ""

				switch tableExpr := expr.LeftExpr.(type) {
				case *sqlparser.AliasedTableExpr:
					switch table := tableExpr.Expr.(type) {
					case sqlparser.TableName:
						left = table.Name.String()
					}
				}

				switch tableExpr := expr.RightExpr.(type) {
				case *sqlparser.AliasedTableExpr:
					switch table := tableExpr.Expr.(type) {
					case sqlparser.TableName:
						right = table.Name.String()
					}
				}

				var cols []*common.Column
				for _, col := range using {
					if left != "" {
						cols = append(cols, &common.Column{
							Name:  col.String(),
							Table: left,
							Alias: make([]string, 0),
						})
					}

					if right != "" {
						cols = append(cols, &common.Column{
							Name:  col.String(),
							Table: right,
							Alias: make([]string, 0),
						})
					}

				}
				columns = append(columns, cols)

			}

		}
		return true, nil
	}, node)
	common.LogIfWarn(err, "")
	return columns
}

// FindEQColsInJoinCond 获取 join condition 中应转为whereEQ条件的列
func FindEQColsInJoinCond(node sqlparser.SQLNode) []*common.Column {
	common.Log.Debug("Enter:  FindEQColsInJoinCond(), Caller: %s", common.Caller())
	var columns []*common.Column
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case sqlparser.JoinCondition:
			columns = append(columns, FindEQColsInWhere(expr)...)
		}
		return true, nil
	}, node)
	common.LogIfWarn(err, "")
	return columns
}

// FindINEQColsInJoinCond 获取 join condition 中应转为whereINEQ条件的列
func FindINEQColsInJoinCond(node sqlparser.SQLNode) []*common.Column {
	common.Log.Debug("Enter:  FindINEQColsInJoinCond(), Caller: %s", common.Caller())
	var columns []*common.Column
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		case sqlparser.JoinCondition:
			columns = append(columns, FindINEQColsInWhere(expr)...)
		}
		return true, nil
	}, node)
	common.LogIfWarn(err, "")
	return columns
}

// FindSubquery 拆分subquery，获取最深层的subquery
// 为索引优化获取subquery中包含的列信息
func FindSubquery(depth int, node sqlparser.SQLNode, queries ...string) []string {
	common.Log.Debug("Enter:  FindSubquery(), Caller: %s", common.Caller())
	if queries == nil {
		queries = make([]string, 0)
	}
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch expr := node.(type) {
		// 查找SQL中的子查询
		case *sqlparser.Subquery:
			noSub := true
			// 查看子查询中是否还包含子查询，如果包含，递归找到最深层的子查询
			err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
				switch sub := node.(type) {
				case *sqlparser.Subquery:
					noSub = false
					// 查找深度depth，超过最大深度后不再向下查找
					depth = depth + 1
					if depth < common.Config.MaxSubqueryDepth {
						queries = append(queries, FindSubquery(depth, sub.Select)...)
					}
				}
				return true, nil
			}, expr.Select)
			common.LogIfWarn(err, "")

			// 如果没有嵌套的子查询了，返回子查询的SQL
			if noSub {
				sql := sqlparser.String(expr)
				// 去除SQL前后的括号
				queries = append(queries, sql[1:len(sql)-1])
			}

		}
		return true, nil
	}, node)
	common.LogIfWarn(err, "")
	return queries
}

// FindAllCondition 获取AST中所有的condition条件
func FindAllCondition(node sqlparser.SQLNode) []interface{} {
	common.Log.Debug("Enter:  FindAllCondition(), Caller: %s", common.Caller())
	var conditions []interface{}
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch node := node.(type) {
		case *sqlparser.ComparisonExpr, *sqlparser.RangeCond, *sqlparser.IsExpr:
			conditions = append(conditions, node)
		}
		return true, nil
	}, node)
	common.LogIfWarn(err, "")
	return conditions
}

// FindAllCols 获取AST中某个节点下所有的columns
func FindAllCols(node sqlparser.SQLNode, targets ...string) []*common.Column {
	var result []*common.Column
	// 获取节点内所有的列
	f := func(node sqlparser.SQLNode) {
		err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch col := node.(type) {
			case *sqlparser.ColName:
				result = common.MergeColumn(result, &common.Column{
					Name:  col.Name.String(),
					Table: col.Qualifier.Name.String(),
					DB:    col.Qualifier.Qualifier.String(),
					Alias: make([]string, 0),
				})
			}
			return true, nil
		}, node)
		common.LogIfWarn(err, "")
	}

	if len(targets) == 0 {
		// 如果不指定具体节点类型，则获取全部的column
		f(node)
	} else {
		// 根据target获取所有的节点
		for _, target := range targets {
			target = strings.Replace(strings.ToLower(target), " ", "", -1)
			err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
				switch node := node.(type) {
				case *sqlparser.Subquery:
					// 忽略子查询
				case *sqlparser.JoinTableExpr:
					if target == "join" {
						f(node)
					}
				case *sqlparser.Where:
					if target == "where" {
						f(node)
					}
				case *sqlparser.GroupBy:
					if target == "groupby" {
						f(node)
					}
				case sqlparser.OrderBy:
					if target == "orderby" {
						f(node)
					}
				}
				return true, nil
			}, node)
			common.LogIfWarn(err, "")
		}
	}

	return result
}

// GetSubqueryDepth 获取一条SQL的嵌套深度
func GetSubqueryDepth(node sqlparser.SQLNode) int {
	depth := 1
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch node.(type) {
		case *sqlparser.Subquery:
			depth++
		}
		return true, nil
	}, node)
	common.LogIfWarn(err, "")
	return depth
}

// getColumnName 获取node中Column具体的定义以及名称
func getColumnName(node sqlparser.SQLNode) (*sqlparser.ColName, string) {
	var colName *sqlparser.ColName
	str := ""
	switch c := node.(type) {
	case *sqlparser.ColName:
		if c.Qualifier.Name.IsEmpty() {
			str = fmt.Sprintf("`%s`", c.Name.String())
		} else {
			if c.Qualifier.Qualifier.IsEmpty() {
				str = fmt.Sprintf("`%s`.`%s`", c.Qualifier.Name.String(), c.Name.String())
			} else {
				str = fmt.Sprintf("`%s`.`%s`.`%s`",
					c.Qualifier.Qualifier.String(), c.Qualifier.Name.String(), c.Name.String())
			}
		}
		colName = c
	}
	return colName, str
}
