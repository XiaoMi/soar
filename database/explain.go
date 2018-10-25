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

package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/XiaoMi/soar/ast"
	"github.com/XiaoMi/soar/common"

	"github.com/tidwall/gjson"
	"vitess.io/vitess/go/vt/sqlparser"
)

// format_type 支持的输出格式
// https://dev.mysql.com/doc/refman/5.7/en/explain-output.html
const (
	TraditionalFormatExplain = iota // 默认输出
	JSONFormatExplain               // JSON格式输出
)

// ExplainFormatType EXPLAIN支持的FORMAT_TYPE
var ExplainFormatType = map[string]int{
	"traditional": 0,
	"json":        1,
}

// explain_type
const (
	TraditionalExplainType = iota // 默认转出
	ExtendedExplainType           // EXTENDED输出
	PartitionsExplainType         // PARTITIONS输出
)

// ExplainType EXPLAIN命令支持的参数
var ExplainType = map[string]int{
	"traditional": 0,
	"extended":    1,
	"partitions":  2,
}

// 为TraditionalFormatExplain准备的结构体 { start

// ExplainInfo 用于存放Explain信息
type ExplainInfo struct {
	SQL           string
	ExplainFormat int
	ExplainRows   []*ExplainRow
	ExplainJSON   *ExplainJSON
	Warnings      []*ExplainWarning
	QueryCost     float64
}

// ExplainRow 单行Explain
type ExplainRow struct {
	ID           int
	SelectType   string
	TableName    string
	Partitions   string // explain partitions
	AccessType   string
	PossibleKeys []string
	Key          string
	KeyLen       string // 索引长度，如果发生了index_merge， KeyLen格式为N,N，所以不能定义为整型
	Ref          []string
	Rows         int
	Filtered     float64 // 5.6 JSON, 5.7+, 5.5 EXTENDED
	Scalability  string  // O(1), O(n), O(log n), O(log n)+
	Extra        string
}

// ExplainWarning explain extended后SHOW WARNINGS输出的结果
type ExplainWarning struct {
	Level   string
	Code    int
	Message string
}

// 为TraditionalFormatExplain准备的结构体 end }

// 为JSONFormatExplain准备的结构体 { start

// ExplainJSONCostInfo JSON
type ExplainJSONCostInfo struct {
	ReadCost        string `json:"read_cost"`
	EvalCost        string `json:"eval_cost"`
	PrefixCost      string `json:"prefix_cost"`
	DataReadPerJoin string `json:"data_read_per_join"`
	QueryCost       string `json:"query_cost"`
	SortCost        string `json:"sort_cost"`
}

// ExplainJSONMaterializedFromSubquery JSON
type ExplainJSONMaterializedFromSubquery struct {
	UsingTemporaryTable bool                   `json:"using_temporary_table"`
	Dependent           bool                   `json:"dependent"`
	Cacheable           bool                   `json:"cacheable"`
	QueryBlock          *ExplainJSONQueryBlock `json:"query_block"`
}

// 该变量用于存放JSON到Traditional模式的所有ExplainJSONTable
var explainJSONTables []*ExplainJSONTable

// ExplainJSONTable JSON
type ExplainJSONTable struct {
	TableName                string                              `json:"table_name"`
	AccessType               string                              `json:"access_type"`
	PossibleKeys             []string                            `json:"possible_keys"`
	Key                      string                              `json:"key"`
	UsedKeyParts             []string                            `json:"used_key_parts"`
	KeyLength                string                              `json:"key_length"`
	Ref                      []string                            `json:"ref"`
	RowsExaminedPerScan      int                                 `json:"rows_examined_per_scan"`
	RowsProducedPerJoin      int                                 `json:"rows_produced_per_join"`
	Filtered                 string                              `json:"filtered"`
	UsingIndex               bool                                `json:"using_index"`
	UsingIndexForGroupBy     bool                                `json:"using_index_for_group_by"`
	CostInfo                 ExplainJSONCostInfo                 `json:"cost_info"`
	UsedColumns              []string                            `json:"used_columns"`
	AttachedCondition        string                              `json:"attached_condition"`
	AttachedSubqueries       []ExplainJSONSubqueries             `json:"attached_subqueries"`
	MaterializedFromSubquery ExplainJSONMaterializedFromSubquery `json:"materialized_from_subquery"`
}

// ExplainJSONNestedLoop JSON
type ExplainJSONNestedLoop struct {
	Table ExplainJSONTable `json:"table"`
}

// ExplainJSONBufferResult JSON
type ExplainJSONBufferResult struct {
	UsingTemporaryTable bool                    `json:"using_temporary_table"`
	NestedLoop          []ExplainJSONNestedLoop `json:"nested_loop"`
}

// ExplainJSONSubqueries JSON
type ExplainJSONSubqueries struct {
	Dependent  bool                  `json:"dependent"`
	Cacheable  bool                  `json:"cacheable"`
	QueryBlock ExplainJSONQueryBlock `json:"query_block"`
}

// ExplainJSONGroupingOperation JSON
type ExplainJSONGroupingOperation struct {
	UsingTemporaryTable bool                    `json:"using_temporary_table"`
	UsingFilesort       bool                    `json:"using_filesort"`
	Table               ExplainJSONTable        `json:"table"`
	CostInfo            ExplainJSONCostInfo     `json:"cost_info"`
	NestedLoop          []ExplainJSONNestedLoop `json:"nested_loop"`
	GroupBySubqueries   []ExplainJSONSubqueries `json:"group_by_subqueries"`
}

