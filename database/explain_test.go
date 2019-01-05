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
	"fmt"
	"testing"

	"github.com/XiaoMi/soar/common"

	"github.com/kr/pretty"
)

var sqls = []string{
	`use sakila`, // not explain able sql, will convert to empty!
	`select * from city where country_id = 44;`,
	`select * from address where address2 is not null;`,
	`select * from address where address2 is null;`,
	`select * from address where address2 >= 44;`,
	`select * from city where country_id between 44 and 107;`,
	`select * from city where city like 'Ad%';`,
	`select * from city where city = 'Aden' and country_id = 107;`,
	`select * from city where country_id > 31 and city = 'Aden';`,
	`select * from address where address_id > 8 and city_id < 400 and district = 'Nantou';`,
	`select * from address where address_id > 8 and city_id < 400;`,
	`select first_name from actor where last_update='2006-02-15 04:34:33' and last_name='CHASE' group by first_name;`,
	`select district from address where last_update >='2014-09-25 22:33:47' group by district;`,
	`select address from address group by address,district;`,
	`select district from address where last_update='2014-09-25 22:30:27' group by district,(address_id+city_id);`,
	`select * from customer where active=1 order by last_name limit 10;`,
	`select * from customer order by last_name limit 10;`,
	`select * from customer where address_id > 224 order by address_id limit 10;`,
	`select * from customer where address_id < 224 order by address_id limit 10;`,
	`select * from customer where active=1 order by last_name;`,
	`select * from customer where address_id > 224 order by address_id;`,
	`select * from customer where address_id in (224,510) order by last_name;`,
	`select city from city where country_id = 44;`,
	`select city,city_id from city where country_id = 44 and last_update='2006-02-15 04:45:25';`,
	`select city from city where country_id > 44 and last_update > '2006-02-15 04:45:25';`,
	`select * from city where country_id=1 and city='Kabul' order by last_update;`,
	`select * from city where country_id>1 and city='Kabul' order by last_update;`,
	`select * from city where city_id>251 order by last_update;`,
	`select * from city i inner join country o on i.country_id=o.country_id;`,
	`select * from city i left join country o on i.city_id=o.country_id;`,
	`select * from city i right join country o on i.city_id=o.country_id;`,
	`select * from city i left join country o on i.city_id=o.country_id where o.country_id is null;`,
	`select * from city i right join country o on i.city_id=o.country_id where i.city_id is null;`,
	`select * from city i left join country o on i.city_id=o.country_id union select * from city i right join country o on i.city_id=o.country_id;`,
	`select * from city i left join country o on i.city_id=o.country_id where o.country_id is null union select * from city i right join country o on i.city_id=o.country_id where i.city_id is null;`,
	`select first_name,last_name,email from customer natural left join address;`,
	`select first_name,last_name,email from customer natural left join address;`,
	`select first_name,last_name,email from customer natural right join address;`,
	`select first_name,last_name,email from customer STRAIGHT_JOIN address on customer.address_id=address.address_id;`,
	`select ID,name from (select address from customer_list where SID=1 order by phone limit 50,10) a join customer_list l on (a.address=l.address) join city c on (c.city=l.city) order by phone desc;`,
	`SELECT a.table_name 表名, a.table_comment 表说明, b.COLUMN_NAME 字段名, b.column_comment 字段说明, b.column_type 字段类型, b.column_key 约束 FROM information_schema.TABLES a LEFT JOIN information_schema. COLUMNS b ON a.table_name = b.TABLE_NAME WHERE a.table_schema IN ('a') AND b.column_comment LIKE '%一%' ORDER BY a.table_name`,
}

