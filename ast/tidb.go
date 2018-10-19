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
	"github.com/pingcap/tidb/ast"
	"github.com/pingcap/tidb/parser"
)

// TiParse TiDB 语法解析
func TiParse(sql, charset, collation string) ([]ast.StmtNode, error) {
	p := parser.New()
	return p.Parse(sql, charset, collation)
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

// TiVisitor TODO
type TiVisitor struct {
	EnterFunc func(node ast.Node) bool
	LeaveFunc func(node ast.Node) bool
}

// Enter TODO
func (visitor *TiVisitor) Enter(n ast.Node) (node ast.Node, skip bool) {
	skip = visitor.EnterFunc(n)
	return
}

// Leave TODO
func (visitor *TiVisitor) Leave(n ast.Node) (node ast.Node, ok bool) {
	ok = visitor.LeaveFunc(n)
	return
}
