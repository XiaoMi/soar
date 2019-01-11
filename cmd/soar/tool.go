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
	"io/ioutil"
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
	connTest, err := database.NewConnector(common.Config.TestDSN)
	if err != nil {
		fmt.Println("test-dsn:", common.Config.TestDSN.Addr, err.Error())
		return 1
	}
	testVersion, err := connTest.Version()
	if err != nil && !common.Config.TestDSN.Disable {
		fmt.Println("test-dsn:", connTest, err.Error())
		return 1
	}
	if common.Config.Verbose {
		if err == nil {
			fmt.Println("test-dsn", connTest, "Version:", testVersion)
		} else {
			fmt.Println("test-dsn", common.Config.TestDSN)
		}
	}

	if !connTest.HasAllPrivilege() {
		fmt.Printf("test-dsn: %s, need all privileges", common.FormatDSN(common.Config.TestDSN))
		return 1
	}
	// OnlineDSN connection check
	connOnline, err := database.NewConnector(common.Config.OnlineDSN)
	if err != nil {
		fmt.Println("test-dsn:", common.Config.OnlineDSN.Addr, err.Error())
		return 1
	}
	onlineVersion, err := connOnline.Version()
	if err != nil && !common.Config.OnlineDSN.Disable {
		fmt.Println("online-dsn:", connOnline, err.Error())
		return 1
	}
	if common.Config.Verbose {
		if err == nil {
			fmt.Println("online-dsn", connOnline, "Version:", onlineVersion)
		} else {
			fmt.Println("online-dsn", common.Config.OnlineDSN)
		}
	}

	if !connOnline.HasSelectPrivilege() {
		fmt.Printf("online-dsn: %s, need all privileges", common.FormatDSN(common.Config.OnlineDSN))
		return 1
	}
	return 0
}

// helpTools help tools in cmd flags
func helpTools() (isContinue bool, exitCode int) {
	// environment error check, eg. MySQL password error
	if common.CheckConfig {
		return false, checkConfig()
	}
	// 打印 SOAR 版本信息
	if common.PrintVersion {
		common.SoarVersion()
		return false, 0
	}
	// 打印已加载配置的各配置项，检查配置是否生效
	if common.PrintConfig {
		common.PrintConfiguration()
		return false, 0
	}
	// 打印支持启发式建议
	if common.Config.ListHeuristicRules {
		advisor.ListHeuristicRules(advisor.HeuristicRules)
		return false, 0
	}
	// 打印支持的 SQL 重写规则
	if common.Config.ListRewriteRules {
		ast.ListRewriteRules(ast.RewriteRules)
		return false, 0
	}
	// 打印所有的测试 SQL
	if common.Config.ListTestSqls {
		advisor.ListTestSQLs()
		return false, 0
	}
	// 打印支持的 report-type
	if common.Config.ListReportTypes {
		common.ListReportTypes()
		return false, 0
	}

	return true, 0
}

// reportTool tools in report type
func reportTool(sql string, bom []byte) (isContinue bool, exitCode int) {
	switch common.Config.ReportType {
	case "html":
		// HTML 格式输入 CSS 加载
		fmt.Println(common.MarkdownHTMLHeader())
		return true, 0
	case "md2html":
		// markdown2html 转换小工具
		fmt.Println(common.MarkdownHTMLHeader())
		fmt.Println(common.Markdown2HTML(sql))
		return false, 0
	case "explain-digest":
		// 当用户输入为 EXPLAIN 信息，只对 Explain 信息进行分析
		// 注意： 这里只能处理一条 SQL 的 EXPLAIN 信息，用户一次反馈多条 SQL 的 EXPLAIN 信息无法处理
		advisor.DigestExplainText(sql)
		return false, 0
	case "chardet":
		// Get charset of input
		charset := common.CheckCharsetByBOM(bom)
		if charset == "" {
			charset = common.Chardet([]byte(sql))
		}
		fmt.Println(charset)
		return false, 0
	case "remove-comment":
		fmt.Println(database.RemoveSQLComments(sql))
		return false, 0
	default:
		return true, 0
	}
}

// initQuery
func initQuery(query string) string {
	// 读入待优化 SQL ，当配置文件或命令行参数未指定 SQL 时从管道读取
	if query == "" {
		// check stdin is pipe or terminal
		// https://stackoverflow.com/questions/22744443/check-if-there-is-something-to-read-on-stdin-in-golang
		stat, err := os.Stdin.Stat()
		if stat == nil {
			common.Log.Critical("os.Stdin.Stat Error: %v", err)
			os.Exit(1)
		}
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			// stdin is from a terminal
			fmt.Println("Args format error, use --help see how to use it!")
			os.Exit(1)
		}
		// read from pipe
		var data []byte
		data, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			common.Log.Critical("ioutil.ReadAll Error: %v", err)
		}
		common.Log.Debug("initQuery get query from os.Stdin")
		return string(data)
	}

	if _, err := os.Stat(query); err == nil {
		var data []byte
		data, err = ioutil.ReadFile(query)
		if err != nil {
			common.Log.Critical("ioutil.ReadFile Error: %v", err)
		}
		common.Log.Debug("initQuery get query from file: %s", query)
		return string(data)
	}

	return query
}

func shutdown(vEnv *env.VirtualEnv, rEnv *database.Connector) {
	if common.Config.DropTestTemporary {
		vEnv.CleanUp()
	}
	err := vEnv.Conn.Close()
	common.LogIfWarn(err, "")
	err = rEnv.Conn.Close()
	common.LogIfWarn(err, "")
	os.Exit(0)
}

func verboseInfo() {
	if !common.Config.Verbose {
		return
	}
	// syntax check verbose mode, add output for success!
	if common.Config.OnlySyntaxCheck {
		fmt.Println("Syntax check OK!")
		return
	}
	switch common.Config.ReportType {
	case "markdown":
		if common.Config.TestDSN.Disable || common.Config.OnlineDSN.Disable {
			fmt.Println("MySQL environment verbose info")
			// TestDSN
			if common.Config.TestDSN.Disable {
				fmt.Println("* test-dsn:", common.Config.TestDSN.Addr, "is disable, please check log.")
			}
			// OnlineDSN
			if common.Config.OnlineDSN.Disable {
				fmt.Println("* online-dsn:", common.Config.OnlineDSN.Addr, "is disable, please check log.")
			}
		}
	}
}