// ExplainJSONDuplicatesRemoval JSON
type ExplainJSONDuplicatesRemoval struct {
	UsingTemporaryTable bool                         `json:"using_temporary_table"`
	UsingFilesort       bool                         `json:"using_filesort"`
	BufferResult        ExplainJSONBufferResult      `json:"buffer_result"`
	GroupingOperation   ExplainJSONGroupingOperation `json:"grouping_operation"`
}

// ExplainJSONOrderingOperation JSON
type ExplainJSONOrderingOperation struct {
	UsingFilesort     bool                         `json:"using_filesort"`
	Table             ExplainJSONTable             `json:"table"`
	DuplicatesRemoval ExplainJSONDuplicatesRemoval `json:"duplicates_removal"`
	GroupingOperation ExplainJSONGroupingOperation `json:"grouping_operation"`
	OderbySubqueries  []ExplainJSONSubqueries      `json:"order_by_subqueries"`
}

// ExplainJSONQueryBlock JSON
type ExplainJSONQueryBlock struct {
	SelectID                int                          `json:"select_id"`
	CostInfo                ExplainJSONCostInfo          `json:"cost_info"`
	Table                   ExplainJSONTable             `json:"table"`
	NestedLoop              []ExplainJSONNestedLoop      `json:"nested_loop"`
	OrderingOperation       ExplainJSONOrderingOperation `json:"ordering_operation"`
	GroupingOperation       ExplainJSONGroupingOperation `json:"grouping_operation"`
	OptimizedAwaySubqueries []ExplainJSONSubqueries      `json:"optimized_away_subqueries"`
	HavingSubqueries        []ExplainJSONSubqueries      `json:"having_subqueries"`
	SelectListSubqueries    []ExplainJSONSubqueries      `json:"select_list_subqueries"`
	UpdateValueSubqueries   []ExplainJSONSubqueries      `json:"update_value_subqueries"`
	QuerySpecifications     []ExplainJSONSubqueries      `json:"query_specifications"`
	UnionResult             ExplainJSONUnionResult       `json:"union_result"`
	Message                 string                       `json:"message"`
}

// ExplainJSONUnionResult JSON
type ExplainJSONUnionResult struct {
	UsingTemporaryTable bool                    `json:"using_temporary_table"`
	TableName           string                  `json:"table_name"`
	AccessType          string                  `json:"access_type"`
	QuerySpecifications []ExplainJSONSubqueries `json:"query_specifications"`
}

// ExplainJSON 根结点
type ExplainJSON struct {
	QueryBlock ExplainJSONQueryBlock `json:"query_block"`
}

// 为JSONFormatExplain准备的结构体 end }

// ExplainKeyWords 需要解释的关键字
var ExplainKeyWords = []string{
	"access_type",
	"attached_condition",
	"attached_subqueries",
	"buffer_result",
	"cacheable",
	"cost_info",
	"data_read_per_join",
	"dependent",
	"duplicates_removal",
	"eval_cost",
	"filtered",
	"group_by_subqueries",
	"grouping_operation",
	"having_subqueries",
	"key",
	"key_length",
	"materialized_from_subquery",
	"message",
	"nested_loop",
	"optimized_away_subqueries",
	"order_by_subqueries",
	"ordering_operation",
	"possible_keys",
	"prefix_cost",
	"query_block",
	"query_cost",
	"query_specifications",
	"read_cost",
	"ref",
	"rows_examined_per_scan",
	"rows_produced_per_join",
	"select_id",
	"select_list_subqueries",
	"sort_cost",
	"table",
	"table_name",
	"union_result",
	"update_value_subqueries",
	"used_columns",
	"used_key_parts",
	"using_filesort",
	"using_index",
	"using_index_for_group_by",
	"using_temporary_table",
}

// ExplainColumnIndent EXPLAIN表头
var ExplainColumnIndent = map[string]string{
	"id":            "id为SELECT的标识符. 它是在SELECT查询中的顺序编号. 如果这一行表示其他行的union结果, 这个值可以为空. 在这种情况下, table列会显示为形如<union M,N>, 表示它是id为M和N的查询行的联合结果.",
	"select_type":   "表示查询的类型. ",
	"table":         "输出行所引用的表.",
	"type":          "type显示连接使用的类型, 有关不同类型的描述, 请参见解释连接类型.",
	"possible_keys": "指出MySQL能在该表中使用哪些索引有助于查询. 如果为空, 说明没有可用的索引.",
	"key":           "MySQL实际从possible_keys选择使用的索引. 如果为NULL, 则没有使用索引. 很少情况下, MySQL会选择优化不足的索引. 这种情况下, 可以在select语句中使用USE INDEX (indexname)来强制使用一个索引或者用IGNORE INDEX (indexname)来强制MySQL忽略索引.",
	"key_len":       "显示MySQL使用索引键的长度. 如果key是NULL, 则key_len为NULL. 使用的索引的长度. 在不损失精确性的情况下, 长度越短越好.",
	"ref":           "显示索引的哪一列被使用了.",
	"rows":          "表示MySQL认为必须检查的用来返回请求数据的行数.",
	"filtered":      "表示返回结果的行占需要读到的行(rows列的值)的百分比.",
	"Extra":         "该列显示MySQL在查询过程中的一些详细信息, MySQL查询优化器执行查询的过程中对查询计划的重要补充信息.",
}

