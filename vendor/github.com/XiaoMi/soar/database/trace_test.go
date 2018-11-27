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
	"flag"
	"testing"

	"github.com/XiaoMi/soar/common"

	"github.com/kr/pretty"
)

var update = flag.Bool("update", false, "update .golden files")

func TestTrace(t *testing.T) {
	common.Config.QueryTimeOut = 1
	res, err := connTest.Trace("select 1")
	if err == nil {
		common.GoldenDiff(func() {
			pretty.Println(res)
		}, t.Name(), update)
	} else {
		t.Error(err)
	}
}

func TestFormatTrace(t *testing.T) {
	res, err := connTest.Trace("select 1")
	if err == nil {
		pretty.Println(FormatTrace(res))
	} else {
		t.Error(err)
	}
}

func TestGetTrace(t *testing.T) {
	res, err := connTest.Trace("select 1")
	if err == nil {
		pretty.Println(getTrace(res))
	} else {
		t.Error(err)
	}
}
