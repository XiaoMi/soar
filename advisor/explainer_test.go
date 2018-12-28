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
	"testing"

	"github.com/XiaoMi/soar/common"
)

func TestDigestExplainText(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	var text = `+----+-------------+---------+-------+---------------------------------------------------------+-------------------+---------+---------------------------+------+-------------+
| id | select_type | table   | type  | possible_keys                                           | key               | key_len | ref                       | rows | Extra       |
+----+-------------+---------+-------+---------------------------------------------------------+-------------------+---------+---------------------------+------+-------------+
|  1 | SIMPLE      | country | index | PRIMARY,country_id                                      | country           | 152     | NULL                      |  109 | Using index |
|  1 | SIMPLE      | city    | ref   | idx_fk_country_id,idx_country_id_city,idx_all,idx_other | idx_fk_country_id | 2       | sakila.country.country_id |    2 | Using index |
+----+-------------+---------+-------+---------------------------------------------------------+-------------------+---------+---------------------------+------+-------------+`
	common.Config.ReportType = "explain-digest"
	err := common.GoldenDiff(func() {
		DigestExplainText(text)
		orgReportType := common.Config.ReportType
		common.Config.ReportType = "html"
		DigestExplainText(text)
		common.Config.ReportType = orgReportType
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