// ExplainSelectType EXPLAIN中SELECT TYPE会出现的类型
var ExplainSelectType = map[string]string{
	"SIMPLE":               "简单SELECT(不使用UNION或子查询等).",
	"PRIMARY":              "最外层的select.",
	"UNION":                "UNION中的第二个或后面的SELECT查询, 不依赖于外部查询的结果集.",
	"DEPENDENT":            "UNION中的第二个或后面的SELECT查询, 依赖于外部查询的结果集.",
	"UNION RESULT":         "UNION查询的结果集.",
	"SUBQUERY":             "子查询中的第一个SELECT查询, 不依赖于外部查询的结果集.",
	"DEPENDENT SUBQUERY":   "子查询中的第一个SELECT查询, 依赖于外部查询的结果集.",
	"DERIVED":              "用于from子句里有子查询的情况. MySQL会递归执行这些子查询, 把结果放在临时表里.",
	"MATERIALIZED":         "Materialized subquery.",
	"UNCACHEABLE SUBQUERY": "结果集不能被缓存的子查询, 必须重新为外层查询的每一行进行评估.",
	"UNCACHEABLE UNION":    "UNION中的第二个或后面的select查询, 属于不可缓存的子查询（类似于UNCACHEABLE SUBQUERY）.",
}

// ExplainAccessType EXPLAIN中ACCESS TYPE会出现的类型
var ExplainAccessType = map[string]string{
	"system":          "这是const连接类型的一种特例, 该表仅有一行数据(=系统表).",
	"const":           `const用于使用常数值比较PRIMARY KEY时, 当查询的表仅有一行时, 使用system. 例:SELECT * FROM tbl WHERE col =1.`,
	"eq_ref":          `除const类型外最好的可能实现的连接类型. 它用在一个索引的所有部分被连接使用并且索引是UNIQUE或PRIMARY KEY, 对于每个索引键, 表中只有一条记录与之匹配. 例:'SELECT * FROM ref_table,tbl WHERE ref_table.key_column=tbl.column;'.`,
	"ref":             `连接不能基于关键字选择单个行, 可能查找到多个符合条件的行. 叫做ref是因为索引要跟某个参考值相比较. 这个参考值或者是一个数, 或者是来自一个表里的多表查询的结果值. 例:'SELECT * FROM tbl WHERE idx_col=expr;'.`,
	"fulltext":        "查询时使用 FULLTEXT 索引.",
	"ref_or_null":     "如同ref, 但是MySQL必须在初次查找的结果里找出null条目, 然后进行二次查找.",
	"index_merge":     `表示使用了索引合并优化方法. 在这种情况下. key列包含了使用的索引的清单, key_len包含了使用的索引的最长的关键元素. 详情请见 8.2.1.4, “Index Merge Optimization”.`,
	"unique_subquery": `在某些IN查询中使用此种类型，而不是常规的ref:'value IN (SELECT primary_key FROM single_table WHERE some_expr)'.`,
	"index_subquery":  "在某些IN查询中使用此种类型, 与unique_subquery类似, 但是查询的是非唯一索引性索引.",
	"range":           `只检索给定范围的行, 使用一个索引来选择行. key列显示使用了哪个索引. key_len包含所使用索引的最长关键元素.`,
	"index":           "全表扫描, 只是扫描表的时候按照索引次序进行而不是行. 主要优点就是避免了排序, 但是开销仍然非常大.",
	"ALL":             `最坏的情况, 从头到尾全表扫描.`,
}

// ExplainScalability ACCESS TYPE对应的运算复杂度 [AccessType]scalability map
var ExplainScalability = map[string]string{
	"ALL":             "O(n)",
	"index":           "O(n)",
	"range":           "O(log n)+",
	"index_subquery":  "O(log n)+",
	"unique_subquery": "O(log n)+",
	"index_merge":     "O(log n)+",
	"ref_or_null":     "O(log n)+",
	"fulltext":        "O(log n)+",
	"ref":             "O(log n)",
	"eq_ref":          "O(log n)",
	"const":           "O(1)",
	"system":          "O(1)",
}

