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
	"fmt"
	"strings"

	"github.com/XiaoMi/soar/ast"
	"github.com/XiaoMi/soar/common"
	"github.com/XiaoMi/soar/database"
	"github.com/XiaoMi/soar/env"

	"github.com/dchest/uniuri"
	"vitess.io/vitess/go/vt/sqlparser"
)

// IndexAdvisor 索引建议需要使用到的所有信息
type IndexAdvisor struct {
	vEnv      *env.VirtualEnv     // 线下虚拟测试环境（测试环境）
	rEnv      database.Connector  // 线上真实环境
	Ast       sqlparser.Statement // Vitess Parser生成的抽象语法树
	where     []*common.Column    // 所有where条件中用到的列
	whereEQ   []*common.Column    // where条件中可以加索引的等值条件列
	whereINEQ []*common.Column    // where条件中可以加索引的非等值条件列
	groupBy   []*common.Column    // group by可以加索引列
	orderBy   []*common.Column    // order by可以加索引列
	joinCond  [][]*common.Column  // 由于join condition跨层级间索引不可共用，需要多一个维度用来维护层级关系
	IndexMeta map[string]map[string]*database.TableIndexInfo
}

// IndexInfo 创建一条索引需要的信息
type IndexInfo struct {
	Name          string           `json:"name"`           // 索引名称
	Database      string           `json:"database"`       // 数据库名
	Table         string           `json:"table"`          // 表名
	DDL           string           `json:"ddl"`            // ALTER, CREATE等类型的DDL语句
	ColumnDetails []*common.Column `json:"column_details"` // 列详情
}

// IndexAdvises IndexAdvises列表
type IndexAdvises []IndexInfo

// mergeAdvices 合并索引建议
func mergeAdvices(dst []IndexInfo, src ...IndexInfo) IndexAdvises {
	if len(src) == 0 {
		return dst
	}

	for _, newIdx := range src {
		has := false
		for _, idx := range dst {
			if newIdx.DDL == idx.DDL {
				common.Log.Debug("merge index %s and %s", idx.Name, newIdx.Name)
				has = true
			}
		}

		if !has {
			dst = append(dst, newIdx)
		}
	}

	return dst
}

// NewAdvisor 构造一个 IndexAdvisor 的时候就会对其本身结构初始化
// 获取 condition 中的等值条件、非等值条件，以及group by 、 order by信息
func NewAdvisor(env *env.VirtualEnv, rEnv database.Connector, q Query4Audit) (*IndexAdvisor, error) {
	common.Log.Debug("Enter: NewAdvisor(), Caller: %s", common.Caller())
	if common.Config.TestDSN.Disable {
		return nil, fmt.Errorf("TestDSN is Disabled: %s", common.Config.TestDSN.Addr)
	}
	// DDL 检测
	switch stmt := q.Stmt.(type) {
	case *sqlparser.DDL:
		// 获取ast中用到的库表
		sqlMeta := ast.GetMeta(q.Stmt, nil)
		for db := range sqlMeta {
			dbRef := db
			if db == "" {
				dbRef = rEnv.Database
			}

			// DDL在Env初始化的时候已经执行过了
			if _, ok := env.TableMap[dbRef]; !ok {
				env.TableMap[dbRef] = make(map[string]string)
			}

			for _, tb := range sqlMeta[db].Table {
				env.TableMap[dbRef][tb.TableName] = tb.TableName
			}
		}

		return nil, nil

	case *sqlparser.DBDDL:
		// 忽略建库语句
		return nil, nil

	case *sqlparser.Use:
		// 如果是use，切基础环境
		env.Database = env.DBHash(stmt.DBName.String())
		return nil, nil
	}

	return &IndexAdvisor{
		vEnv: env,
		rEnv: rEnv,
		Ast:  q.Stmt,

		// 所有的FindXXXXCols尽最大可能先排除不需要加索引的列，但由于元数据在此阶段尚未补齐，给出的列有可能也无法添加索引
		// 后续需要通过CompleteColumnsInfo + calcCardinality补全后再进一步判断
		joinCond:  ast.FindJoinCols(q.Stmt),
		whereEQ:   ast.FindWhereEQ(q.Stmt),
		whereINEQ: ast.FindWhereINEQ(q.Stmt),
		groupBy:   ast.FindGroupByCols(q.Stmt),
		orderBy:   ast.FindOrderByCols(q.Stmt),
		where:     ast.FindAllCols(q.Stmt, "where"),
		IndexMeta: make(map[string]map[string]*database.TableIndexInfo),
	}, nil
}

