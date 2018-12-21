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
	"fmt"
	"strings"

	"github.com/XiaoMi/soar/common"
)

// CurrentUser get current user with current_user() function
func (db *Connector) CurrentUser() (string, string, error) {
	var user, host string
	res, err := db.Query("select current_user()")
	if err != nil {
		return user, host, err
	}
	if res.Rows.Next() {
		var currentUser string
		err = res.Rows.Scan(&currentUser)
		if err != nil {
			return user, host, err
		}
		res.Rows.Close()

		cols := strings.Split(currentUser, "@")
		if len(cols) == 2 {
			user = strings.Trim(cols[0], "'")
			host = strings.Trim(cols[1], "'")
			if strings.Contains(user, "'") || strings.Contains(host, "'") {
				return "", "", errors.New("user or host contains irregular character")
			}
			return user, host, nil
		}
		return user, host, errors.New("user or host contains irregular character")
	}
	return user, host, errors.New("no privilege info")
}

// HasSelectPrivilege if user has select privilege
func (db *Connector) HasSelectPrivilege() bool {
	user, host, err := db.CurrentUser()
	if err != nil {
		common.Log.Error("User: %s, HasSelectPrivilege: %s", db.User, err.Error())
		return false
	}
	res, err := db.Query(fmt.Sprintf("select Select_priv from mysql.user where user='%s' and host='%s'", user, host))
	if err != nil {
		common.Log.Error("HasSelectPrivilege, DSN: %s, Error: %s", db.Addr, err.Error())
		return false
	}
	// Select_priv
	if res.Rows.Next() {
		var selectPrivilege string
		err = res.Rows.Scan(&selectPrivilege)
		if err != nil {
			common.Log.Error("HasSelectPrivilege, Scan Error: %s", err.Error())
			return false
		}
		res.Rows.Close()

		if selectPrivilege == "Y" {
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
	res, err := db.Query("SELECT GROUP_CONCAT(COLUMN_NAME) from information_schema.COLUMNS where TABLE_SCHEMA='mysql' and TABLE_NAME='user' and COLUMN_NAME like '%%_priv'")
	if err != nil {
		common.Log.Error("HasAllPrivilege, DSN: %s, Error: %s", db.Addr, err.Error())
		return false
	}

	var priv string
	if res.Rows.Next() {
		err = res.Rows.Scan(&priv)
		if err != nil {
			common.Log.Error("HasAllPrivilege, DSN: %s, Scan error", db.Addr)
			return false
		}
		res.Rows.Close()
	}

	// get all privilege status
	res, err = db.Query(fmt.Sprintf("select concat("+priv+") from mysql.user where user='%s' and host='%s'", user, host))
	if err != nil {
		common.Log.Error("HasAllPrivilege, DSN: %s, Error: %s", db.Addr, err.Error())
		return false
	}

	// %_priv
	if res.Rows.Next() {
		err = res.Rows.Scan(&priv)
		if err != nil {
			common.Log.Error("HasAllPrivilege, DSN: %s, Scan error", db.Addr)
			return false
		}
		res.Rows.Close()
		if strings.Replace(priv, "Y", "", -1) == "" {
			return true
		}
	}
	return false
}
