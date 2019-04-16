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

package common

import (
	"strconv"
	"strings"
)

// Meta 以 'database' 为 key, DB 的 map, 按 db->table->column 组织的元数据
type Meta map[string]*DB

// DB 数据库相关的结构体
type DB struct {
	Name  string
	Table map[string]*Table // ['table_name']*TableName
}

// NewDB 用于初始化*DB
func NewDB(db string) *DB {
	return &DB{
		Name:  db,
		Table: make(map[string]*Table),
	}
}

// Table 含有表的属性
type Table struct {
	TableName    string
	TableAliases []string
	Column       map[string]*Column
}

// NewTable 初始化*TableName
func NewTable(tb string) *Table {
	return &Table{
		TableName:    tb,
		TableAliases: make([]string, 0),
		Column:       make(map[string]*Column),
	}
}

// KeyType 用于标志每个Key的类别
type KeyType int

// Column 含有列的定义属性
type Column struct {
	Name        string   `json:"col_name"`    // 列名
	Alias       []string `json:"alias"`       // 别名
	Table       string   `json:"tb_name"`     // 表名
	DB          string   `json:"db_name"`     // 数据库名称
	DataType    string   `json:"data_type"`   // 数据类型
	Character   string   `json:"character"`   // 字符集
	Collation   string   `json:"collation"`   // collation
	Cardinality float64  `json:"cardinality"` // 散粒度
	Null        string   `json:"null"`        // 是否为空: YES/NO
	Key         string   `json:"key"`         // 键类型
	Default     string   `json:"default"`     // 默认值
	Extra       string   `json:"extra"`       // 其他
	Comment     string   `json:"comment"`     // 备注
	Privileges  string   `json:"privileges"`  // 权限
}

// TableColumns 这个结构体中的元素是有序的  map[db]map[table][]columns
type TableColumns map[string]map[string][]*Column

// Equal 判断两个column是否相等
func (col *Column) Equal(column *Column) bool {
	return col.Name == column.Name &&
		col.Table == column.Table &&
		col.DB == column.DB
}

// IsColsPart 判断两个column队列是否是包含关系（包括相等）
func IsColsPart(a, b []*Column) bool {
	times := len(a)
	if len(b) < times {
		times = len(b)
	}

	for i := 0; i < times; i++ {
		if strings.ToLower(a[i].DB) != strings.ToLower(b[i].DB) ||
			strings.ToLower(a[i].Table) != strings.ToLower(b[i].Table) ||
			strings.ToLower(a[i].Name) != strings.ToLower(b[i].Name) {
			return false
		}
	}

	return true
}

// JoinColumnsName 将所有的列合并
func JoinColumnsName(cols []*Column, sep string) string {
	name := ""
	for _, col := range cols {
		name += col.Name + sep
	}
	return strings.Trim(name, sep)
}

// Tables 获取 Meta 中指定 db 的所有表名
// Input:数据库名
// Output:表名组成的list
func (b Meta) Tables(db string) []string {
	var result []string
	if b[db] != nil {
		for tb := range b[db].Table {
			result = append(result, tb)
		}

	}
	return result
}

// SetDefault 设置默认值
func (b Meta) SetDefault(defaultDB string) Meta {
	if defaultDB == "" {
		return b
	}

	for db := range b {
		if db == "" {
			// 当获取到的 join 中的 DB 为空的时候，说明 SQL 未显示的指定 DB，即使用的是 rEnv 默认 DB, 需要将表合并到原 DB 中
			if _, ok := b[defaultDB]; ok {
				for tbName, table := range b[""].Table {
					if _, ok := b[defaultDB].Table[tbName]; ok {
						b[defaultDB].Table[tbName].TableAliases = append(
							b[defaultDB].Table[tbName].TableAliases,
							table.TableAliases...,
						)
						continue
					}
					b[defaultDB].Table[tbName] = table
				}
				delete(b, "")
			}

			// 如果没有出现DB指定不一致的情况，直接进行合并
			b[defaultDB] = b[""]
			delete(b, "")
		}
	}

	return b
}

// MergeColumn 将使用到的列按 db->table 组织去重
// 注意：Column 中的 db, table 信息可能为空，需要提前通过env环境补齐再调用该函数。
// @input: 目标列list, 源列list（可以将多个源合并到一个目标列list）
// @output: 合并后的列list
func MergeColumn(dst []*Column, src ...*Column) []*Column {
	var tmp []*Column
	for _, newCol := range src {
		if len(dst) == 0 {
			tmp = append(tmp, newCol)
			continue
		}

		has := false
		for _, oldCol := range dst {
			if (newCol.Name == oldCol.Name) && (newCol.Table == oldCol.Table) && (newCol.DB == oldCol.DB) {
				has = true
			}
		}

		if !has {
			tmp = append(tmp, newCol)
		}

	}
	return append(dst, tmp...)
}