/*

关于如何添加索引：
在《Relational Database Index Design and the Optimizers》一书中，作者提出著名的的三星索引理论（Three-Star Index）

To Qualify for the First Star:
Pick the columns from all equal predicates (WHERE COL = . . .).
Make these the first columns of the index—in any order. For CURSOR41, the three-star index will begin with
columns LNAME, CITY or CITY, LNAME. In both cases the index slice that must be scanned will be as thin as possible.

To Qualify for the Second Star:
Add the ORDER BY columns. Do not change the order of these columns, but ignore columns that were already
picked in step 1. For example, if CURSOR41 had redundant columns in the ORDER BY, say ORDER BY LNAME,
FNAME or ORDER BY FNAME, CITY, only FNAME would be added in this step. When FNAME is the third index column,
the result table will be in the right order without sorting. The first FETCH call will return the row with
the smallest FNAME value.

To Qualify for the Third Star:
Add all the remaining columns from the SELECT statement. The order of the columns added in this step
has no impact on the performance of the SELECT, but the cost of updates should be reduced by placing volatile
columns at the end. Now the index contains all the columns required for an index-only access path.

索引添加算法正是以这个理想化索策略添为基础，尽可能的给予"三星"索引建议。

但又如《High Performance MySQL》一书中所说，索引并不总是最好的工具。只有当索引帮助存储引擎快速查找到记录带来的好处大于其
带来的额外工作时，索引才是有效的。

因此，在三星索引理论的基础上引入启发式索引算法，在第二颗星的实现上做了部分改进，对于非等值条件只会添加散粒度最高的一列到索引中，
并基于总体列的使用情况作出判断，按需对order by、group by添加索引，由此来想`增强索引建议的通用性。

*/

// IndexAdvise 索引优化建议算法入口主函数
// TODO 索引顺序该如何确定
func (idxAdv *IndexAdvisor) IndexAdvise() IndexAdvises {
	// 支持不依赖DB的索引建议分析
	if common.Config.TestDSN.Disable {
		// 未开启Env原数据依赖，信息不全的情况下可能会给予错误的索引建议，请人工进行核查。
		common.Log.Warn("TestDSN.Disable = true")
	}

	// 检查否是否含有子查询
	subQueries := ast.FindSubquery(0, idxAdv.Ast)
	var subQueryAdvises []IndexInfo
	// 含有子查询对子查询进行单独评审，子查询评审建议报错忽略
	if len(subQueries) > 0 {
		for _, subSQL := range subQueries {
			stmt, err := sqlparser.Parse(subSQL)
			if err != nil {
				continue
			}
			q := Query4Audit{
				Query: subSQL,
				Stmt:  stmt,
			}
			subIdxAdv, _ := NewAdvisor(idxAdv.vEnv, idxAdv.rEnv, q)
			subQueryAdvises = append(subQueryAdvises, subIdxAdv.IndexAdvise()...)
		}
	}

	// 变量初始化，用于存放索引信息，按照db.tb.[cols]组织
	indexList := make(map[string]map[string][]*common.Column)

	// 为用到的每一列填充库名，表名等信息
	var joinCond [][]*common.Column
	for _, joinCols := range idxAdv.joinCond {
		joinCond = append(joinCond, CompleteColumnsInfo(idxAdv.Ast, joinCols, idxAdv.vEnv))
	}
	idxAdv.joinCond = joinCond

	idxAdv.where = CompleteColumnsInfo(idxAdv.Ast, idxAdv.where, idxAdv.vEnv)
	idxAdv.whereEQ = CompleteColumnsInfo(idxAdv.Ast, idxAdv.whereEQ, idxAdv.vEnv)
	idxAdv.whereINEQ = CompleteColumnsInfo(idxAdv.Ast, idxAdv.whereINEQ, idxAdv.vEnv)
	idxAdv.groupBy = CompleteColumnsInfo(idxAdv.Ast, idxAdv.groupBy, idxAdv.vEnv)
	idxAdv.orderBy = CompleteColumnsInfo(idxAdv.Ast, idxAdv.orderBy, idxAdv.vEnv)

	// 只要在开启使用env元数据的时候才会计算散粒度
	if !common.Config.TestDSN.Disable {
		// 计算joinCond, whereEQ, whereINEQ用到的每一列的散粒度，并排序，方便后续添加复合索引
		// groupBy, orderBy列按书写顺序给索引建议，不需要按散粒度排序
		idxAdv.calcCardinality(idxAdv.whereEQ)
		idxAdv.calcCardinality(idxAdv.whereINEQ)
		idxAdv.calcCardinality(idxAdv.orderBy)
		idxAdv.calcCardinality(idxAdv.groupBy)

		for i, joinCols := range idxAdv.joinCond {
			idxAdv.calcCardinality(joinCols)
			joinCols = common.ColumnSort(joinCols)
			idxAdv.joinCond[i] = joinCols
		}

		// 根据散粒度进行排序
		// 对所有列进行排序，按散粒度由大到小排序
		idxAdv.whereEQ = common.ColumnSort(idxAdv.whereEQ)
		idxAdv.whereINEQ = common.ColumnSort(idxAdv.whereINEQ)
		idxAdv.orderBy = common.ColumnSort(idxAdv.orderBy)
		idxAdv.groupBy = common.ColumnSort(idxAdv.groupBy)

	}

	// 是否指定Where条件，打标签
	hasWhere := false
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch where := node.(type) {
		case *sqlparser.Subquery:
			return false, nil
		case *sqlparser.Where:
			if where != nil {
				hasWhere = true
			}
		}
		return true, nil
	}, idxAdv.Ast)
	common.LogIfError(err, "")
	// 获取哪些列被忽略
	var ignore []*common.Column
	usedCols := append(idxAdv.whereINEQ, idxAdv.whereEQ...)

	for _, whereCol := range idxAdv.where {
		isUsed := false
		for _, used := range usedCols {
			if whereCol.Equal(used) {
				isUsed = true
			}
		}

		if !isUsed {
			common.Log.Debug("column %s in `%s`.`%s` will ignore when adding index", whereCol.DB, whereCol.Table, whereCol.Name)
			ignore = append(ignore, whereCol)
		}

	}

	// 索引优化算法入口，从这里开始放大招
	if hasWhere {
		// 有Where条件的先分析 等值条件
		for _, index := range idxAdv.whereEQ {
			// 对应列在前面已经按散粒度由大到小排序好了
			mergeIndex(indexList, index)
		}
		// 若存在非等值查询条件，可以给第一个非等值条件添加索引
		if len(idxAdv.whereINEQ) > 0 {
			mergeIndex(indexList, idxAdv.whereINEQ[0])
		}
		// 有WHERE条件，但WHERE条件未能给出索引建议就不能再加GROUP BY和ORDER BY建议了
		if len(ignore) == 0 {
			// 没有非等值查询条件时可以再为GroupBy和OrderBy添加索引
			for _, index := range idxAdv.groupBy {
				mergeIndex(indexList, index)
			}

			// OrderBy
			// 没有GroupBy时可以为OrderBy加索引
			if len(idxAdv.groupBy) == 0 {
				for _, index := range idxAdv.orderBy {
					mergeIndex(indexList, index)
				}
			}
		}
	} else {
		// 未指定Where条件的，只需要GroupBy和OrderBy的索引建议
		for _, index := range idxAdv.groupBy {
			mergeIndex(indexList, index)
		}

		// OrderBy
		// 没有GroupBy时可以为OrderBy加索引
		// 没有where条件时OrderBy的索引仅能够在索引覆盖的情况下被使用

		// if len(idxAdv.groupBy) == 0 {
		// 	for _, index := range idxAdv.orderBy {
		// 		mergeIndex(indexList, index)
		// 	}
		// }
	}

	// 开始整合索引信息，添加索引
	var indexes []IndexInfo

	// 为join添加索引
	// 获取 join condition 中需要加索引的表有哪些
	defaultDB := ""
	if !common.Config.TestDSN.Disable {
		defaultDB = idxAdv.vEnv.RealDB(idxAdv.vEnv.Database)
	}
	if !common.Config.OnlineDSN.Disable {
		defaultDB = idxAdv.rEnv.Database
	}

	// 根据join table的信息给予优化建议
	joinTableMeta := ast.FindJoinTable(idxAdv.Ast, nil).SetDefault(idxAdv.rEnv.Database).SetDefault(defaultDB)
	indexes = mergeAdvices(indexes, idxAdv.buildJoinIndex(joinTableMeta)...)

	if common.Config.TestDSN.Disable || common.Config.OnlineDSN.Disable {
		// 无env环境下只提供单列索引，无法确定table时不给予优化建议
		// 仅有table信息时给出的建议不包含DB信息
		indexes = mergeAdvices(indexes, idxAdv.buildIndexWithNoEnv(indexList)...)
	} else {
		// 给出尽可能详细的索引建议
		indexes = mergeAdvices(indexes, idxAdv.buildIndex(indexList)...)
	}

	indexes = mergeAdvices(indexes, subQueryAdvises...)

	// 在开启env的情况下，检查数据库版本，字段类型，索引总长度
	indexes = idxAdv.idxColsTypeCheck(indexes)

	// 在开启env的情况下，会对索引进行检查，对全索引进行过滤
	// 在前几步都不会对idx生成DDL语句，DDL语句在这里生成
	return idxAdv.mergeIndexes(indexes)
}

