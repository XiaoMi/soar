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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoMi/soar/common"
)

// SHOW TABLE STATUS Syntax
// https://dev.mysql.com/doc/refman/5.7/en/show-table-status.html

// TableStatInfo 用以保存 show table status 之后获取的table信息
type TableStatInfo struct {
	Name string
	Rows []tableStatusRow
}

// tableStatusRow 用于 show table status value
type tableStatusRow struct {
	Name         string // 表名
	Engine       string // 该表使用的存储引擎
	Version      int    // 该表的 .frm 文件版本号
	RowFormat    string // 该表使用的行存储格式
	Rows         int64  // 表行数，InnoDB 引擎中为预估值，甚至可能会有40%~50%的数值偏差
	AvgRowLength int    // 平均行长度

	// MyISAM: Data_length 为数据文件的大小，单位为 bytes
	// InnoDB: Data_length 为聚簇索引分配的近似内存量，单位为 bytes, 计算方式为聚簇索引数量乘以 InnoDB 页面大小
	// 其他不同的存储引擎中该值的意义可能不尽相同
	DataLength int

	// MyISAM: Max_data_length 为数据文件长度的最大值。这是在给定使用的数据指针大小的情况下，可以存储在表中的数据的最大字节数
	// InnoDB: 未使用
	// 其他不同的存储引擎中该值的意义可能不尽相同
	MaxDataLength int

	// MyISAM: Index_length 为 index 文件的大小，单位为 bytes
	// InnoDB: Index_length 为非聚簇索引分配的近似内存量，单位为 bytes，计算方式为非聚簇索引数量乘以 InnoDB 页面大小
	// 其他不同的存储引擎中该值的意义可能不尽相同
	IndexLength int

	DataFree      int       // 已分配但未使用的字节数
	AutoIncrement int       // 下一个自增值
	CreateTime    time.Time // 创建时间
	UpdateTime    time.Time // 最近一次更新时间，该值不准确
	CheckTime     time.Time // 上次检查时间
	Collation     string    // 字符集及排序规则信息
	Checksum      string    // 校验和
	CreateOptions string    // 创建表的时候的时候一切其他属性
	Comment       string    // 注释
}

// newTableStat 构造 table Stat 对象
func newTableStat(tableName string) *TableStatInfo {
	return &TableStatInfo{
		Name: tableName,
		Rows: make([]tableStatusRow, 0),
	}
}

// ShowTables 执行 show tables
func (db *Connector) ShowTables() ([]string, error) {
	defer func() {
		err := recover()
		if err != nil {
			common.Log.Error("recover ShowTableStatus()", err)
		}
	}()

	// 执行 show table status
	res, err := db.Query("show tables")
	if err != nil {
		return []string{}, err
	}

	// 获取值
	var tables []string
	for _, row := range res.Rows {
		tables = append(tables, row.Str(0))
	}

	return tables, err
}

// ShowTableStatus 执行 show table status
func (db *Connector) ShowTableStatus(tableName string) (*TableStatInfo, error) {
	defer func() {
		err := recover()
		if err != nil {
			common.Log.Error("recover ShowTableStatus()", err)
		}
	}()

	// 初始化struct
	ts := newTableStat(tableName)

	// 执行 show table status
	res, err := db.Query("show table status where name = '%s'", ts.Name)
	if err != nil {
		return ts, err
	}

	rs := res.Result.Map("Rows")
	name := res.Result.Map("Name")
	df := res.Result.Map("Data_free")
	sum := res.Result.Map("Checksum")
	engine := res.Result.Map("Engine")
	version := res.Result.Map("Version")
	comment := res.Result.Map("Comment")
	ai := res.Result.Map("Auto_increment")
	collation := res.Result.Map("Collation")
	rowFormat := res.Result.Map("Row_format")
	checkTime := res.Result.Map("Check_time")
	dataLength := res.Result.Map("Data_length")
	idxLength := res.Result.Map("Index_length")
	createTime := res.Result.Map("Create_time")
	updateTime := res.Result.Map("Update_time")
	options := res.Result.Map("Create_options")
	avgRowLength := res.Result.Map("Avg_row_length")
	maxDataLength := res.Result.Map("Max_data_length")

	// 获取值
	for _, row := range res.Rows {
		value := tableStatusRow{
			Name:          row.Str(name),
			Engine:        row.Str(engine),
			Version:       row.Int(version),
			Rows:          row.Int64(rs),
			RowFormat:     row.Str(rowFormat),
			AvgRowLength:  row.Int(avgRowLength),
			DataLength:    row.Int(dataLength),
			MaxDataLength: row.Int(maxDataLength),
			IndexLength:   row.Int(idxLength),
			DataFree:      row.Int(df),
			AutoIncrement: row.Int(ai),
			CreateTime:    row.Time(createTime, time.Local),
			UpdateTime:    row.Time(updateTime, time.Local),
			CheckTime:     row.Time(checkTime, time.Local),
			Collation:     row.Str(collation),
			Checksum:      row.Str(sum),
			CreateOptions: row.Str(options),
			Comment:       row.Str(comment),
		}
		ts.Rows = append(ts.Rows, value)
	}

	return ts, err
}

