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
	"errors"
	"strings"

	"github.com/XiaoMi/soar/common"
)

// CurrentUser get current user with current_user() function
func (db *Connector) CurrentUser() (string, string, error) {
	res, err := db.Query("select current_user()")
	if err != nil {
		return "", "", err
	}
	if len(res.Rows) > 0 {
		cols := strings.Split(res.Rows[0].Str(0), "@")
		if len(cols) == 2 {
			user := strings.Trim(cols[0], "'")
			host := strings.Trim(cols[1], "'")
			if strings.Contains(user, "'") || strings.Contains(host, "'") {
				return "", "", errors.New("user or host contains irregular character")
			}
			return user, host, nil
		}
		return "", "", errors.New("user or host contains irregular character")
	}
	return "", "", errors.New("no privilege info")
}

// HasSelectPrivilege if user has select privilege
func (db *Connector) HasSelectPrivilege() bool {
	user, host, err := db.CurrentUser()
	if err != nil {
		common.Log.Error("User: %s, HasSelectPrivilege: %s", db.User, err.Error())
		return false
	}
	res, err := db.Query("select Select_priv from mysql.user where user='%s' and host='%s'", user, host)
	if err != nil {
		common.Log.Error("HasSelectPrivilege, DSN: %s, Error: %s", db.Addr, err.Error())
		return false
	}
	// Select_priv
	if len(res.Rows) > 0 {
		if res.Rows[0].Str(0) == "Y" {
			return true
		}
	}
	return false
}

// HasAllPrivilege if user has all privileges
func (db *Connector) HasAllPrivilege() bool {
	user, host, err := db.CurrentUser()
	if err != nil {
		common.Log.Error("User: %s, HasAllPrivilege: %s", db.User, err.Error())
		return false
	}

	// concat privilege columns
	res, err := db.Query("SELECT GROUP_CONCAT(COLUMN_NAME) from information_schema.COLUMNS where TABLE_SCHEMA='mysql' and TABLE_NAME='user' and COLUMN_NAME like '%_priv'")
	if err != nil {
		common.Log.Error("HasAllPrivilege, DSN: %s, Error: %s", db.Addr, err.Error())
		return false
	}
	var priv string
	if len(res.Rows) > 0 {
		priv = res.Rows[0].Str(0)
	} else {
		common.Log.Error("HasAllPrivilege, DSN: %s, get privilege string error", db.Addr)
		return false
	}

	// get all privilege status
	res, err = db.Query("select concat("+priv+") from mysql.user where user='%s' and host='%s'", user, host)
	if err != nil {
		common.Log.Error("HasAllPrivilege, DSN: %s, Error: %s", db.Addr, err.Error())
		return false
	}

	// %_priv
	if len(res.Rows) > 0 {
		if strings.Replace(res.Rows[0].Str(0), "Y", "", -1) == "" {
			return true
		}
	}
	return false
}
