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

func TestCurrentUser(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	user, host, err := connTest.CurrentUser()
	if err != nil {
		t.Error(err.Error())
	}
	if user != "root" || host != "%" {
		t.Errorf("Want user: root, host: %%. Get user: %s, host: %s", user, host)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestHasSelectPrivilege(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	if !connTest.HasSelectPrivilege() {
		t.Errorf("DSN: %s, User: %s, should has select privilege", connTest.Addr, connTest.User)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestHasAllPrivilege(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	if !connTest.HasAllPrivilege() {
		t.Errorf("DSN: %s, User: %s, should has all privilege", connTest.Addr, connTest.User)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