// https://dev.mysql.com/doc/refman/5.7/en/show-index.html

// TableIndexInfo 用以保存 show index 之后获取的 index 信息
type TableIndexInfo struct {
	TableName string
	IdxRows   []TableIndexRow
}

// TableIndexRow 用以存放show index之后获取的每一条index信息
type TableIndexRow struct {
	Table        string // 表名
	NonUnique    int    // 0：unique key，1：not unique
	KeyName      string // index的名称，如果是主键则为 "PRIMARY"
	SeqInIndex   int    // 该列在索引中的位置。计数从 1 开始
	ColumnName   string // 列名
	Collation    string // A or Null
	Cardinality  int    // 索引中唯一值的数量，"ANALYZE TABLE" 可更新该值
	SubPart      int    // 索引前缀字节数
	Packed       int
	Null         string // 表示该列是否可以为空，如果可以为 'YES'，反之''
	IndexType    string // BTREE, FULLTEXT, HASH, RTREE
	Comment      string
	IndexComment string
}

// NewTableIndexInfo 构造 TableIndexInfo
func NewTableIndexInfo(tableName string) *TableIndexInfo {
	return &TableIndexInfo{
		TableName: tableName,
		IdxRows:   make([]TableIndexRow, 0),
	}
}

// ShowIndex show Index
func (db *Connector) ShowIndex(tableName string) (*TableIndexInfo, error) {
	tbIndex := NewTableIndexInfo(tableName)

	// 执行 show create table
	res, err := db.Query("show index from `%s`.`%s`", db.Database, tableName)
	if err != nil {
		return nil, err
	}

	table := res.Result.Map("Table")
	unique := res.Result.Map("Non_unique")
	keyName := res.Result.Map("Key_name")
	seq := res.Result.Map("Seq_in_index")
	cName := res.Result.Map("Column_name")
	collation := res.Result.Map("Collation")
	cardinality := res.Result.Map("Cardinality")
	subPart := res.Result.Map("Sub_part")
	packed := res.Result.Map("Packed")
	null := res.Result.Map("Null")
	idxType := res.Result.Map("Index_type")
	comment := res.Result.Map("Comment")
	idxComment := res.Result.Map("Index_comment")

	// 获取值
	for _, row := range res.Rows {
		value := TableIndexRow{
			Table:        row.Str(table),
			NonUnique:    row.Int(unique),
			KeyName:      row.Str(keyName),
			SeqInIndex:   row.Int(seq),
			ColumnName:   row.Str(cName),
			Collation:    row.Str(collation),
			Cardinality:  row.Int(cardinality),
			SubPart:      row.Int(subPart),
			Packed:       row.Int(packed),
			Null:         row.Str(null),
			IndexType:    row.Str(idxType),
			Comment:      row.Str(comment),
			IndexComment: row.Str(idxComment),
		}
		tbIndex.IdxRows = append(tbIndex.IdxRows, value)
	}
	return tbIndex, err
}

// IndexSelectKey 用以对 TableIndexInfo 进行查询
type IndexSelectKey string

// 索引相关
const (
	IndexKeyName    = IndexSelectKey("KeyName")    // 索引名称
	IndexColumnName = IndexSelectKey("ColumnName") // 索引列名称
	IndexIndexType  = IndexSelectKey("IndexType")  // 索引类型
	IndexNonUnique  = IndexSelectKey("NonUnique")  // 唯一索引
)

// FindIndex 获取TableIndexInfo中需要的索引
func (tbIndex *TableIndexInfo) FindIndex(arg IndexSelectKey, value string) []TableIndexRow {
	var result []TableIndexRow
	if tbIndex == nil {
		return result
	}

	value = strings.ToLower(value)

	switch arg {
	case IndexKeyName:
		for _, index := range tbIndex.IdxRows {
			if strings.ToLower(index.KeyName) == value {
				result = append(result, index)
			}
		}

	case IndexColumnName:
		for _, index := range tbIndex.IdxRows {
			if strings.ToLower(index.ColumnName) == value {
				result = append(result, index)
			}
		}

	case IndexIndexType:
		for _, index := range tbIndex.IdxRows {
			if strings.ToLower(index.IndexType) == value {
				result = append(result, index)
			}
		}

	case IndexNonUnique:
		for _, index := range tbIndex.IdxRows {
			unique := strconv.Itoa(index.NonUnique)
			if unique == value {
				result = append(result, index)
			}
		}

	default:
		common.Log.Error("no such args: TableIndexRow")
	}

	return result
}