// idxColsTypeCheck 对超长的字段添加前缀索引，剔除无法添索引字段的列
// TODO 暂不支持fulltext索引，
func (idxAdv *IndexAdvisor) idxColsTypeCheck(idxList []IndexInfo) []IndexInfo {
	if common.Config.TestDSN.Disable {
		return rmSelfDupIndex(idxList)
	}

	var indexes []IndexInfo

	for _, idx := range idxList {
		var newCols []*common.Column
		var newColInfo []string
		// 索引总长度
		idxBytesTotal := 0
		isOverFlow := false
		for _, col := range idx.ColumnDetails {
			// 获取字段bytes
			bytes := col.GetDataBytes(common.Config.OnlineDSN.Version)
			tmpCol := col.Name
			overFlow := 0
			// 加上该列后是否索引长度过长
			if bytes < 0 {
				// bytes < 0 说明字段的长度是无法计算的
				common.Log.Warning("%s.%s data type not support %s, can't add index",
					col.Table, col.Name, col.DataType)
				continue
			}

			// idx bytes over flow
			if total := idxBytesTotal + bytes; total > common.Config.MaxIdxBytes {

				common.Log.Debug("bytes: %d, idxBytesTotal: %d, total: %d, common.Config.MaxIdxBytes: %d",
					bytes, idxBytesTotal, total, common.Config.MaxIdxBytes)

				overFlow = total - common.Config.MaxIdxBytes
				isOverFlow = true

			} else {
				idxBytesTotal = total
			}

			// common.Config.MaxIdxColBytes 默认大小 767
			if bytes > common.Config.MaxIdxBytesPerColumn || isOverFlow {
				// In 5.6, you may not include a column that equates to
				// bigger than 767 bytes: VARCHAR(255) CHARACTER SET utf8 or VARCHAR(191) CHARACTER SET utf8mb4.
				// In 5.7  you may not include a column that equates to
				// bigger than 3072 bytes.

				// v : 在 col.Character 字符集下每个字符占用 v bytes
				v, ok := common.CharSets[strings.ToLower(col.Character)]
				if !ok {
					// 找不到对应字符集，不添加索引
					// 如果出现不认识的字符集，认为每个字符占用4个字节
					common.Log.Warning("%s.%s(%s) charset not support yet %s, use default 4 bytes length",
						col.Table, col.Name, col.DataType, col.Character)
					v = 4
				}

				// 保留两个字节的安全余量
				length := (common.Config.MaxIdxBytesPerColumn - 2) / v
				if isOverFlow {
					// 在索引中添加该列会导致索引长度过长，建议根据需求转换为合理的前缀索引
					// _OPR_SPLIT_ 是自定的用于后续处理的特殊分隔符
					common.Log.Warning("adding index '%s(%s)' to table '%s' causes the index to be too long, overflow is %d",
						col.Name, col.DataType, col.Table, overFlow)
					tmpCol += fmt.Sprintf("_OPR_SPLIT_(N)")
				} else {
					// 索引没有过长，可以加一个最长的前缀索引
					common.Log.Warning("index column too large: %s.%s --> %s.%s(%d), data type: %s",
						col.Table, col.Name, col.Table, tmpCol, length, col.DataType)
					tmpCol += fmt.Sprintf("_OPR_SPLIT_(%d)", length)
				}

			}

			newCols = append(newCols, col)
			newColInfo = append(newColInfo, tmpCol)
		}

		// 为新索引重建索引语句
		idxName := "idx_"
		idxCols := ""
		for i, newCol := range newColInfo {
			// 对名称和可能存在的长度进行拼接
			// 用等号进行分割
			tmp := strings.Split(newCol, "_OPR_SPLIT_")
			idxName += tmp[0]
			if len(tmp) > 1 {
				idxCols += tmp[0] + "`" + tmp[1]
			} else {
				idxCols += tmp[0] + "`"
			}

			if i+1 < len(newColInfo) {
				idxName += "_"
				idxCols += ",`"
			}
		}

		// 索引名称最大长度64
		if len(idxName) > 64 {
			common.Log.Warn("index '%s' name large than 64", idxName)
			idxName = strings.TrimRight(idxName[:64], "_")
		}

		// 新的alter语句
		newDDL := fmt.Sprintf("alter table `%s`.`%s` add index `%s` (`%s)", idxAdv.vEnv.RealDB(idx.Database),
			idx.Table, idxName, idxCols)

		// 将筛选改造后的索引信息信息加入到新的索引列表中
		idx.ColumnDetails = newCols
		idx.DDL = newDDL
		indexes = append(indexes, idx)
	}

	return indexes
}

