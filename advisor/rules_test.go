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
	"flag"
	"testing"

	"github.com/XiaoMi/soar/common"
)

var update = flag.Bool("update", false, "update .golden files")

func TestListTestSQLs(t *testing.T) {
	err := common.GoldenDiff(func() { ListTestSQLs() }, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
}

func TestListHeuristicRules(t *testing.T) {
	err := common.GoldenDiff(func() { ListHeuristicRules(HeuristicRules) }, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
}

func TestInBlackList(t *testing.T) {
	common.BlackList = []string{"select"}
	if !InBlackList("select 1") {
		t.Error("should be true")
	}
}

func TestIsIgnoreRule(t *testing.T) {
	common.Config.IgnoreRules = []string{"test"}
	if !IsIgnoreRule("test") {
		t.Error("should be true")
	}
}
