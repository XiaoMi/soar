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
	"encoding/json"

	"github.com/XiaoMi/soar/common"

	"github.com/kr/pretty"
	"vitess.io/vitess/go/vt/sqlparser"
)

// PrintPrettyVitessStmtNode print vitess AST struct data
func PrintPrettyVitessStmtNode(sql string) {
	tree, err := sqlparser.Parse(sql)
	if err != nil {
		common.Log.Warning(err.Error())
	} else {
		_, err = pretty.Println(tree)
		common.LogIfWarn(err, "")
	}
}

// VitessStmtNode2JSON vitess AST tree into json format
func VitessStmtNode2JSON(sql string) string {
	var str string
	tree, err := sqlparser.Parse(sql)
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
