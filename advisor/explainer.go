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
	"fmt"
	"strings"

	"github.com/XiaoMi/soar/common"
	"github.com/XiaoMi/soar/database"
)

var explainRuleID int

// [EXP.XXX]Rule
var explainRules map[string]Rule

// [table_name]"suggest text"
var tablesSuggests map[string][]string

// explain建议的形式
// Item: EXP.XXX
// Severity: L[0-8]
// Summary: full table scan, not use index, full index scan...
// Content: XX TABLE xxx

// checkExplainSelectType
func checkExplainSelectType(exp *database.ExplainInfo) {
	// 判断是否跳过不检查
	if len(common.Config.ExplainWarnSelectType) == 1 {
		if common.Config.ExplainWarnSelectType[0] == "" {
			return
		}
	} else if len(common.Config.ExplainWarnSelectType) < 1 {
		return
	}

	if exp.ExplainFormat == database.JSONFormatExplain {
		// TODO
		// JSON 形式遍历分析不方便，转成 Row 格式也没有 SelectType 暂不处理
		return
	}
	for _, v := range common.Config.ExplainWarnSelectType {
		for _, row := range exp.ExplainRows {
			if row.SelectType == v && v != "" {
				tablesSuggests[row.TableName] = append(tablesSuggests[row.TableName], fmt.Sprintf("SelectType:%s", row.SelectType))
			}
		}
	}
}

// checkExplainAccessType 用户可以设置AccessType的建议级别，匹配到的查询会给出建议
func checkExplainAccessType(exp *database.ExplainInfo) {
	// 判断是否跳过不检查
	if len(common.Config.ExplainWarnAccessType) == 1 {
		if common.Config.ExplainWarnAccessType[0] == "" {
			return
		}
	} else if len(common.Config.ExplainWarnAccessType) < 1 {
		return
	}

	rows := exp.ExplainRows
	if exp.ExplainFormat == database.JSONFormatExplain {
		// JSON形式遍历分析不方便，转成Row格式统一处理
		rows = database.ConvertExplainJSON2Row(exp.ExplainJSON)
	}
	for _, v := range common.Config.ExplainWarnAccessType {
		for _, row := range rows {
			if row.AccessType == v && v != "" {
				tablesSuggests[row.TableName] = append(tablesSuggests[row.TableName], fmt.Sprintf("Scalability:%s", row.Scalability))
			}
		}
	}
}

/*
// TODO:
func checkExplainPossibleKeys(exp *database.ExplainInfo) {
}

func checkExplainKeyLen(exp *database.ExplainInfo) {
}

func checkExplainKey(exp *database.ExplainInfo) {
	// 小于最小使用试用key数量
	//return intval($explainResult) < intval($userCond);
	//explain-min-keys int
}

func checkExplainExtra(exp *database.ExplainInfo) {
	// 包含用户配置的逗号分隔关键词之一则提醒
	// return self::contains($explainResult, $userCond);
	// explain-warn-extra []string
}
*/

// checkExplainRef ...
func checkExplainRef(exp *database.ExplainInfo) {
	rows := exp.ExplainRows
	if exp.ExplainFormat == database.JSONFormatExplain {
		// JSON形式遍历分析不方便，转成Row格式统一处理
		rows = database.ConvertExplainJSON2Row(exp.ExplainJSON)
	}
	for i, row := range rows {
		if strings.Join(row.Ref, "") == "NULL" || strings.Join(row.Ref, "") == "" {
			if i == 0 && len(rows) > 1 {
				continue
			}
			tablesSuggests[row.TableName] = append(tablesSuggests[row.TableName], fmt.Sprintf("Ref:null"))
		}
	}
}

// checkExplainRows ...
func checkExplainRows(exp *database.ExplainInfo) {
	// 判断是否跳过不检查
	if common.Config.ExplainMaxRows <= 0 {
		return
	}

	rows := exp.ExplainRows
	if exp.ExplainFormat == database.JSONFormatExplain {
		// JSON形式遍历分析不方便，转成Row格式统一处理
		rows = database.ConvertExplainJSON2Row(exp.ExplainJSON)
	}

	for _, row := range rows {
		if row.Rows >= common.Config.ExplainMaxRows {
			tablesSuggests[row.TableName] = append(tablesSuggests[row.TableName], fmt.Sprintf("Rows:%d", row.Rows))
		}
	}
}

// checkExplainFiltered ...
func checkExplainFiltered(exp *database.ExplainInfo) {
	// 判断是否跳过不检查
	if common.Config.ExplainMaxFiltered <= 0.001 {
		return
	}

	rows := exp.ExplainRows
	if exp.ExplainFormat == database.JSONFormatExplain {
		// JSON形式遍历分析不方便，转成Row格式统一处理
		rows = database.ConvertExplainJSON2Row(exp.ExplainJSON)
	}
	for i, row := range rows {
		if i == 0 && len(rows) > 1 {
			continue
		}
		if row.Filtered >= common.Config.ExplainMaxFiltered {
			tablesSuggests[row.TableName] = append(tablesSuggests[row.TableName], fmt.Sprintf("Filtered:%.2f%s", row.Filtered, "%"))
		}
	}
}

// ExplainAdvisor 基于explain信息给出建议
func ExplainAdvisor(exp *database.ExplainInfo) map[string]Rule {
	common.Log.Debug("ExplainAdvisor SQL: %v", exp.SQL)
	explainRuleID = 0
	explainRules = make(map[string]Rule)
	tablesSuggests = make(map[string][]string)

	checkExplainSelectType(exp)
	checkExplainAccessType(exp)
	checkExplainFiltered(exp)
	checkExplainRef(exp)
	checkExplainRows(exp)

	// 打印explain table
	content := database.PrintMarkdownExplainTable(exp)

	if common.Config.ShowWarnings {
		content += "\n" + database.MySQLExplainWarnings(exp)
	}

	// 对explain table中各项难于理解的值做解释
	cases := database.ExplainInfoTranslator(exp)

	// 添加last_query_cost
	if common.Config.ShowLastQueryCost {
		content += "\n" + database.MySQLExplainQueryCost(exp)
	}

	if content != "" {
		explainRules["EXP.000"] = Rule{
			Item:     "EXP.000",
			Severity: "L0",
			Summary:  "Explain信息",
			Content:  content,
			Case:     cases,
			Func:     (*Query4Audit).RuleOK,
		}
	}
	// TODO: 检查explain对应的表是否需要跳过，如dual,空表等
	return explainRules
}

// DigestExplainText 分析用户输入的EXPLAIN信息
func DigestExplainText(text string) {
	// explain信息就不要显示完美了，美不美自己看吧。
	common.Config.IgnoreRules = append(common.Config.IgnoreRules, "OK")

	if !IsIgnoreRule("EXP.") {
		explainInfo, err := database.ParseExplainText(text)
		if err != nil {
			common.Log.Error("main ParseExplainText Error: %v", err)
			return
		}
		expSuggest := ExplainAdvisor(explainInfo)
		_, output := FormatSuggest("", "", common.Config.ReportType, expSuggest)
		if common.Config.ReportType == "html" {
			fmt.Println(common.MarkdownHTMLHeader())
			fmt.Println(common.Markdown2HTML(output))
		} else {
			fmt.Println(output)
		}
	}
}
