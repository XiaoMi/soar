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
	"flag"
	"runtime"
	"testing"

	"github.com/kr/pretty"
)

var update = flag.Bool("update", false, "update .golden files")

func TestGetDefaultLogOutput(t *testing.T) {
	output := getDefaultLogOutput()
	if runtime.GOOS == "windows" && output != "nul" {
		t.Error("windows default -log-output not nul")
	}

	if runtime.GOOS != "windows" && output != "/dev/stderr" {
		t.Error("default -log-output not /dev/stderr")
	}
}

func TestParseConfig(t *testing.T) {
	err := ParseConfig("")
	if err != nil {
		t.Error("sqlparser.Parse Error:", err)
	}
}

func TestReadConfigFile(t *testing.T) {
	if Config == nil {
		Config = new(Configration)
	}
	Config.readConfigFile("../soar.yaml")
}

func TestParseDSN(t *testing.T) {
	var dsns = []string{
		"",
		"user:password@hostname:3307/database",
		"user:password@hostname:3307",
		"user:password@hostname:/database",
		"user:password@:3307/database",
		"user:password@",
		"hostname:3307/database",
		"@hostname:3307/database",
		"@hostname",
		"hostname",
		"@/database",
		"@hostname:3307",
		"@:3307/database",
		":3307/database",
		"/database",
	}

	GoldenDiff(func() {
		for _, dsn := range dsns {
			pretty.Println(parseDSN(dsn, nil))
		}
	}, t.Name(), update)
}

func TestListReportTypes(t *testing.T) {
	err := GoldenDiff(func() { ListReportTypes() }, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
}