// ExplainExtra Extra信息解读
// https://dev.mysql.com/doc/refman/8.0/en/explain-output.html
// sql/opt_explain_traditional.cc:traditional_extra_tags
var ExplainExtra = map[string]string{
	"Using temporary":                   "表示MySQL在对查询结果排序时使用临时表. 常见于排序order by和分组查询group by.",
	"Using filesort":                    "MySQL会对结果使用一个外部索引排序,而不是从表里按照索引次序读到相关内容. 可能在内存或者磁盘上进行排序. MySQL中无法利用索引完成的排序操作称为'文件排序'.",
	"Using index condition":             "在5.6版本后加入的新特性（Index Condition Pushdown）。Using index condition 会先条件过滤索引，过滤完索引后找到所有符合索引条件的数据行，随后用 WHERE 子句中的其他条件去过滤这些数据行。",
	"Range checked for each record":     "MySQL没有发现好的可以使用的索引,但发现如果来自前面的表的列值已知,可能部分索引可以使用。",
	"Using where with pushed condition": "这是一个仅仅在NDBCluster存储引擎中才会出现的信息，打开condition pushdown优化功能才可能被使用。",
	"Using MRR":                         "使用了 MRR Optimization IO 层面进行了优化，减少 IO 方面的开销。",
	"Skip_open_table":                   "Tables are read using the Multi-Range Read optimization strategy.",
	"Open_frm_only":                     "Table files do not need to be opened. The information is already available from the data dictionary.",
	"Open_full_table":                   "Unoptimized information lookup. Table information must be read from the data dictionary and by reading table files.",
	"Scanned":                           "This indicates how many directory scans the server performs when processing a query for INFORMATION_SCHEMA tables.",
	"Using index for group-by":          "Similar to the Using index table access method, Using index for group-by indicates that MySQL found an index that can be used to retrieve all columns of a GROUP BY or DISTINCT query without any extra disk access to the actual table. Additionally, the index is used in the most efficient way so that for each group, only a few index entries are read.",
	"Start temporary":                   "This indicates temporary table use for the semi-join Duplicate Weedout strategy.Start",
	"End temporary":                     "This indicates temporary table use for the semi-join Duplicate Weedout strategy.End",
	"FirstMatch":                        "The semi-join FirstMatch join shortcutting strategy is used for tbl_name.",
	"Materialize":                       "Materialized subquery",
	"Start materialize":                 "Materialized subquery Start",
	"End materialize":                   "Materialized subquery End",
	"unique row not found":              "For a query such as SELECT ... FROM tbl_name, no rows satisfy the condition for a UNIQUE index or PRIMARY KEY on the table.",
	//"Scan":                                                "",
	//"Impossible ON condition":                             "",
	//"Ft_hints:":                                           "",
	//"Backward index scan":                                 "",
	//"Recursive":                                           "",
	//"Table function:":                                     "",
	"Index dive skipped due to FORCE":                     "This item applies to NDB tables only. It means that MySQL Cluster is using the Condition Pushdown optimization to improve the efficiency of a direct comparison between a nonindexed column and a constant. In such cases, the condition is “pushed down” to the cluster's data nodes and is evaluated on all data nodes simultaneously. This eliminates the need to send nonmatching rows over the network, and can speed up such queries by a factor of 5 to 10 times over cases where Condition Pushdown could be but is not used.",
	"Impossible WHERE noticed after reading const tables": "查询了所有const(和system)表, 但发现WHERE查询条件不起作用.",
	"Using where":                              "WHERE条件用于筛选出与下一个表匹配的数据然后返回给客户端. 除非故意做的全表扫描, 否则连接类型是ALL或者是index, 且在Extra列的值中没有Using Where, 则该查询可能是有问题的.",
	"Using join buffer":                        "从已有连接中找被读入缓存的数据, 并且通过缓存来完成与当前表的连接.",
	"Using index":                              "只需通过索引就可以从表中获取列的信息, 无需额外去读取真实的行数据. 如果查询使用的列值仅仅是一个简单索引的部分值, 则会使用这种策略来优化查询.",
	"const row not found":                      "空表做类似 SELECT ... FROM tbl_name 的查询操作.",
	"Distinct":                                 "MySQL is looking for distinct values, so it stops searching for more rows for the current row combination after it has found the first matching row.",
	"Full scan on NULL key":                    "子查询中的一种优化方式, 常见于无法通过索引访问null值.",
	"Impossible HAVING":                        "HAVING条件过滤没有效果, 返回已有查询的结果集.",
	"Impossible WHERE":                         "WHERE条件过滤没有效果, 最终是全表扫描.",
	"LooseScan":                                "使用半连接LooseScan策略.",
	"No matching min/max row":                  "没有行满足查询的条件, 如 SELECT MIN(...) FROM ... WHERE condition.",
	"no matching row in const table":           "对于连接查询, 列未满足唯一索引的条件或表为空.",
	"No matching rows after partition pruning": "对于DELETE 或 UPDATE, 优化器在分区之后, 未发现任何要删除或更新的内容. 类似查询 Impossible WHERE.",
	"No tables used":                           "查询没有FROM子句, 或者有一个 FROM DUAL子句.",
	"Not exists":                               "MySQL能够对LEFT JOIN查询进行优化, 并且在查找到符合LEFT JOIN条件的行后, 则不再查找更多的行.",
	"Plan isn't ready yet":                     "This value occurs with EXPLAIN FOR CONNECTION when the optimizer has not finished creating the execution plan for the statement executing in the named connection. If execution plan output comprises multiple lines, any or all of them could have this Extra value, depending on the progress of the optimizer in determining the full execution plan.",
	"Using intersect":                          "开启了index merge，即：对多个索引分别进行条件扫描，然后将它们各自的结果进行合并，使用的算法为：index_merge_intersection",
	"Using union":                              "开启了index merge，即：对多个索引分别进行条件扫描，然后将它们各自的结果进行合并，使用的算法为：index_merge_union",
	"Using sort_union":                         "开启了index merge，即：对多个索引分别进行条件扫描，然后将它们各自的结果进行合并，使用的算法为：index_merge_sort_union",
}

// 提取ExplainJSON中所有的ExplainJSONTable, 将其写入全局变量explainJSONTables
// depth只是用于debug，逻辑上并未使用
func findTablesInJSON(explainJSON string, depth int) {
	common.Log.Debug("findTablesInJSON Enter: depth(%d), json(%s)", depth, explainJSON)
	// 去除注释，语法检查
	explainJSON = string(RemoveSQLComments([]byte(explainJSON)))
	if !gjson.Valid(explainJSON) {
		return
	}
	// 提取所有ExplainJSONTable struct
	for _, key := range ExplainKeyWords {
		result := gjson.Get(explainJSON, key)
		if result.String() == "" {
			continue
		}

		if key == "table" {
			table := new(ExplainJSONTable)
			common.Log.Debug("findTablesInJSON FindTable: depth(%d), table(%s)", depth, result.String)
			err := json.Unmarshal([]byte(result.Raw), table)
			common.LogIfError(err, "")
			if table.TableName != "" {
				explainJSONTables = append(explainJSONTables, table)
			}
			findTablesInJSON(result.String(), depth+1)
		} else {
			common.Log.Debug("findTablesInJSON ScanOther: depth(%d), key(%s), array_len(%d), json(%s)", depth, key, len(result.Array()), result.String)
			for _, val := range result.Array() {
				if val.String() != "" {
					findTablesInJSON(val.String(), depth+1)
				}
			}
			findTablesInJSON(result.String(), depth+1)
		}
	}
}

