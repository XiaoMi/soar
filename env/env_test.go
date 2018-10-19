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

package env

import (
	"flag"
	"testing"

	"github.com/XiaoMi/soar/common"
	"github.com/XiaoMi/soar/database"
	"github.com/kr/pretty"
	"github.com/ziutek/mymysql/mysql"
)

var connTest *database.Connector
var update = flag.Bool("update", false, "update .golden files")

func init() {
	common.BaseDir = common.DevPath
	err := common.ParseConfig("")
	common.LogIfError(err, "init ParseConfig")
	connTest = &database.Connector{
		Addr:     common.Config.TestDSN.Addr,
		User:     common.Config.TestDSN.User,
		Pass:     common.Config.TestDSN.Password,
		Database: common.Config.TestDSN.Schema,
		Charset:  common.Config.TestDSN.Charset,
	}
}

func TestNewVirtualEnv(t *testing.T) {
	testSQL := []string{
		"create table t(id int,c1 varchar(20),PRIMARY KEY (id));",
		"alter table t add index `idx_c1`(c1);",
		"alter table t add index `idx_c1`(c1);",
		"select * from city where country_id = 44;",
		"select * from address where address2 is not null;",
		"select * from address where address2 is null;",
		"select * from address where address2 >= 44;",
		"select * from city where country_id between 44 and 107;",
		"select * from city where city like 'Ad%';",
		"select * from city where city = 'Aden' and country_id = 107;",
		"select * from city where country_id > 31 and city = 'Aden';",
		"select * from address where address_id > 8 and city_id < 400 and district = 'Nantou';",
		"select * from address where address_id > 8 and city_id < 400;",
		"select * from actor where last_update='2006-02-15 04:34:33' and last_name='CHASE' group by first_name;",
		"select * from address where last_update >='2014-09-25 22:33:47' group by district;",
		"select * from address group by address,district;",
		"select * from address where last_update='2014-09-25 22:30:27' group by district,(address_id+city_id);",
		"select * from customer where active=1 order by last_name limit 10;",
		"select * from customer order by last_name limit 10;",
		"select * from customer where address_id > 224 order by address_id limit 10;",
		"select * from customer where address_id < 224 order by address_id limit 10;",
		"select * from customer where active=1 order by last_name;",
		"select * from customer where address_id > 224 order by address_id;",
		"select * from customer where address_id in (224,510) order by last_name;",
		"select city from city where country_id = 44;",
		"select city,city_id from city where country_id = 44 and last_update='2006-02-15 04:45:25';",
		"select city from city where country_id > 44 and last_update > '2006-02-15 04:45:25';",
		"select * from city where country_id=1 and city='Kabul' order by last_update;",
		"select * from city where country_id>1 and city='Kabul' order by last_update;",
		"select * from city where city_id>251 order by last_update; ",
		"select * from city i inner join country o on i.country_id=o.country_id;",
		"select * from city i left join country o on i.city_id=o.country_id;",
		"select * from city i right join country o on i.city_id=o.country_id;",
		"select * from city i left join country o on i.city_id=o.country_id where o.country_id is null;",
		"select * from city i right join country o on i.city_id=o.country_id where i.city_id is null;",
		"select * from city i left join country o on i.city_id=o.country_id union select * from city i right join country o on i.city_id=o.country_id;",
		"select * from city i left join country o on i.city_id=o.country_id where o.country_id is null union select * from city i right join country o on i.city_id=o.country_id where i.city_id is null;",
		"select first_name,last_name,email from customer natural left join address;",
		"select first_name,last_name,email from customer natural left join address;",
		"select first_name,last_name,email from customer natural right join address;",
		"select first_name,last_name,email from customer STRAIGHT_JOIN address on customer.address_id=address.address_id;",
		"select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;",
	}

	rEnv := connTest

	env := NewVirtualEnv(connTest)
	defer env.CleanUp()
	common.GoldenDiff(func() {
		for _, sql := range testSQL {
			env.BuildVirtualEnv(rEnv, sql)
			switch err := env.Error.(type) {
			case nil:
				pretty.Println(sql, "OK")
			case error:
				// unexpected EOF
				// 测试环境无法访问，或者被Disable的时候会进入这个分支
				pretty.Println(sql, err)
			case *mysql.Error:
				if err.Code != 1061 {
					t.Error(err)
				}
			default:
				t.Error(err)
			}
		}
	}, t.Name(), update)
}

func TestGenTableColumns(t *testing.T) {
	vEnv, rEnv := BuildEnv()
	defer vEnv.CleanUp()

	pretty.Println(common.Config.TestDSN.Disable)
	if common.Config.TestDSN.Disable {
		common.Log.Warn("common.Config.TestDSN.Disable=true, by pass TestGenTableColumns")
		return
	}

	// 只能对sakila数据库进行测试
	if rEnv.Database == "sakila" {
		testSQL := []string{
			"select * from city where country_id = 44;",
			"select country_id from city where country_id = 44;",
			"select country_id from city where country_id > 44;",
		}

		metaList := []common.Meta{
			{
				"": &common.DB{
					Table: map[string]*common.Table{
						"city": common.NewTable("city"),
					},
				},
			},
			{
				"sakila": &common.DB{
					Table: map[string]*common.Table{
						"city": common.NewTable("city"),
					},
				},
			},
			{
				"sakila": &common.DB{
					Table: map[string]*common.Table{
						"city": {
							TableName: "city",
							Column: map[string]*common.Column{
								"country_id": {
									Name: "country_id",
								},
							},
						},
					},
				},
			},
		}

		for i, sql := range testSQL {
			vEnv.BuildVirtualEnv(rEnv, sql)
			tFlag := false
			columns := vEnv.GenTableColumns(metaList[i])
			if _, ok := columns["sakila"]; ok {
				if _, okk := columns["sakila"]["city"]; okk {
					if length := len(columns["sakila"]["city"]); length >= 1 {
						tFlag = true
					}
				}
			}

			if !tFlag {
				t.Errorf("columns: \n%s", pretty.Sprint(columns))
			}
		}
	}
}
