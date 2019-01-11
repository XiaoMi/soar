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
	"github.com/XiaoMi/soar/common"

	"github.com/kr/pretty"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"

	json "github.com/CorgiMan/json2"

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