// FormatJSONIntoTraditional 将JSON形式转换为TRADITIONAL形式，方便前端展现
func FormatJSONIntoTraditional(explainJSON string) []*ExplainRow {
	// 查找JSON中的所有ExplainJSONTable
	explainJSONTables = []*ExplainJSONTable{}
	findTablesInJSON(explainJSON, 0)

	var explainRows []*ExplainRow
	id := -1
	for _, table := range explainJSONTables {
		keyLen := table.KeyLength
		filtered, err := strconv.ParseFloat(table.Filtered, 64)
		if err != nil {
			filtered = 0.00
		}
		if filtered > 100.00 {
			filtered = 100.00
		}
		explainRows = append(explainRows, &ExplainRow{
			ID:           id + 1,
			SelectType:   "",
			TableName:    table.TableName,
			Partitions:   "NULL",
			AccessType:   table.AccessType,
			PossibleKeys: table.PossibleKeys,
			Key:          table.Key,
			KeyLen:       keyLen,
			Ref:          table.Ref,
			Rows:         table.RowsExaminedPerScan,
			Filtered:     filtered,
			Scalability:  ExplainScalability[table.AccessType],
			Extra:        "",
		})
	}
	return explainRows
}

// ConvertExplainJSON2Row 将JSON格式转成ROW格式，为方便统一做优化建议
// 但是会损失一些JSON特有的分析结果
func ConvertExplainJSON2Row(explainJSON *ExplainJSON) []*ExplainRow {
	buf, err := json.Marshal(explainJSON)
	if err != nil {
		return nil
	}
	return FormatJSONIntoTraditional(string(buf))
}

// 用于检测MySQL版本是否低于MySQL5.6
// 低于5.6 返回 true， 表示需要改写非SELECT的SQL --> SELECT
func (db *Connector) supportExplainWrite() (bool, error) {
	defer func() {
		err := recover()
		if err != nil {
			common.Log.Error("Recover supportExplainWrite() Error:", err)
		}
	}()

	// 5.6以上版本支持EXPLAIN UPDATE/DELETE等语句，但需要开启写入
	// 如开启了read_only，EXPLAIN UPDATE/DELETE也会受限制
	if common.Config.TestDSN.Version >= 560 {
		readOnly, err := db.SingleIntValue("read_only")
		if err != nil {
			return false, err
		}
		superReadOnly, err := db.SingleIntValue("super_read_only")
		// Percona, MariaDB 5.6就已经有super_read_only了，但社区版5.6还没有这个参数
		if strings.Contains(err.Error(), "Unknown system variable") {
			superReadOnly = readOnly
		} else if err != nil {
			return false, err
		}

		if readOnly == 1 || superReadOnly == 1 {
			return true, nil
		}

		return false, nil
	}

	return true, nil
}

// 将SQL语句转换为可以被Explain的语句，如：写转读
// 当输出为空时，表示语法错误或不支持EXPLAIN
func (db *Connector) explainAbleSQL(sql string) (string, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		common.Log.Error("explainAbleSQL sqlparser.Parse Error: %v", err)
		return sql, err
	}

	switch stmt.(type) {
	case *sqlparser.Insert, *sqlparser.Update, *sqlparser.Delete: // REPLACE和INSERT的AST基本相同，只是Action不同
		// 判断Explain的SQL是否需要被改写
		need, err := db.supportExplainWrite()
		if err != nil {
			common.Log.Error("explainAbleSQL db.supportExplainWrite Error: %v", err)
			return "", err
		}
		if need {
			rw := ast.NewRewrite(sql)
			if rw != nil {
				return rw.RewriteDML2Select().NewSQL, nil
			}
		}
		return sql, nil

	case *sqlparser.Union, *sqlparser.ParenSelect, *sqlparser.Select, sqlparser.SelectStatement:
		return sql, nil
	default:
	}
	return "", nil
}

// 执行explain请求，返回mysql.Result执行结果
func (db *Connector) executeExplain(sql string, explainType int, formatType int) (*QueryResult, error) {
	var err error
	sql, _ = db.explainAbleSQL(sql)
	if sql == "" {
		return nil, err
	}

	// 5.6以上支持FORMAT=JSON
	explainFormat := ""
	switch formatType {
	case JSONFormatExplain:
		if common.Config.TestDSN.Version >= 560 {
			explainFormat = "FORMAT=JSON"
		}
	}
	// 执行explain
	var res *QueryResult
	switch explainType {
	case ExtendedExplainType:
		// 5.6以上extended关键字已经不推荐使用，8.0废弃了这个关键字
		if common.Config.TestDSN.Version >= 560 {
			res, err = db.Query("explain %s", sql)
		} else {
			res, err = db.Query("explain extended %s", sql)
		}
	case PartitionsExplainType:
		res, err = db.Query("explain partitions %s", sql)

	default:
		res, err = db.Query("explain %s %s", explainFormat, sql)
	}
	return res, err
}

// MySQLExplainWarnings WARNINGS信息中包含的优化器信息
func MySQLExplainWarnings(exp *ExplainInfo) string {
	content := "## MySQL优化器调优结果\n\n```sql\n"
	for _, row := range exp.Warnings {
		content += "\n" + row.Message + "\n"
	}
	content += "\n```"
	return content
}

// MySQLExplainQueryCost 将last_query_cost信息补充到评审结果中
func MySQLExplainQueryCost(exp *ExplainInfo) string {
	var content string
	if exp.QueryCost > 0 {

		tmp := fmt.Sprintf("%.3f\n", exp.QueryCost)

		content = "Query cost: "
		if exp.QueryCost > float64(common.Config.MaxQueryCost) {
			content += fmt.Sprintf("☠️ **%s**", tmp)
		} else {
			content += tmp
		}

	}
	return content
}

