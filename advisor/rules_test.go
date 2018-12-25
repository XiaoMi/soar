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

package advisor

import (
	"strings"
	"testing"

	"github.com/XiaoMi/soar/common"
)

func TestListTestSQLs(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	for _, sql := range common.TestSQLs {
		if !strings.HasSuffix(sql, ";") {
			t.Errorf("%s should end with ';'", sql)
		}
	}
	err := common.GoldenDiff(func() { ListTestSQLs() }, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestListHeuristicRules(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	err := common.GoldenDiff(func() { ListHeuristicRules(HeuristicRules) }, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestInBlackList(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		"select",
		"select 1",
	}
	common.BlackList = []string{"select"}
	for _, sql := range sqls {
		if !InBlackList(sql) {
			t.Error("should be true")
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestIsIgnoreRule(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	common.Config.IgnoreRules = []string{"test"}
	if !IsIgnoreRule("test") {
		t.Error("should be true")
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
