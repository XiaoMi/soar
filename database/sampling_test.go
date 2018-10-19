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
	"testing"

	"github.com/XiaoMi/soar/common"
)

func init() {
	common.BaseDir = common.DevPath
}

func TestSamplingData(t *testing.T) {
	online := &Connector{
		Addr:     common.Config.OnlineDSN.Addr,
		User:     common.Config.OnlineDSN.User,
		Pass:     common.Config.OnlineDSN.Password,
		Database: common.Config.OnlineDSN.Schema,
	}

	offline := &Connector{
		Addr:     common.Config.TestDSN.Addr,
		User:     common.Config.TestDSN.User,
		Pass:     common.Config.TestDSN.Password,
		Database: common.Config.TestDSN.Schema,
	}

	offline.Database = "test"

	err := connTest.SamplingData(*online, "film")
	if err != nil {
		t.Error(err)
	}
}