// ExplainInfoTranslator 将explain信息翻译成人能读懂的
func ExplainInfoTranslator(exp *ExplainInfo) string {
	var buf []string
	var selectTypeBuf []string
	var accessTypeBuf []string
	var extraTypeBuf []string
	buf = append(buf, fmt.Sprint("### Explain信息解读\n"))
	rows := exp.ExplainRows
	if exp.ExplainFormat == JSONFormatExplain {
		// JSON形式遍历分析不方便，转成Row格式统一处理
		rows = ConvertExplainJSON2Row(exp.ExplainJSON)
	}
	if len(rows) == 0 {
		return ""
	}

	// SelectType信息解读
	explainSelectType := make(map[string]string)
	for k, v := range ExplainSelectType {
		explainSelectType[k] = v
	}
	for _, row := range rows {
		if _, ok := explainSelectType[row.SelectType]; ok {
			desc := fmt.Sprintf("* **%s**: %s\n", row.SelectType, explainSelectType[row.SelectType])
			selectTypeBuf = append(selectTypeBuf, desc)
			delete(explainSelectType, row.SelectType)
		}
	}
	if len(selectTypeBuf) > 0 {
		buf = append(buf, fmt.Sprint("#### SelectType信息解读\n"))
		buf = append(buf, strings.Join(selectTypeBuf, "\n"))
	}

	// #### Type信息解读
	explainAccessType := make(map[string]string)
	for k, v := range ExplainAccessType {
		explainAccessType[k] = v
	}
	for _, row := range rows {
		if _, ok := explainAccessType[row.AccessType]; ok {
			var warn bool
			var desc string
			for _, t := range common.Config.ExplainWarnAccessType {
				if row.AccessType == t {
					warn = true
				}
			}
			if warn {
				desc = fmt.Sprintf("* ☠️ **%s**: %s\n", row.AccessType, explainAccessType[row.AccessType])
			} else {
				desc = fmt.Sprintf("* **%s**: %s\n", row.AccessType, explainAccessType[row.AccessType])
			}

			accessTypeBuf = append(accessTypeBuf, desc)
			delete(explainAccessType, row.AccessType)
		}
	}
	if len(accessTypeBuf) > 0 {
		buf = append(buf, fmt.Sprint("#### Type信息解读\n"))
		buf = append(buf, strings.Join(accessTypeBuf, "\n"))
	}

	// #### Extra信息解读
	if exp.ExplainFormat != JSONFormatExplain {
		explainExtra := make(map[string]string)
		for k, v := range ExplainExtra {
			explainExtra[k] = v
		}
		for _, row := range rows {
			for k, c := range explainExtra {
				if strings.Contains(row.Extra, k) {
					if k == "Impossible WHERE" {
						if strings.Contains(row.Extra, "Impossible WHERE noticed after reading const tables") {
							continue
						}
					}
					warn := false
					for _, w := range common.Config.ExplainWarnExtra {
						if k == w {
							warn = true
						}
					}
					if warn {
						extraTypeBuf = append(extraTypeBuf, fmt.Sprintf("* ☠️ **%s**: %s\n", k, c))
					} else {
						extraTypeBuf = append(extraTypeBuf, fmt.Sprintf("* **%s**: %s\n", k, c))
					}
					delete(explainExtra, k)
				}
			}
		}
	}
	if len(extraTypeBuf) > 0 {
		buf = append(buf, fmt.Sprint("#### Extra信息解读\n"))
		buf = append(buf, strings.Join(extraTypeBuf, "\n"))
	}

	return strings.Join(buf, "\n")
}

// ParseExplainText 解析explain文本信息（很可能是用户复制粘贴得到），返回格式化数据
func ParseExplainText(content string) (exp *ExplainInfo, err error) {
	exp = &ExplainInfo{ExplainFormat: TraditionalFormatExplain}

	content = strings.TrimSpace(content)
	verticalFormat := strings.HasPrefix(content, "*")
	jsonFormat := strings.HasPrefix(content, "{")
	traditionalFormat := strings.HasPrefix(content, "+")

	if verticalFormat && traditionalFormat && jsonFormat {
		return nil, errors.New("not supported explain type")
	}

	if verticalFormat {
		exp.ExplainRows, err = parseVerticalExplainText(content)
	}

	if jsonFormat {
		exp.ExplainFormat = JSONFormatExplain
		exp.ExplainJSON, err = parseJSONExplainText(content)
	}

	if traditionalFormat {
		exp.ExplainRows, err = parseTraditionalExplainText(content)
	}
	return exp, err
}

