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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/XiaoMi/soar/advisor"
	"github.com/XiaoMi/soar/ast"
	"github.com/XiaoMi/soar/common"
	"github.com/XiaoMi/soar/database"
	"github.com/XiaoMi/soar/env"

	"github.com/go-sql-driver/mysql"
	"github.com/kr/pretty"
	"github.com/percona/go-mysql/query"
)

func main() {
	// 全局变量
	var err error
	var sql string                                            // 单条评审指定的 sql 或 explain
	var currentDB string                                      // 当前 SQL 使用的 database
	sqlCounter := 1                                           // SQL 计数器
	lineCounter := 1                                          // 行计数器
	var alterSQLs []string                                    // 待评审的 SQL 中所有 ALTER 请求
	alterTableTimes := make(map[string]int)                   // 待评审的 SQL 中同一经表 ALTER 请求计数器
	suggestMerged := make(map[string]map[string]advisor.Rule) // 优化建议去重, key 为 sql 的 fingerprint.ID
	var suggestStr []string                                   // string 形式格式化之后的优化建议，用于 -report-type json
	tables := make(map[string][]string)                       // SQL 使用的库表名

	// 配置文件&命令行参数解析
	initConfig()

	// 命令行帮助工具，如 -list-report-types, -check-config等。
	if isContinue, exitCode := helpTools(); !isContinue {
		os.Exit(exitCode)
	}

	// 环境初始化，连接检查线上环境+构建测试环境
	vEnv, rEnv := env.BuildEnv()

	// 使用 -cleanup-test-database 命令手动清理残余的 optimizer_xxx 数据库
	if common.Config.CleanupTestDatabase {
		vEnv.CleanupTestDatabase()
		return
	}

	// 如果使用到测试环境，在这里环境清理
	if common.Config.DropTestTemporary {
		defer vEnv.CleanUp()
	}

	// 当程序卡死的时候，或者由于某些原因程序没有退出，可以通过捕获信号量的形式让程序优雅退出并且清理测试环境
	common.HandleSignal(func() {
		shutdown(vEnv, rEnv)
	})

	// 对指定的库表进行索引重复检查
	if common.Config.ReportType == "duplicate-key-checker" {
		dupKeySuggest := advisor.DuplicateKeyChecker(rEnv)
		_, str := advisor.FormatSuggest("", currentDB, common.Config.ReportType, dupKeySuggest)
		if str == "" {
			fmt.Printf("%s/%s 未发现重复索引\n", common.Config.OnlineDSN.Addr, common.Config.OnlineDSN.Schema)
		} else {
			fmt.Println(str)
		}
		return
	}

	// 读入待优化 SQL ，当配置文件或命令行参数未指定 SQL 时从管道读取
	buf := initQuery(common.Config.Query)
	lineCounter += ast.LeftNewLines([]byte(buf))
	buf = strings.TrimSpace(buf)

	// remove bom from file header
	var bom []byte
	buf, bom = common.RemoveBOM([]byte(buf))

	if isContinue, exitCode := reportTool(buf, bom); !isContinue {
		os.Exit(exitCode)
	}

	// 逐条SQL给出优化建议
	for ; ; sqlCounter++ {
		var id string                                     // fingerprint.ID
		heuristicSuggest := make(map[string]advisor.Rule) // 启发式建议
		expSuggest := make(map[string]advisor.Rule)       // EXPLAIN 解读
		idxSuggest := make(map[string]advisor.Rule)       // 索引建议
		proSuggest := make(map[string]advisor.Rule)       // Profiling 信息
		traceSuggest := make(map[string]advisor.Rule)     // Trace 信息
		mysqlSuggest := make(map[string]advisor.Rule)     // MySQL 返回的 ERROR 信息

		if buf == "" {
			common.Log.Debug("Ending, buf: '%s', sql: '%s'", buf, sql)
			break
		}
		// 查询请求切分
		orgSQL, sql, bufBytes := ast.SplitStatement([]byte(buf), []byte(common.Config.Delimiter))
		// lineCounter
		lc := ast.NewLines([]byte(orgSQL))
		// leftLineCounter
		llc := ast.LeftNewLines([]byte(orgSQL))
		lineCounter += llc
		buf = string(bufBytes)

		// 去除无用的备注和空格
		sql = database.RemoveSQLComments(sql)
		if sql == "" {
			common.Log.Debug("empty query or comment, buf: %s", buf)
			continue
		}
		common.Log.Debug("main loop SQL: %s", sql)

		// +++++++++++++++++++++小工具集[开始]+++++++++++++++++++++++{
		fingerprint := strings.TrimSpace(query.Fingerprint(sql))
		// SQL 签名
		id = query.Id(fingerprint)
		currentDB = env.CurrentDB(sql, currentDB)
		switch common.Config.ReportType {
		case "fingerprint":
			// SQL 指纹
			if common.Config.Verbose {
				fmt.Printf("-- ID: %s\n", id)
			}
			fmt.Println(fingerprint)
			continue
		case "pretty":
			// SQL 美化
			fmt.Println(ast.Pretty(sql, "builtin") + common.Config.Delimiter)
			continue
		case "compress":
			// SQL 压缩
			fmt.Println(ast.Compress(sql) + common.Config.Delimiter)
			continue
		case "ast":
			// print vitess AST data struct
			ast.PrintPrettyVitessStmtNode(sql)
			continue
		case "ast-json":
			// print vitess SQL AST into json format
			fmt.Println(ast.VitessStmtNode2JSON(sql))
			continue
		case "tiast":
			// print TiDB AST data struct
			ast.PrintPrettyStmtNode(sql, "", "")
			continue
		case "tiast-json":
			// print TiDB SQL AST into json format
			fmt.Println(ast.StmtNode2JSON(sql, "", ""))
			continue
		case "tokenize":
			// SQL 切词
			_, err = pretty.Println(ast.Tokenize(sql))
			common.LogIfWarn(err, "")
			continue
		default:
			// 建议去重，减少评审整个文件耗时
			// TODO: 由于 a = 11 和 a = '11' 的 fingerprint 相同，这里一旦跳过即无法检查有些建议了，如： ARG.003
			if _, ok := suggestMerged[id]; ok {
				// `use ?` 不可以去重，去重后将导致无法切换数据库
				if !strings.HasPrefix(fingerprint, "use") {
					continue
				}
			}
			// 黑名单中的SQL不给建议
			if advisor.InBlackList(fingerprint) {
				// `use ?` 不可以出现在黑名单中
				if !strings.HasPrefix(fingerprint, "use") {
					continue
				}
			}
		}
		tables[id] = ast.SchemaMetaInfo(sql, currentDB)
		// +++++++++++++++++++++小工具集[结束]+++++++++++++++++++++++}

		// +++++++++++++++++++++语法检查[开始]+++++++++++++++++++++++{
		q, syntaxErr := advisor.NewQuery4Audit(sql)
		stmt := q.Stmt

		// 如果语法检查出错则不需要给优化建议
		if syntaxErr != nil {
			errContent := fmt.Sprintf("At SQL %d : %v", sqlCounter, syntaxErr)
			common.Log.Warning(errContent)
			if common.Config.OnlySyntaxCheck || common.Config.ReportType == "rewrite" ||
				common.Config.ReportType == "query-type" {
				fmt.Println(errContent)
				os.Exit(1)
			}
			// tidb parser 语法检查给出的建议 ERR.000
			mysqlSuggest["ERR.000"] = advisor.RuleMySQLError("ERR.000", syntaxErr)
		}
		// 如果只想检查语法直接跳过后面的步骤
		if common.Config.OnlySyntaxCheck {
			continue
		}
		// +++++++++++++++++++++语法检查[结束]+++++++++++++++++++++++}

		switch common.Config.ReportType {
		case "tables":
			continue
		case "query-type":
			fmt.Println(syntaxErr)
			// query type by first key word
			fmt.Println(ast.QueryType(sql))
			continue
		}

		// +++++++++++++++++++++启发式规则建议[开始]+++++++++++++++++++++++{
		common.Log.Debug("start of heuristic advisor Query: %s", q.Query)
		for item, rule := range advisor.HeuristicRules {
			// 去除忽略的建议检查
			okFunc := (*advisor.Query4Audit).RuleOK
			if !advisor.IsIgnoreRule(item) && &rule.Func != &okFunc {
				r := rule.Func(q)
				if r.Item == item {
					heuristicSuggest[item] = r
				}
			}
		}
		common.Log.Debug("end of heuristic advisor Query: %s", q.Query)
		// +++++++++++++++++++++启发式规则建议[结束]+++++++++++++++++++++++}

		// +++++++++++++++++++++索引优化建议[开始]+++++++++++++++++++++++{
		// 如果配置了索引建议过滤规则，不进行索引优化建议
		// 在配置文件 ignore-rules 中添加 'IDX.*' 即可屏蔽索引优化建议
		common.Log.Debug("start of index advisor Query: %s", q.Query)
		if !advisor.IsIgnoreRule("IDX.") {
			if vEnv.BuildVirtualEnv(rEnv, q.Query) {
				idxAdvisor, err := advisor.NewAdvisor(vEnv, *rEnv, *q)
				if err != nil || (idxAdvisor == nil && vEnv.Error == nil) {
					if idxAdvisor == nil {
						// 如果 SQL 是 DDL 语句，则返回的 idxAdvisor 为 nil，可以忽略不处理
						// TODO alter table add index 语句检查索引是否已经存在
						common.Log.Debug("idxAdvisor by pass Query: %s", q.Query)
					} else {
						common.Log.Warning("advisor.NewAdvisor Error: %v", err)
					}
				} else {
					// 创建环境时没有出现错误，生成索引建议
					if vEnv.Error == nil {
						idxSuggest = idxAdvisor.IndexAdvise().Format()

						// 依赖数据字典的启发式建议
						for i, r := range idxAdvisor.HeuristicCheck(*q) {
							heuristicSuggest[i] = r
						}
					} else {
						// 根据错误号输出建议
						switch vEnv.Error.(*mysql.MySQLError).Number {
						case 1061:
							idxSuggest["IDX.001"] = advisor.Rule{
								Item:     "IDX.001",
								Severity: "L2",
								Summary:  "索引名称已存在",
								Content:  strings.Trim(strings.Split(vEnv.Error.Error(), ":")[1], " "),
								Case:     sql,
							}
						default:
							// vEnv.VEnvBuild 阶段给出的 ERROR 是 ERR.001
							delete(mysqlSuggest, "ERR.000")
							mysqlSuggest["ERR.001"] = advisor.RuleMySQLError("ERR.001", vEnv.Error)
							common.Log.Error("BuildVirtualEnv DDL Execute Error : %v", vEnv.Error)
						}
					}
				}
			} else {
				common.Log.Error("vEnv.BuildVirtualEnv Error: prepare SQL '%s' in vEnv failed.", q.Query)
			}
		}
		common.Log.Debug("end of index advisor Query: %s", q.Query)
		// +++++++++++++++++++++索引优化建议[结束]+++++++++++++++++++++++}

		// +++++++++++++++++++++EXPLAIN 建议[开始]+++++++++++++++++++++++{
		// 如果未配置 Online 或 Test 无法给 Explain 建议
		common.Log.Debug("start of explain Query: %s", q.Query)
		if !common.Config.OnlineDSN.Disable && !common.Config.TestDSN.Disable {
			// 因为 EXPLAIN 依赖数据库环境，所以把这段逻辑放在启发式建议和索引建议后面
			if common.Config.Explain {
				// 执行 EXPLAIN
				explainInfo, err := rEnv.Explain(q.Query,
					database.ExplainType[common.Config.ExplainType],
					database.ExplainFormatType[common.Config.ExplainFormat])
				if err != nil {
					// 线上环境执行失败才到测试环境 EXPLAIN，比如在用户提供建表语句及查询语句的场景
					common.Log.Warn("rEnv.Explain Warn: %v", err)
					explainInfo, err = vEnv.Explain(q.Query,
						database.ExplainType[common.Config.ExplainType],
						database.ExplainFormatType[common.Config.ExplainFormat])
					if err != nil {
						// EXPLAIN 阶段给出的 ERROR 是 ERR.002
						mysqlSuggest["ERR.002"] = advisor.RuleMySQLError("ERR.002", err)
						common.Log.Error("vEnv.Explain Error: %v", err)
					}
				}
				// 分析 EXPLAIN 结果
				if explainInfo != nil {
					expSuggest = advisor.ExplainAdvisor(explainInfo)
				} else {
					common.Log.Warn("rEnv&vEnv.Explain explainInfo nil, SQL: %s", q.Query)
				}
			}
		}
		common.Log.Debug("end of explain Query: %s", q.Query)
		// +++++++++++++++++++++ EXPLAIN 建议[结束]+++++++++++++++++++++++}

		// +++++++++++++++++++++ Profiling [开始]+++++++++++++++++++++++++{
		common.Log.Debug("start of profiling Query: %s", q.Query)
		if common.Config.Profiling {
			res, err := vEnv.Profiling(q.Query)
			if err == nil {
				proSuggest["PRO.001"] = advisor.Rule{
					Item:     "PRO.001",
					Severity: "L0",
					Content:  database.FormatProfiling(res),
				}
			} else {
				common.Log.Error("Profiling Error: %v", err)
			}
		}
		common.Log.Debug("end of profiling Query: %s", q.Query)
		// +++++++++++++++++++++ Profiling [结束]++++++++++++++++++++++++++}

		// +++++++++++++++++++++ Trace [开始]+++++++++++++++++++++++++{
		common.Log.Debug("start of trace Query: %s", q.Query)
		if common.Config.Trace {
			res, err := vEnv.Trace(q.Query)
			if err == nil {
				traceSuggest["TRA.001"] = advisor.Rule{
					Item:     "TRA.001",
					Severity: "L0",
					Content:  database.FormatTrace(res),
				}
			} else {
				common.Log.Error("Trace Error: %v", err)
			}
		}
		common.Log.Debug("end of trace Query: %s", q.Query)
		// +++++++++++++++++++++Trace [结束]++++++++++++++++++++++++++}

		// +++++++++++++++++++++SQL 重写[开始]+++++++++++++++++++++++++{
		common.Log.Debug("start of rewrite Query: %s", q.Query)
		if common.Config.ReportType == "rewrite" {
			if strings.HasPrefix(strings.TrimSpace(strings.ToLower(sql)), "create") ||
				strings.HasPrefix(strings.TrimSpace(strings.ToLower(sql)), "alter") ||
				strings.HasPrefix(strings.TrimSpace(strings.ToLower(sql)), "rename") {
				// 依赖上下文件的 SQL 重写，如：多条 ALTER SQL 合并
				// vitess 对 DDL 语法的支持不好，大部分 DDL 会语法解析出错，但即使出错了还是会生成一个 stmt 而且里面的 db.table 还是准确的。

				alterSQLs = append(alterSQLs, sql)
				alterTbl := ast.AlterAffectTable(stmt)
				if alterTbl != "" && alterTbl != "dual" {
					if _, ok := alterTableTimes[alterTbl]; ok {
						heuristicSuggest["ALT.002"] = advisor.HeuristicRules["ALT.002"]
						alterTableTimes[alterTbl] = alterTableTimes[alterTbl] + 1
					} else {
						alterTableTimes[alterTbl] = 1
					}
				}
			} else {
				// 其他不依赖上下文件的 SQL 重写
				rw := ast.NewRewrite(sql)
				if rw == nil {
					// 都到这一步了 sql 不会语法不正确，因此 rw 一般不会为 nil
					common.Log.Critical("NewRewrite nil point error, SQL: %s", sql)
					os.Exit(1)
				}
				// SQL 转写需要的源信息采集，如果没有配置环境则只做有限改写
				meta := ast.GetMeta(rw.Stmt, nil)
				rw.Columns = vEnv.GenTableColumns(meta)
				// 执行定义好的 SQL 重写规则
				rw.Rewrite()
				fmt.Println(strings.TrimSpace(rw.NewSQL))
			}
		}
		common.Log.Debug("end of rewrite Query: %s", q.Query)
		// +++++++++++++++++++++ SQL 重写[结束]++++++++++++++++++++++++++}

		// +++++++++++++++++++++打印单条 SQL 优化建议[开始]++++++++++++++++++++++++++{
		common.Log.Debug("start of print suggestions, Query: %s", q.Query)
		if strings.HasPrefix(fingerprint, "use") {
			continue
		}
		sug, str := advisor.FormatSuggest(q.Query, currentDB, common.Config.ReportType, heuristicSuggest, idxSuggest, expSuggest, proSuggest, traceSuggest, mysqlSuggest)
		suggestMerged[id] = sug
		switch common.Config.ReportType {
		case "json":
			suggestStr = append(suggestStr, str)
		case "tables":
		case "duplicate-key-checker":
		case "rewrite":
		case "lint":
			for _, s := range strings.Split(str, "\n") {
				// ignore empty output
				if strings.TrimSpace(s) == "" {
					continue
				}

				if common.Config.Query != "" {
					if _, err = os.Stat(common.Config.Query); err == nil {
						fmt.Printf("%s:%d:%s\n", common.Config.Query, lineCounter, s)
					} else {
						fmt.Printf("null:%d:%s\n", lineCounter, s)
					}
				} else {
					fmt.Printf("stdin:%d:%s\n", lineCounter, s)
				}
			}
			lineCounter += lc - llc
		case "html":
			fmt.Println(common.Markdown2HTML(str))
		default:
			fmt.Println(str)
		}
		common.Log.Debug("end of print suggestions, Query: %s", q.Query)
		// +++++++++++++++++++++打印单条 SQL 优化建议[结束]++++++++++++++++++++++++++}
	}

	// 同一张表的多条 ALTER 语句合并为一条
	if ast.RewriteRuleMatch("mergealter") {
		for _, v := range ast.MergeAlterTables(alterSQLs...) {
			fmt.Println(strings.TrimSpace(v))
		}
		return
	}

	// 以 JSON 格式化输出
	if common.Config.ReportType == "json" {
		fmt.Println("[\n", strings.Join(suggestStr, ",\n"), "\n]")
	}

	// 以 JSON 格式输出 SQL 影响的库表名
	if common.Config.ReportType == "tables" {
		js, err := json.MarshalIndent(tables, "", "  ")
		if err == nil {
			fmt.Println(string(js))
		} else {
			common.Log.Error("FormatSuggest json.Marshal Error: %v", err)
		}
		return
	}

	verboseInfo()
}
