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
	"testing"

	"github.com/XiaoMi/soar/common"

	"github.com/kr/pretty"
)

func TestTrace(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select 1",
		"explain select 1",
		"show create table film",
	}
	err := common.GoldenDiff(func() {
		for _, sql := range sqls {
			res, err := connTest.Trace(sql)
			pretty.Println(sql, res, err)
		}
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFormatTrace(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	res, err := connTest.Trace("select 1")
	if err != nil {
		t.Error(err)
	}

	err = common.GoldenDiff(func() {
		pretty.Println(FormatTrace(res))
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