// mergeIndexes 与线上环境对比，将给出的索引建议进行去重
func (idxAdv *IndexAdvisor) mergeIndexes(idxList []IndexInfo) []IndexInfo {
	// TODO 暂不支持前缀索引去重
	if common.Config.TestDSN.Disable {
		return rmSelfDupIndex(idxList)
	}

	var indexes []IndexInfo
	for _, idx := range idxList {
		// 将DB替换成vEnv中的数据库名称
		dbInVEnv := idx.Database
		if _, ok := idxAdv.vEnv.DBRef[idx.Database]; ok {
			dbInVEnv = idxAdv.vEnv.DBRef[idx.Database]
		}

		// 检测索引添加的表是否是视图
		if idxAdv.vEnv.IsView(idx.Table) {
			common.Log.Info("%s.%s is a view. no need indexed", idx.Database, idx.Table)
			continue
		}

		// 检测是否存在重复索引
		indexMeta := idxAdv.IndexMeta[dbInVEnv][idx.Table]
		isExisted := false

		// 检测无索引列的情况
		if len(idx.ColumnDetails) < 1 {
			continue
		}

		if existedIndexes := indexMeta.FindIndex(database.IndexColumnName, idx.ColumnDetails[0].Name); len(existedIndexes) > 0 {
			for _, existedIdx := range existedIndexes {
				// flag: 用于标记已存在的索引是否是约束条件
				isConstraint := false

				var cols []string
				var colsDetail []*common.Column

				// 把已经存在的key摘出来遍历一遍对比是否是包含关系
				for _, col := range indexMeta.FindIndex(database.IndexKeyName, existedIdx.KeyName) {
					cols = append(cols, col.ColumnName)
					colsDetail = append(colsDetail, &common.Column{
						Name:  col.ColumnName,
						Table: idx.Table,
						DB:    idx.ColumnDetails[0].DB,
					})
				}

				// 判断已存在的索引是否属于约束条件(唯一索引、主键)
				// 这里可以忽略是否含有外键的情况，因为索引已经重复了，添加了新索引后原先重复的索引是可以删除的。
				if existedIdx.NonUnique == 0 {
					common.Log.Debug("%s.%s表%s为约束条件", dbInVEnv, idx.Table, existedIdx.KeyName)
					isConstraint = true
				}

				// 如果已存在的索引与索引建议存在重叠，则说明无需添加新索引或可能需要给出删除索引的建议
				if common.IsColsPart(colsDetail, idx.ColumnDetails) {
					idxName := existedIdx.KeyName
					// 如果已经存在的索引包含需要添加的索引，则无需添加
					if len(colsDetail) >= len(idx.ColumnDetails) {
						common.Log.Info(" `%s`.`%s` %s already had a index `%s`",
							idx.Database, idx.Table, strings.Join(cols, ","), idxName)
						isExisted = true
						continue
					}

					// 库、表、列名需要用反撇转义
					// TODO 关于外键索引去重的优雅解决方案
					if !isConstraint {
						if common.Config.AllowDropIndex {
							alterSQL := fmt.Sprintf("alter table `%s`.`%s` drop index `%s`", idx.Database, idx.Table, idxName)
							indexes = append(indexes, IndexInfo{
								Name:          idxName,
								Database:      idx.Database,
								Table:         idx.Table,
								DDL:           alterSQL,
								ColumnDetails: colsDetail,
							})
						} else {
							common.Log.Warning("In table `%s`, the new index of column `%s` contains index `%s`,"+
								" maybe you could drop one of them.", existedIdx.Table,
								strings.Join(cols, ","), idxName)
						}
					}
				}
			}
		}

		if !isExisted {
			// 检测索引名称是否重复?
			if existedIndexes := indexMeta.FindIndex(database.IndexKeyName, idx.Name); len(existedIndexes) > 0 {
				var newName string
				if len(idx.Name) < 59 {
					newName = idx.Name + "_" + uniuri.New()[:4]
				} else {
					newName = idx.Name[:59] + "_" + uniuri.New()[:4]
				}

				common.Log.Warning("duplicate index name '%s', new name is '%s'", idx.Name, newName)
				idx.DDL = strings.Replace(idx.DDL, idx.Name, newName, -1)
				idx.Name = newName
			}

			// 添加合并
			indexes = mergeAdvices(indexes, idx)
		}

	}

	// 对索引进行去重
	return rmSelfDupIndex(indexes)
}