var exp = []string{
	`+----+-------------+---------+-------+---------------------------------------------------------+-------------------+---------+---------------------------+------+-------------+
| id | select_type | table   | type  | possible_keys                                           | key               | key_len | ref                       | rows | Extra       |
+----+-------------+---------+-------+---------------------------------------------------------+-------------------+---------+---------------------------+------+-------------+
|  1 | SIMPLE      | country | index | PRIMARY,country_id                                      | country           | 152     | NULL                      |  109 | Using index |
|  1 | SIMPLE      | city    | ref   | idx_fk_country_id,idx_country_id_city,idx_all,idx_other | idx_fk_country_id | 2       | sakila.country.country_id |    2 | Using index |
+----+-------------+---------+-------+---------------------------------------------------------+-------------------+---------+---------------------------+------+-------------+`,
	`+----+-------------+---------+------------+-------+-------------------+-------------------+---------+---------------------------+------+----------+-------------+
| id | select_type | table   | partitions | type  | possible_keys     | key               | key_len | ref                       | rows | filtered | Extra       |
+----+-------------+---------+------------+-------+-------------------+-------------------+---------+---------------------------+------+----------+-------------+
|  1 | SIMPLE      | country | NULL       | index | PRIMARY           | PRIMARY           | 2       | NULL                      |  109 |   100.00 | Using index |
|  1 | SIMPLE      | city    | NULL       | ref   | idx_fk_country_id | idx_fk_country_id | 2       | sakila.country.country_id |    5 |   100.00 | Using index |
+----+-------------+---------+------------+-------+-------------------+-------------------+---------+---------------------------+------+----------+-------------+`,
	`*************************** 1. row ***************************
           id: 1
  select_type: SIMPLE
        table: country
         type: index
possible_keys: PRIMARY,country_id
          key: country
      key_len: 152
          ref: NULL
         rows: 109
        Extra: Using index
*************************** 2. row ***************************
           id: 1
  select_type: SIMPLE
        table: city
         type: ref
possible_keys: idx_fk_country_id,idx_country_id_city,idx_all,idx_other
          key: idx_fk_country_id
      key_len: 2
          ref: sakila.country.country_id
         rows: 2
        Extra: Using index`,
	`+----+-------------+---------+------------+-------+---------------------------------------------------------+-------------------+---------+---------------------------+------+-------------+
| id | select_type | table   | partitions | type  | possible_keys                                           | key               | key_len | ref                       | rows | Extra       |
+----+-------------+---------+------------+-------+---------------------------------------------------------+-------------------+---------+---------------------------+------+-------------+
|  1 | SIMPLE      | country | NULL       | index | PRIMARY,country_id                                      | country           | 152     | NULL                      |  109 | Using index |
|  1 | SIMPLE      | city    | NULL       | ref   | idx_fk_country_id,idx_country_id_city,idx_all,idx_other | idx_fk_country_id | 2       | sakila.country.country_id |    2 | Using index |
+----+-------------+---------+------------+-------+---------------------------------------------------------+-------------------+---------+---------------------------+------+-------------+`,
	`{
  "query_block": {
    "select_id": 1,
    "message": "No tables used"
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "message": "no matching row in const table"
  }
}`,
	`{
  "query_block": {
    "select_id": 1,
    "table": {
      "insert": true,
      "table_name": "t1",
      "access_type": "ALL"
    } /* table */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "message": "no matching row in const table"
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "message": "no matching row in const table"
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "13.50"
    } /* cost_info */,
    "table": {
      "table_name": "a4",
      "access_type": "ALL",
      "rows_examined_per_scan": 14,
      "rows_produced_per_join": 14,
      "filtered": "100.00",
      "cost_info": {
        "read_cost": "10.70",
        "eval_cost": "2.80",
        "prefix_cost": "13.50",
        "data_read_per_join": "224"
      } /* cost_info */,
      "used_columns": [
        "i"
      ] /* used_columns */,
      "materialized_from_subquery": {
        "using_temporary_table": true,
        "dependent": false,
        "cacheable": true,
        "query_block": {
          "select_id": 2,
          "cost_info": {
            "query_cost": "13.50"
          } /* cost_info */,
          "table": {
            "table_name": "a3",
            "access_type": "ALL",
            "rows_examined_per_scan": 14,
            "rows_produced_per_join": 14,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "10.70",
              "eval_cost": "2.80",
              "prefix_cost": "13.50",
              "data_read_per_join": "224"
            } /* cost_info */,
            "used_columns": [
              "i"
            ] /* used_columns */,
            "materialized_from_subquery": {
              "using_temporary_table": true,
              "dependent": false,
              "cacheable": true,
              "query_block": {
                "select_id": 3,
                "cost_info": {
                  "query_cost": "13.50"
                } /* cost_info */,
                "table": {
                  "table_name": "a2",
                  "access_type": "ALL",
                  "rows_examined_per_scan": 14,
                  "rows_produced_per_join": 14,
                  "filtered": "100.00",
                  "cost_info": {
                    "read_cost": "10.70",
                    "eval_cost": "2.80",
                    "prefix_cost": "13.50",
                    "data_read_per_join": "224"
                  } /* cost_info */,
                  "used_columns": [
                    "i"
                  ] /* used_columns */,
                  "materialized_from_subquery": {
                    "using_temporary_table": true,
                    "dependent": false,
                    "cacheable": true,
                    "query_block": {
                      "select_id": 4,
                      "cost_info": {
                        "query_cost": "15.55"
                      } /* cost_info */,
                      "nested_loop": [
                        {
                          "table": {
                            "table_name": "t2",
                            "access_type": "ALL",
                            "rows_examined_per_scan": 2,
                            "rows_produced_per_join": 2,
                            "filtered": "100.00",
                            "cost_info": {
                              "read_cost": "2.00",
                              "eval_cost": "0.40",
                              "prefix_cost": "2.40",
                              "data_read_per_join": "16"
                            } /* cost_info */
                          } /* table */
                        },
                        {
                          "table": {
                            "table_name": "a1",
                            "access_type": "ALL",
                            "rows_examined_per_scan": 7,
                            "rows_produced_per_join": 14,
                            "filtered": "100.00",
                            "using_join_buffer": "Block Nested Loop",
                            "cost_info": {
                              "read_cost": "10.35",
                              "eval_cost": "2.80",
                              "prefix_cost": "15.55",
                              "data_read_per_join": "224"
                            } /* cost_info */,
                            "used_columns": [
                              "i"
                            ] /* used_columns */,
                            "materialized_from_subquery": {
                              "using_temporary_table": true,
                              "dependent": false,
                              "cacheable": true,
                              "query_block": {
                                "select_id": 5,
                                "cost_info": {
                                  "query_cost": "3.41"
                                } /* cost_info */,
                                "table": {
                                  "table_name": "t1",
                                  "access_type": "ALL",
                                  "rows_examined_per_scan": 7,
                                  "rows_produced_per_join": 7,
                                  "filtered": "100.00",
                                  "cost_info": {
                                    "read_cost": "2.01",
                                    "eval_cost": "1.40",
                                    "prefix_cost": "3.41",
                                    "data_read_per_join": "56"
                                  } /* cost_info */,
                                  "used_columns": [
                                    "i"
                                  ] /* used_columns */
                                } /* table */
                              } /* query_block */
                            } /* materialized_from_subquery */
                          } /* table */
                        }
                      ] /* nested_loop */
                    } /* query_block */
                  } /* materialized_from_subquery */
                } /* table */
              } /* query_block */
            } /* materialized_from_subquery */
          } /* table */
        } /* query_block */
      } /* materialized_from_subquery */
    } /* table */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "5.81"
    } /* cost_info */,
    "nested_loop": [
      {
        "table": {
          "table_name": "t1",
          "access_type": "ALL",
          "rows_examined_per_scan": 7,
          "rows_produced_per_join": 0,
          "filtered": "14.29",
          "cost_info": {
            "read_cost": "3.21",
            "eval_cost": "0.20",
            "prefix_cost": "3.41",
            "data_read_per_join": "7"
          } /* cost_info */,
          "used_columns": [
            "i"
          ] /* used_columns */,
          "attached_condition": "(test.t1.i = 10)"
        } /* table */
      },
      {
        "table": {
          "table_name": "t2",
          "access_type": "ALL",
          "rows_examined_per_scan": 2,
          "rows_produced_per_join": 0,
          "filtered": "50.00",
          "first_match": "t1",
          "using_join_buffer": "Block Nested Loop",
          "cost_info": {
            "read_cost": "2.20",
            "eval_cost": "0.20",
            "prefix_cost": "5.82",
            "data_read_per_join": "7"
          } /* cost_info */,
          "used_columns": [
            "i"
          ] /* used_columns */,
          "attached_condition": "(test.t2.i = 10)"
        } /* table */
      }
    ] /* nested_loop */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "3.41"
    } /* cost_info */,
    "table": {
      "table_name": "t1",
      "access_type": "ALL",
      "rows_examined_per_scan": 7,
      "rows_produced_per_join": 7,
      "filtered": "100.00",
      "cost_info": {
        "read_cost": "2.01",
        "eval_cost": "1.40",
        "prefix_cost": "3.41",
        "data_read_per_join": "56"
      } /* cost_info */,
      "used_columns": [
        "i"
      ] /* used_columns */,
      "attached_condition": "(<in_optimizer>(test.t1.i ,<exists>(/* select#2 */ select 1 from test.t2 where ((test.t1.i = 10) and (<cache>(test.t1.i) = test.t2.i)))) or <in_optimizer>(test.t1.i.test.t1.i in ( <materialize> (/* select#3 */ select NULL from test.t4 where 1 ), <primary_index_lookup>(test.t1.i in <temporary table> on <auto_key> where ((test.t1.i = materialized-subquery.i))))))",
      "attached_subqueries": [
        {
          "table": {
            "table_name": "<materialized_subquery>",
            "access_type": "eq_ref",
            "key": "<auto_key>",
            "key_length": "5",
            "rows_examined_per_scan": 1,
            "materialized_from_subquery": {
              "using_temporary_table": true,
              "dependent": true,
              "cacheable": false,
              "query_block": {
                "select_id": 3,
                "message": "no matching row in const table"
              } /* query_block */
            } /* materialized_from_subquery */
          } /* table */
        },
        {
          "dependent": true,
          "cacheable": false,
          "query_block": {
            "select_id": 2,
            "cost_info": {
              "query_cost": "2.40"
            } /* cost_info */,
            "table": {
              "table_name": "t2",
              "access_type": "ALL",
              "rows_examined_per_scan": 2,
              "rows_produced_per_join": 1,
              "filtered": "50.00",
              "cost_info": {
                "read_cost": "2.00",
                "eval_cost": "0.20",
                "prefix_cost": "2.40",
                "data_read_per_join": "8"
              } /* cost_info */,
              "used_columns": [
                "i"
              ] /* used_columns */,
              "attached_condition": "((test.t1.i = 10) and (<cache>(test.t1.i) = test.t2.i))"
            } /* table */
          } /* query_block */
        }
      ] /* attached_subqueries */
    } /* table */
  } /* query_block */
}`,
	`{
  "query_block": {
    "union_result": {
      "using_temporary_table": true,
      "table_name": "<union1,2,3>",
      "access_type": "ALL",
      "query_specifications": [
        {
          "dependent": false,
          "cacheable": true,
          "query_block": {
            "select_id": 1,
            "cost_info": {
              "query_cost": "3.41"
            } /* cost_info */,
            "table": {
              "table_name": "t1",
              "access_type": "ALL",
              "rows_examined_per_scan": 7,
              "rows_produced_per_join": 7,
              "filtered": "100.00",
              "cost_info": {
                "read_cost": "2.01",
                "eval_cost": "1.40",
                "prefix_cost": "3.41",
                "data_read_per_join": "56"
              } /* cost_info */,
              "used_columns": [
                "i"
              ] /* used_columns */
            } /* table */
          } /* query_block */
        },
        {
          "dependent": false,
          "cacheable": true,
          "query_block": {
            "select_id": 2,
            "cost_info": {
              "query_cost": "2.40"
            } /* cost_info */,
            "table": {
              "table_name": "t2",
              "access_type": "ALL",
              "rows_examined_per_scan": 2,
              "rows_produced_per_join": 2,
              "filtered": "100.00",
              "cost_info": {
                "read_cost": "2.00",
                "eval_cost": "0.40",
                "prefix_cost": "2.40",
                "data_read_per_join": "16"
              } /* cost_info */,
              "used_columns": [
                "i"
              ] /* used_columns */
            } /* table */
          } /* query_block */
        },
        {
          "dependent": false,
          "cacheable": true,
          "query_block": {
            "select_id": 3,
            "message": "no matching row in const table"
          } /* query_block */
        }
      ] /* query_specifications */
    } /* union_result */
  } /* query_block */
}`,
	`{
  "query_block": {
    "union_result": {
      "using_temporary_table": false,
      "query_specifications": [
        {
          "dependent": false,
          "cacheable": true,
          "query_block": {
            "select_id": 1,
            "cost_info": {
              "query_cost": "7.21"
            } /* cost_info */,
            "nested_loop": [
              {
                "table": {
                  "table_name": "t2",
                  "access_type": "ALL",
                  "rows_examined_per_scan": 2,
                  "rows_produced_per_join": 2,
                  "filtered": "100.00",
                  "cost_info": {
                    "read_cost": "2.00",
                    "eval_cost": "0.40",
                    "prefix_cost": "2.40",
                    "data_read_per_join": "16"
                  } /* cost_info */
                } /* table */
              },
              {
                "table": {
                  "table_name": "t1",
                  "access_type": "ALL",
                  "rows_examined_per_scan": 7,
                  "rows_produced_per_join": 14,
                  "filtered": "100.00",
                  "using_join_buffer": "Block Nested Loop",
                  "cost_info": {
                    "read_cost": "2.01",
                    "eval_cost": "2.80",
                    "prefix_cost": "7.22",
                    "data_read_per_join": "112"
                  } /* cost_info */,
                  "used_columns": [
                    "i"
                  ] /* used_columns */
                } /* table */
              }
            ] /* nested_loop */
          } /* query_block */
        },
        {
          "dependent": false,
          "cacheable": true,
          "query_block": {
            "select_id": 2,
            "message": "no matching row in const table"
          } /* query_block */
        }
      ] /* query_specifications */
    } /* union_result */
  } /* query_block */
}`,
	`{
  "query_block": {
    "ordering_operation": {
      "using_filesort": true,
      "union_result": {
        "using_temporary_table": true,
        "table_name": "<union1,2>",
        "access_type": "ALL",
        "query_specifications": [
          {
            "dependent": false,
            "cacheable": true,
            "query_block": {
              "select_id": 1,
              "cost_info": {
                "query_cost": "3.41"
              } /* cost_info */,
              "table": {
                "table_name": "t1",
                "access_type": "ALL",
                "rows_examined_per_scan": 7,
                "rows_produced_per_join": 7,
                "filtered": "100.00",
                "cost_info": {
                  "read_cost": "2.01",
                  "eval_cost": "1.40",
                  "prefix_cost": "3.41",
                  "data_read_per_join": "56"
                } /* cost_info */,
                "used_columns": [
                  "i"
                ] /* used_columns */
              } /* table */
            } /* query_block */
          },
          {
            "dependent": false,
            "cacheable": true,
            "query_block": {
              "select_id": 2,
              "cost_info": {
                "query_cost": "2.40"
              } /* cost_info */,
              "table": {
                "table_name": "t2",
                "access_type": "ALL",
                "rows_examined_per_scan": 2,
                "rows_produced_per_join": 2,
                "filtered": "100.00",
                "cost_info": {
                  "read_cost": "2.00",
                  "eval_cost": "0.40",
                  "prefix_cost": "2.40",
                  "data_read_per_join": "16"
                } /* cost_info */,
                "used_columns": [
                  "i"
                ] /* used_columns */
              } /* table */
            } /* query_block */
          }
        ] /* query_specifications */
      } /* union_result */,
      "order_by_subqueries": [
        {
          "dependent": true,
          "cacheable": false,
          "query_block": {
            "select_id": 3,
            "message": "No tables used"
          } /* query_block */
        }
      ] /* order_by_subqueries */
    } /* ordering_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "3.41"
    } /* cost_info */,
    "ordering_operation": {
      "using_filesort": false,
      "table": {
        "table_name": "t1",
        "access_type": "ALL",
        "rows_examined_per_scan": 7,
        "rows_produced_per_join": 7,
        "filtered": "100.00",
        "cost_info": {
          "read_cost": "2.01",
          "eval_cost": "1.40",
          "prefix_cost": "3.41",
          "data_read_per_join": "56"
        } /* cost_info */,
        "used_columns": [
          "i"
        ] /* used_columns */
      } /* table */,
      "optimized_away_subqueries": [
        {
          "dependent": false,
          "cacheable": true,
          "query_block": {
            "select_id": 2,
            "cost_info": {
              "query_cost": "2.40"
            } /* cost_info */,
            "table": {
              "table_name": "t2",
              "access_type": "ALL",
              "rows_examined_per_scan": 2,
              "rows_produced_per_join": 2,
              "filtered": "100.00",
              "cost_info": {
                "read_cost": "2.00",
                "eval_cost": "0.40",
                "prefix_cost": "2.40",
                "data_read_per_join": "16"
              } /* cost_info */
            } /* table */
          } /* query_block */
        }
      ] /* optimized_away_subqueries */
    } /* ordering_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "3.41"
    } /* cost_info */,
    "table": {
      "table_name": "t1",
      "access_type": "ALL",
      "rows_examined_per_scan": 7,
      "rows_produced_per_join": 7,
      "filtered": "100.00",
      "cost_info": {
        "read_cost": "2.01",
        "eval_cost": "1.40",
        "prefix_cost": "3.41",
        "data_read_per_join": "56"
      } /* cost_info */,
      "used_columns": [
        "i"
      ] /* used_columns */
    } /* table */,
    "having_subqueries": [
      {
        "dependent": false,
        "cacheable": true,
        "query_block": {
          "select_id": 3,
          "cost_info": {
            "query_cost": "2.40"
          } /* cost_info */,
          "table": {
            "table_name": "t2",
            "access_type": "ALL",
            "rows_examined_per_scan": 2,
            "rows_produced_per_join": 2,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "2.00",
              "eval_cost": "0.40",
              "prefix_cost": "2.40",
              "data_read_per_join": "16"
            } /* cost_info */,
            "used_columns": [
              "i"
            ] /* used_columns */
          } /* table */
        } /* query_block */
      },
      {
        "dependent": false,
        "cacheable": true,
        "query_block": {
          "select_id": 2,
          "cost_info": {
            "query_cost": "2.40"
          } /* cost_info */,
          "table": {
            "table_name": "t2",
            "access_type": "ALL",
            "rows_examined_per_scan": 2,
            "rows_produced_per_join": 2,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "2.00",
              "eval_cost": "0.40",
              "prefix_cost": "2.40",
              "data_read_per_join": "16"
            } /* cost_info */,
            "used_columns": [
              "i"
            ] /* used_columns */
          } /* table */
        } /* query_block */
      }
    ] /* having_subqueries */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "10.41"
    } /* cost_info */,
    "grouping_operation": {
      "using_temporary_table": true,
      "using_filesort": true,
      "cost_info": {
        "sort_cost": "7.00"
      } /* cost_info */,
      "table": {
        "table_name": "t1",
        "access_type": "ALL",
        "rows_examined_per_scan": 7,
        "rows_produced_per_join": 7,
        "filtered": "100.00",
        "cost_info": {
          "read_cost": "2.01",
          "eval_cost": "1.40",
          "prefix_cost": "3.41",
          "data_read_per_join": "56"
        } /* cost_info */,
        "used_columns": [
          "i"
        ] /* used_columns */
      } /* table */,
      "group_by_subqueries": [
        {
          "dependent": true,
          "cacheable": false,
          "query_block": {
            "select_id": 3,
            "cost_info": {
              "query_cost": "2.40"
            } /* cost_info */,
            "table": {
              "table_name": "t2",
              "access_type": "ALL",
              "rows_examined_per_scan": 2,
              "rows_produced_per_join": 2,
              "filtered": "100.00",
              "cost_info": {
                "read_cost": "2.00",
                "eval_cost": "0.40",
                "prefix_cost": "2.40",
                "data_read_per_join": "16"
              } /* cost_info */,
              "used_columns": [
                "i"
              ] /* used_columns */,
              "attached_condition": "<if>(outer_field_is_not_null, ((<cache>(test.t1.i) >= test.t2.i) or isnull(test.t2.i)), true)"
            } /* table */
          } /* query_block */
        },
        {
          "dependent": true,
          "cacheable": false,
          "query_block": {
            "select_id": 2,
            "cost_info": {
              "query_cost": "2.40"
            } /* cost_info */,
            "table": {
              "table_name": "t2",
              "access_type": "ALL",
              "rows_examined_per_scan": 2,
              "rows_produced_per_join": 2,
              "filtered": "100.00",
              "cost_info": {
                "read_cost": "2.00",
                "eval_cost": "0.40",
                "prefix_cost": "2.40",
                "data_read_per_join": "16"
              } /* cost_info */,
              "used_columns": [
                "i"
              ] /* used_columns */,
              "attached_condition": "<if>(outer_field_is_not_null, ((<cache>(test.t1.i) <= test.t2.i) or isnull(test.t2.i)), true)"
            } /* table */
          } /* query_block */
        }
      ] /* group_by_subqueries */
    } /* grouping_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "3.41"
    } /* cost_info */,
    "table": {
      "table_name": "t1",
      "access_type": "ALL",
      "rows_examined_per_scan": 7,
      "rows_produced_per_join": 7,
      "filtered": "100.00",
      "cost_info": {
        "read_cost": "2.01",
        "eval_cost": "1.40",
        "prefix_cost": "3.41",
        "data_read_per_join": "56"
      } /* cost_info */,
      "used_columns": [
        "i"
      ] /* used_columns */
    } /* table */,
    "select_list_subqueries": [
      {
        "dependent": false,
        "cacheable": false,
        "query_block": {
          "select_id": 2,
          "cost_info": {
            "query_cost": "3.41"
          } /* cost_info */,
          "ordering_operation": {
            "using_temporary_table": true,
            "using_filesort": true,
            "table": {
              "table_name": "t1",
              "access_type": "ALL",
              "rows_examined_per_scan": 7,
              "rows_produced_per_join": 7,
              "filtered": "100.00",
              "cost_info": {
                "read_cost": "2.01",
                "eval_cost": "1.40",
                "prefix_cost": "3.41",
                "data_read_per_join": "56"
              } /* cost_info */,
              "used_columns": [
                "i"
              ] /* used_columns */
            } /* table */
          } /* ordering_operation */
        } /* query_block */
      }
    ] /* select_list_subqueries */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "message": "no matching row in const table"
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "5.21"
    } /* cost_info */,
    "nested_loop": [
      {
        "table": {
          "table_name": "t1",
          "access_type": "ALL",
          "rows_examined_per_scan": 2,
          "rows_produced_per_join": 2,
          "filtered": "100.00",
          "cost_info": {
            "read_cost": "2.00",
            "eval_cost": "0.40",
            "prefix_cost": "2.40",
            "data_read_per_join": "32"
          } /* cost_info */,
          "used_columns": [
            "a",
            "b"
          ] /* used_columns */,
          "attached_condition": "<nop>(<in_optimizer>((/* select#3 */ select test.t3.e from test.t3),<exists>(/* select#4 */ select 1 from test.t3 where (test.t1.b and <if>(outer_field_is_not_null, ((<cache>((/* select#3 */ select test.t3.e from test.t3)) < test.t3.e) or isnull(test.t3.e)), true)) having <if>(outer_field_is_not_null, <is_not_null_test>(test.t3.e), true))))",
          "attached_subqueries": [
            {
              "dependent": true,
              "cacheable": false,
              "query_block": {
                "select_id": 4,
                "cost_info": {
                  "query_cost": "2.40"
                } /* cost_info */,
                "table": {
                  "table_name": "t3",
                  "access_type": "ALL",
                  "rows_examined_per_scan": 2,
                  "rows_produced_per_join": 2,
                  "filtered": "100.00",
                  "cost_info": {
                    "read_cost": "2.00",
                    "eval_cost": "0.40",
                    "prefix_cost": "2.40",
                    "data_read_per_join": "16"
                  } /* cost_info */,
                  "used_columns": [
                    "e"
                  ] /* used_columns */,
                  "attached_condition": "(test.t1.b and <if>(outer_field_is_not_null, ((<cache>((/* select#3 */ select test.t3.e from test.t3)) < test.t3.e) or isnull(test.t3.e)), true))"
                } /* table */
              } /* query_block */
            },
            {
              "dependent": false,
              "cacheable": true,
              "query_block": {
                "select_id": 3,
                "cost_info": {
                  "query_cost": "2.40"
                } /* cost_info */,
                "table": {
                  "table_name": "t3",
                  "access_type": "ALL",
                  "rows_examined_per_scan": 2,
                  "rows_produced_per_join": 2,
                  "filtered": "100.00",
                  "cost_info": {
                    "read_cost": "2.00",
                    "eval_cost": "0.40",
                    "prefix_cost": "2.40",
                    "data_read_per_join": "16"
                  } /* cost_info */,
                  "used_columns": [
                    "e"
                  ] /* used_columns */
                } /* table */
              } /* query_block */
            }
          ] /* attached_subqueries */
        } /* table */
      },
      {
        "table": {
          "table_name": "t2",
          "access_type": "ALL",
          "rows_examined_per_scan": 2,
          "rows_produced_per_join": 2,
          "filtered": "50.00",
          "first_match": "t1",
          "using_join_buffer": "Block Nested Loop",
          "cost_info": {
            "read_cost": "2.00",
            "eval_cost": "0.40",
            "prefix_cost": "5.21",
            "data_read_per_join": "32"
          } /* cost_info */,
          "used_columns": [
            "c"
          ] /* used_columns */,
          "attached_condition": "(test.t2.c = test.t1.a)"
        } /* table */
      }
    ] /* nested_loop */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "35.44"
    } /* cost_info */,
    "nested_loop": [
      {
        "table": {
          "table_name": "t1",
          "access_type": "ALL",
          "rows_examined_per_scan": 12,
          "rows_produced_per_join": 12,
          "filtered": "100.00",
          "cost_info": {
            "read_cost": "2.02",
            "eval_cost": "2.40",
            "prefix_cost": "4.42",
            "data_read_per_join": "96"
          } /* cost_info */,
          "used_columns": [
            "a"
          ] /* used_columns */,
          "attached_condition": "((test.t1.a is not null) and (test.t1.a is not null))"
        } /* table */
      },
      {
        "table": {
          "table_name": "<subquery3>",
          "access_type": "eq_ref",
          "key": "<auto_key>",
          "key_length": "5",
          "ref": [
            "test.t1.a"
          ] /* ref */,
          "rows_examined_per_scan": 1,
          "materialized_from_subquery": {
            "using_temporary_table": true,
            "query_block": {
              "nested_loop": [
                {
                  "table": {
                    "table_name": "t4",
                    "access_type": "ALL",
                    "rows_examined_per_scan": 12,
                    "rows_produced_per_join": 3,
                    "filtered": "33.33",
                    "cost_info": {
                      "read_cost": "3.62",
                      "eval_cost": "0.80",
                      "prefix_cost": "4.42",
                      "data_read_per_join": "31"
                    } /* cost_info */,
                    "used_columns": [
                      "a"
                    ] /* used_columns */,
                    "attached_condition": "(test.t4.a > 0)"
                  } /* table */
                },
                {
                  "table": {
                    "table_name": "t3",
                    "access_type": "ALL",
                    "rows_examined_per_scan": 12,
                    "rows_produced_per_join": 4,
                    "filtered": "10.00",
                    "using_join_buffer": "Block Nested Loop",
                    "cost_info": {
                      "read_cost": "2.02",
                      "eval_cost": "0.96",
                      "prefix_cost": "16.04",
                      "data_read_per_join": "38"
                    } /* cost_info */,
                    "used_columns": [
                      "a"
                    ] /* used_columns */,
                    "attached_condition": "(test.t3.a = test.t4.a)"
                  } /* table */
                }
              ] /* nested_loop */
            } /* query_block */
          } /* materialized_from_subquery */
        } /* table */
      },
      {
        "table": {
          "table_name": "<subquery2>",
          "access_type": "eq_ref",
          "key": "<auto_key>",
          "key_length": "5",
          "ref": [
            "test.t1.a"
          ] /* ref */,
          "rows_examined_per_scan": 1,
          "materialized_from_subquery": {
            "using_temporary_table": true,
            "query_block": {
              "table": {
                "table_name": "t2",
                "access_type": "ALL",
                "rows_examined_per_scan": 12,
                "rows_produced_per_join": 3,
                "filtered": "33.33",
                "cost_info": {
                  "read_cost": "3.62",
                  "eval_cost": "0.80",
                  "prefix_cost": "4.42",
                  "data_read_per_join": "31"
                } /* cost_info */,
                "used_columns": [
                  "a"
                ] /* used_columns */,
                "attached_condition": "(test.t2.a > 0)"
              } /* table */
            } /* query_block */
          } /* materialized_from_subquery */
        } /* table */
      }
    ] /* nested_loop */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "1.20"
    } /* cost_info */,
    "table": {
      "table_name": "t1",
      "access_type": "ALL",
      "rows_examined_per_scan": 1,
      "rows_produced_per_join": 1,
      "filtered": "100.00",
      "cost_info": {
        "read_cost": "1.00",
        "eval_cost": "0.20",
        "prefix_cost": "1.20",
        "data_read_per_join": "8"
      } /* cost_info */,
      "used_columns": [
        "i1",
        "c1"
      ] /* used_columns */,
      "attached_condition": "exists(/* select#2 */ select test.t2.c1 from test.t2 join test.t3 where ((test.t2.c1 = test.t3.c1) and (test.t2.c2 = (/* select#3 */ select min(test.t3.c1) from test.t3)) and ((/* select#3 */ select min(test.t3.c1) from test.t3) <> test.t1.c1)))",
      "attached_subqueries": [
        {
          "dependent": true,
          "cacheable": false,
          "query_block": {
            "select_id": 2,
            "cost_info": {
              "query_cost": "2.40"
            } /* cost_info */,
            "nested_loop": [
              {
                "table": {
                  "table_name": "t3",
                  "access_type": "ALL",
                  "rows_examined_per_scan": 1,
                  "rows_produced_per_join": 1,
                  "filtered": "100.00",
                  "cost_info": {
                    "read_cost": "1.00",
                    "eval_cost": "0.20",
                    "prefix_cost": "1.20",
                    "data_read_per_join": "8"
                  } /* cost_info */,
                  "used_columns": [
                    "c1"
                  ] /* used_columns */,
                  "attached_condition": "((/* select#3 */ select min(test.t3.c1) from test.t3) <> test.t1.c1)",
                  "attached_subqueries": [
                    {
                      "dependent": false,
                      "cacheable": true,
                      "query_block": {
                        "select_id": 3,
                        "cost_info": {
                          "query_cost": "1.20"
                        } /* cost_info */,
                        "table": {
                          "table_name": "t3",
                          "access_type": "ALL",
                          "rows_examined_per_scan": 1,
                          "rows_produced_per_join": 1,
                          "filtered": "100.00",
                          "cost_info": {
                            "read_cost": "1.00",
                            "eval_cost": "0.20",
                            "prefix_cost": "1.20",
                            "data_read_per_join": "8"
                          } /* cost_info */,
                          "used_columns": [
                            "c1"
                          ] /* used_columns */
                        } /* table */
                      } /* query_block */
                    }
                  ] /* attached_subqueries */
                } /* table */
              },
              {
                "table": {
                  "table_name": "t2",
                  "access_type": "ref",
                  "possible_keys": [
                    "c1"
                  ] /* possible_keys */,
                  "key": "c1",
                  "used_key_parts": [
                    "c1"
                  ] /* used_key_parts */,
                  "key_length": "3",
                  "ref": [
                    "test.t3.c1"
                  ] /* ref */,
                  "rows_examined_per_scan": 1,
                  "rows_produced_per_join": 0,
                  "filtered": "50.00",
                  "cost_info": {
                    "read_cost": "1.00",
                    "eval_cost": "0.10",
                    "prefix_cost": "2.40",
                    "data_read_per_join": "8"
                  } /* cost_info */,
                  "used_columns": [
                    "c1",
                    "c2"
                  ] /* used_columns */,
                  "attached_condition": "(test.t2.c2 = (/* select#3 */ select min(test.t3.c1) from test.t3))",
                  "attached_subqueries": [
                    {
                      "dependent": false,
                      "cacheable": true,
                      "query_block": {
                        "select_id": 3,
                        "cost_info": {
                          "query_cost": "1.20"
                        } /* cost_info */,
                        "table": {
                          "table_name": "t3",
                          "access_type": "ALL",
                          "rows_examined_per_scan": 1,
                          "rows_produced_per_join": 1,
                          "filtered": "100.00",
                          "cost_info": {
                            "read_cost": "1.00",
                            "eval_cost": "0.20",
                            "prefix_cost": "1.20",
                            "data_read_per_join": "8"
                          } /* cost_info */,
                          "used_columns": [
                            "c1"
                          ] /* used_columns */
                        } /* table */
                      } /* query_block */
                    }
                  ] /* attached_subqueries */
                } /* table */
              }
            ] /* nested_loop */
          } /* query_block */
        }
      ] /* attached_subqueries */
    } /* table */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "20.82"
    } /* cost_info */,
    "duplicates_removal": {
      "using_temporary_table": true,
      "nested_loop": [
        {
          "table": {
            "table_name": "t5",
            "access_type": "ALL",
            "rows_examined_per_scan": 3,
            "rows_produced_per_join": 3,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "2.01",
              "eval_cost": "0.60",
              "prefix_cost": "2.61",
              "data_read_per_join": "24"
            } /* cost_info */,
            "used_columns": [
              "c"
            ] /* used_columns */
          } /* table */
        },
        {
          "table": {
            "table_name": "t2",
            "access_type": "ALL",
            "rows_examined_per_scan": 3,
            "rows_produced_per_join": 3,
            "filtered": "33.33",
            "using_join_buffer": "Block Nested Loop",
            "cost_info": {
              "read_cost": "2.01",
              "eval_cost": "0.60",
              "prefix_cost": "6.41",
              "data_read_per_join": "48"
            } /* cost_info */,
            "used_columns": [
              "c",
              "c_key"
            ] /* used_columns */,
            "attached_condition": "(test.t2.c = test.t5.c)"
          } /* table */
        },
        {
          "table": {
            "table_name": "t1",
            "access_type": "index",
            "possible_keys": [
              "c_key"
            ] /* possible_keys */,
            "key": "c_key",
            "used_key_parts": [
              "c_key"
            ] /* used_key_parts */,
            "key_length": "5",
            "rows_examined_per_scan": 3,
            "rows_produced_per_join": 3,
            "filtered": "33.33",
            "using_index": true,
            "using_join_buffer": "Block Nested Loop",
            "cost_info": {
              "read_cost": "2.01",
              "eval_cost": "0.60",
              "prefix_cost": "12.22",
              "data_read_per_join": "24"
            } /* cost_info */,
            "used_columns": [
              "c_key"
            ] /* used_columns */,
            "attached_condition": "(test.t1.c_key = test.t2.c_key)"
          } /* table */
        },
        {
          "table": {
            "table_name": "t4",
            "access_type": "ALL",
            "rows_examined_per_scan": 3,
            "rows_produced_per_join": 3,
            "filtered": "33.33",
            "using_join_buffer": "Block Nested Loop",
            "cost_info": {
              "read_cost": "2.01",
              "eval_cost": "0.60",
              "prefix_cost": "16.02",
              "data_read_per_join": "48"
            } /* cost_info */,
            "used_columns": [
              "c",
              "c_key"
            ] /* used_columns */,
            "attached_condition": "((test.t4.c = test.t5.c) and (test.t4.c_key is not null))"
          } /* table */
        },
        {
          "table": {
            "table_name": "t3",
            "access_type": "ref",
            "possible_keys": [
              "c_key"
            ] /* possible_keys */,
            "key": "c_key",
            "used_key_parts": [
              "c_key"
            ] /* used_key_parts */,
            "key_length": "5",
            "ref": [
              "test.t4.c_key"
            ] /* ref */,
            "rows_examined_per_scan": 1,
            "rows_produced_per_join": 3,
            "filtered": "100.00",
            "using_index": true,
            "cost_info": {
              "read_cost": "3.00",
              "eval_cost": "0.60",
              "prefix_cost": "20.82",
              "data_read_per_join": "24"
            } /* cost_info */,
            "used_columns": [
              "c_key"
            ] /* used_columns */
          } /* table */
        }
      ] /* nested_loop */
    } /* duplicates_removal */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "table": {
      "update": true,
      "table_name": "t1",
      "access_type": "ALL",
      "rows_examined_per_scan": 1,
      "filtered": "100.00"
    } /* table */,
    "update_value_subqueries": [
      {
        "dependent": false,
        "cacheable": true,
        "query_block": {
          "select_id": 2,
          "cost_info": {
            "query_cost": "1.00"
          } /* cost_info */,
          "table": {
            "table_name": "t2",
            "access_type": "system",
            "rows_examined_per_scan": 1,
            "rows_produced_per_join": 1,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "0.00",
              "eval_cost": "0.20",
              "prefix_cost": "0.00",
              "data_read_per_join": "8"
            } /* cost_info */,
            "used_columns": [
              "i"
            ] /* used_columns */
          } /* table */
        } /* query_block */
      }
    ] /* update_value_subqueries */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "1.00"
    } /* cost_info */,
    "nested_loop": [
      {
        "table": {
          "update": true,
          "table_name": "t1",
          "access_type": "system",
          "rows_examined_per_scan": 1,
          "rows_produced_per_join": 1,
          "filtered": "100.00",
          "cost_info": {
            "read_cost": "0.00",
            "eval_cost": "0.20",
            "prefix_cost": "0.00",
            "data_read_per_join": "8"
          } /* cost_info */,
          "used_columns": [
            "i"
          ] /* used_columns */
        } /* table */
      },
      {
        "table": {
          "table_name": "t2",
          "access_type": "system",
          "rows_examined_per_scan": 1,
          "rows_produced_per_join": 1,
          "filtered": "100.00",
          "cost_info": {
            "read_cost": "0.00",
            "eval_cost": "0.20",
            "prefix_cost": "0.00",
            "data_read_per_join": "8"
          } /* cost_info */
        } /* table */
      }
    ] /* nested_loop */,
    "update_value_subqueries": [
      {
        "dependent": false,
        "cacheable": true,
        "query_block": {
          "select_id": 2,
          "cost_info": {
            "query_cost": "1.00"
          } /* cost_info */,
          "table": {
            "table_name": "t3",
            "access_type": "system",
            "rows_examined_per_scan": 1,
            "rows_produced_per_join": 1,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "0.00",
              "eval_cost": "0.20",
              "prefix_cost": "0.00",
              "data_read_per_join": "8"
            } /* cost_info */,
            "used_columns": [
              "i"
            ] /* used_columns */
          } /* table */
        } /* query_block */
      }
    ] /* update_value_subqueries */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "1.00"
    } /* cost_info */,
    "table": {
      "insert": true,
      "table_name": "t1",
      "access_type": "ALL"
    } /* table */,
    "insert_from": {
      "table": {
        "table_name": "t2",
        "access_type": "system",
        "rows_examined_per_scan": 1,
        "rows_produced_per_join": 1,
        "filtered": "100.00",
        "cost_info": {
          "read_cost": "0.00",
          "eval_cost": "0.20",
          "prefix_cost": "0.00",
          "data_read_per_join": "8"
        } /* cost_info */,
        "used_columns": [
          "i"
        ] /* used_columns */
      } /* table */
    } /* insert_from */,
    "update_value_subqueries": [
      {
        "dependent": false,
        "cacheable": true,
        "query_block": {
          "select_id": 2,
          "cost_info": {
            "query_cost": "1.00"
          } /* cost_info */,
          "table": {
            "table_name": "t2",
            "access_type": "system",
            "rows_examined_per_scan": 1,
            "rows_produced_per_join": 1,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "0.00",
              "eval_cost": "0.20",
              "prefix_cost": "0.00",
              "data_read_per_join": "8"
            } /* cost_info */,
            "used_columns": [
              "i"
            ] /* used_columns */
          } /* table */
        } /* query_block */
      }
    ] /* update_value_subqueries */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "table": {
      "insert": true,
      "table_name": "t1",
      "access_type": "ALL"
    } /* table */,
    "update_value_subqueries": [
      {
        "dependent": false,
        "cacheable": true,
        "query_block": {
          "select_id": 2,
          "cost_info": {
            "query_cost": "1.00"
          } /* cost_info */,
          "table": {
            "table_name": "t2",
            "access_type": "system",
            "rows_examined_per_scan": 1,
            "rows_produced_per_join": 1,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "0.00",
              "eval_cost": "0.20",
              "prefix_cost": "0.00",
              "data_read_per_join": "8"
            } /* cost_info */,
            "used_columns": [
              "i"
            ] /* used_columns */
          } /* table */
        } /* query_block */
      }
    ] /* update_value_subqueries */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "table": {
      "insert": true,
      "table_name": "t3",
      "access_type": "ALL"
    } /* table */,
    "optimized_away_subqueries": [
      {
        "dependent": false,
        "cacheable": true,
        "query_block": {
          "select_id": 3,
          "cost_info": {
            "query_cost": "1.00"
          } /* cost_info */,
          "table": {
            "table_name": "t2",
            "access_type": "system",
            "rows_examined_per_scan": 1,
            "rows_produced_per_join": 1,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "0.00",
              "eval_cost": "0.20",
              "prefix_cost": "0.00",
              "data_read_per_join": "8"
            } /* cost_info */,
            "used_columns": [
              "i"
            ] /* used_columns */
          } /* table */
        } /* query_block */
      },
      {
        "dependent": false,
        "cacheable": true,
        "query_block": {
          "select_id": 2,
          "cost_info": {
            "query_cost": "1.00"
          } /* cost_info */,
          "table": {
            "table_name": "t1",
            "access_type": "system",
            "rows_examined_per_scan": 1,
            "rows_produced_per_join": 1,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "0.00",
              "eval_cost": "0.20",
              "prefix_cost": "0.00",
              "data_read_per_join": "8"
            } /* cost_info */,
            "used_columns": [
              "i"
            ] /* used_columns */
          } /* table */
        } /* query_block */
      }
    ] /* optimized_away_subqueries */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "10.50"
    } /* cost_info */,
    "ordering_operation": {
      "using_filesort": true,
      "grouping_operation": {
        "using_temporary_table": true,
        "using_filesort": false,
        "table": {
          "table_name": "t1",
          "access_type": "ALL",
          "rows_examined_per_scan": 2,
          "rows_produced_per_join": 2,
          "filtered": "100.00",
          "cost_info": {
            "read_cost": "10.10",
            "eval_cost": "0.40",
            "prefix_cost": "10.50",
            "data_read_per_join": "48"
          } /* cost_info */,
          "used_columns": [
            "a",
            "b"
          ] /* used_columns */,
          "materialized_from_subquery": {
            "using_temporary_table": true,
            "dependent": false,
            "cacheable": true,
            "query_block": {
              "union_result": {
                "using_temporary_table": false,
                "query_specifications": [
                  {
                    "dependent": false,
                    "cacheable": true,
                    "query_block": {
                      "select_id": 2,
                      "message": "No tables used"
                    } /* query_block */
                  },
                  {
                    "dependent": false,
                    "cacheable": true,
                    "query_block": {
                      "select_id": 3,
                      "message": "No tables used"
                    } /* query_block */
                  }
                ] /* query_specifications */
              } /* union_result */
            } /* query_block */
          } /* materialized_from_subquery */
        } /* table */
      } /* grouping_operation */
    } /* ordering_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "4.40"
    } /* cost_info */,
    "grouping_operation": {
      "using_temporary_table": true,
      "using_filesort": true,
      "cost_info": {
        "sort_cost": "2.00"
      } /* cost_info */,
      "table": {
        "table_name": "t1",
        "access_type": "ALL",
        "rows_examined_per_scan": 2,
        "rows_produced_per_join": 2,
        "filtered": "100.00",
        "cost_info": {
          "read_cost": "2.00",
          "eval_cost": "0.40",
          "prefix_cost": "2.40",
          "data_read_per_join": "32"
        } /* cost_info */,
        "used_columns": [
          "a"
        ] /* used_columns */
      } /* table */,
      "group_by_subqueries": [
        {
          "dependent": false,
          "cacheable": true,
          "query_block": {
            "select_id": 2,
            "cost_info": {
              "query_cost": "1.00"
            } /* cost_info */,
            "table": {
              "table_name": "d",
              "access_type": "system",
              "rows_examined_per_scan": 1,
              "rows_produced_per_join": 1,
              "filtered": "100.00",
              "cost_info": {
                "read_cost": "0.00",
                "eval_cost": "0.20",
                "prefix_cost": "0.00",
                "data_read_per_join": "16"
              } /* cost_info */,
              "used_columns": [
                "b"
              ] /* used_columns */,
              "materialized_from_subquery": {
                "using_temporary_table": true,
                "dependent": false,
                "cacheable": true,
                "query_block": {
                  "select_id": 3,
                  "cost_info": {
                    "query_cost": "5.21"
                  } /* cost_info */,
                  "ordering_operation": {
                    "using_temporary_table": true,
                    "using_filesort": true,
                    "nested_loop": [
                      {
                        "table": {
                          "table_name": "t1",
                          "access_type": "ALL",
                          "rows_examined_per_scan": 2,
                          "rows_produced_per_join": 2,
                          "filtered": "100.00",
                          "cost_info": {
                            "read_cost": "2.00",
                            "eval_cost": "0.40",
                            "prefix_cost": "2.40",
                            "data_read_per_join": "32"
                          } /* cost_info */,
                          "used_columns": [
                            "a",
                            "b"
                          ] /* used_columns */
                        } /* table */
                      },
                      {
                        "table": {
                          "table_name": "t2",
                          "access_type": "ALL",
                          "rows_examined_per_scan": 2,
                          "rows_produced_per_join": 4,
                          "filtered": "100.00",
                          "using_join_buffer": "Block Nested Loop",
                          "cost_info": {
                            "read_cost": "2.00",
                            "eval_cost": "0.80",
                            "prefix_cost": "5.21",
                            "data_read_per_join": "64"
                          } /* cost_info */
                        } /* table */
                      }
                    ] /* nested_loop */
                  } /* ordering_operation */
                } /* query_block */
              } /* materialized_from_subquery */
            } /* table */
          } /* query_block */
        }
      ] /* group_by_subqueries */
    } /* grouping_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "2.40"
    } /* cost_info */,
    "table": {
      "table_name": "t1",
      "access_type": "ALL",
      "rows_examined_per_scan": 2,
      "rows_produced_per_join": 2,
      "filtered": "100.00",
      "cost_info": {
        "read_cost": "2.00",
        "eval_cost": "0.40",
        "prefix_cost": "2.40",
        "data_read_per_join": "16"
      } /* cost_info */
    } /* table */,
    "optimized_away_subqueries": [
      {
        "dependent": false,
        "cacheable": true,
        "query_block": {
          "select_id": 3,
          "cost_info": {
            "query_cost": "4.40"
          } /* cost_info */,
          "grouping_operation": {
            "using_temporary_table": true,
            "using_filesort": true,
            "cost_info": {
              "sort_cost": "2.00"
            } /* cost_info */,
            "table": {
              "table_name": "t1",
              "access_type": "ALL",
              "rows_examined_per_scan": 2,
              "rows_produced_per_join": 2,
              "filtered": "100.00",
              "cost_info": {
                "read_cost": "2.00",
                "eval_cost": "0.40",
                "prefix_cost": "2.40",
                "data_read_per_join": "16"
              } /* cost_info */,
              "used_columns": [
                "f1"
              ] /* used_columns */
            } /* table */
          } /* grouping_operation */
        } /* query_block */
      }
    ] /* optimized_away_subqueries */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "4.02"
    } /* cost_info */,
    "ordering_operation": {
      "using_filesort": true,
      "table": {
        "table_name": "t1",
        "access_type": "ALL",
        "rows_examined_per_scan": 10,
        "rows_produced_per_join": 10,
        "filtered": "100.00",
        "cost_info": {
          "read_cost": "2.02",
          "eval_cost": "2.00",
          "prefix_cost": "4.02",
          "data_read_per_join": "80"
        } /* cost_info */,
        "used_columns": [
          "i"
        ] /* used_columns */
      } /* table */,
      "order_by_subqueries": [
        {
          "dependent": true,
          "cacheable": false,
          "query_block": {
            "select_id": 2,
            "cost_info": {
              "query_cost": "4.02"
            } /* cost_info */,
            "table": {
              "table_name": "t2",
              "access_type": "ALL",
              "rows_examined_per_scan": 10,
              "rows_produced_per_join": 1,
              "filtered": "10.00",
              "cost_info": {
                "read_cost": "2.02",
                "eval_cost": "0.20",
                "prefix_cost": "4.02",
                "data_read_per_join": "16"
              } /* cost_info */,
              "used_columns": [
                "i",
                "j"
              ] /* used_columns */,
              "attached_condition": "(test.t2.i = test.t1.i)"
            } /* table */
          } /* query_block */
        }
      ] /* order_by_subqueries */
    } /* ordering_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "4.02"
    } /* cost_info */,
    "grouping_operation": {
      "using_temporary_table": true,
      "using_filesort": true,
      "table": {
        "table_name": "t1",
        "access_type": "ALL",
        "rows_examined_per_scan": 10,
        "rows_produced_per_join": 10,
        "filtered": "100.00",
        "cost_info": {
          "read_cost": "2.02",
          "eval_cost": "2.00",
          "prefix_cost": "4.02",
          "data_read_per_join": "80"
        } /* cost_info */,
        "used_columns": [
          "i"
        ] /* used_columns */
      } /* table */,
      "group_by_subqueries": [
        {
          "dependent": true,
          "cacheable": false,
          "query_block": {
            "select_id": 2,
            "cost_info": {
              "query_cost": "4.02"
            } /* cost_info */,
            "table": {
              "table_name": "t2",
              "access_type": "ALL",
              "rows_examined_per_scan": 10,
              "rows_produced_per_join": 1,
              "filtered": "10.00",
              "cost_info": {
                "read_cost": "2.02",
                "eval_cost": "0.20",
                "prefix_cost": "4.02",
                "data_read_per_join": "16"
              } /* cost_info */,
              "used_columns": [
                "i",
                "j"
              ] /* used_columns */,
              "attached_condition": "(test.t2.i = test.t1.i)"
            } /* table */
          } /* query_block */
        }
      ] /* group_by_subqueries */
    } /* grouping_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "6.50"
    } /* cost_info */,
    "ordering_operation": {
      "using_temporary_table": true,
      "using_filesort": true,
      "grouping_operation": {
        "using_filesort": false,
        "table": {
          "table_name": "t1",
          "access_type": "range",
          "possible_keys": [
            "k1"
          ] /* possible_keys */,
          "key": "k1",
          "used_key_parts": [
            "a"
          ] /* used_key_parts */,
          "key_length": "4",
          "rows_examined_per_scan": 11,
          "rows_produced_per_join": 11,
          "filtered": "100.00",
          "using_index_for_group_by": true,
          "cost_info": {
            "read_cost": "4.30",
            "eval_cost": "2.20",
            "prefix_cost": "6.50",
            "data_read_per_join": "176"
          } /* cost_info */,
          "used_columns": [
            "a",
            "b"
          ] /* used_columns */
        } /* table */
      } /* grouping_operation */
    } /* ordering_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "6.20"
    } /* cost_info */,
    "ordering_operation": {
      "using_temporary_table": true,
      "using_filesort": true,
      "grouping_operation": {
        "using_filesort": true,
        "nested_loop": [
          {
            "table": {
              "table_name": "t1",
              "access_type": "ALL",
              "possible_keys": [
                "PRIMARY"
              ] /* possible_keys */,
              "rows_examined_per_scan": 3,
              "rows_produced_per_join": 3,
              "filtered": "100.00",
              "cost_info": {
                "read_cost": "2.01",
                "eval_cost": "0.60",
                "prefix_cost": "2.61",
                "data_read_per_join": "48"
              } /* cost_info */,
              "used_columns": [
                "a",
                "b"
              ] /* used_columns */
            } /* table */
          },
          {
            "table": {
              "table_name": "t2",
              "access_type": "ref",
              "possible_keys": [
                "PRIMARY"
              ] /* possible_keys */,
              "key": "PRIMARY",
              "used_key_parts": [
                "a"
              ] /* used_key_parts */,
              "key_length": "4",
              "ref": [
                "test.t1.a"
              ] /* ref */,
              "rows_examined_per_scan": 1,
              "rows_produced_per_join": 3,
              "filtered": "100.00",
              "using_index": true,
              "cost_info": {
                "read_cost": "3.00",
                "eval_cost": "0.60",
                "prefix_cost": "6.21",
                "data_read_per_join": "48"
              } /* cost_info */,
              "used_columns": [
                "a",
                "b"
              ] /* used_columns */
            } /* table */
          }
        ] /* nested_loop */
      } /* grouping_operation */
    } /* ordering_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "12.82"
    } /* cost_info */,
    "grouping_operation": {
      "using_filesort": true,
      "cost_info": {
        "sort_cost": "9.00"
      } /* cost_info */,
      "table": {
        "table_name": "t1",
        "access_type": "ALL",
        "rows_examined_per_scan": 9,
        "rows_produced_per_join": 9,
        "filtered": "100.00",
        "cost_info": {
          "read_cost": "2.02",
          "eval_cost": "1.80",
          "prefix_cost": "3.82",
          "data_read_per_join": "144"
        } /* cost_info */,
        "used_columns": [
          "a",
          "b"
        ] /* used_columns */
      } /* table */
    } /* grouping_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "3.01"
    } /* cost_info */,
    "ordering_operation": {
      "using_filesort": true,
      "duplicates_removal": {
        "using_temporary_table": true,
        "using_filesort": false,
        "grouping_operation": {
          "using_temporary_table": true,
          "using_filesort": false,
          "table": {
            "table_name": "t1",
            "access_type": "ALL",
            "rows_examined_per_scan": 5,
            "rows_produced_per_join": 5,
            "filtered": "100.00",
            "cost_info": {
              "read_cost": "2.01",
              "eval_cost": "1.00",
              "prefix_cost": "3.01",
              "data_read_per_join": "80"
            } /* cost_info */,
            "used_columns": [
              "a",
              "b"
            ] /* used_columns */
          } /* table */
        } /* grouping_operation */
      } /* duplicates_removal */
    } /* ordering_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "2.40"
    } /* cost_info */,
    "ordering_operation": {
      "using_filesort": false,
      "duplicates_removal": {
        "using_temporary_table": true,
        "using_filesort": false,
        "buffer_result": {
          "using_temporary_table": true,
          "nested_loop": [
            {
              "table": {
                "table_name": "t1",
                "access_type": "system",
                "rows_examined_per_scan": 1,
                "rows_produced_per_join": 1,
                "filtered": "100.00",
                "cost_info": {
                  "read_cost": "0.00",
                  "eval_cost": "0.20",
                  "prefix_cost": "0.00",
                  "data_read_per_join": "8"
                } /* cost_info */,
                "used_columns": [
                  "a"
                ] /* used_columns */
              } /* table */
            },
            {
              "table": {
                "table_name": "t2",
                "access_type": "index",
                "key": "PRIMARY",
                "used_key_parts": [
                  "a"
                ] /* used_key_parts */,
                "key_length": "4",
                "rows_examined_per_scan": 2,
                "rows_produced_per_join": 2,
                "filtered": "100.00",
                "using_index": true,
                "distinct": true,
                "cost_info": {
                  "read_cost": "2.00",
                  "eval_cost": "0.40",
                  "prefix_cost": "2.40",
                  "data_read_per_join": "16"
                } /* cost_info */,
                "used_columns": [
                  "a"
                ] /* used_columns */
              } /* table */
            }
          ] /* nested_loop */
        } /* buffer_result */
      } /* duplicates_removal */
    } /* ordering_operation */
  } /* query_block */
}`,
	`{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "6.41"
    } /* cost_info */,
    "nested_loop": [
      {
        "table": {
          "table_name": "t1",
          "access_type": "ALL",
          "possible_keys": [
            "PRIMARY"
          ] /* possible_keys */,
          "rows_examined_per_scan": 4,
          "rows_produced_per_join": 3,
          "filtered": "75.00",
          "cost_info": {
            "read_cost": "2.21",
            "eval_cost": "0.60",
            "prefix_cost": "2.81",
            "data_read_per_join": "48"
          } /* cost_info */,
          "used_columns": [
            "a",
            "b"
          ] /* used_columns */,
          "attached_condition": "(test.t1.b <> 30)"
        } /* table */
      },
      {
        "table": {
          "table_name": "t2",
          "access_type": "eq_ref",
          "possible_keys": [
            "PRIMARY"
          ] /* possible_keys */,
          "key": "PRIMARY",
          "used_key_parts": [
            "a"
          ] /* used_key_parts */,
          "key_length": "4",
          "ref": [
            "test.t1.a"
          ] /* ref */,
          "rows_examined_per_scan": 1,
          "rows_produced_per_join": 3,
          "filtered": "100.00",
          "using_index": true,
          "cost_info": {
            "read_cost": "3.00",
            "eval_cost": "0.60",
            "prefix_cost": "6.41",
            "data_read_per_join": "24"
          } /* cost_info */,
          "used_columns": [
            "a"
          ] /* used_columns */
        } /* table */
      }
    ] /* nested_loop */
  } /* query_block */
}`,
}

