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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kr/pretty"
)

var update = flag.Bool("update", false, "update .golden files")

func TestMain(m *testing.M) {
	// 初始化 init
	if DevPath == "" {
		_, file, _, _ := runtime.Caller(0)
		DevPath, _ = filepath.Abs(filepath.Dir(filepath.Join(file, ".."+string(filepath.Separator))))
	}
	BaseDir = DevPath
	err := ParseConfig("")
	LogIfError(err, "init ParseConfig")
	Log.Debug("mysql_test init")

	// 分割线
	flag.Parse()
	m.Run()

	// 环境清理
	//
}

func TestParseConfig(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	err := ParseConfig("")
	if err != nil {
		t.Error("sqlparser.Parse Error:", err)
	}
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func TestReadConfigFile(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	if Config == nil {
		Config = new(Configuration)
	}
	Config.readConfigFile(filepath.Join(DevPath, "etc/soar.yaml"))
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func TestParseDSN(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	var dsns = []string{
		// version < 0.11.0
		"",
		"user:password@hostname:3307/database",
		"user:password@hostname:3307/database?charset=utf8",
		"user:password@hostname:3307",
		"user:password@hostname:/database",
		"user:password@:3307/database",
		"user@hostname/database",
		"user:pwd:pwd@pwd/pwd@hostname/database",
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
		// go-sql-driver dsn
		"user@unix(/path/to/socket)/dbname",
		"root:pw@unix(/tmp/mysql.sock)/myDatabase?loc=Local",
		"user:password@tcp(localhost:5555)/dbname?tls=skip-verify&autocommit=true",
		"user:password@/dbname?sql_mode=TRADITIONAL",
		"user:password@tcp([de:ad:be:ef::ca:fe]:80)/dbname?timeout=90s&collation=utf8mb4_unicode_ci",
		"id:password@tcp(your-amazonaws-uri.com:3306)/dbname",
		"user@cloudsql(project-id:instance-name)/dbname",
		"user@cloudsql(project-id:regionname:instance-name)/dbname",
		"user:password@tcp/dbname?charset=utf8mb4,utf8&sys_var=esc%40ped",
		"user:password@/dbname",
		"user:password@/",
		"user:password@tcp(localhost:3307)/database?charset=utf8&timeout=5s",
	}

	err := GoldenDiff(func() {
		for _, dsn := range dsns {
			pretty.Println(dsn)
			pretty.Println(ParseDSN(dsn, nil))
		}
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func TestListReportTypes(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	err := GoldenDiff(func() { ListReportTypes() }, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func TestArgConfig(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	testArgs1 := [][]string{
		{"soar", "-config", "=", "soar.yaml"},
		{"soar", "-print-config", "-config", "soar.yaml"},
	}
	testArgs2 := [][]string{
		{"soar", "-config", "soar.yaml"},
		{"soar", "-config", "=soar.yaml"},
		{"soar", "-config=soar.yaml"},
	}
	for _, args := range testArgs1 {
		os.Args = args
		configFile := ArgConfig()
		if configFile != "" {
			t.Errorf("should return '', but got %s", configFile)
		}
	}
	for _, args := range testArgs2 {
		os.Args = args
		configFile := ArgConfig()
		if configFile != "soar.yaml" {
			t.Errorf("should return soar.yaml, but got %s", configFile)
		}
	}
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func TestPrintConfiguration(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	Config.readConfigFile(filepath.Join(DevPath, "etc/soar.yaml"))
	oldLogOutput := Config.LogOutput
	Config.LogOutput = "soar.log"
	err := GoldenDiff(func() {
		PrintConfiguration()
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	Config.LogOutput = oldLogOutput
	Log.Debug("Exiting function: %s", GetFunctionName())
}
