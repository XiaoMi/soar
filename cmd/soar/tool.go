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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/XiaoMi/soar/advisor"
	"github.com/XiaoMi/soar/ast"
	"github.com/XiaoMi/soar/common"
	"github.com/XiaoMi/soar/database"
	"github.com/XiaoMi/soar/env"
)

// initConfig load config from default->file->cmdFlag
func initConfig() {
	// 更新 binary 文件所在路径为 BaseDir
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	common.BaseDir = filepath.Dir(ex)

	for i, c := range os.Args {
		// 如果指定了 -config, 它必须是第一个参数
		if strings.HasPrefix(c, "-config") && i != 1 {
			fmt.Println("-config must be the first arg")
			os.Exit(1)
		}
		// 等号两边请不要加空格
		if c == "=" {
			// -config = soar.yaml not support
			fmt.Println("wrong format, no space between '=', eg: -config=soar.yaml")
			os.Exit(1)
		}
	}

	// 加载配置文件，处理命令行参数
	err = common.ParseConfig(common.ArgConfig())
	// 检查配置文件及命令行参数是否正确
	if common.CheckConfig && err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	common.LogIfWarn(err, "")
}

// checkConfig for `-check-config` flag
// if error found return non-zero, no error return zero
func checkConfig() int {
	// TestDSN connection check
	testConn := &database.Connector{
		Addr:     common.Config.TestDSN.Addr,
		User:     common.Config.TestDSN.User,
		Pass:     common.Config.TestDSN.Password,
		Database: common.Config.TestDSN.Schema,
		Charset:  common.Config.TestDSN.Charset,
	}
	testVersion, err := testConn.Version()
	if err != nil && !common.Config.TestDSN.Disable {
		fmt.Println("test-dsn:", testConn, err.Error())
		return 1
	}
	if common.Config.Verbose {
		if err == nil {
			fmt.Println("test-dsn", testConn, "Version:", testVersion)
		} else {
			fmt.Println("test-dsn", common.Config.TestDSN)
		}
	}
	// OnlineDSN connection check
	onlineConn := &database.Connector{
		Addr:     common.Config.OnlineDSN.Addr,
		User:     common.Config.OnlineDSN.User,
		Pass:     common.Config.OnlineDSN.Password,
		Database: common.Config.OnlineDSN.Schema,
		Charset:  common.Config.OnlineDSN.Charset,
	}
	onlineVersion, err := onlineConn.Version()
	if err != nil && !common.Config.OnlineDSN.Disable {
		fmt.Println("online-dsn:", onlineConn, err.Error())
		return 1
	}
	if common.Config.Verbose {
		if err == nil {
			fmt.Println("online-dsn", onlineConn, "Version:", onlineVersion)
		} else {
			fmt.Println("online-dsn", common.Config.OnlineDSN)
		}
	}
	return 0
}

// helpTools help tools in cmd flags
func helpTools() {
	// environment error check, eg. MySQL password error
	if common.CheckConfig {
		os.Exit(checkConfig())
	}
	// 打印 SOAR 版本信息
	if common.PrintVersion {
		common.SoarVersion()
		os.Exit(0)
	}
	// 打印已加载配置的各配置项，检查配置是否生效
	if common.PrintConfig {
		common.PrintConfiguration()
		os.Exit(0)
	}
	// 打印支持启发式建议
	if common.Config.ListHeuristicRules {
		advisor.ListHeuristicRules(advisor.HeuristicRules)
		os.Exit(0)
	}
	// 打印支持的 SQL 重写规则
	if common.Config.ListRewriteRules {
		ast.ListRewriteRules(ast.RewriteRules)
		os.Exit(0)
	}
	// 打印所有的测试 SQL
	if common.Config.ListTestSqls {
		advisor.ListTestSQLs()
		os.Exit(0)
	}
	// 打印支持的 report-type
	if common.Config.ListReportTypes {
		common.ListReportTypes()
		os.Exit(0)
	}
}

func shutdown(vEnv *env.VirtualEnv) {
	if common.Config.DropTestTemporary {
		vEnv.CleanUp()
	}
	os.Exit(0)
}
