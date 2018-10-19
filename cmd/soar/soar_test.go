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
	"testing"

	"github.com/XiaoMi/soar/common"
)

func init() {
	common.Config.OnlineDSN.Schema = "sakila"
}

func Test_Main(_ *testing.T) {
	common.Config.OnlineDSN.Disable = true
	common.Config.LogLevel = 0
	common.Config.Query = "select * from film;alter table city add index idx_country_id(country_id);"
	main()
}

func Test_Main_More(_ *testing.T) {
	common.Config.LogLevel = 0
	common.Config.Profiling = true
	common.Config.Explain = true
	common.Config.Query = "select * from film where country_id = 1;use sakila;alter table city add index idx_country_id(country_id);"
	for _, typ := range []string{
		"json", "html", "markdown", "fingerprint", "compress", "pretty", "rewrite",
	} {
		common.Config.ReportType = typ
		main()
	}
}
