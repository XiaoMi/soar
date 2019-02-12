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
// use []byte instead of string, because []byte allow to be null, string not
type tableStatusRow struct {
	Name         string // 表名
	Engine       []byte // 该表使用的存储引擎
	Version      []byte // 该表的 .frm 文件版本号
	RowFormat    []byte // 该表使用的行存储格式
	Rows         uint64 // 表行数, InnoDB 引擎中为预估值，甚至可能会有40%~50%的数值偏差
	AvgRowLength uint64 // 平均行长度

	// MyISAM: Data_length 为数据文件的大小，单位为 bytes
	// InnoDB: Data_length 为聚簇索引分配的近似内存量，单位为 bytes, 计算方式为聚簇索引数量乘以 InnoDB 页面大小
	// 其他不同的存储引擎中该值的意义可能不尽相同
	DataLength uint64

	// MyISAM: Max_data_length 为数据文件长度的最大值。这是在给定使用的数据指针大小的情况下，可以存储在表中的数据的最大字节数
	// InnoDB: 未使用
	// 其他不同的存储引擎中该值的意义可能不尽相同
	MaxDataLength uint64

	// MyISAM: Index_length 为 index 文件的大小，单位为 bytes
	// InnoDB: Index_length 为非聚簇索引分配的近似内存量，单位为 bytes，计算方式为非聚簇索引数量乘以 InnoDB 页面大小
	// 其他不同的存储引擎中该值的意义可能不尽相同
	IndexLength uint64

	DataFree      uint64 // 已分配但未使用的字节数
	AutoIncrement []byte // 下一个自增值
	CreateTime    []byte // 创建时间
	UpdateTime    []byte // 最近一次更新时间，该值不准确
	CheckTime     []byte // 上次检查时间
	Collation     []byte // 字符集及排序规则信息
	Checksum      []byte // 校验和
	CreateOptions []byte // 创建表的时候的时候一切其他属性
	Comment       []byte // 注释
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
			common.Log.Error("recover ShowTables()", err)
		}
	}()

	// 执行 show table status
	res, err := db.Query("show tables")
	if err != nil {
		return []string{}, err
	}

	// 获取值
	var tables []string
	for res.Rows.Next() {
		var table string
		err = res.Rows.Scan(&table)
		if err != nil {
			break
		}
		tables = append(tables, table)
	}
	res.Rows.Close()
	return tables, err
}

// ShowTableStatus 执行 show table status
func (db *Connector) ShowTableStatus(tableName string) (*TableStatInfo, error) {
	// 初始化struct
	tbStatus := newTableStat(tableName)

	// 执行 show table status
	res, err := db.Query(fmt.Sprintf("show table status where name = '%s'", Escape(tbStatus.Name, false)))
	if err != nil {
		return tbStatus, err
	}

	// columns info
	ts := tableStatusRow{}
	statusFields := make([]interface{}, 0)
	fields := map[string]interface{}{
		"Name":            &ts.Name,
		"Engine":          &ts.Engine,
		"Version":         &ts.Version,
		"Row_format":      &ts.RowFormat,
		"Rows":            &ts.Rows,
		"Avg_row_length":  &ts.AvgRowLength,
		"Data_length":     &ts.DataLength,
		"Max_data_length": &ts.MaxDataLength,
		"Index_length":    &ts.IndexLength,
		"Data_free":       &ts.DataFree,
		"Auto_increment":  &ts.AutoIncrement,
		"Create_time":     &ts.CreateTime,
		"Update_time":     &ts.UpdateTime,
		"Check_time":      &ts.CheckTime,
		"Collation":       &ts.Collation,
		"Checksum":        &ts.Checksum,
		"Create_options":  &ts.CreateOptions,
		"Comment":         &ts.Comment,
	}
	cols, err := res.Rows.Columns()
	common.LogIfError(err, "")
	var colByPass []byte
	for _, col := range cols {
		if _, ok := fields[col]; ok {
			statusFields = append(statusFields, fields[col])
		} else {
			common.Log.Debug("ShowTableStatus by pass column %s", col)
			statusFields = append(statusFields, &colByPass)
		}
	}
	// 获取值
	for res.Rows.Next() {
		err := res.Rows.Scan(statusFields...)
		if err != nil {
			// MariaDB 中视图的 STATUS 信息大部分表都为 NULL，此时会打印如下 DEBUG 级别日志信息，看到后忽略即可。
			// sql: Scan error on column index 4: converting driver.Value type <nil> ("<nil>") to uint64: invalid syntax
			common.Log.Debug(err.Error())
		}
		tbStatus.Rows = append(tbStatus.Rows, ts)
	}
	res.Rows.Close()
	return tbStatus, err
}

