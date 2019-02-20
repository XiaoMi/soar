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

	"github.com/XiaoMi/soar/common"

	json "github.com/CorgiMan/json2"
	"github.com/kr/pretty"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	"github.com/tidwall/gjson"

	// for pingcap parser
	_ "github.com/pingcap/tidb/types/parser_driver"
)

// TiParse TiDB 语法解析
func TiParse(sql, charset, collation string) ([]ast.StmtNode, error) {
	p := parser.New()
	stmt, warn, err := p.Parse(sql, charset, collation)
	// TODO: bypass warning info
	for _, w := range warn {
		common.Log.Warn(w.Error())
	}
	return stmt, err
}

// PrintPrettyStmtNode 打印TiParse语法树
func PrintPrettyStmtNode(sql, charset, collation string) {
	tree, err := TiParse(sql, charset, collation)
	if err != nil {
		common.Log.Warning(err.Error())
	} else {
		_, err = pretty.Println(tree)
		common.LogIfWarn(err, "")
	}
}

// StmtNode2JSON TiParse AST tree into json format
func StmtNode2JSON(sql, charset, collation string) string {
	var str string
	tree, err := TiParse(sql, charset, collation)
	if err != nil {
		common.Log.Warning(err.Error())
	} else {
		b, err := json.MarshalIndent(tree, "", "  ")
		if err != nil {
			common.Log.Error(err.Error())
		} else {
			str = string(b)
		}
	}
	return str
}

// SchemaMetaInfo get used database, table name from SQL
func SchemaMetaInfo(sql string, defaultDatabase string) []string {
	var tables []string
	tree, err := TiParse(sql, "", "")
	if err != nil {
		return tables
	}

	jsonString := StmtNode2JSON(sql, "", "")

	for _, node := range tree {
		switch n := node.(type) {
		case *ast.UseStmt:
			tables = append(tables, fmt.Sprintf("`%s`.`dual`", n.DBName))
		case *ast.InsertStmt, *ast.SelectStmt, *ast.UnionStmt, *ast.UpdateStmt, *ast.DeleteStmt:
			// DML/DQL: INSERT, SELECT, UPDATE, DELETE
			for _, tableRef := range common.JSONFind(jsonString, "TableRefs") {
				for _, source := range common.JSONFind(tableRef, "Source") {
					database := gjson.Get(source, "Schema.O")
					table := gjson.Get(source, "Name.O")
					if database.String() == "" {
						if table.String() != "" {
							tables = append(tables, fmt.Sprintf("`%s`.`%s`", defaultDatabase, table.String()))
						}
					} else {
						if table.String() != "" {
							tables = append(tables, fmt.Sprintf("`%s`.`%s`", database.String(), table.String()))
						} else {
							tables = append(tables, fmt.Sprintf("`%s`.`dual`", database.String()))
						}
					}
				}
			}
		case *ast.DropTableStmt:
			// DDL: DROP TABLE|VIEW
			schemas := common.JSONFind(jsonString, "Tables")
			for _, tabs := range schemas {
				for _, table := range gjson.Parse(tabs).Array() {
					db := gjson.Get(table.String(), "Schema.O")
					tb := gjson.Get(table.String(), "Name.O")
					if db.String() == "" {
						if tb.String() != "" {
							tables = append(tables, fmt.Sprintf("`%s`.%s`", defaultDatabase, tb.String()))
						}
					} else {
						if tb.String() != "" {
							tables = append(tables, fmt.Sprintf("`%s`.`%s`", db.String(), tb.String()))
						}
					}
				}
			}
		case *ast.DropDatabaseStmt, *ast.CreateDatabaseStmt:
			// DDL: DROP|CREATE DATABASE
			schemas := common.JSONFind(jsonString, "Name")
			for _, schema := range schemas {
				tables = append(tables, fmt.Sprintf("`%s`.`dual`", schema))
			}
		default:
			// DDL: CREATE TABLE|DATABASE|INDEX|VIEW, DROP INDEX
			schemas := common.JSONFind(jsonString, "Table")
			for _, table := range schemas {
				db := gjson.Get(table, "Schema.O")
				tb := gjson.Get(table, "Name.O")
				if db.String() == "" {
					if tb.String() != "" {
						tables = append(tables, fmt.Sprintf("`%s`.%s`", defaultDatabase, tb.String()))
					}
				} else {
					if tb.String() != "" {
						tables = append(tables, fmt.Sprintf("`%s`.`%s`", db.String(), tb.String()))
					}
				}
			}
		}
	}
	return common.RemoveDuplicatesItem(tables)
}
