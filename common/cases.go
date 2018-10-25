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

package common

// TestSQLs 测试SQL大集合
var TestSQLs []string

func init() {
	// 所有的SQL都要以分号结尾，-list-test-sqls参数会打印这个list，以分号结尾可方便测试
	// 如：./soar -list-test-sql | ./soar
	TestSQLs = []string{
		// single equality
		"SELECT * FROM film WHERE length = 86;",    // index(length)
		"SELECT * FROM film WHERE length IS NULL;", // index(length)
		"SELECT * FROM film HAVING title = 'abc';", // 无法使用索引

		// single inequality
		"SELECT * FROM sakila.film WHERE length >= 60;",   // any of <, <=, >=, >; but not <>, !=, IS NOT NULL"
		"SELECT * FROM sakila.film WHERE length >= '60';", // Implicit Conversion
		"SELECT * FROM film WHERE length BETWEEN 60 AND 84;",
		"SELECT * FROM film WHERE title LIKE 'AIR%';", // but not LIKE '%blah'",
		"SELECT * FROM film WHERE title IS NOT NULL;",

		// multiple equalities
		"SELECT * FROM film WHERE length = 114 and title = 'ALABAMA DEVIL';", // index(title,length) or index(length,title)",

		// equality and inequality
		"SELECT * FROM film WHERE length > 100 and title = 'ALABAMA DEVIL';", // index(title, length)",

		// multiple inequality
		"SELECT * FROM film WHERE length > 100 and language_id < 10 and title = 'xyz';", // index(d, b) or index(d, c) 依赖数据",
		"SELECT * FROM film WHERE length > 100 and language_id < 10;",                   // index(b) or index(c)",

		// GROUP BY
		"SELECT release_year, sum(length) FROM film WHERE length = 123 AND language_id = 1 GROUP BY release_year;",  // INDEX(length, language_id, release_year) or INDEX(language_id, length, release_year)",
		"SELECT release_year, sum(length) FROM film WHERE length >= 123 GROUP BY release_year;",                     // INDEX(length)",
		"SELECT release_year, language_id, sum(length) FROM film GROUP BY release_year, language_id;",               // INDEX(release_year, language_id) (no WHERE)",
		"SELECT release_year, sum(length) FROM film WHERE length = 123 GROUP BY release_year,(length+language_id);", // INDEX(length) expression in GROUP BY, so no use including even release_year.",
		"SELECT release_year, sum(film_id) FROM film GROUP BY release_year;",                                        // INDEX(`release_year`)
		"SELECT * FROM address GROUP BY address,district;",                                                          // INDEX(address, district)
		"SELECT title FROM film WHERE ABS(language_id) = 3 GROUP BY title;",                                         // 无法使用索引

		// ORDER BY
		"SELECT language_id FROM film WHERE length = 123 GROUP BY release_year ORDER BY language_id;",            //  INDEX(length, release_year) should have stopped with Step 2b",
		"SELECT release_year FROM film WHERE length = 123 GROUP BY release_year ORDER BY release_year;",          //  INDEX(length, release_year) the release_year will be used for both GROUP BY and ORDER BY",
		"SELECT * FROM film WHERE length = 123 ORDER BY release_year ASC, language_id DESC;",                     //  INDEX(length) mixture of ASC and DESC.",
		"SELECT release_year FROM film WHERE length = 123 GROUP BY release_year ORDER BY release_year LIMIT 10;", //  INDEX(length, release_year)",
		"SELECT * FROM film WHERE length = 123 ORDER BY release_year LIMIT 10;",                                  //  INDEX(length, release_year)",
		"SELECT * FROM film ORDER BY release_year LIMIT 10;",                                                     //  不能单独给release_year加索引
		"SELECT film_id FROM film ORDER BY release_year LIMIT 10;",                                               //  TODO: INDEX(release_year)，film_id是主键查询列满足索引覆盖的情况才会使用到release_year索引
		"SELECT * FROM film WHERE length > 100 ORDER BY length LIMIT 10;",                                        //  INDEX(length) This "range" is compatible with ORDER BY
		"SELECT * FROM film WHERE length < 100 ORDER BY length LIMIT 10;",                                        //  INDEX(length) also works
		"SELECT * FROM customer WHERE address_id in (224,510) ORDER BY last_name;",                               //  INDEX(address_id)
		"SELECT * FROM film WHERE release_year = 2016 AND length != 1 ORDER BY title;",                           //  INDEX(`release_year`, `length`, `title`)

		// "Covering" IdxRows
		"SELECT title FROM film WHERE release_year = 1995;",                               //  INDEX(release_year, title)",
		"SELECT title, replacement_cost FROM film WHERE language_id = 5 AND length = 70;", //  INDEX(language_id, length, title, replacement_cos film ), title, replacement_cost顺序无关，language_id, length顺序视散粒度情况.
		"SELECT title FROM film WHERE language_id > 5 AND length > 70;",                   //  INDEX(language_id, length, title) language_id or length first (that's as far as the Algorithm goes), then the other two fields afterwards.

		// equalities and sort
		"SELECT * FROM film WHERE length = 100 and title = 'xyz' ORDER BY release_year;", // 依赖数据特征，index(length, title, release_year) or index(title, length, release_year)需要评估

		// inequality and sort
		"SELECT * FROM film WHERE length > 100 and title = 'xyz' ORDER BY release_year;", // 依赖数据特征， index(title, release_year)，index(title, length)需要评估
		"SELECT * FROM film WHERE length > 100 ORDER BY release_year;",                   // 依赖数据特征， index(length)，index(release_year)需要评估

		// Join
		// 内连接 INNER JOIN
		// 在mysql中，inner join...on , join...on , 逗号...WHERE ，cross join...on是一样的含义。
		// 但是在标准SQL中，它们并不等价，标准SQL中INNER JOIN与ON共同使用, CROSS JOIN用于其他情况。
		// 逗号不支持on和using语法, 逗号的优先级要低于INNER JOIN, CROSS JOIN, LEFT JOIN
		// ON子句的语法格式为：tb1.col1 = tb2.col2列名可以不同，筛选连接后的结果，两表的对应列值相同才在结果集中。
		// 当模式设计对联接表的列采用了相同的命名样式时，就可以使用 USING 语法来简化 ON 语法

		// join, inner join, cross join等价，优先选择小结果集条件表为驱动表
		// left [outer] join左表为驱动表
		// right [outer] join右表为驱动表
		// 驱动表连接列如果没其他条件可以不考虑加索引，反正是需要foreach
		// 被驱动表连接列需要加索引。即:left [outer] join的右表连接列需要加索引，right [outer] join的左表连接列需要加索引，inner join结果集较大表的连接列需要加索引
		// 其他索引添加算法与单表索引优化算法相同
		// 总结：被驱动表列需要添加索引
		// 建议：将无索引的表通常作为驱动表

		"SELECT * FROM city a INNER JOIN country b ON a.country_id=b.country_id;",

		// 左外连接 LEFT [OUTER] JOIN
		"SELECT * FROM city a LEFT JOIN country b ON a.country_id=b.country_id;",

		// 右外连接 RIGHT [OUTER] JOIN
		"SELECT * FROM city a RIGHT JOIN country b ON a.country_id=b.country_id;",

		// 左连接
		"SELECT * FROM city a LEFT JOIN country b ON a.country_id=b.country_id WHERE b.last_update IS NULL;",

		// 右连接
		"SELECT * FROM city a RIGHT JOIN country b ON a.country_id=b.country_id WHERE a.last_update IS NULL;",

		// 全连接 FULL JOIN 因为在mysql中并不支持，所以我们用union实现
		"SELECT * FROM city a LEFT JOIN country b ON a.country_id=b.country_id " +
			"UNION " +
			"SELECT * FROM city a RIGHT JOIN country b ON a.country_id=b.country_id;",

		// 两张表中不共同满足的数据集
		"SELECT * FROM city a RIGHT JOIN country b ON a.country_id=b.country_id WHERE a.last_update IS NULL " +
			"UNION " +
			"SELECT * FROM city a LEFT JOIN country b ON a.country_id=b.country_id WHERE b.last_update IS NULL;",

		// NATURAL JOIN 默认是同名字段完全匹配的INNER JOIN
		"SELECT country_id, last_update FROM city NATURAL JOIN country;",

		// NATURAL LEFT JOIN
		"SELECT country_id, last_update FROM city NATURAL LEFT JOIN country;",

		// NATURAL RIGHT JOIN
		"SELECT country_id, last_update FROM city NATURAL RIGHT JOIN country;",

		// STRAIGHT_JOIN 实际上与内连接 INNER JOIN 表现完全一致，
		// 不同的是使用了 STRAIGHT_JOIN后指定表载入的顺序，city先于country载入
		"SELECT a.country_id, a.last_update FROM city a STRAIGHT_JOIN country b ON a.country_id=b.country_id;",

		// SEMI JOIN
		// 半连接： 当一张表在另一张表找到匹配的记录之后，半连接（semi-join）返回第一张表中的记录。
		// 与条件连接相反，即使在右节点中找到几条匹配的记录，左节点的表也只会返回一条记录。
		// 另外，右节点的表一条记录也不会返回。半连接通常使用IN  或 EXISTS 作为连接条件
		"SELECT d.deptno,d.dname,d.loc FROM scott.dept d WHERE d.deptno IN  (SELECT e.deptno FROM scott.emp e);",

		// Delayed Join
		// https://www.percona.com/blog/2007/04/06/using-delayed-join-to-optimize-count-and-limit-queries/
		`SELECT visitor_id, url FROM (SELECT id FROM log WHERE ip="123.45.67.89" order by tsdesc limit 50, 10) I JOIN log ON (I.id=log.id) JOIN url ON (url.id=log.url_id) order by TS desc;`,

		// DELETE
		"DELETE city, country FROM city INNER JOIN country using (country_id) WHERE city.city_id = 1;",
		"DELETE city FROM city LEFT JOIN country ON city.country_id = country.country_id WHERE country.country IS NULL;",
		"DELETE a1, a2 FROM city AS a1 INNER JOIN country AS a2 WHERE a1.country_id=a2.country_id;",
		"DELETE FROM a1, a2 USING city AS a1 INNER JOIN country AS a2 WHERE a1.country_id=a2.country_id;",
		"DELETE FROM film WHERE length > 100;",

		// UPDATE
		"UPDATE city INNER JOIN country USING(country_id) SET city.city = 'Abha', city.last_update = '2006-02-15 04:45:25', country.country = 'Afghanistan' WHERE city.city_id=10;",
		"UPDATE city INNER JOIN country ON city.country_id = country.country_id INNER JOIN address ON city.city_id = address.city_id SET city.city = 'Abha', city.last_update = '2006-02-15 04:45:25', country.country = 'Afghanistan' WHERE city.city_id=10;",
		"UPDATE city, country SET city.city = 'Abha', city.last_update = '2006-02-15 04:45:25', country.country = 'Afghanistan' WHERE city.country_id = country.country_id AND city.city_id=10;",
		"UPDATE film SET length = 10 WHERE language_id = 20;",

		// INSERT
		"INSERT INTO city (country_id) SELECT country_id FROM country;",
		"INSERT INTO city (country_id) VALUES (1),(2),(3);",
		"INSERT INTO city (country_id) VALUES (10);",
		"INSERT INTO city (country_id) SELECT 10 FROM DUAL;",

		// REPLACE
		"REPLACE INTO city (country_id) SELECT country_id FROM country;",
		"REPLACE INTO city (country_id) VALUES (1),(2),(3);",
		"REPLACE INTO city (country_id) VALUES (10);",
		"REPLACE INTO city (country_id) SELECT 10 FROM DUAL;",

		// DEPTH
		"SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM ( SELECT film_id FROM  film ) film ) film ) film ) film ) film ) film ) film ) film ) film ) film ) film ) film ) film ) film ) film ) film;",

		// SUBQUERY
		"SELECT * FROM film WHERE language_id = (SELECT language_id FROM language LIMIT 1);",
		// "SELECT COUNT(*) /* no hint */ FROM t2 WHERE NOT EXISTS (SELECT * FROM t3 WHERE ROW(5 * t2.s1, 77) = (SELECT 50, 11 * s1 FROM t4 UNION SELECT 50, 77 FROM (SELECT * FROM t5) AS t5 ) ) ;",
		"SELECT * FROM city i left JOIN country o ON i.city_id=o.country_id union SELECT * FROM city i right JOIN country o ON i.city_id=o.country_id;",
		"SELECT * FROM (SELECT * FROM actor WHERE last_update='2006-02-15 04:34:33' and last_name='CHASE') t WHERE last_update='2006-02-15 04:34:33' and last_name='CHASE' GROUP BY first_name;",
		"SELECT * FROM city i left JOIN country o ON i.city_id=o.country_id union SELECT * FROM city i right JOIN country o ON i.city_id=o.country_id;",
		"SELECT * FROM city i left JOIN country o ON i.city_id=o.country_id WHERE o.country_id is null union SELECT * FROM city i right JOIN country o ON i.city_id=o.country_id WHERE i.city_id is null;",
		"SELECT first_name,last_name,email FROM customer STRAIGHT_JOIN address ON customer.address_id=address.address_id;",
		"SELECT ID,name FROM (SELECT address FROM customer_list WHERE SID=1 order by phone limit 50,10) a JOIN customer_list l ON (a.address=l.address) JOIN city c ON (c.city=l.city) order by phone desc;",

		// function in conditions
		"SELECT * FROM film WHERE date(last_update)='2006-02-15';",
		"SELECT last_update FROM film GROUP BY date(last_update);",
		"SELECT last_update FROM film order by date(last_update);",

		// CLA.004
		"SELECT description FROM film WHERE description IN('NEWS','asd') GROUP BY description;",

		// ALTER TABLE ADD INDEX
		// 已经存在索引的列应该提醒索引已存在
		"alter table address add index idx_city_id(city_id);",
		"alter table inventory add index `idx_store_film` (`store_id`,`film_id`);",
		"alter table inventory add index `idx_store_film` (`store_id`,`film_id`),add index `idx_store_film` (`store_id`,`film_id`),add index `idx_store_film` (`store_id`,`film_id`);",

		// https://github.com/XiaoMi/soar/issues/47
		`SELECT	DATE_FORMAT(t.atm, '%Y-%m-%d'),	COUNT(DISTINCT (t.usr))	FROM usr_terminal t WHERE t.atm > '2018-10-22 00:00:00'	AND t.agent LIKE '%Chrome%'	AND t.system = 'eip' GROUP BY DATE_FORMAT(t.atm, '%Y-%m-%d')	ORDER BY DATE_FORMAT(t.atm, '%Y-%m-%d')`,
		// https://github.com/XiaoMi/soar/issues/17
		"create table hello.t (id int unsigned);",
	}
}