// desc table
// https://dev.mysql.com/doc/refman/5.7/en/show-columns.html

// TableDesc show columns from rental;
type TableDesc struct {
	Name       string
	DescValues []TableDescValue
}

// TableDescValue 含有每一列的属性
type TableDescValue struct {
	Field      string // 列名
	Type       string // 数据类型
	Null       string // 是否有NULL（NO、YES）
	Collation  string // 字符集
	Privileges string // 权限s
	Key        string // 键类型
	Default    string // 默认值
	Extra      string // 其他
	Comment    string // 备注
}

// NewTableDesc 初始化一个*TableDesc
func NewTableDesc(tableName string) *TableDesc {
	return &TableDesc{
		Name:       tableName,
		DescValues: make([]TableDescValue, 0),
	}
}

// ShowColumns 获取DB中所有的columns
func (db *Connector) ShowColumns(tableName string) (*TableDesc, error) {
	tbDesc := NewTableDesc(tableName)

	// 执行 show create table
	res, err := db.Query("show full columns from `%s`.`%s`", db.Database, tableName)
	if err != nil {
		return nil, err
	}

	field := res.Result.Map("Field")
	tp := res.Result.Map("Type")
	null := res.Result.Map("Null")
	key := res.Result.Map("Key")
	def := res.Result.Map("Default")
	extra := res.Result.Map("Extra")
	collation := res.Result.Map("Collation")
	privileges := res.Result.Map("Privileges")
	comm := res.Result.Map("Comment")

	// 获取值
	for _, row := range res.Rows {
		value := TableDescValue{
			Field:      row.Str(field),
			Type:       row.Str(tp),
			Null:       row.Str(null),
			Key:        row.Str(key),
			Default:    row.Str(def),
			Extra:      row.Str(extra),
			Privileges: row.Str(privileges),
			Collation:  row.Str(collation),
			Comment:    row.Str(comm),
		}
		tbDesc.DescValues = append(tbDesc.DescValues, value)
	}
	return tbDesc, err
}

// Columns 用于获取TableDesc中所有列的名称
func (td TableDesc) Columns() []string {
	var cols []string
	for _, col := range td.DescValues {
		cols = append(cols, col.Field)
	}
	return cols
}

// showCreate show create
func (db *Connector) showCreate(createType, name string) (string, error) {
	// 执行 show create table
	res, err := db.Query("show create %s `%s`", createType, name)
	if err != nil {
		return "", err
	}

	// 获取ddl
	var ddl string
	for _, row := range res.Rows {
		ddl = row.Str(1)
	}

	return ddl, err
}

// ShowCreateDatabase show create database
func (db *Connector) ShowCreateDatabase(dbName string) (string, error) {
	defer func() {
		err := recover()
		if err != nil {
			common.Log.Error("recover ShowCreateDatabase()", err)
		}
	}()
	return db.showCreate("database", dbName)
}

// ShowCreateTable show create table
func (db *Connector) ShowCreateTable(tableName string) (string, error) {
	defer func() {
		err := recover()
		if err != nil {
			common.Log.Error("recover ShowCreateTable()", err)
		}
	}()

	ddl, err := db.showCreate("table", tableName)

	// 去除外键关联条件
	var noConstraint []string
	relationReg, _ := regexp.Compile("CONSTRAINT")
	for _, line := range strings.Split(ddl, "\n") {

		if relationReg.Match([]byte(line)) {
			continue
		}

		// 去除外键语句会使DDL中多一个','导致语法错误，要把多余的逗号去除
		if strings.Index(line, ")") == 0 {
			lineWrongSyntax := noConstraint[len(noConstraint)-1]
			// 如果')'前一句的末尾是',' 删除 ',' 保证语法正确性
			if strings.Index(lineWrongSyntax, ",") == len(lineWrongSyntax)-1 {
				noConstraint[len(noConstraint)-1] = lineWrongSyntax[:len(lineWrongSyntax)-1]
			}
		}

		noConstraint = append(noConstraint, line)
	}

	return strings.Join(noConstraint, "\n"), err
}