// https://dev.mysql.com/doc/refman/5.7/en/show-index.html

// TableIndexInfo 用以保存 show index 之后获取的 index 信息
type TableIndexInfo struct {
	TableName string
	Rows      []TableIndexRow
}

// TableIndexRow 用以存放show index 之后获取的每一条 index 信息
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
	Visible      string
	Expression   []byte
}

// NewTableIndexInfo 构造 TableIndexInfo
func NewTableIndexInfo(tableName string) *TableIndexInfo {
	return &TableIndexInfo{
		TableName: tableName,
		Rows:      make([]TableIndexRow, 0),
	}
}

// ShowIndex show Index
func (db *Connector) ShowIndex(tableName string) (*TableIndexInfo, error) {
	tbIndex := NewTableIndexInfo(tableName)

	if db.Database == "" || tableName == "" {
		return nil, fmt.Errorf("database('%s') or table('%s') name should not empty", db.Database, tableName)
	}

	// 执行 show create table
	res, err := db.Query(fmt.Sprintf("show index from `%s`.`%s`", Escape(db.Database, false), Escape(tableName, false)))
	if err != nil {
		return nil, err
	}

	// columns info
	ti := TableIndexRow{}
	indexFields := make([]interface{}, 0)
	fields := map[string]interface{}{
		"Table":         &ti.Table,
		"Non_unique":    &ti.NonUnique,
		"Key_name":      &ti.KeyName,
		"Seq_in_index":  &ti.SeqInIndex,
		"Column_name":   &ti.ColumnName,
		"Collation":     &ti.Collation,
		"Cardinality":   &ti.Cardinality,
		"Sub_part":      &ti.SubPart,
		"Packed":        &ti.Packed,
		"Null":          &ti.Null,
		"Index_type":    &ti.IndexType,
		"Comment":       &ti.Comment,
		"Index_comment": &ti.IndexComment,
		"Visible":       &ti.Visible,
		"Expression":    &ti.Expression,
	}
	cols, err := res.Rows.Columns()
	common.LogIfError(err, "")
	var colByPass []byte
	for _, col := range cols {
		if _, ok := fields[col]; ok {
			indexFields = append(indexFields, fields[col])
		} else {
			common.Log.Debug("ShowIndex by pass column %s", col)
			indexFields = append(indexFields, &colByPass)
		}
	}
	// 获取值
	for res.Rows.Next() {
		err := res.Rows.Scan(indexFields...)
		if err != nil {
			common.Log.Debug(err.Error())
		}
		tbIndex.Rows = append(tbIndex.Rows, ti)
	}
	res.Rows.Close()
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

// FindIndex 获取 TableIndexInfo 中需要的索引
func (tbIndex *TableIndexInfo) FindIndex(arg IndexSelectKey, value string) []TableIndexRow {
	var result []TableIndexRow
	if tbIndex == nil {
		return result
	}

	value = strings.ToLower(value)

	switch arg {
	case IndexKeyName:
		for _, index := range tbIndex.Rows {
			if strings.ToLower(index.KeyName) == value {
				result = append(result, index)
			}
		}

	case IndexColumnName:
		for _, index := range tbIndex.Rows {
			if strings.ToLower(index.ColumnName) == value {
				result = append(result, index)
			}
		}

	case IndexIndexType:
		for _, index := range tbIndex.Rows {
			if strings.ToLower(index.IndexType) == value {
				result = append(result, index)
			}
		}

	case IndexNonUnique:
		for _, index := range tbIndex.Rows {
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
	Collation  []byte // 字符集
	Null       string // 是否有NULL（NO、YES）
	Key        string // 键类型
	Default    []byte // 默认值
	Extra      string // 其他
	Privileges string // 权限
	Comment    string // 备注
}

// NewTableDesc 初始化一个*TableDesc
func NewTableDesc(tableName string) *TableDesc {
	return &TableDesc{
		Name:       tableName,
		DescValues: make([]TableDescValue, 0),
	}
}

// ShowColumns 获取 DB 中所有的 columns
func (db *Connector) ShowColumns(tableName string) (*TableDesc, error) {
	tbDesc := NewTableDesc(tableName)

	// 执行 show create table
	res, err := db.Query(fmt.Sprintf("show full columns from `%s`.`%s`", Escape(db.Database, false), Escape(tableName, false)))
	if err != nil {
		return nil, err
	}

	// columns info
	tc := TableDescValue{}
	columnFields := make([]interface{}, 0)
	fields := map[string]interface{}{
		"Field":      &tc.Field,
		"Type":       &tc.Type,
		"Collation":  &tc.Collation,
		"Null":       &tc.Null,
		"Key":        &tc.Key,
		"Default":    &tc.Default,
		"Extra":      &tc.Extra,
		"Privileges": &tc.Privileges,
		"Comment":    &tc.Comment,
	}
	cols, err := res.Rows.Columns()
	common.LogIfError(err, "")
	var colByPass []byte
	for _, col := range cols {
		if _, ok := fields[col]; ok {
			columnFields = append(columnFields, fields[col])
		} else {
			common.Log.Debug("ShowColumns by pass column %s", col)
			columnFields = append(columnFields, &colByPass)
		}
	}
	// 获取值
	for res.Rows.Next() {
		err := res.Rows.Scan(columnFields...)
		if err != nil {
			common.Log.Debug(err.Error())
		}
		tbDesc.DescValues = append(tbDesc.DescValues, tc)
	}
	res.Rows.Close()
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
	// SHOW CREATE DATABASE db_name
	// SHOW CREATE EVENT event_name
	// SHOW CREATE FUNCTION func_name
	// SHOW CREATE PROCEDURE proc_name
	// SHOW CREATE TABLE tbl_name
	// SHOW CREATE TRIGGER trigger_name
	// SHOW CREATE VIEW view_name
	res, err := db.Query(fmt.Sprintf("SHOW CREATE %s `%s`", createType, Escape(name, false)))
	if err != nil {
		return "", err
	}

	// columns info
	var create string
	createFields := make([]interface{}, 0)
	fields := map[string]interface{}{
		"Create View":      &create,
		"Create Table":     &create,
		"Create Database":  &create,
		"Create Event":     &create,
		"Statement":        &create, // show create trigger
		"Create Function":  &create,
		"Create Procedure": &create,
	}
	cols, err := res.Rows.Columns()
	common.LogIfError(err, "")
	var colByPass []byte
	for _, col := range cols {
		if _, ok := fields[col]; ok {
			createFields = append(createFields, fields[col])
		} else {
			common.Log.Debug("showCreate Type: %s, Name: %s, by pass column `%s`", createType, name, col)
			createFields = append(createFields, &colByPass)
		}
	}

	// 获取 CREATE 语句
	for res.Rows.Next() {
		err := res.Rows.Scan(createFields...)
		if err != nil {
			common.Log.Debug(err.Error())
		}
	}
	res.Rows.Close()
	return create, err
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

	ddl, err := db.showCreate("TABLE", tableName)

	// 去除外键关联条件
	lines := strings.Split(ddl, "\n")
	// CREATE VIEW ONLY 1 LINE
	if len(lines) > 2 {
		var noConstraint []string
		relationReg, _ := regexp.Compile("CONSTRAINT")
		for _, line := range lines[1 : len(lines)-1] {
			if relationReg.Match([]byte(line)) {
				continue
			}
			line = strings.TrimSuffix(line, ",")
			noConstraint = append(noConstraint, line)
		}

		// 去除外键语句会使DDL中多一个','导致语法错误，要把多余的逗号去除
		ddl = fmt.Sprint(
			lines[0], "\n",
			strings.Join(noConstraint, ",\n"), "\n",
			lines[len(lines)-1],
		)
	}
	return ddl, err
}

// FindColumn find column
func (db *Connector) FindColumn(name, dbName string, tables ...string) ([]*common.Column, error) {
	// 执行 show create table
	var columns []*common.Column
	sql := fmt.Sprintf("SELECT "+
		"c.TABLE_NAME,c.TABLE_SCHEMA,c.COLUMN_TYPE,c.CHARACTER_SET_NAME, c.COLLATION_NAME "+
		"FROM `INFORMATION_SCHEMA`.`COLUMNS` as c where c.COLUMN_NAME = '%s' ", Escape(name, false))

	if dbName != "" {
		sql += fmt.Sprintf(" and c.table_schema = '%s'", Escape(dbName, false))
	}

	if len(tables) > 0 {
		var tmp []string
		for _, table := range tables {
			tmp = append(tmp, "'"+Escape(table, false)+"'")
		}
		sql += fmt.Sprintf(" and c.table_name in (%s)", strings.Join(tmp, ","))
	}

	common.Log.Debug("FindColumn, execute SQL: %s", sql)
	res, err := db.Query(sql)
	if err != nil {
		common.Log.Error("(db *Connector) FindColumn Error : ", err)
		return columns, err
	}

	var col common.Column
	for res.Rows.Next() {
		var character, collation []byte
		err = res.Rows.Scan(&col.Table, &col.DB, &col.DataType, &character, &collation)
		if err != nil {
			break
		}
		col.Name = name
		col.Character = string(character)
		col.Collation = string(collation)
		// 填充字符集和排序规则
		if col.Character == "" {
			// 当从 `INFORMATION_SCHEMA`.`COLUMNS` 表中查询不到相关列的 character 和 collation 的信息时
			// 认为该列使用的 character 和 collation 与其所处的表一致
			// 由于 `INFORMATION_SCHEMA`.`TABLES` 表中未找到表的 character，所以从按照 MySQL 中 collation 的规则从中截取 character

			sql = fmt.Sprintf("SELECT `t`.`TABLE_COLLATION` FROM `INFORMATION_SCHEMA`.`TABLES` AS `t` "+
				"WHERE `t`.`TABLE_NAME`='%s' AND `t`.`TABLE_SCHEMA` = '%s'", Escape(col.Table, false), Escape(col.DB, false))

			common.Log.Debug("FindColumn, execute SQL: %s", sql)
			var newRes QueryResult
			newRes, err = db.Query(sql)
			if err != nil {
				common.Log.Error("(db *Connector) FindColumn Error : ", err)
				return columns, err
			}

			var tbCollation []byte
			if newRes.Rows.Next() {
				err = newRes.Rows.Scan(&tbCollation)
				if err != nil {
					break
				}
			}
			newRes.Rows.Close()
			if string(tbCollation) != "" {
				col.Character = strings.Split(string(tbCollation), "_")[0]
				col.Collation = string(tbCollation)
			}
		}
		columns = append(columns, &col)
	}
	res.Rows.Close()
	return columns, err
}

// IsForeignKey 判断列是否是外键
func (db *Connector) IsForeignKey(dbName, tbName, column string) bool {
	sql := fmt.Sprintf("SELECT REFERENCED_COLUMN_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE C "+
		"WHERE REFERENCED_TABLE_SCHEMA <> 'NULL' AND"+
		" TABLE_NAME='%s' AND"+
		" TABLE_SCHEMA='%s' AND"+
		" COLUMN_NAME='%s'", Escape(tbName, false), Escape(dbName, false), Escape(column, false))

	common.Log.Debug("IsForeignKey, execute SQL: %s", sql)
	res, err := db.Query(sql)
	if err != nil {
		common.Log.Error("IsForeignKey, Error: %s", err.Error())
		return false
	}
	if res.Rows.Next() {
		return true
	}

	return false
}

// Reference 用于存储关系
type Reference map[string][]ReferenceValue

// ReferenceValue 用于处理表之间的关系
type ReferenceValue struct {
	ReferencedTableSchema string // 夫表所属数据库
	ReferencedTableName   string // 父表
	TableSchema           string // 子表所属数据库
	TableName             string // 子表
	ConstraintName        string // 关系名称
}

// ShowReference 查找所有的外键信息
func (db *Connector) ShowReference(dbName string, tbName ...string) ([]ReferenceValue, error) {
	var referenceValues []ReferenceValue
	sql := `SELECT DISTINCT C.REFERENCED_TABLE_SCHEMA,C.REFERENCED_TABLE_NAME,C.TABLE_SCHEMA,C.TABLE_NAME,C.CONSTRAINT_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE C JOIN INFORMATION_SCHEMA. TABLES T ON T.TABLE_NAME = C.TABLE_NAME WHERE C.REFERENCED_TABLE_NAME IS NOT NULL`
	sql = sql + fmt.Sprintf(` AND C.TABLE_SCHEMA = "%s"`, Escape(dbName, false))

	var tables []string
	for _, tb := range tbName {
		tables = append(tables, "'"+Escape(tb, false)+"'")
	}
	if len(tbName) > 0 {
		extra := fmt.Sprintf(` AND C.TABLE_NAME IN ("%s")`, strings.Join(tables, ","))
		sql = sql + extra
	}

	common.Log.Debug("ShowReference, execute SQL: %s", sql)
	// 执行SQL查找外键关联关系
	res, err := db.Query(sql)
	if err != nil {
		return referenceValues, err
	}

	// 获取值
	for res.Rows.Next() {
		var rv ReferenceValue
		err = res.Rows.Scan(&rv.ReferencedTableSchema, &rv.ReferencedTableName, &rv.TableSchema, &rv.TableName, &rv.ConstraintName)
		if err != nil {
			break
		}
		referenceValues = append(referenceValues, rv)
	}
	res.Rows.Close()
	return referenceValues, err
}
