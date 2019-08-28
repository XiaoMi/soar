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

package ast

import (
	"fmt"
	"sort"
	"testing"

	"github.com/XiaoMi/soar/common"
)

func TestRewrite(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgTestDSNStatus := common.Config.TestDSN.Disable
	common.Config.TestDSN.Disable = false
	testSQL := []map[string]string{
		{
			"input":  `SELECT * FROM film`,
			"output": `select film.film_id, film.title, film.description, film.release_year, film.language_id, film.original_language_id, film.rental_duration from film;`,
		},
		{
			"input":  `SELECT film.*, actor.actor_id FROM film,actor`,
			"output": `select film.film_id, film.title, film.description, film.release_year, film.language_id, film.original_language_id, film.rental_duration, actor.actor_id from film, actor;`,
		},
		{
			"input":  `insert into film values(1,2,3,4,5)`,
			"output": `insert into film(film_id, title, description, release_year, language_id) values (1, 2, 3, 4, 5);`,
		},
		{
			"input":  `insert into sakila.film values(1,2)`,
			"output": `insert into sakila.film(film_id, title) values (1, 2);`,
		},
		{
			"input":  `replace into sakila.film select id from tb`,
			"output": `replace into sakila.film(film_id) select id from tb;`,
		},
		{
			"input":  `replace into sakila.film select id, title, description from tb`,
			"output": `replace into sakila.film(film_id, title, description) select id, title, description from tb;`,
		},
		{
			"input":  `insert into film values(1,2,3,4,5)`,
			"output": `insert into film(film_id, title, description, release_year, language_id) values (1, 2, 3, 4, 5);`,
		},
		{
			"input":  `insert into sakila.film values(1,2)`,
			"output": `insert into sakila.film(film_id, title) values (1, 2);`,
		},
		{
			"input":  `replace into sakila.film select id from tb`,
			"output": `replace into sakila.film(film_id) select id from tb;`,
		},
		{
			"input":  `replace into sakila.film select id, title, description from tb`,
			"output": `replace into sakila.film(film_id, title, description) select id, title, description from tb;`,
		},
		{
			"input":  "DELETE FROM tbl WHERE col1=1 ORDER BY col",
			"output": "delete from tbl where col1 = 1;",
		},
		{
			"input":  "UPDATE tbl SET col =1 WHERE col1=1 ORDER BY col",
			"output": "update tbl set col = 1 where col1 = 1;",
		},
	}

	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"])
		rw.Columns = map[string]map[string][]*common.Column{
			"sakila": {
				"film": {
					{Name: "film_id", Table: "film"},
					{Name: "title", Table: "film"},
					{Name: "description", Table: "film"},
					{Name: "release_year", Table: "film"},
					{Name: "language_id", Table: "film"},
					{Name: "original_language_id", Table: "film"},
					{Name: "rental_duration", Table: "film"},
				},
			},
		}
		rw.Rewrite()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Config.TestDSN.Disable = orgTestDSNStatus
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteStar2Columns(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgTestDSNStatus := common.Config.TestDSN.Disable
	common.Config.TestDSN.Disable = false
	testSQL := []map[string]string{
		{
			"input":  `SELECT * FROM film`,
			"output": `select film_id, title from film`,
		},
		{
			"input":  `SELECT film.* FROM film`,
			"output": `select film_id, title from film`,
		},
	}

	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"])
		rw.Columns = map[string]map[string][]*common.Column{
			"sakila": {
				"film": {
					{Name: "film_id", Table: "film"},
					{Name: "title", Table: "film"},
				},
			},
		}
		rw.RewriteStar2Columns()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}

	testSQL2 := []map[string]string{
		{
			"input":  `SELECT film.* FROM film, actor`,
			"output": `select film.film_id, film.title from film, actor`,
		},
		{
			"input":  `SELECT film.*, actor.actor_id FROM film, actor`,
			"output": `select film.film_id, film.title, actor.actor_id from film, actor`,
		},
	}

	for _, sql := range testSQL2 {
		rw := NewRewrite(sql["input"])
		rw.Columns = map[string]map[string][]*common.Column{
			"sakila": {
				"film": {
					{Name: "film_id", Table: "film"},
					{Name: "title", Table: "film"},
				},
				"actor": {
					{Name: "actor_id", Table: "actor"},
				},
			},
		}
		rw.RewriteStar2Columns()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Config.TestDSN.Disable = orgTestDSNStatus
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteInsertColumns(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  `insert into film values(1,2,3,4,5)`,
			"output": `insert into film(film_id, title, description, release_year, language_id) values (1, 2, 3, 4, 5)`,
		},
		{
			"input":  `insert into sakila.film values(1,2)`,
			"output": `insert into sakila.film(film_id, title) values (1, 2)`,
		},
		{
			"input":  `replace into sakila.film select id from tb`,
			"output": `replace into sakila.film(film_id) select id from tb`,
		},
		{
			"input":  `replace into sakila.film select id, title, description from tb`,
			"output": `replace into sakila.film(film_id, title, description) select id, title, description from tb`,
		},
	}

	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"])
		rw.Columns = map[string]map[string][]*common.Column{
			"sakila": {
				"film": {
					{Name: "film_id", Table: "film"},
					{Name: "title", Table: "film"},
					{Name: "description", Table: "film"},
					{Name: "release_year", Table: "film"},
					{Name: "language_id", Table: "film"},
					{Name: "original_language_id", Table: "film"},
					{Name: "rental_duration", Table: "film"},
				},
			},
		}
		rw.RewriteInsertColumns()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteHaving(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  `SELECT state, COUNT(*) FROM Drivers GROUP BY state HAVING state IN ('GA', 'TX') ORDER BY state`,
			"output": "select state, COUNT(*) from Drivers where state in ('GA', 'TX') group by state order by state asc",
		},
		{
			"input":  `SELECT state, COUNT(*) FROM Drivers WHERE col =1 GROUP BY state HAVING state IN ('GA', 'TX') ORDER BY state`,
			"output": "select state, COUNT(*) from Drivers where (col = 1) and state in ('GA', 'TX') group by state order by state asc",
		},
		{
			"input":  `SELECT state, COUNT(*) FROM Drivers WHERE col =1 or col1 =2 GROUP BY state HAVING state IN ('GA', 'TX') ORDER BY state`,
			"output": "select state, COUNT(*) from Drivers where (col = 1 or col1 = 2) and state in ('GA', 'TX') group by state order by state asc",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteHaving()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteAddOrderByNull(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  "SELECT sum(col1) FROM tbl GROUP BY col",
			"output": "select sum(col1) from tbl group by col order by null",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteAddOrderByNull()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteRemoveDMLOrderBy(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  "DELETE FROM tbl WHERE col1=1 ORDER BY col",
			"output": "delete from tbl where col1 = 1",
		},
		{
			"input":  "UPDATE tbl SET col =1 WHERE col1=1 ORDER BY col",
			"output": "update tbl set col = 1 where col1 = 1",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteRemoveDMLOrderBy()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteGroupByConst(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  "select 1;",
			"output": "select 1 from dual",
		},
		/*
				{
					"input":  "SELECT col1 FROM tbl GROUP BY 1;",
					"output": "select col1 from tbl GROUP BY col1",
				},
			    {
					"input":  "SELECT col1, col2 FROM tbl GROUP BY 1, 2;",
					"output": "select col1, col2 from tbl GROUP BY col1, col2",
				},
			    {
					"input":  "SELECT col1, col2, col3 FROM tbl GROUP BY 1, 3;",
					"output": "select col1, col2, col3 from tbl GROUP BY col1, col3",
				},
		*/
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteGroupByConst()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteStandard(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  "SELECT sum(col1) FROM tbl GROUP BY 1;",
			"output": "select sum(col1) from tbl group by 1",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteStandard()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteCountStar(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  "SELECT count(col) FROM tbl GROUP BY 1;",
			"output": "select count(*) from tbl group by 1",
		},
		{
			"input":  "SELECT COUNT(tb.col) FROM tbl GROUP BY 1;",
			"output": "select COUNT(tb.*) from tbl group by 1",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteCountStar()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteInnoDB(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  "CREATE TABLE t1(id bigint(20) NOT NULL AUTO_INCREMENT);",
			"output": "create table t1 (\n\tid bigint(20) not null auto_increment\n) ENGINE=InnoDB ",
		},
		{
			"input":  "create table t1 (\n\tid bigint(20) not null auto_increment\n) ENGINE=memory ",
			"output": "create table t1 (\n\tid bigint(20) not null auto_increment\n) ENGINE=InnoDB ",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteInnoDB()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteAutoIncrement(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  "CREATE TABLE t1(id bigint(20) NOT NULL AUTO_INCREMENT) ENGINE=InnoDB AUTO_INCREMENT=123802;",
			"output": "create table t1 (\n\tid bigint(20) not null auto_increment\n) ENGINE=InnoDB auto_increment=1 ",
		},
		{
			"input":  "create table t1 (\n\tid bigint(20) not null auto_increment\n) ENGINE=InnoDB",
			"output": "create table t1 (\n\tid bigint(20) not null auto_increment\n) ENGINE=InnoDB",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteAutoIncrement()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteIntWidth(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  "CREATE TABLE t1(id bigint(10) NOT NULL AUTO_INCREMENT) ENGINE=InnoDB AUTO_INCREMENT=123802;",
			"output": "create table t1 (\n\tid bigint(20) not null auto_increment\n) ENGINE=InnoDB auto_increment=123802",
		},
		{
			"input":  "CREATE TABLE t1(id bigint NOT NULL AUTO_INCREMENT) ENGINE=InnoDB AUTO_INCREMENT=123802;",
			"output": "create table t1 (\n\tid bigint(20) not null auto_increment\n) ENGINE=InnoDB auto_increment=123802",
		},
		{
			"input":  "create table t1(id int(20) not null auto_increment) ENGINE=InnoDB;",
			"output": "create table t1 (\n\tid int(10) not null auto_increment\n) ENGINE=InnoDB",
		},
		{
			"input":  "create table t1(id int not null auto_increment) ENGINE=InnoDB;",
			"output": "create table t1 (\n\tid int not null auto_increment\n) ENGINE=InnoDB",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteIntWidth()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteAlwaysTrue(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  "SELECT count(col) FROM tbl where 1=1;",
			"output": "select count(col) from tbl",
		},
		{
			"input":  "SELECT count(col) FROM tbl where col=col;",
			"output": "select count(col) from tbl where col = col",
		},
		{
			"input":  "SELECT count(col) FROM tbl where col=col2;",
			"output": "select count(col) from tbl where col = col2",
		},
		{
			"input":  "SELECT count(col) FROM tbl where 1>=1;",
			"output": "select count(col) from tbl",
		},
		{
			"input":  "SELECT count(col) FROM tbl where 2>1;",
			"output": "select count(col) from tbl",
		},
		{
			"input":  "SELECT count(col) FROM tbl where 1<=1;",
			"output": "select count(col) from tbl",
		},
		{
			"input":  "SELECT count(col) FROM tbl where 1<2;",
			"output": "select count(col) from tbl",
		},
		{
			"input":  "SELECT count(col) FROM tbl where 1=1 and 2=2;",
			"output": "select count(col) from tbl",
		},
		{
			"input":  "SELECT count(col) FROM tbl where 1=1 or 2=3;",
			"output": "select count(col) from tbl where 2 = 3",
		},
		{
			"input":  "SELECT count(col) FROM tbl where 1=1 and 3=3 or 2=3;",
			"output": "select count(col) from tbl where 2 = 3",
		},
		{
			"input":  "SELECT count(col) FROM tbl where 1=1 and 3=3 or 2!=3;",
			"output": "select count(col) from tbl",
		},
		{
			"input":  "SELECT count(col) FROM tbl where 1=1 or 2=3 and 3=3 ;",
			"output": "select count(col) from tbl where 2 = 3",
		},
		{
			"input":  "SELECT count(col) FROM tbl where (1=1);",
			"output": "select count(col) from tbl",
		},
		{
			"input":  "SELECT count(col) FROM tbl where a=1;",
			"output": "select count(col) from tbl where a = 1",
		},
		{
			"input":  "SELECT count(col) FROM tbl where ('a'= 'a' or 'b' = 'b') and a = 'b';",
			"output": "select count(col) from tbl where a = 'b'",
		},
		{
			"input":  "SELECT count(col) FROM tbl where (('a'= 'a' or 'b' = 'b') and a = 'b');",
			"output": "select count(col) from tbl where (a = 'b')",
		},
		{
			"input":  "SELECT count(col) FROM tbl where 'a'= 'a' or ('b' = 'b' and a = 'b');",
			"output": "select count(col) from tbl where (a = 'b')",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteAlwaysTrue()
		if rw == nil {
			t.Errorf("NoRw")
		} else if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

// TODO:
func TestRewriteSubQuery2Join(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	orgTestDSNStatus := common.Config.TestDSN.Disable
	common.Config.TestDSN.Disable = true
	testSQL := []map[string]string{
		{
			// 这个case是官方文档给的，但不一定正确，需要视表结构的定义来进行判断
			"input":  `SELECT * FROM t1 WHERE id IN (SELECT id FROM t2);`,
			"output": "",
			//"output": `SELECT DISTINCT t1.* FROM t1, t2 WHERE t1.id=t2.id;`,
		},
		{
			"input":  `SELECT * FROM t1 WHERE id NOT IN (SELECT id FROM t2);`,
			"output": "",
			//"output": `SELECT table1.* FROM t1 LEFT JOIN t2 ON t1.id=t2.id WHERE t2.id IS NULL;`,
		},
		{
			"input":  `SELECT * FROM t1 WHERE NOT EXISTS (SELECT id FROM t2 WHERE t1.id=t2.id);`,
			"output": "",
			//"output": `SELECT table1.* FROM table1 LEFT JOIN table2 ON table1.id=table2.id WHERE table2.id IS NULL;`,
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteSubQuery2Join()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Config.TestDSN.Disable = orgTestDSNStatus
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteDML2Select(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  "DELETE city, country FROM city INNER JOIN country using (country_id) WHERE city.city_id = 1;",
			"output": "select * from city join country using (country_id) where city.city_id = 1",
		}, {
			"input":  "DELETE city FROM city LEFT JOIN country ON city.country_id = country.country_id WHERE country.country IS NULL;",
			"output": "select * from city left join country on city.country_id = country.country_id where country.country is null",
		}, {
			"input":  "DELETE a1, a2 FROM city AS a1 INNER JOIN country AS a2 WHERE a1.country_id=a2.country_id",
			"output": "select * from city as a1 join country as a2 where a1.country_id = a2.country_id",
		}, {
			"input":  "DELETE FROM a1, a2 USING city AS a1 INNER JOIN country AS a2 WHERE a1.country_id=a2.country_id",
			"output": "select * from city as a1 join country as a2 where a1.country_id = a2.country_id",
		}, {
			"input":  "DELETE FROM film WHERE length > 100;",
			"output": "select * from film where length > 100",
		}, {
			"input":  "UPDATE city INNER JOIN country USING(country_id) SET city.city = 'Abha', city.last_update = '2006-02-15 04:45:25', country.country = 'Afghanistan' WHERE city.city_id=10;",
			"output": "select * from city join country using (country_id) where city.city_id = 10",
		}, {
			"input":  "UPDATE city INNER JOIN country ON city.country_id = country.country_id INNER JOIN address ON city.city_id = address.city_id SET city.city = 'Abha', city.last_update = '2006-02-15 04:45:25', country.country = 'Afghanistan' WHERE city.city_id=10;",
			"output": "select * from city join country on city.country_id = country.country_id join address on city.city_id = address.city_id where city.city_id = 10",
		}, {
			"input":  "UPDATE city, country SET city.city = 'Abha', city.last_update = '2006-02-15 04:45:25', country.country = 'Afghanistan' WHERE city.country_id = country.country_id AND city.city_id=10;",
			"output": "select * from city, country where city.country_id = country.country_id and city.city_id = 10",
		}, {
			"input":  "UPDATE film SET length = 10 WHERE language_id = 20;",
			"output": "select * from film where language_id = 20",
		}, {
			"input":  "INSERT INTO city (country_id) SELECT country_id FROM country;",
			"output": "select country_id from country",
		}, {
			"input":  "INSERT INTO city (country_id) VALUES (1),(2),(3);",
			"output": "select 1 from DUAL",
		}, {
			"input":  "INSERT INTO city (country_id) VALUES (10);",
			"output": "select 1 from DUAL",
		}, {
			"input":  "INSERT INTO city (country_id) SELECT 10 FROM DUAL;",
			"output": "select 10 from dual",
		}, {
			"input":  "replace INTO city (country_id) SELECT 10 FROM DUAL;",
			"output": "select 10 from dual",
		},
	}

	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteDML2Select()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteDistinctStar(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  `SELECT DISTINCT * FROM film;`,
			"output": "SELECT * FROM film;",
		},
		{
			"input":  `SELECT COUNT(DISTINCT *) FROM film;`,
			"output": "SELECT COUNT(*) FROM film;",
		},
		{
			"input":  `SELECT DISTINCT film.* FROM film;`,
			"output": "SELECT * FROM film;",
		},
		{
			"input":  "SELECT DISTINCT col FROM film;",
			"output": "SELECT DISTINCT col FROM film;",
		},
		{
			"input":  "SELECT DISTINCT film.* FROM film, tbl;",
			"output": "SELECT DISTINCT film.* FROM film, tbl;",
		},
		{

			"input":  "SELECT DISTINCT * FROM film, tbl;",
			"output": "SELECT DISTINCT * FROM film, tbl;",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteDistinctStar()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestMergeAlterTables(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	sqls := []string{
		// ADD|DROP INDEX
		// TODO: PRIMARY KEY, [UNIQUE|FULLTEXT|SPATIAL] INDEX
		"CREATE INDEX part_of_name ON customer (name(10));",
		"create index idx_test_cca on test_bb(test_cc);", // https://github.com/XiaoMi/soar/issues/205
		"alter table `sakila`.`t1` add index `idx_col`(`col`)",
		"alter table `sakila`.`t1` add UNIQUE index `idx_col`(`col`)",
		"alter table `sakila`.`t1` add index `idx_ID`(`ID`)",

		// ADD|DROP COLUMN
		"ALTER TABLE t2 DROP COLUMN c, DROP COLUMN d;",
		"ALTER TABLE T2 ADD COLUMN C int;",
		"ALTER TABLE T2 ADD COLUMN D int FIRST;",
		"ALTER TABLE T2 ADD COLUMN E int AFTER D;",

		// RENAME COLUMN
		"ALTER TABLE t1 RENAME COLUMN a TO b",

		// RENAME INDEX
		"ALTER TABLE t1 RENAME INDEX idx_a TO idx_b",
		"ALTER TABLE t1 RENAME KEY idx_a TO idx_b",

		// RENAME TABLE
		"ALTER TABLE db.old_table RENAME new_table;",
		"ALTER TABLE old_table RENAME TO new_table;",
		"ALTER TABLE old_table RENAME AS new_table;",

		// MODIFY & CHANGE
		"ALTER TABLE t1 MODIFY col1 BIGINT UNSIGNED DEFAULT 1 COMMENT 'my column';",
		"ALTER TABLE t1 CHANGE b a INT NOT NULL;",

		// table name quote in back ticks
		"alter table `t3`add index `idx_a`(a)",
		"alter table`t3`drop index`idx_b`",
	}

	alterSQLs := MergeAlterTables(sqls...)
	var sortedAlterSQLs []string
	for item := range alterSQLs {
		sortedAlterSQLs = append(sortedAlterSQLs, item)
	}
	sort.Strings(sortedAlterSQLs)

	err := common.GoldenDiff(func() {
		for _, tb := range sortedAlterSQLs {
			fmt.Println(tb, ":", alterSQLs[tb])
		}
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteUnionAll(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  `select country_id from city union select country_id from country;`,
			"output": "select country_id from city union all select country_id from country",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteUnionAll()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
func TestRewriteTruncate(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  `delete from tbl;`,
			"output": "truncate table tbl",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteTruncate()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRewriteOr2In(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  `select country_id from city where country_id = 1 or country_id = 2 or country_id = 3;`,
			"output": "select country_id from city where country_id in (1, 2, 3)",
		},
		// TODO or中的恒真条件
		{
			"input":  `select country_id from city where country_id != 1 or country_id != 2 or country_id = 3;`,
			"output": "select country_id from city where country_id != 1 or country_id != 2 or country_id = 3",
		},
		// col = 1 or col is null不可转为IN
		{
			"input":  `select country_id from city where col = 1 or col is null;`,
			"output": "select country_id from city where col = 1 or col is null",
		},
		{
			"input":  `select country_id from city where col1 = 1 or col2 = 1 or col2 = 2;`,
			"output": "select country_id from city where col1 = 1 or col2 in (1, 2)",
		},
		{
			"input":  `select country_id from city where col1 = 1 or col2 = 1 or col2 = 2 or col1 = 3;`,
			"output": "select country_id from city where col2 in (1, 2) or col1 in (1, 3)",
		},
		{
			"input":  `select country_id from city where (col1 = 1 or col2 = 1 or col2 = 2 ) or col1 = 3;`,
			"output": "select country_id from city where (col1 = 1 or col2 in (1, 2)) or col1 = 3",
		},
		{
			"input":  `select country_id from city where col1 = 1 or (col2 = 1 or col2 = 2 ) or col1 = 3;`,
			"output": "select country_id from city where (col2 in (1, 2)) or col1 in (1, 3)",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteOr2In()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestRmParenthesis(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	testSQL := []map[string]string{
		{
			"input":  `select country_id from city where (country_id = 1);`,
			"output": "select country_id from city where country_id = 1",
		},
		{
			"input":  `select * from city where a = 1 and (country_id = 1);`,
			"output": "select * from city where a = 1 and country_id = 1",
		},
		{
			"input":  `select country_id from city where (country_id = 1) or country_id = 1 ;`,
			"output": "select country_id from city where country_id = 1 or country_id = 1",
		},
		{
			"input":  `select country_id from city where col = 1 or (country_id = 1) or country_id = 1 ;`,
			"output": "select country_id from city where col = 1 or country_id = 1 or country_id = 1",
		},
	}
	for _, sql := range testSQL {
		rw := NewRewrite(sql["input"]).RewriteRmParenthesis()
		if rw.NewSQL != sql["output"] {
			t.Errorf("want: %s\ngot: %s", sql["output"], rw.NewSQL)
		}
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}

func TestListRewriteRules(t *testing.T) {
	common.Log.Debug("Entering function: %s", common.GetFunctionName())
	err := common.GoldenDiff(func() {
		ListRewriteRules(RewriteRules)
		orgReportType := common.Config.ReportType
		common.Config.ReportType = "json"
		ListRewriteRules(RewriteRules)
		common.Config.ReportType = orgReportType
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	common.Log.Debug("Exiting function: %s", common.GetFunctionName())
}