// rmSelfDupIndex 去重传入的[]IndexInfo中重复的索引
func rmSelfDupIndex(indexes []IndexInfo) []IndexInfo {
	var resultIndex []IndexInfo
	tmpIndexList := indexes
	for _, a := range indexes {
		tmp := a
		for i, b := range tmpIndexList {
			if common.IsColsPart(tmp.ColumnDetails, b.ColumnDetails) && tmp.Name != b.Name {
				if len(b.ColumnDetails) > len(tmp.ColumnDetails) {
					common.Log.Debug("remove duplicate index: %s", tmp.Name)
					tmp = b
				}

				if i < len(tmpIndexList) {
					tmpIndexList = append(tmpIndexList[:i], tmpIndexList[i+1:]...)
				} else {
					tmpIndexList = tmpIndexList[:i]
				}

			}
		}
		resultIndex = mergeAdvices(resultIndex, tmp)
	}

	return resultIndex
}

// buildJoinIndex 检查Join中使用的库表是否需要添加索引并给予索引建议
func (idxAdv *IndexAdvisor) buildJoinIndex(meta common.Meta) []IndexInfo {
	var indexes []IndexInfo
	for _, IndexCols := range idxAdv.joinCond {
		// 如果该列的库表为join condition中需要添加索引的库表
		indexColsList := make(map[string]map[string][]*common.Column)
		for _, col := range IndexCols {
			mergeIndex(indexColsList, col)

		}

		if common.Config.TestDSN.Disable || common.Config.OnlineDSN.Disable {
			indexes = mergeAdvices(indexes, idxAdv.buildIndexWithNoEnv(indexColsList)...)
			continue
		}

		indexes = mergeAdvices(indexes, idxAdv.buildIndex(indexColsList)...)
	}
	return indexes
}

// buildIndex 尽可能的将 map[string]map[string][]*common.Column 转换成 []IndexInfo
// 此处不判断索引是否重复
func (idxAdv *IndexAdvisor) buildIndex(idxList map[string]map[string][]*common.Column) []IndexInfo {
	var indexes []IndexInfo
	for db, tbs := range idxList {
		for tb, cols := range tbs {

			// 单个索引中含有的列收 config 中参数限制
			if len(cols) > common.Config.MaxIdxColsCount {
				cols = cols[:common.Config.MaxIdxColsCount]
			}

			var colNames []string
			for _, col := range cols {
				if col.DB == "" || col.Table == "" {
					common.Log.Warn("can not get the meta info of column '%s'", col.Name)
					continue
				}
				colNames = append(colNames, col.Name)
			}

			if len(colNames) == 0 {
				continue
			}

			idxName := "idx_" + strings.Join(colNames, "_")

			// 索引名称最大长度64
			if len(idxName) > 64 {
				common.Log.Warn("index '%s' name large than 64", idxName)
				idxName = strings.TrimRight(idxName[:64], "_")
			}

			alterSQL := fmt.Sprintf("alter table `%s`.`%s` add index `%s` (`%s`)", idxAdv.vEnv.RealDB(db), tb,
				idxName, strings.Join(colNames, "`,`"))

			indexes = append(indexes, IndexInfo{
				Name:          idxName,
				Database:      idxAdv.vEnv.RealDB(db),
				Table:         tb,
				DDL:           alterSQL,
				ColumnDetails: cols,
			})
		}
	}
	return indexes
}