// 解析文本形式传统形式Explain信息
func parseTraditionalExplainText(content string) (explainRows []*ExplainRow, err error) {
	LS := regexp.MustCompile(`^\+`) // 华丽的分隔线:)

	// 格式正确性检查
	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return nil, errors.New("explain Rows less than 3")
	}

	// 提取头部，用于后续list到map的转换
	var header []string
	for _, h := range strings.Split(strings.Trim(lines[1], "|"), "|") {
		header = append(header, strings.TrimSpace(h))
	}
	colIdx := make(map[string]int)
	for i, item := range header {
		colIdx[strings.ToLower(item)] = i
	}

	// explain format=json未把外面的框去了
	if strings.ToLower(header[0]) == "explain" {
		return nil, errors.New("json format explain need remove")
	}

	// 将每一列填充至ExplainRow结构体
	colsMap := make(map[string]string)
	for _, l := range lines[3:] {
		var keylen string
		var rows int
		var filtered float64
		var partitions string
		// 跳过分割线
		if LS.MatchString(l) || strings.TrimSpace(l) == "" {
			continue
		}

		// list到map的转换
		var cols []string
		for _, c := range strings.Split(strings.Trim(l, "|"), "|") {
			cols = append(cols, strings.TrimSpace(c))
		}
		for item, i := range colIdx {
			colsMap[item] = cols[i]
		}

		// 值类型转换
		id, err := strconv.Atoi(colsMap["id"])
		if err != nil {
			return nil, err
		}

		// 不存在字段给默认值
		if colsMap["partitions"] == "" {
			partitions = "NULL"
		} else {
			partitions = colsMap["partitions"]
		}

		keylen = colsMap["key_len"]

		rows, err = strconv.Atoi(colsMap["Rows"])
		if err != nil {
			rows = 0
		}

		filtered, err = strconv.ParseFloat(colsMap["filtered"], 64)
		if err != nil {
			filtered = 0.00
		}
		// filtered may larger than 100.00
		// https://bugs.mysql.com/bug.php?id=34124
		if filtered > 100.00 {
			filtered = 100.00
		}

		// 拼接结构体
		explainRows = append(explainRows, &ExplainRow{
			ID:           id,
			SelectType:   colsMap["select_type"],
			TableName:    colsMap["table"],
			Partitions:   partitions,
			AccessType:   colsMap["type"],
			PossibleKeys: strings.Split(colsMap["possible_keys"], ","),
			Key:          colsMap["key"],
			KeyLen:       keylen,
			Ref:          strings.Split(colsMap["ref"], ","),
			Rows:         rows,
			Filtered:     filtered,
			Scalability:  ExplainScalability[colsMap["type"]],
			Extra:        colsMap["extra"],
		})
	}
	return explainRows, nil
}

// 解析文本形式竖排版 Explain信息
func parseVerticalExplainText(content string) (explainRows []*ExplainRow, err error) {
	var lines []string
	explainRow := &ExplainRow{
		Partitions: "NULL",
		Filtered:   0.00,
	}
	LS := regexp.MustCompile(`^\*.*\*$`) // 华丽的分隔线:)

	// 格式正确性检查
	for _, l := range strings.Split(content, "\n") {
		lines = append(lines, strings.TrimSpace(l))
	}
	if len(lines) < 11 {
		return nil, errors.New("explain rows less than 11")
	}

	// 将每一行填充至ExplainRow结构体
	for _, l := range lines {
		if LS.MatchString(l) || strings.TrimSpace(l) == "" {
			continue
		}
		if strings.HasPrefix(l, "id:") {
			id := strings.TrimPrefix(l, "id: ")
			explainRow.ID, err = strconv.Atoi(id)
			if err != nil {
				return nil, err
			}
		}
		if strings.HasPrefix(l, "select_type:") {
			explainRow.SelectType = strings.TrimPrefix(l, "select_type: ")
		}
		if strings.HasPrefix(l, "table:") {
			explainRow.TableName = strings.TrimPrefix(l, "table: ")
		}
		if strings.HasPrefix(l, "partitions:") {
			explainRow.AccessType = strings.TrimPrefix(l, "partitions: ")
		}
		if strings.HasPrefix(l, "type:") {
			explainRow.AccessType = strings.TrimPrefix(l, "type: ")
			explainRow.Scalability = ExplainScalability[explainRow.AccessType]
		}
		if strings.HasPrefix(l, "possible_keys:") {
			explainRow.PossibleKeys = strings.Split(strings.TrimPrefix(l, "possible_keys: "), ",")
		}
		if strings.HasPrefix(l, "key:") {
			explainRow.Key = strings.TrimPrefix(l, "key: ")
		}
		if strings.HasPrefix(l, "key_len:") {
			keyLen := strings.TrimPrefix(l, "key_len: ")
			explainRow.KeyLen = keyLen
		}
		if strings.HasPrefix(l, "ref:") {
			explainRow.Ref = strings.Split(strings.TrimPrefix(l, "ref: "), ",")
		}
		if strings.HasPrefix(l, "Rows:") {
			rows := strings.TrimPrefix(l, "Rows: ")
			explainRow.Rows, err = strconv.Atoi(rows)
			if err != nil {
				explainRow.Rows = 0
			}
		}
		if strings.HasPrefix(l, "filtered:") {
			filtered := strings.TrimPrefix(l, "filtered: ")
			explainRow.Filtered, err = strconv.ParseFloat(filtered, 64)
			if err != nil {
				return nil, err
			} else if explainRow.Filtered > 100.00 {
				explainRow.Filtered = 100.00
			}
		}
		if strings.HasPrefix(l, "Extra:") {
			explainRow.Extra = strings.TrimPrefix(l, "Extra: ")
			explainRows = append(explainRows, explainRow)
		}
	}
	return explainRows, err
}

// 解析文本形式JSON Explain信息
func parseJSONExplainText(content string) (*ExplainJSON, error) {
	explainJSON := new(ExplainJSON)
	err := json.Unmarshal(RemoveSQLComments([]byte(content)), explainJSON)
	return explainJSON, err
}