// ColumnSort 通过散粒度对 colList 进行排序， 散粒度排序由大到小
func ColumnSort(colList []*Column) []*Column {
	// 使用冒泡排序保持相等情况下左右两边顺序不变
	if len(colList) < 2 {
		return colList
	}

	for i := 0; i < len(colList)-1; i++ {
		for j := i + 1; j < len(colList); j++ {
			if colList[i].Cardinality < colList[j].Cardinality {
				colList[i], colList[j] = colList[j], colList[i]
			}
		}
	}

	return colList
}

// GetDataTypeBase 获取dataType中的数据类型，忽略长度
func GetDataTypeBase(dataType string) string {
	if i := strings.Index(dataType, "("); i > 0 {
		return dataType[0:i]
	}

	return dataType
}

// GetDataTypeLength 获取dataType中的数据类型长度
func GetDataTypeLength(dataType string) []int {
	var length []int
	if si := strings.Index(dataType, "("); si > 0 {
		dataLength := dataType[si+1:]
		if ei := strings.Index(dataLength, ")"); ei > 0 {
			dataLength = dataLength[:ei]
			if strings.HasPrefix(dataType, "enum") ||
				strings.HasPrefix(dataType, "set") {
				// set('one', 'two'), enum('G','PG','PG-13','R','NC-17')
				length = []int{len(strings.Split(dataLength, ","))}
			} else {
				// char(10), varchar(10)
				for _, l := range strings.Split(dataLength, ",") {
					v, err := strconv.Atoi(l)
					if err != nil {
						Log.Debug("GetDataTypeLength() Error: %v", err)
						return []int{-1}
					}
					length = append(length, v)
				}
			}
		}
	}

	if len(length) == 0 {
		length = []int{-1}
	}

	return length
}

// GetDataBytes 计算数据类型字节数
// https://dev.mysql.com/doc/refman/8.0/en/storage-requirements.html
// return -1 表示该列无法计算数据大小
func (col *Column) GetDataBytes(dbVersion int) int {
	if col.DataType == "" {
		Log.Warning("Can't get %s.%s data type", col.Table, col.Name)
		return -1
	}
	switch strings.ToLower(GetDataTypeBase(col.DataType)) {
	case "tinyint", "smallint", "mediumint",
		"int", "integer", "bigint",
		"double", "real", "float", "decimal",
		"numeric", "bit":
		// numeric
		return numericStorageReq(col.DataType)

	case "year", "date", "time", "datetime", "timestamp":
		// date & time
		return timeStorageReq(col.DataType, dbVersion)

	case "char", "binary", "varchar", "varbinary", "enum", "set":
		// string
		return StringStorageReq(col.DataType, col.Character)
	case "tinyblob", "tinytext", "blob", "text", "mediumblob", "mediumtext",
		"longblob", "longtext":
		// strings length depend on it's values
		// 这些字段为不定长字段，添加索引时必须指定前缀，索引前缀与字符集相关
		return Config.MaxIdxBytesPerColumn + 1
	default:
		Log.Warning("Type %s not support:", col.DataType)
		return -1
	}
}

// Numeric Type Storage Requirements
// return bytes count
func numericStorageReq(dataType string) int {
	typeLength := GetDataTypeLength(dataType)
	baseType := strings.ToLower(GetDataTypeBase(dataType))

	switch baseType {
	case "tinyint":
		return 1
	case "smallint":
		return 2
	case "mediumint":
		return 3
	case "int", "integer":
		return 4
	case "bigint", "double", "real":
		return 8
	case "float":
		if typeLength[0] == -1 || typeLength[0] >= 0 && typeLength[0] <= 24 {
			// 4 bytes if 0 <= p <= 24
			return 4
		}
		// 8 bytes if no p || 25 <= p <= 53
		return 8
	case "decimal", "numeric":
		// Values for DECIMAL (and NUMERIC) columns are represented using a binary format
		// that packs nine decimal (base 10) digits into four bytes. Storage for the integer
		// and fractional parts of each value are determined separately. Each multiple of nine
		// digits requires four bytes, and the “leftover” digits require some fraction of four bytes.

		if typeLength[0] == -1 {
			return 4
		}

		leftover := func(leftover int) int {
			if leftover > 0 && leftover <= 2 {
				return 1
			} else if leftover > 2 && leftover <= 4 {
				return 2
			} else if leftover > 4 && leftover <= 6 {
				return 3
			} else if leftover > 6 && leftover <= 8 {
				return 4
			} else {
				return 4
			}
		}

		integer := typeLength[0]/9*4 + leftover(typeLength[0]%9)
		fractional := typeLength[1]/9*4 + leftover(typeLength[1]%9)

		return integer + fractional

	case "bit":
		// approximately (M+7)/8 bytes
		if typeLength[0] == -1 {
			return 1
		}
		return (typeLength[0] + 7) / 8

	default:
		Log.Error("No such numeric type: %s", baseType)
		return 8
	}
}