// buildIndexWithNoEnv 忽略原数据，给予最基础的索引
func (idxAdv *IndexAdvisor) buildIndexWithNoEnv(indexList map[string]map[string][]*common.Column) []IndexInfo {
	// 如果不获取数据库原信息，则不去判断索引是否重复，且只给单列加索引
	var indexes []IndexInfo
	for _, tableIndex := range indexList {
		for _, indexCols := range tableIndex {
			for _, col := range indexCols {
				if col.Table == "" {
					common.Log.Warn("can not get the meta info of column '%s'", col.Name)
					continue
				}
				idxName := "idx_" + col.Name
				// 库、表、列名需要用反撇转义
				alterSQL := fmt.Sprintf("alter table `%s`.`%s` add index `%s` (`%s`)", idxAdv.vEnv.RealDB(col.DB), col.Table, idxName, col.Name)
				if col.DB == "" {
					alterSQL = fmt.Sprintf("alter table `%s` add index `%s` (`%s`)", col.Table, idxName, col.Name)
				}

				indexes = append(indexes, IndexInfo{
					Name:          idxName,
					Database:      idxAdv.vEnv.RealDB(col.DB),
					Table:         col.Table,
					DDL:           alterSQL,
					ColumnDetails: []*common.Column{col},
				})
			}

		}
	}
	return indexes
}

// mergeIndex 将索引用到的列去重后合并到一起
func mergeIndex(idxList map[string]map[string][]*common.Column, column *common.Column) {
	db := column.DB
	tb := column.Table
	if idxList[db] == nil {
		idxList[db] = make(map[string][]*common.Column)
	}
	if idxList[db][tb] == nil {
		idxList[db][tb] = make([]*common.Column, 0)
	}

	// 去除重复列Append
	exist := false
	for _, cl := range idxList[db][tb] {
		if cl.Name == column.Name {
			exist = true
		}
	}
	if !exist {
		idxList[db][tb] = append(idxList[db][tb], column)
	}
}

// CompleteColumnsInfo 补全索引可能会用到列的所属库名、表名等信息
func CompleteColumnsInfo(stmt sqlparser.Statement, cols []*common.Column, env *env.VirtualEnv) []*common.Column {
	// 如果传过来的列是空的，没必要跑逻辑
	if len(cols) == 0 {
		return cols
	}

	// 从Ast中拿到DBStructure，包含所有表的相关信息
	dbs := ast.GetMeta(stmt, nil)

	// 此处生成的meta信息中不应该含有""db的信息，若DB为空则认为是已传入的db为默认db并进行信息补全
	// BUG Fix:
	// 修补dbs中空DB的导致后续补全列信息时无法获取正确table名称的问题
	if _, ok := dbs[""]; ok {
		dbs[env.Database] = dbs[""]
		delete(dbs, "")
	}

	tableCount := 0
	for db := range dbs {
		for tb := range dbs[db].Table {
			if tb != "" {
				tableCount++
			}
		}
	}

	var noEnvTmp []*common.Column
	for _, col := range cols {
		for db := range dbs {
			// 对每一列进行比对，将别名转换为正确的名称
			find := false
			for _, tb := range dbs[db].Table {
				for _, tbAlias := range tb.TableAliases {
					if col.Table != "" && col.Table == tbAlias {
						common.Log.Debug("column '%s' prefix change: %s --> %s", col.Name, col.Table, tb.TableName)
						find = true
						col.Table = tb.TableName
						col.DB = db
						break
					}
				}
				if find {
					break
				}

			}

			// 如果不依赖env环境，利用ast中包含的信息推理列的库表信息
			if common.Config.TestDSN.Disable {
				if tableCount == 1 {
					for _, tb := range dbs[db].Table {
						col.Table = tb.TableName

						// 因为tableMeta是按照库表组织的树状结构，db变量贯穿全局
						// 只有在最终赋值前才能根据逻辑变更补全
						if db == "" {
							db = env.Database
						}
						col.DB = db
					}
				}

				// 如果SQL中含有的表大于一个，则使用的列中必须含有前缀，不然无法判断该列属于哪个表
				// 如果某一列未含有前缀信息，则认为两张表中都含有该列，需要由人去判断
				if tableCount > 1 {
					if col.Table == "" {
						for _, tb := range dbs[db].Table {
							if tb.TableName == "" {
								common.Log.Warn("can not get the meta info of column '%s'", col.Name)
							}

							if db == "" {
								db = env.RealDB(env.Database)
							}
							col.Table = tb.TableName
							col.DB = db

							tmp := *col
							tmp.Table = tb.TableName
							tmp.DB = db

							noEnvTmp = append(noEnvTmp, &tmp)
						}
					}

					if col.DB == "" {
						if db == "" {
							db = env.Database
						}
						col.DB = db
					}
				}

				break
			}

			// 将已经获取到正确表信息的列信息带入到env中，利用show columns where table 获取库表信息
			// 此出会传入之前从ast中，该 db 下获取的所有表来作为where限定条件，
			// 防止与SQL无关的库表信息干扰准确性
			// 此处传入的是测试环境，DB是经过变换的，所以在寻找列名的时候需要将DB名称转换成测试环境中经过hash的DB名称
			// 不然会找不到col的信息
			realCols, err := env.FindColumn(col.Name, env.DBHash(db), dbs.Tables(db)...)
			if err != nil {
				common.Log.Warn("%v", err)
				continue
			}

			// 对比 column 信息中的表名与从 env 中获取的库表名的一致性
			for _, realCol := range realCols {
				if col.Name == realCol.Name {
					// 如果查询到了列名一致，但从ast中获取的列的前缀与env中的表信息不符
					// 1.存在一个同名列，但不同表，该情况下忽略
					// 2.存在一个未正确转换的别名(如表名为)，该情况下修正，大概率是正确的
					if col.Table != "" && col.Table != realCol.Table {
						has, _ := env.FindColumn(col.Name, env.DBHash(db), col.Table)
						if len(has) > 0 {
							realCol = has[0]
						}
					}

					col.DataType = realCol.DataType
					col.Table = realCol.Table
					col.DB = env.RealDB(realCol.DB)
					col.Character = realCol.Character
					col.Collation = realCol.Collation

				}
			}
		}

	}

	// 如果不依赖env环境，将可能存在的列也加入到索引预处理列表中
	if common.Config.TestDSN.Disable {
		cols = append(cols, noEnvTmp...)
	}

	return cols
}

