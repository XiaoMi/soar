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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/XiaoMi/soar/advisor"
	"github.com/XiaoMi/soar/ast"
	"github.com/XiaoMi/soar/common"
	"github.com/XiaoMi/soar/database"
	"github.com/XiaoMi/soar/env"

	"github.com/kr/pretty"
	"github.com/percona/go-mysql/query"
	"github.com/ziutek/mymysql/mysql"
	"vitess.io/vitess/go/vt/sqlparser"
)

func main() {
	// 全局变量
	var sql string                                            // 单条评审指定的sql或explain
	sqlCounter := 1                                           // SQL 计数器
	lineCounter := 1                                          // 行计数器
	var alterSqls []string                                    // 待评审的SQL中所有ALTER请求
	alterTableTimes := make(map[string]int)                   // 待评审的SQL中同一经表ALTER请求计数器
	suggestMerged := make(map[string]map[string]advisor.Rule) // 优化建议去重,key为sql的fingerprint.ID

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	common.BaseDir = filepath.Dir(ex) // binary文件所在路径

	// 配置文件&命令行参数解析
	err = common.ParseConfig("")
	common.LogIfWarn(err, "")

	// 打印支持启发式建议
	if common.Config.ListHeuristicRules {
		// 只打印支持的优化建议
		advisor.ListHeuristicRules(advisor.HeuristicRules)
		return
	}
	// 打印支持的SQL重写规则
	if common.Config.ListRewriteRules {
		ast.ListRewriteRules(ast.RewriteRules)
		return
	}
	// 打印所有的测试SQL
	if common.Config.ListTestSqls {
		advisor.ListTestSQLs()
		return
	}
	// 打印支持的report-type
	if common.Config.ListReportTypes {
		common.ListReportTypes()
		return
	}

	// 环境初始化，连接检查线上环境+构建测试环境
	vEnv, rEnv := env.BuildEnv()

	// 如果使用到测试环境，在这里环境清理
	if common.Config.DropTestTemporary {
		defer vEnv.CleanUp()
	}

	// 对指定的库表进行索引重复检查
	if common.Config.ReportType == "duplicate-key-checker" {
		dupKeySuggest := advisor.DuplicateKeyChecker(rEnv)
		_, str := advisor.FormatSuggest("", common.Config.ReportType, dupKeySuggest)
		if str == "" {
			fmt.Printf("%s/%s 未发现重复索引\n", common.Config.OnlineDSN.Addr, common.Config.OnlineDSN.Schema)
		} else {
			fmt.Println(str)
		}
		return
	}

	// 读入待优化SQL，当配置文件或命令行参数未指定SQL时从管道读取
	if common.Config.Query == "" {
		var data []byte
		data, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			common.Log.Critical("ioutil.ReadAll Error: %v", err)
		}
		lineCounter += ast.LeftNewLines(data)
		sql = strings.TrimSpace(string(data))
	} else {
		if _, err = os.Stat(common.Config.Query); err == nil {
			var data []byte
			data, err = ioutil.ReadFile(common.Config.Query)
			if err != nil {
				common.Log.Critical("ioutil.ReadFile Error: %v", err)
			}
			lineCounter += ast.LeftNewLines(data)
			sql = strings.TrimSpace(string(data))
		} else {
			lineCounter += ast.LeftNewLines([]byte(common.Config.Query))
			sql = strings.TrimSpace(common.Config.Query)
		}
	}

	switch common.Config.ReportType {
	case "html":
		// HTML格式输入CSS加载
		fmt.Println(common.MarkdownHTMLHeader())
	case "md2html":
		// markdown2html 转换小工具
		fmt.Println(common.MarkdownHTMLHeader())
		fmt.Println(common.Markdown2HTML(sql))
		return
	case "explain-digest":
		// 当用户输入为EXPLAIN信息，只对Explain信息进行分析
		// 注意： 这里只能处理一条SQL的EXPLAIN信息，用户一次反馈多条SQL的EXPLAIN信息无法处理
		advisor.DigestExplainText(sql)
		return
	case "remove-comment":
		fmt.Println(string(database.RemoveSQLComments([]byte(sql))))
		return
	}

	// 逐条SQL给出优化建议
	lineCounter += ast.LeftNewLines([]byte(sql))
	buf := strings.TrimSpace(sql)
	for ; ; sqlCounter++ {
		var id string                                     // fingerprint.ID
		heuristicSuggest := make(map[string]advisor.Rule) // 启发式建议
		expSuggest := make(map[string]advisor.Rule)       // EXPLAIN解读
		idxSuggest := make(map[string]advisor.Rule)       // 索引建议
		proSuggest := make(map[string]advisor.Rule)       // Profiling信息
		traceSuggest := make(map[string]advisor.Rule)     // Trace信息
		mysqlSuggest := make(map[string]advisor.Rule)     // MySQL返回的ERROR信息

		if buf == "" {
			break
		}
		// 查询请求切分
		sql, bufBytes := ast.SplitStatement([]byte(buf), []byte(common.Config.Delimiter))
		// lineCounter
		lc := ast.NewLines([]byte(sql))
		// leftLineCounter
		llc := ast.LeftNewLines([]byte(sql))
		lineCounter += llc
		buf = string(bufBytes)

		// 去除无用的备注和空格
		sql = strings.TrimSpace(sql)
		sql = string(database.RemoveSQLComments([]byte(sql)))

		common.Log.Debug("main loop SQL: %s", sql)

		// +++++++++++++++++++++小工具集[开始]+++++++++++++++++++++++{
		fingerprint := strings.TrimSpace(query.Fingerprint(sql))
		switch common.Config.ReportType {
		case "fingerprint":
			// SQL指纹
			fmt.Println(fingerprint)
			continue
		case "pretty":
			// SQL美化
			fmt.Println(ast.Pretty(sql, "builtin") + common.Config.Delimiter)
			continue
		case "compress":
			// SQL压缩
			fmt.Println(ast.Compress(sql) + common.Config.Delimiter)
			continue
		case "ast":
			// SQL 抽象语法树
			var tree sqlparser.Statement
			tree, err = sqlparser.Parse(sql)
			if err != nil {
				fmt.Println(err)
			} else {
				_, err = pretty.Println(tree)
				common.LogIfWarn(err, "")
			}
			continue
		case "tiast":
			// TiDB SQL 抽象语法树
			ast.PrintPrettyStmtNode(sql, "", "")
			continue
		case "tokenize":
			// SQL 切词
			_, err = pretty.Println(ast.Tokenize(sql))
			common.LogIfWarn(err, "")
			continue
		default:
			// SQL签名
			id = query.Id(fingerprint)
			// 建议去重，减少评审整个文件耗时
			// TODO: 由于 a = 11和a = '11'的fingerprint相同，这里一旦跳过即无法检查有些建议了，如：ARG.003
			if _, ok := suggestMerged[id]; ok {
				continue
			}
			// 黑名单中的SQL不给建议
			if advisor.InBlackList(fingerprint) {
				continue
			}
		}
		// +++++++++++++++++++++小工具集[结束]+++++++++++++++++++++++}

		// +++++++++++++++++++++语法检查[开始]+++++++++++++++++++++++{
		q, syntaxErr := advisor.NewQuery4Audit(sql)
		stmt := q.Stmt

		switch stmt.(type) {
		case *sqlparser.DDL:
			// 因为vitess的parser对于DDL语法支持不好，通过在测试环境执行辅助进行语法检查
			if common.Config.OnlySyntaxCheck && vEnv.BuildVirtualEnv(rEnv, sql) {
				syntaxErr = vEnv.Error
			}
		}

		// 如果语法检查出错则不需要给优化建议
		if syntaxErr != nil {
			errContent := fmt.Sprintf("At SQL %d : %v", sqlCounter, syntaxErr)
			common.Log.Warning(errContent)
			if common.Config.OnlySyntaxCheck {
				if !strings.Contains(errContent, `doesn't exist`) {
					fmt.Println(errContent)
				}
			}
			if !common.Config.DryRun {
				os.Exit(1)
			}
			// vitess 语法检查给出的建议ERR.000
			if common.Config.TestDSN.Disable {
				mysqlSuggest["ERR.000"] = advisor.RuleMySQLError("ERR.000", syntaxErr)
			}
		}
		// 如果只想检查语法直接跳过后面的步骤
		if common.Config.OnlySyntaxCheck {
			continue
		}

		// +++++++++++++++++++++语法检查[结束]+++++++++++++++++++++++}

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
		// 在配置文件ignore-rules中添加 'IDX.*'即可屏蔽索引优化建议
		common.Log.Debug("start of index advisor Query: %s", q.Query)
		if !advisor.IsIgnoreRule("IDX.") {
			if vEnv.BuildVirtualEnv(rEnv, q.Query) {
				idxAdvisor, err := advisor.NewAdvisor(vEnv, *rEnv, *q)
				if err != nil || (idxAdvisor == nil && vEnv.Error == nil) {
					if idxAdvisor == nil {
						// 如果SQL是DDL语句，则返回的idxAdvisor为nil，可以忽略不处理
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
						switch vEnv.Error.(*mysql.Error).Code {
						case 1061:
							idxSuggest["IDX.001"] = advisor.Rule{
								Item:     "IDX.001",
								Severity: "L2",
								Summary:  "索引名称已存在",
								Content:  strings.Trim(strings.Split(vEnv.Error.Error(), ":")[1], " "),
								Case:     sql,
							}
						default:
							// vEnv.VEnvBuild阶段给出的ERROR是ERR.001
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

		// +++++++++++++++++++++EXPLAIN建议[开始]+++++++++++++++++++++++{
		// 如果未配置Online或Test无法给Explain建议
		common.Log.Debug("start of explain Query: %s", q.Query)
		if !common.Config.OnlineDSN.Disable && !common.Config.TestDSN.Disable {
			// 因为EXPLAIN依赖数据库环境，所以把这段逻辑放在启发式建议和索引建议后面
			if common.Config.Explain {
				// 执行EXPLAIN
				explainInfo, err := rEnv.Explain(q.Query,
					database.ExplainType[common.Config.ExplainType],
					database.ExplainFormatType[common.Config.ExplainFormat])
				if err != nil {
					// 线上环境执行失败才到测试环境EXPLAIN，比如在用户提供建表语句及查询语句的场景
					common.Log.Warn("rEnv.Explain Warn: %v", err)
					explainInfo, err = vEnv.Explain(q.Query,
						database.ExplainType[common.Config.ExplainType],
						database.ExplainFormatType[common.Config.ExplainFormat])
					if err != nil {
						// EXPLAIN阶段给出的ERROR是ERR.002
						mysqlSuggest["ERR.002"] = advisor.RuleMySQLError("ERR.002", err)
						common.Log.Error("vEnv.Explain Error: %v", err)
						continue
					}
				}
				// 分析EXPLAIN结果
				if explainInfo != nil {
					expSuggest = advisor.ExplainAdvisor(explainInfo)
				} else {
					common.Log.Warn("rEnv&vEnv.Explain explainInfo nil, SQL: %s", q.Query)
				}
			}
		}
		common.Log.Debug("end of explain Query: %s", q.Query)
		// +++++++++++++++++++++EXPLAIN建议[结束]+++++++++++++++++++++++}

		// +++++++++++++++++++++Profiling[开始]+++++++++++++++++++++++++{
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
		// +++++++++++++++++++++Profiling[结束]++++++++++++++++++++++++++}

		// +++++++++++++++++++++Trace [开始]+++++++++++++++++++++++++{
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

		// +++++++++++++++++++++SQL重写[开始]+++++++++++++++++++++++++{
		common.Log.Debug("start of rewrite Query: %s", q.Query)
		if common.Config.ReportType == "rewrite" {
			if strings.HasPrefix(strings.TrimSpace(strings.ToLower(sql)), "create") ||
				strings.HasPrefix(strings.TrimSpace(strings.ToLower(sql)), "alter") ||
				strings.HasPrefix(strings.TrimSpace(strings.ToLower(sql)), "rename") {
				// 依赖上下文件的SQL重写，如：多条ALTER SQL合并
				// vitess对DDL语法的支持不好，大部分DDL会语法解析出错，但即使出错了还是会生成一个stmt而且里面的db.table还是准确的。

				alterSqls = append(alterSqls, sql)
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
				// 其他不依赖上下文件的SQL重写
				rw := ast.NewRewrite(sql)
				if rw == nil {
					// 都到这一步了sql不会语法不正确，因此rw一般不会为nil
					common.Log.Critical("NewRewrite nil point error, SQL: %s", sql)
					os.Exit(1)
				}
				// SQL转写需要的源信息采集，如果没有配置环境则只做有限改写
				meta := ast.GetMeta(rw.Stmt, nil)
				rw.Columns = vEnv.GenTableColumns(meta)
				// 执行定义好的SQL重写规则
				rw.Rewrite()
				fmt.Println(strings.TrimSpace(rw.NewSQL))
			}
		}
		common.Log.Debug("end of rewrite Query: %s", q.Query)
		// +++++++++++++++++++++SQL重写[结束]++++++++++++++++++++++++++}

		// +++++++++++++++++++++打印单条SQL优化建议[开始]++++++++++++++++++++++++++{
		common.Log.Debug("start of print suggestions, Query: %s", q.Query)
		sug, str := advisor.FormatSuggest(q.Query, common.Config.ReportType, heuristicSuggest, idxSuggest, expSuggest, proSuggest, traceSuggest, mysqlSuggest)
		suggestMerged[id] = sug
		switch common.Config.ReportType {
		case "json":
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
		// +++++++++++++++++++++打印单条SQL优化建议[结束]++++++++++++++++++++++++++}
	}

	// 同一张表的多条ALTER语句合并为一条
	if ast.RewriteRuleMatch("mergealter") {
		for _, v := range ast.MergeAlterTables(alterSqls...) {
			fmt.Println(strings.TrimSpace(v))
		}
		return
	}

	// 以JSON格式化输出
	if common.Config.ReportType == "json" {
		js, err := json.MarshalIndent(suggestMerged, "", "  ")
		if err == nil {
			fmt.Println(string(js))
		} else {
			common.Log.Error("FormatSuggest json.Marshal Error: %v", err)
		}
		return
	}
}