// FindColumn find column
func (db *Connector) FindColumn(name, dbName string, tables ...string) ([]*common.Column, error) {
	// 执行 show create table
	var columns []*common.Column
	sql := fmt.Sprintf("SELECT "+
		"c.TABLE_NAME,c.TABLE_SCHEMA,c.COLUMN_TYPE,c.CHARACTER_SET_NAME, c.COLLATION_NAME "+
		"FROM `INFORMATION_SCHEMA`.`COLUMNS` as c where c.COLUMN_NAME = '%s' ", name)

	if len(tables) > 0 {
		var tmp []string
		for _, table := range tables {
			tmp = append(tmp, "'"+table+"'")
		}
		sql += fmt.Sprintf(" and c.table_name in (%s)", strings.Join(tmp, ","))
	}

	if dbName != "" {
		sql += fmt.Sprintf(" and c.table_schema = '%s'", dbName)
	}

	res, err := db.Query(sql)
	if err != nil {
		common.Log.Error("(db *Connector) FindColumn Error : ", err)
		return columns, err
	}

	tbName := res.Result.Map("TABLE_NAME")
	schema := res.Result.Map("TABLE_SCHEMA")
	colTyp := res.Result.Map("COLUMN_TYPE")
	colCharset := res.Result.Map("CHARACTER_SET_NAME")
	collation := res.Result.Map("COLLATION_NAME")

	// 获取ddl
	for _, row := range res.Rows {
		col := &common.Column{
			Name:      name,
			Table:     row.Str(tbName),
			DB:        row.Str(schema),
			DataType:  row.Str(colTyp),
			Character: row.Str(colCharset),
			Collation: row.Str(collation),
		}

		// 填充字符集和排序规则
		if col.Character == "" {
			// 当从`INFORMATION_SCHEMA`.`COLUMNS`表中查询不到相关列的character和collation的信息时
			// 认为该列使用的character和collation与其所处的表一致
			// 由于`INFORMATION_SCHEMA`.`TABLES`表中未找到表的character，所以从按照MySQL中collation的规则从中截取character

			sql = fmt.Sprintf("SELECT `t`.`TABLE_COLLATION` FROM `INFORMATION_SCHEMA`.`TABLES` AS `t` "+
				"WHERE `t`.`TABLE_NAME`='%s' AND `t`.`TABLE_SCHEMA` = '%s'", col.Table, col.DB)
			var newRes *QueryResult
			newRes, err = db.Query(sql)
			if err != nil {
				common.Log.Error("(db *Connector) FindColumn Error : ", err)
				return columns, err
			}

			tbCollation := newRes.Rows[0].Str(0)
			if tbCollation != "" {
				col.Character = strings.Split(tbCollation, "_")[0]
				col.Collation = tbCollation
			}
		}

		columns = append(columns, col)
	}

	return columns, err
}

// IsFKey 判断列是否是外键
func (db *Connector) IsFKey(dbName, tbName, column string) bool {
	sql := fmt.Sprintf("SELECT REFERENCED_COLUMN_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE C "+
		"WHERE REFERENCED_TABLE_SCHEMA <> 'NULL' AND"+
		" TABLE_NAME='%s' AND"+
		" TABLE_SCHEMA='%s' AND"+
		" COLUMN_NAME='%s'", tbName, dbName, column)

	res, err := db.Query(sql)
	if err == nil && len(res.Rows) == 0 {
		return false
	}

	return true
}

// Reference 用于存储关系
type Reference map[string][]ReferenceValue

// ReferenceValue 用于处理表之间的关系
type ReferenceValue struct {
	RefDBName      string // 夫表所属数据库
	RefTable       string // 父表
	DBName         string // 子表所属数据库
	Table          string // 子表
	ConstraintName string // 关系名称
}

// ShowReference 查找所有的外键信息
func (db *Connector) ShowReference(dbName string, tbName ...string) ([]ReferenceValue, error) {
	var referenceValues []ReferenceValue
	sql := `SELECT C.REFERENCED_TABLE_SCHEMA,C.REFERENCED_TABLE_NAME,C.TABLE_SCHEMA,C.TABLE_NAME,C.CONSTRAINT_NAME 
FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE C JOIN INFORMATION_SCHEMA. TABLES T ON T.TABLE_NAME = C.TABLE_NAME 
WHERE C.REFERENCED_TABLE_NAME IS NOT NULL`
	sql = sql + fmt.Sprintf(` AND C.TABLE_SCHEMA = "%s"`, dbName)

	if len(tbName) > 0 {
		extra := fmt.Sprintf(` AND C.TABLE_NAME IN ("%s")`, strings.Join(tbName, `","`))
		sql = sql + extra
	}

	// 执行SQL查找外键关联关系
	res, err := db.Query(sql)
	if err != nil {
		return referenceValues, err
	}

	refDb := res.Result.Map("REFERENCED_TABLE_SCHEMA")
	refTb := res.Result.Map("REFERENCED_TABLE_NAME")
	schema := res.Result.Map("TABLE_SCHEMA")
	tb := res.Result.Map("TABLE_NAME")
	cName := res.Result.Map("CONSTRAINT_NAME")

	// 获取值
	for _, row := range res.Rows {
		value := ReferenceValue{
			RefDBName:      row.Str(refDb),
			RefTable:       row.Str(refTb),
			DBName:         row.Str(schema),
			Table:          row.Str(tb),
			ConstraintName: row.Str(cName),
		}
		referenceValues = append(referenceValues, value)
	}

	return referenceValues, err

}