// calcCardinality 计算每一列的散粒度
// 这个函数需要在补全列的库表信息之后再调用，否则无法确定要计算列的归属
func (idxAdv *IndexAdvisor) calcCardinality(cols []*common.Column) []*common.Column {
	common.Log.Debug("Enter: calcCardinality(), Caller: %s", common.Caller())
	tmpDB := *idxAdv.vEnv
	for _, col := range cols {
		// 补全对应列的库->表->索引信息到IndexMeta
		// 这将在后面用于判断某一列是否为主键或单列唯一索引，快速返回散粒度
		if col.DB == "" {
			col.DB = idxAdv.vEnv.Database
		}
		realDB := idxAdv.vEnv.DBHash(col.DB)
		if idxAdv.IndexMeta[realDB] == nil {
			idxAdv.IndexMeta[realDB] = make(map[string]*database.TableIndexInfo)
		}

		if idxAdv.IndexMeta[realDB][col.Table] == nil {
			tmpDB.Database = realDB
			indexInfo, err := tmpDB.ShowIndex(col.Table)
			if err != nil {
				// 如果是不存在的表就会报错，报错的可能性有三个：
				// 1.数据库错误  2.表不存在  3.临时表
				// 而这三种错误都是不需要在这一层关注的，直接跳过
				common.Log.Debug("calcCardinality error: %v", err)
				continue
			}

			// 将获取的索引信息以db.tb维度组织到IndexMeta中
			idxAdv.IndexMeta[realDB][col.Table] = indexInfo
		}

		// 检查对应列是否为主键或单列唯一索引，如果满足直接返回1，不再重复计算，提高效率
		// 多列复合唯一索引不能跳过计算，单列普通索引不能跳过计算
		for _, index := range idxAdv.IndexMeta[realDB][col.Table].IdxRows {
			// 根据索引的名称判断该索引包含的列数，列数大于1即为复合索引
			columnCount := len(idxAdv.IndexMeta[realDB][col.Table].FindIndex(database.IndexKeyName, index.KeyName))
			if col.Name == index.ColumnName {
				// 主键、唯一键 无需计算散粒度
				if (index.KeyName == "PRIMARY" || index.NonUnique == 0) && columnCount == 1 {
					common.Log.Debug("column '%s' is PK or UK, no need to calculate cardinality.", col.Name)
					col.Cardinality = 1
					break
				}
			}

		}

		// 给非 PRIMARY、UNIQUE 的列计算散粒度
		if col.Cardinality != 1 {
			col.Cardinality = idxAdv.vEnv.ColumnCardinality(col.Table, col.Name)
		}
	}

	return cols
}

// Format 用于格式化输出索引建议
func (idxAdvs IndexAdvises) Format() map[string]Rule {
	rulesMap := make(map[string]Rule)
	number := 1
	rules := make(map[string]*Rule)
	sqls := make(map[string][]string)

	for _, advise := range idxAdvs {
		advKey := advise.Database + advise.Table

		if _, ok := sqls[advKey]; !ok {
			sqls[advKey] = make([]string, 0)
		}

		sqls[advKey] = append(sqls[advKey], advise.DDL)

		if _, ok := rules[advKey]; !ok {
			summary := fmt.Sprintf("为%s库的%s表添加索引", advise.Database, advise.Table)
			if advise.Database == "" {
				summary = fmt.Sprintf("为%s表添加索引", advise.Table)
			}

			rules[advKey] = &Rule{
				Summary:  summary,
				Content:  "",
				Severity: "L2",
			}
		}

		for _, col := range advise.ColumnDetails {
			// 为了更好地显示效果
			if common.Config.Sampling {
				cardinal := fmt.Sprintf("%0.2f", col.Cardinality*100)
				if cardinal != "0.00" {
					rules[advKey].Content += fmt.Sprintf("为列%s添加索引,散粒度为: %s%%; ",
						col.Name, cardinal)
				}
			} else {
				rules[advKey].Content += fmt.Sprintf("为列%s添加索引;", col.Name)
			}
		}
		// 清理多余的标点
		rules[advKey].Content = strings.Trim(rules[advKey].Content, common.Config.Delimiter)
	}

	for adv := range rules {
		key := fmt.Sprintf("IDX.%03d", number)
		ddl := ast.MergeAlterTables(sqls[adv]...)
		// 由于传入合并的SQL都是一张表的，所以一定只会输出一条ddl语句
		for _, v := range ddl {
			rules[adv].Case = v
		}

		// set item
		rules[adv].Item = key

		rulesMap[key] = *rules[adv]

		number++
	}

	return rulesMap
}

