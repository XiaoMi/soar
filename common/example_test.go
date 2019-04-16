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

import "fmt"

func ExampleFormatDSN() {
	Log.Debug("Entering function: %s", GetFunctionName())
	dsxExp := newDSN(nil)
	dsxExp.Addr = "127.0.0.1:3306"
	dsxExp.Schema = "mysql"
	dsxExp.User = "root"
	dsxExp.Password = "1t'sB1g3rt"
	dsxExp.Charset = "utf8mb4"
	dsxExp.Disable = false

	// 根据 &dsn 生成 dsnStr
	fmt.Println(FormatDSN(dsxExp))

	// Output: root:1t'sB1g3rt@tcp(127.0.0.1:3306)/mysql?charset=utf8mb4
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func ExampleIsColsPart() {
	Log.Debug("Entering function: %s", GetFunctionName())
	// IsColsPart() 会 按照顺序 检查两个Column队列是否是包含（或相等）关系。
	a := []*Column{{Name: "1"}, {Name: "2"}, {Name: "3"}}
	b := []*Column{{Name: "1"}, {Name: "2"}}
	c := []*Column{{Name: "1"}, {Name: "3"}}
	d := []*Column{{Name: "1"}, {Name: "2"}, {Name: "3"}, {Name: "4"}}
	id := []*Column{{Name: "id"}}
	iD := []*Column{{Name: "iD"}}

	ab := IsColsPart(a, b)
	ac := IsColsPart(a, c)
	ad := IsColsPart(a, d)
	idiD := IsColsPart(id, iD) // 大小写对比

	fmt.Println(ab, ac, ad, idiD)
	// Output: true false true true
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func ExampleSortedKey() {
	Log.Debug("Entering function: %s", GetFunctionName())
	ages := map[string]int{
		"a": 1,
		"c": 3,
		"d": 4,
		"b": 2,
	}
	for _, name := range SortedKey(ages) {
		fmt.Print(ages[name])
	}
	// Output: 1234
	Log.Debug("Exiting function: %s", GetFunctionName())
}