// ParseExplainResult 分析mysql执行explain的结果，返回ExplainInfo结构化数据
func ParseExplainResult(res *QueryResult, formatType int) (exp *ExplainInfo, err error) {
	exp = &ExplainInfo{
		ExplainFormat: formatType,
	}
	// JSON格式直接调用文本方式解析
	if formatType == JSONFormatExplain {
		exp.ExplainJSON, err = parseJSONExplainText(res.Rows[0].Str(0))
		return exp, err
	}

	// 生成表头
	colIdx := make(map[int]string)
	for i, f := range res.Result.Fields() {
		colIdx[i] = strings.ToLower(f.Name)
	}
	// 补全ExplainRows
	var explainrows []*ExplainRow
	for _, row := range res.Rows {
		expRow := &ExplainRow{Partitions: "NULL", Filtered: 0.00}
		// list到map的转换
		for i := range row {
			switch colIdx[i] {
			case "id":
				expRow.ID = row.ForceInt(i)
			case "select_type":
				expRow.SelectType = row.Str(i)
			case "table":
				expRow.TableName = row.Str(i)
				if expRow.TableName == "" {
					expRow.TableName = "NULL"
				}
			case "type":
				expRow.AccessType = row.Str(i)
				if expRow.AccessType == "" {
					expRow.AccessType = "NULL"
				}
				expRow.Scalability = ExplainScalability[expRow.AccessType]
			case "possible_keys":
				expRow.PossibleKeys = strings.Split(row.Str(i), ",")
			case "key":
				expRow.Key = row.Str(i)
				if expRow.Key == "" {
					expRow.Key = "NULL"
				}
			case "key_len":
				expRow.KeyLen = row.Str(i)
			case "ref":
				expRow.Ref = strings.Split(row.Str(i), ",")
			case "rows":
				expRow.Rows = row.ForceInt(i)
			case "extra":
				expRow.Extra = row.Str(i)
				if expRow.Extra == "" {
					expRow.Extra = "NULL"
				}
			case "filtered":
				expRow.Filtered = row.ForceFloat(i)
				// MySQL bug: https://bugs.mysql.com/bug.php?id=34124
				if expRow.Filtered > 100.00 {
					expRow.Filtered = 100.00
				}
			}
		}
		explainrows = append(explainrows, expRow)
	}
	exp.ExplainRows = explainrows
	for _, w := range res.Warning {
		// 'EXTENDED' is deprecated and will be removed in a future release.
		if w.Int(1) != 1681 {
			exp.Warnings = append(exp.Warnings, &ExplainWarning{Level: w.Str(0), Code: w.Int(1), Message: w.Str(2)})
		}
	}

	// 添加 last_query_cost
	exp.QueryCost = res.QueryCost

	return exp, err
}

// Explain 获取SQL的explain信息
func (db *Connector) Explain(sql string, explainType int, formatType int) (exp *ExplainInfo, err error) {
	exp = &ExplainInfo{SQL: sql}
	if explainType != TraditionalExplainType {
		formatType = TraditionalFormatExplain
	}
	defer func() {
		if e := recover(); e != nil {
			const size = 4096
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			common.Log.Error("Recover Explain() Error: %v\n%v", e, string(buf))
			err = errors.New(fmt.Sprint(e))
		}
	}()

	// 执行EXPLAIN请求
	res, err := db.executeExplain(sql, explainType, formatType)
	if err != nil || res == nil {
		return exp, err
	}

	// 解析mysql结果，输出ExplainInfo
	exp, err = ParseExplainResult(res, formatType)

	return exp, err
}

// PrintMarkdownExplainTable 打印markdown格式的explain table
func PrintMarkdownExplainTable(exp *ExplainInfo) string {
	var buf []string
	rows := exp.ExplainRows
	// JSON转换为TRADITIONAL格式
	if exp.ExplainFormat == JSONFormatExplain {
		buf = append(buf, fmt.Sprint("以下为JSON格式转为传统格式EXPLAIN表格", "\n\n"))
		rows = ConvertExplainJSON2Row(exp.ExplainJSON)
	}

	// explain出错
	if len(rows) == 0 {
		return ""
	}
	if exp.ExplainFormat == JSONFormatExplain {
		buf = append(buf, fmt.Sprintln("| table | partitions | type | possible\\_keys | key | key\\_len | ref | rows | filtered | scalability | Extra |"))
		buf = append(buf, fmt.Sprintln("|---|---|---|---|---|---|---|---|---|---|---|"))
		for _, row := range rows {
			buf = append(buf, fmt.Sprintln("|", row.TableName, "|", row.Partitions, "|", row.AccessType,
				"|", strings.Join(row.PossibleKeys, ","), "|", row.Key, "|", row.KeyLen, "|",
				strings.Join(row.Ref, ","), "|", row.Rows, "|", fmt.Sprintf("%.2f%s", row.Filtered, "%"),
				"|", row.Scalability, "|", row.Extra, "|"))
		}
	} else {
		buf = append(buf, fmt.Sprintln("| id | select\\_type | table | partitions | type | possible_keys | key | key\\_len | ref | rows | filtered | scalability | Extra |"))
		buf = append(buf, fmt.Sprintln("|---|---|---|---|---|---|---|---|---|---|---|---|---|"))
		for _, row := range rows {
			// 加粗
			rows := fmt.Sprint(row.Rows)
			if row.Rows >= common.Config.ExplainMaxRows {
				rows = "☠️ **" + rows + "**"
			}
			filtered := fmt.Sprintf("%.2f%s", row.Filtered, "%")
			if row.Filtered >= common.Config.ExplainMaxFiltered {
				filtered = "☠️ **" + filtered + "**"
			}
			scalability := row.Scalability
			for _, s := range common.Config.ExplainWarnScalability {
				scalability = "☠️ **" + s + "**"
			}
			buf = append(buf, fmt.Sprintln("|", row.ID, " |",
				common.MarkdownEscape(row.SelectType),
				"| *"+common.MarkdownEscape(row.TableName)+"* |",
				common.MarkdownEscape(row.Partitions), "|",
				common.MarkdownEscape(row.AccessType), "|",
				common.MarkdownEscape(strings.Join(row.PossibleKeys, ",<br>")), "|",
				common.MarkdownEscape(row.Key), "|",
				row.KeyLen, "|",
				common.MarkdownEscape(strings.Join(row.Ref, ",<br>")),
				"|", rows, "|",
				filtered, "|", scalability, "|",
				strings.Replace(common.MarkdownEscape(row.Extra), ",", ",<br>", -1),
				"|"))
		}
	}
	buf = append(buf, "\n")
	return strings.Join(buf, "")
}