func TestExplain(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	// TraditionalFormatExplain
	for idx, sql := range sqls {
		exp, err := connTest.Explain(sql, TraditionalExplainType, TraditionalFormatExplain)
		if err != nil {
			t.Error(err)
		}
		pretty.Println("No.:", idx, "\nOld: ", sql, "\nNew: ", exp.SQL)
		pretty.Println(exp)
	}
	// JSONFormatExplain
	for idx, sql := range sqls {
		exp, err := connTest.Explain(sql, TraditionalExplainType, JSONFormatExplain)
		if err != nil {
			t.Error(err)
		}
		pretty.Println("No.:", idx, "\nOld: ", sql, "\nNew: ", exp.SQL)
		pretty.Println(exp)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestParseExplainText(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	for _, content := range exp {
		pretty.Println(RemoveSQLComments(content))
		pretty.Println(ParseExplainText(content))
	}
	/*
		//length := len(exp)
		pretty.Println(string(RemoveSQLComments([]byte(exp[9]))))
		explainInfo, err := ParseExplainText(exp[9])
		pretty.Println(explainInfo)
		fmt.Println(err)
	*/
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFindTablesInJson(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	idx := 9
	for _, j := range exp[idx : idx+1] {
		pretty.Println(j)
		findTablesInJSON(j, 0)
	}
	pretty.Println(len(explainJSONTables), explainJSONTables)
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestFormatJsonIntoTraditional(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	idx := 11
	for _, j := range exp[idx : idx+1] {
		pretty.Println(j)
		pretty.Println(FormatJSONIntoTraditional(j))
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestPrintMarkdownExplainTable(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	expInfo, err := connTest.Explain("select 1", TraditionalExplainType, TraditionalFormatExplain)
	if err != nil {
		t.Error(err)
	}

	err = common.GoldenDiff(func() {
		PrintMarkdownExplainTable(expInfo)
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestExplainInfoTranslator(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	expInfo, err := connTest.Explain("select 1", TraditionalExplainType, TraditionalFormatExplain)
	if err != nil {
		t.Error(err)
	}
	err = common.GoldenDiff(func() {
		ExplainInfoTranslator(expInfo)
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestMySQLExplainWarnings(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	expInfo, err := connTest.Explain("select 1", TraditionalExplainType, TraditionalFormatExplain)
	if err != nil {
		t.Error(err)
	}
	err = common.GoldenDiff(func() {
		MySQLExplainWarnings(expInfo)
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestMySQLExplainQueryCost(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	err := common.GoldenDiff(func() {
		expInfo, err := connTest.Explain("select 1", TraditionalExplainType, TraditionalFormatExplain)
		fmt.Println(err, MySQLExplainQueryCost(expInfo))
		expInfo, err = connTest.Explain("select 1", ExtendedExplainType, TraditionalFormatExplain)
		fmt.Println(err, MySQLExplainQueryCost(expInfo))
		expInfo, err = connTest.Explain("select 1", TraditionalExplainType, JSONFormatExplain)
		fmt.Println(err, MySQLExplainQueryCost(expInfo))
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestSupportExplainWrite(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	_, err := connTest.supportExplainWrite()
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestExplainAbleSQL(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	for _, sql := range sqls {
		if _, err := connTest.explainAbleSQL(sql); err != nil {
			t.Errorf("SQL: %s, not explain able", sql)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