// HeuristicCheck 依赖数据字典的启发式检查
// IndexAdvisor会构建测试环境和数据字典，所以放在这里实现
func (idxAdv *IndexAdvisor) HeuristicCheck(q Query4Audit) map[string]Rule {
	var rule Rule
	heuristicSuggest := make(map[string]Rule)
	if common.Config.OnlineDSN.Disable && common.Config.TestDSN.Disable {
		return heuristicSuggest
	}

	ruleFuncs := []func(*IndexAdvisor) Rule{
		(*IndexAdvisor).RuleImplicitConversion, // ARG.003
		// (*IndexAdvisor).RuleImpossibleOuterJoin, // TODO: JOI.003, JOI.004
		(*IndexAdvisor).RuleGroupByConst,     // CLA.004
		(*IndexAdvisor).RuleOrderByConst,     // CLA.005
		(*IndexAdvisor).RuleUpdatePrimaryKey, // CLA.016
	}

	for _, f := range ruleFuncs {
		rule = f(idxAdv)
		if rule.Item != "OK" {
			heuristicSuggest[rule.Item] = rule
		}
	}
	return heuristicSuggest
}

// DuplicateKeyChecker 对所有用到的库表检查是否存在重复索引
func DuplicateKeyChecker(conn *database.Connector, databases ...string) map[string]Rule {
	common.Log.Debug("Enter:  DuplicateKeyChecker, Caller: %s", common.Caller())
	// 复制一份online connector,防止环境切换影响其他功能的使用
	tmpOnline := *conn
	ruleMap := make(map[string]Rule)
	number := 1

	// 错误处理，用于汇总所有的错误
	funcErrCheck := func(err error) {
		if err != nil {
			if sug, ok := ruleMap["ERR.003"]; ok {
				sug.Content += fmt.Sprintf("; %s", err.Error())
			} else {
				ruleMap["ERR.003"] = Rule{
					Item:     "ERR.003",
					Severity: "L8",
					Content:  err.Error(),
				}
			}
		}
	}

	// 不指定DB的时候检查online dsn中的DB
	if len(databases) == 0 {
		databases = append(databases, tmpOnline.Database)
	}

	for _, db := range databases {
		// 获取所有的表
		tmpOnline.Database = db
		tables, err := tmpOnline.ShowTables()

		if err != nil {
			funcErrCheck(err)
			if !common.Config.DryRun {
				return ruleMap
			}
		}

		for _, tb := range tables {
			// 获取表中所有的索引
			idxMap := make(map[string][]*common.Column)
			idxInfo, err := tmpOnline.ShowIndex(tb)
			if err != nil {
				funcErrCheck(err)
				if !common.Config.DryRun {
					return ruleMap
				}
			}

			// 枚举所有的索引信息，提取用到的列
			for _, idx := range idxInfo.IdxRows {
				if _, ok := idxMap[idx.KeyName]; !ok {
					idxMap[idx.KeyName] = make([]*common.Column, 0)
					for _, col := range idxInfo.FindIndex(database.IndexKeyName, idx.KeyName) {
						idxMap[idx.KeyName] = append(idxMap[idx.KeyName], &common.Column{
							Name:  col.ColumnName,
							Table: tb,
							DB:    db,
						})
					}
				}
			}

			// 对索引进行重复检查
			hasDup := false
			content := ""

			for k1, cl1 := range idxMap {
				for k2, cl2 := range idxMap {
					if k1 != k2 && common.IsColsPart(cl1, cl2) {
						hasDup = true
						col1Str := common.JoinColumnsName(cl1, ", ")
						col2Str := common.JoinColumnsName(cl2, ", ")
						content += fmt.Sprintf("索引%s(%s)与%s(%s)重复;", k1, col1Str, k2, col2Str)
						common.Log.Debug(" %s.%s has duplicate index %s(%s) <--> %s(%s)", db, tb, k1, col1Str, k2, col2Str)
					}
				}
				delete(idxMap, k1)
			}

			// TODO 重复索引检查添加对约束及索引的判断，提供重复索引的删除功能
			if hasDup {
				tmpOnline.Database = db
				ddl, _ := tmpOnline.ShowCreateTable(tb)
				key := fmt.Sprintf("IDX.%03d", number)
				ruleMap[key] = Rule{
					Item:     key,
					Severity: "L2",
					Summary:  fmt.Sprintf("%s.%s存在重复的索引", db, tb),
					Content:  content,
					Case:     ddl,
				}
				number++
			}
		}
	}

	return ruleMap
}