// Date and Time Type Storage Requirements
// return bytes count
func timeStorageReq(dataType string, version int) int {
	/*
			https://dev.mysql.com/doc/refman/8.0/en/storage-requirements.html
			*   ============================================================================================
			*   |	Data Type |	Storage Required Before MySQL 5.6.4	| Storage Required as of MySQL 5.6.4   |
			*   | ---------------------------------------------------------------------------------------- |
			*   |	YEAR	  |	1 byte	                            | 1 byte                               |
			*   |	DATE	  | 3 bytes	                            | 3 bytes                              |
			*   |	TIME	  | 3 bytes	                            | 3 bytes + fractional seconds storage |
			*   |	DATETIME  | 8 bytes	                            | 5 bytes + fractional seconds storage |
			*   |	TIMESTAMP |	4 bytes	                            | 4 bytes + fractional seconds storage |
			*   ============================================================================================
			*	|  Fractional Seconds Precision |Storage Required  |
			*   | ------------------------------------------------ |
			*	|  0	    					|0 bytes		   |
			*	|  1, 2						    |1 byte            |
			*	|  3, 4						    |2 bytes           |
			*	|  5, 6						    |3 bytes           |
		    *   ====================================================
	*/

	typeLength := GetDataTypeLength(dataType)

	extr := func(length int) int {
		if length > 0 && length <= 2 {
			return 1
		} else if length > 2 && length <= 4 {
			return 2
		} else if length > 4 && length <= 6 || length > 6 {
			return 3
		}
		return 0
	}

	switch strings.ToLower(GetDataTypeBase(dataType)) {
	case "year":
		return 1
	case "date":
		return 3
	case "time":
		if version < 50604 {
			return 3
		}
		// 3 bytes + fractional seconds storage
		return 3 + extr(typeLength[0])
	case "datetime":
		if version < 50604 {
			return 8
		}
		// 5 bytes + fractional seconds storage
		return 5 + extr(typeLength[0])
	case "timestamp":
		if version < 50604 {
			return 4
		}
		// 4 bytes + fractional seconds storage
		return 4 + extr(typeLength[0])
	default:
		return 8
	}
}

// SHOW CHARACTER SET

// CharSets character bytes per charcharacter bytes per char
var CharSets = map[string]int{
	"armscii8": 1,
	"ascii":    1,
	"big5":     2,
	"binary":   1,
	"cp1250":   1,
	"cp1251":   1,
	"cp1256":   1,
	"cp1257":   1,
	"cp850":    1,
	"cp852":    1,
	"cp866":    1,
	"cp932":    2,
	"dec8":     1,
	"eucjpms":  3,
	"euckr":    2,
	"gb18030":  4,
	"gb2312":   2,
	"gbk":      2,
	"geostd8":  1,
	"greek":    1,
	"hebrew":   1,
	"hp8":      1,
	"keybcs2":  1,
	"koi8r":    1,
	"koi8u":    1,
	"latin1":   1,
	"latin2":   1,
	"latin5":   1,
	"latin7":   1,
	"macce":    1,
	"macroman": 1,
	"sjis":     2,
	"swe7":     1,
	"tis620":   1,
	"ucs2":     2,
	"ujis":     3,
	"utf16":    4,
	"utf16le":  4,
	"utf32":    4,
	"utf8":     3,
	"utf8mb4":  4,
}

// StringStorageReq String Type Storage Requirements return bytes count
func StringStorageReq(dataType string, charset string) int {
	// get bytes per character, default 1
	bysPerChar := 1
	if _, ok := CharSets[strings.ToLower(charset)]; ok {
		bysPerChar = CharSets[strings.ToLower(charset)]
	}

	// get length
	typeLength := GetDataTypeLength(dataType)
	if typeLength[0] == -1 {
		return 0
	}

	// get type
	baseType := strings.ToLower(GetDataTypeBase(dataType))

	switch baseType {
	case "char":
		// Otherwise, M × w bytes, <= M <= 255,
		// where w is the number of bytes required for the maximum-length character in the character set.
		if typeLength[0] > 255 {
			typeLength[0] = 255
		}
		return typeLength[0] * bysPerChar
	case "binary":
		// M bytes, 0 <= M <= 255
		if typeLength[0] > 255 {
			typeLength[0] = 255
		}
		return typeLength[0]
	case "varchar", "varbinary":
		if typeLength[0] < 255 {
			return typeLength[0]*bysPerChar + 1
		}
		return typeLength[0]*bysPerChar + 2
	case "enum":
		// 1 or 2 bytes, depending on the number of enumeration values (65,535 values maximum)
		return typeLength[0]/(2^15) + 1
	case "set":
		// 1, 2, 3, 4, or 8 bytes, depending on the number of set members (64 members maximum)
		return typeLength[0]/8 + 1
	default:
		return 0
	}
}
