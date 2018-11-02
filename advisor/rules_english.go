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
	"fmt"
	"strings"

	"github.com/XiaoMi/soar/common"
)

// HeuristicRulesEnglish HeuristicRules in English
var HeuristicRulesEnglish map[string]Rule

func init() {
	// TODO: Translate Chinese into English
	HeuristicRulesEnglish = map[string]Rule{
		"OK": {
			Summary: "OK",
			Content: `OK`,
		},
		"ALI.001": {
			Summary: "It is recommended to declare an alias using the AS keyword.",
			Content: `In the alias of the column or table(such as "tbl AS alias"),it is easier to understand to use the keyword of AS clearly than implicit alias(such as "tbl alias")`,
		},
		"ALI.002": {
			Summary: "An alias is not recommended for the column wildcard '*'.",
			Content: `Eg:"SELECT tbl.* col1, col2",the SQL above set alias for wildcard of column,which may cause logical error.You may want to query col1,but the result will be renamed last column of tbl.`,
		},
		"ALI.003": {
			Summary: "Alias shouldn't be the same with column name or table name.",
			Content:`It will be hard to distinguish Alias of table or colomn when the alias is the same as it's real name.`,
		},
		"ALT.001": {
			Summary: "modify the default character set will not modify the character set of each field in the table.",
			Content:`Many beginner would think that "ALTER TABLE tbl_name [DEFAULT] CHARACTER SET 'UTF8' " will modify character set of all fields,but actually it can only influence new fileds but not that already exits.You may use "ALTER TABLE tbl_name CONVERT TO CHARACTER SET charset_name;" if you want to modify character set of all fields.`,
		},
		"ALT.002": {
			Summary: "recommend to merge multiple ALTER request if it's in the same table.",
			Content:`Once the structure of table changed,the online service will be influenced.Please merge requests of ALTER to reduce the number of operations even though you can adjust it by online tools.`,
		},
		"ALT.003": {
			Summary: "It's danger to delete column,please check if the business logic is still dependent before operation.",
			Content:`If the business logic dosen't eliminated compeletly, you may can't write data or query the data of column begin deleted after column deleted,in which case you may lost data requested by users even by backup and recover.`,
		},
		"ALT.004": {
			Summary: "It's danger to delete primary key and foreign key ,please check with DBA before operation.",
			Content:`Primary key and foreign key are two important constraint in relational database,delete existing constraints may broken existing business logic,you may check the influence with DBA.` ,
		},
		"ARG.001": {
			Summary: "The use of the preceding wildcard query is not recommended.",
			Content: `Eg:"%foo",parameter of query has preceding wildcard will not use existing index.`,
		},
		"ARG.002": {
			Summary: "LIKE query without wildcard.",
			Content: `LIKE query without wildcard may cause logic error,because it is the same as the equivalent query logically`,
		},
		"ARG.003": {
			Summary: "Parameter comparisons contain implicit conversions, and indexes cannot be used.",
			Content: "Implicit conversions has risk of unable to hit index,which may cause serious problem in case of high concurrency and large amount of data",
		},
		"ARG.004": {
			Summary: "IN (NULL)/NOT IN (NULL) , always false.",
			Content: `You may use "col IN ('val1', 'val2', 'val3') OR col IS NULL"`,
		},
		"ARG.005": {
			Summary: "Use IN with caution,too many elements will lead to a full table scan.",
			Content: ` Eg："select id from t where num in(1,2,3)" as mentioned above,use BETWEEN instead of IN as you can for continuous values,such as "select id from t where num between 1 and 3".because MYSQL may scan the whole table when there are too many IN values,which may make sharp decline in performance`,
		},
		"ARG.006": {
			Summary: "Try to avoid judge NULL values in WHERE clause.",
			Content: `The engine may scan the whole table instead of use index when you use "IS NULL" or "IS NOT NULL",such as "select id from t where num is null".You can set num as 0 by default to make sure there is no null value in the colnumn of num,then you can use "select id from t where num=0"`,
		},
		"ARG.007": {
			Summary: "Avoid string match",
			Content: `Performance are the biggest problem of using pattern matching operators,another problem that it may return unexpected results will be caused by using LIKE or regular expression to pattern matching,so a better program is to use special search engine technology instead of SQL such as "Apache Lucene".Besides,another optional program is to save results to reduce overhead from Repeated search.If you really want to use SQL,please consider to use third party extension like FULLTEXT index in MySQL.Acctually,SQL is not the one way to solve all problem.`,
		},
		"ARG.008": {
			Summary: "Try to use IN when you execute an OR query in an index column.",
			Content: `The predicate of IN-list can be used by index to search,and optimizer can sort to IN-list to match the sorted sequence of index,which can make more effective search.Please pay attention to that IN-list must include constant only,or keep value of constant during query block execution such as external reference.`,
		},
		"ARG.009": {
			Summary: "The string in the quotation mark contains white space at the beginning or the ending.",
			Content: `The space before or after the column of VARCHAR may cause logic problem,for example, it will be considered the same that 'a' and 'a ' in MySQL 5.5`,
		},
		"ARG.010": {
			Summary: "Don't use hint，such as : sql_no_cache, force index, ignore key, straight join.",
			Content: `hint is used to force SQL executes according to a executing plan,but we can't promise which we predict at the beginning  correct as data changed`,
		},
		"ARG.011": {
			Summary: "Don't use negative query，such as：NOT IN/NOT LIKE",
			Content: `Don't use negative query,which may cause whole table scanned and affect query performance`,
		},
		"CLA.001": {
			Summary: "No where condition for select statement",
			Content: `SELECT语句没有WHERE子句，可能检查比预期更多的行(全表扫描)。对于SELECT COUNT(*)类型的请求如果不要求精度，建议使用SHOW TABLE STATUS或EXPLAIN替代。`,
		},
		"CLA.002": {
			Summary: "Use ORDER BY RAND() isn't recommended.",
			Content: `ORDER BY RAND()是从结果集中检索随机行的一种非常低效的方法，因为它会对整个结果进行排序并丢弃其大部分数据。`,
		},

		"CLA.003": {
			Summary: "The use of LIMIT query with OFFSET is not recommended.",
			Content: `使用LIMIT和OFFSET对结果集分页的复杂度是O(n^2)，并且会随着数据增大而导致性能问题。采用“书签”扫描的方法实现分页效率更高。`,
		},
		"CLA.004": {
			Summary: "GROUP BY is not recommended for constants.",
			Content: `GROUP BY 1 表示按第一列进行GROUP BY。如果在GROUP BY子句中使用数字，而不是表达式或列名称，当查询列顺序改变时，可能会导致问题。`,
		},
		"CLA.005": {
			Summary: "ORDER BY constant column does not make any sense",
			Content: `SQL逻辑上可能存在错误; 最多只是一个无用的操作，不会更改查询结果。`,
		},
		"CLA.006": {
			Summary: "Using GROUP BY or ORDER BY in different table.",
			Content: `这将强制使用临时表和filesort，可能产生巨大性能隐患，并且可能消耗大量内存和磁盘上的临时空间。`,
		},
		"CLA.007": {
			Summary: "Can't use index,because of ORDER BY different directions for multiple different conditions.",
			Content: `ORDER BY子句中的所有表达式必须按统一的ASC或DESC方向排序，以便利用索引。`,
		},
		"CLA.008": {
			Summary: "Please add ORDER BY condition explicitly for GROUP BY.",
			Content: `默认MySQL会对'GROUP BY col1, col2, ...'请求按如下顺序排序'ORDER BY col1, col2, ...'。如果GROUP BY语句不指定ORDER BY条件会导致无谓的排序产生，如果不需要排序建议添加'ORDER BY NULL'。`,
		},
		"CLA.009": {
			Summary: "The condition of ORDER BY is an expression.",
			Content: `当ORDER BY条件为表达式或函数时会使用到临时表，如果在未指定WHERE或WHERE条件返回的结果集较大时性能会很差。`,
		},
		"CLA.010": {
			Summary: "The condition of GROUP BY is an expression.",
			Content: `当GROUP BY条件为表达式或函数时会使用到临时表，如果在未指定WHERE或WHERE条件返回的结果集较大时性能会很差。`,
		},
		"CLA.011": {
			Summary: "Recommend add comments to the table",
			Content: `为表添加注释能够使得表的意义更明确，从而为日后的维护带来极大的便利。`,
		},
		"CLA.012": {
			Summary: "Decompose complex query into several simple queries",
			Content: `SQL是一门极具表现力的语言，您可以在单个SQL查询或者单条语句中完成很多事情。但这并不意味着必须强制只使用一行代码，或者认为使用一行代码就搞定每个任务是个好主意。通过一个查询来获得所有结果的常见后果是得到了一个笛卡儿积。当查询中的两张表之间没有条件限制它们的关系时，就会发生这种情况。没有对应的限制而直接使用两张表进行联结查询，就会得到第一张表中的每一行和第二张表中的每一行的一个组合。每一个这样的组合就会成为结果集中的一行，最终您就会得到一个行数很多的结果集。重要的是要考虑这些查询很难编写、难以修改和难以调试。数据库查询请求的日益增加应该是预料之中的事。经理们想要更复杂的报告以及在用户界面上添加更多的字段。如果您的设计很复杂，并且是一个单一查询，要扩展它们就会很费时费力。不论对您还是项目来说，时间花在这些事情上面不值得。将复杂的意大利面条式查询分解成几个简单的查询。当您拆分一个复杂的SQL查询时，得到的结果可能是很多类似的查询，可能仅仅在数据类型上有所不同。编写所有的这些查询是很乏味的，因此，最好能够有个程序自动生成这些代码。SQL代码生成是一个很好的应用。尽管SQL支持用一行代码解决复杂的问题，但也别做不切实际的事情。`,
		},
		/*
			https://www.datacamp.com/community/tutorials/sql-tutorial-query
			The HAVING Clause
			The HAVING clause was originally added to SQL because the WHERE keyword could not be used with aggregate functions. HAVING is typically used with the GROUP BY clause to restrict the groups of returned rows to only those that meet certain conditions. However, if you use this clause in your query, the index is not used, which -as you already know- can result in a query that doesn't really perform all that well.

			If you’re looking for an alternative, consider using the WHERE clause. Consider the following queries:

			SELECT state, COUNT(*)
			  FROM Drivers
			 WHERE state IN ('GA', 'TX')
			 GROUP BY state
			 ORDER BY state
			SELECT state, COUNT(*)
			  FROM Drivers
			 GROUP BY state
			HAVING state IN ('GA', 'TX')
			 ORDER BY state
			The first query uses the WHERE clause to restrict the number of rows that need to be summed, whereas the second query sums up all the rows in the table and then uses HAVING to throw away the sums it calculated. In these types of cases, the alternative with the WHERE clause is obviously the better one, as you don’t waste any resources.

			You see that this is not about limiting the result set, rather about limiting the intermediate number of records within a query.

			Note that the difference between these two clauses lies in the fact that the WHERE clause introduces a condition on individual rows, while the HAVING clause introduces a condition on aggregations or results of a selection where a single result, such as MIN, MAX, SUM,… has been produced from multiple rows.
		*/
		"CLA.013": {
			Summary: "The use of HAVING clause is not recommended.",
			Content: `将查询的HAVING子句改写为WHERE中的查询条件，可以在查询处理期间使用索引。`,
		},
		"CLA.014": {
			Summary: "Recommend to use TRUNCATE instead of DELETE.",
			Content: `删除全表时建议使用TRUNCATE替代DELETE`,
		},
		"CLA.015": {
			Summary: "UPDATE without WHERE condition.",
			Content: `UPDATE不指定WHERE条件一般是致命的，请您三思后行`,
		},
		"CLA.016": {
			Summary: "Don't UPDATE primary key.",
			Content: `主键是数据表中记录的唯一标识符，不建议频繁更新主键列，这将影响元数据统计信息进而影响正常的查询。`,
		},
		"CLA.017": {
			Summary: "The use of stored procedure,view,trigger,temporary table is not recommended.",
			Content: `这些功能的使用在一定程度上会使得程序难以调试和拓展，更没有移植性，且会极大的增加出现BUG的概率。`,
		},
		"COL.001": {
			Summary: "SELECT * is not good",
			Content: `当表结构变更时，使用*通配符选择所有列将导致查询的含义和行为会发生更改，可能导致查询返回更多的数据。`,
		},
		"COL.002": {
			Summary: "INSERT without specify a column name.",
			Content: `当表结构发生变更，如果INSERT或REPLACE请求不明确指定列名，请求的结果将会与预想的不同; 建议使用“INSERT INTO tbl(col1，col2)VALUES ...”代替。`,
		},
		"COL.003": {
			Summary: "Recommend to modify auto increment Id to unsigned type.",
			Content: `建议修改自增ID为无符号类型`,
		},
		"COL.004": {
			Summary: "Please set default values to the column.",
			Content: `请为列添加默认值，如果是ALTER操作，请不要忘记将原字段的默认值写上。字段无默认值，当表较大时无法在线变更表结构。`,
		},
		"COL.005": {
			Summary: "column without comments.",
			Content: `建议对表中每个列添加注释，来明确每个列在表中的含义及作用。`,
		},
		"COL.006": {
			Summary: "table contains too many columns.",
			Content: `表中包含有太多的列`,
		},
		"COL.008": {
			Summary: "use VARCHAR instead of CHAR，VARBINARY instead of BINARY",
			Content: `为首先变长字段存储空间小，可以节省存储空间。其次对于查询来说，在一个相对较小的字段内搜索效率显然要高些。`,
		},
		"COL.009": {
			Summary: "Precise data types are recommended.",
			Content: `实际上，任何使用FLOAT、REAL或DOUBLE PRECISION数据类型的设计都有可能是反模式。大多数应用程序使用的浮点数的取值范围并不需要达到IEEE 754标准所定义的最大/最小区间。在计算总量时，非精确浮点数所积累的影响是严重的。使用SQL中的NUMERIC或DECIMAL类型来代替FLOAT及其类似的数据类型进行固定精度的小数存储。这些数据类型精确地根据您定义这一列时指定的精度来存储数据。尽可能不要使用浮点数。`,
		},
		"COL.010": {
			Summary: "The use of enum Type is not recommend.",
			Content: `ENUM定义了列中值的类型，使用字符串表示ENUM里的值时，实际存储在列中的数据是这些值在定义时的序数。因此，这列的数据是字节对齐的，当您进行一次排序查询时，结果是按照实际存储的序数值排序的，而不是按字符串值的字母顺序排序的。这可能不是您所希望的。没有什么语法支持从ENUM或者check约束中添加或删除一个值；您只能使用一个新的集合重新定义这一列。如果您打算废弃一个选项，您可能会为历史数据而烦恼。作为一种策略，改变元数据——也就是说，改变表和列的定义——应该是不常见的，并且要注意测试和质量保证。有一个更好的解决方案来约束一列中的可选值:创建一张检查表，每一行包含一个允许在列中出现的候选值；然后在引用新表的旧表上声明一个外键约束。`,
		},
		// 这个建议从sqlcheck迁移来的，实际生产环境每条建表SQL都会给这条建议，看多了会不开心。
		"COL.011": {
			Summary: "NULL is used only when a unique constraint is required, and NOT NULL is used only when the column cannot have a missing value.",
			Content: `NULL和0是不同的，10乘以NULL还是NULL。NULL和空字符串是不一样的。将一个字符串和标准SQL中的NULL联合起来的结果还是NULL。NULL和FALSE也是不同的。AND、OR和NOT这三个布尔操作如果涉及NULL，其结果也让很多人感到困惑。当您将一列声明为NOT NULL时，也就是说这列中的每一个值都必须存在且是有意义的。使用NULL来表示任意类型不存在的空值。 当您将一列声明为NOT NULL时，也就是说这列中的每一个值都必须存在且是有意义的。`,
		},
		"COL.012": {
			Summary: "Never set BLOB and TEXT to NULL.",
			Content: `BLOB和TEXT类型的字段不可设置为NULL`,
		},
		"COL.013": {
			Summary: "TIMESTAMP type without default value.",
			Content: `TIMESTAMP类型未设置默认值`,
		},
		"COL.014": {
			Summary: "specify character set for columns.",
			Content: `建议列与表使用同一个字符集，不要单独指定列的字符集。`,
		},
		"COL.015": {
			Summary: "The default value cannot be specified for a field of BLOB type.",
			Content: `BLOB类型的字段不可指定默认值`,
		},
		"COL.016": {
			Summary: "Recommend to use INT(10) or BIGINT(20) specify the integer definition.",
			Content: `INT(M) 在 integer 数据类型中，M 表示最大显示宽度。 在 INT(M) 中，M 的值跟 INT(M) 所占多少存储空间并无任何关系。 INT(3)、INT(4)、INT(8) 在磁盘上都是占用 4 bytes 的存储空间。`,
		},
		"COL.017": {
			Summary: "Define varchar's length too long.",
			Content: fmt.Sprintf(`varchar 是可变长字符串，不预先分配存储空间，长度不要超过%d，如果存储长度过长MySQL将定义字段类型为text，独立出来一张表，用主键来对应，避免影响其它字段索引效率。`, common.Config.MaxVarcharLength),
		},
		"DIS.001": {
			Summary: "remove unnecessary DISTINCT condition.",
			Content: `太多DISTINCT条件是复杂的裹脚布式查询的症状。考虑将复杂查询分解成许多简单的查询，并减少DISTINCT条件的数量。如果主键列是列的结果集的一部分，则DISTINCT条件可能没有影响。`,
		},
		"DIS.002": {
			Summary: "COUNT(DISTINCT) multiple column may get the result unexpected.",
			Content: `COUNT(DISTINCT col)计算该列除NULL之外的不重复行数，注意COUNT(DISTINCT col, col2)如果其中一列全为NULL那么即使另一列有不同的值，也返回0。`,
		},
		// DIS.003灵感来源于如下链接
		// http://www.ijstr.org/final-print/oct2015/Query-Optimization-Techniques-Tips-For-Writing-Efficient-And-Faster-Sql-Queries.pdf
		"DIS.003": {
			Summary: "DISTINCT * dosen’t make sense for a table with a primary key.",
			Content: `当表已经有主键时，对所有列进行DISTINCT的输出结果与不进行DISTINCT操作的结果相同，请不要画蛇添足。`,
		},
		"FUN.001": {
			Summary: "Avoid using functions or other operation in WHERE conditions.",
			Content: `虽然在SQL中使用函数可以简化很多复杂的查询，但使用了函数的查询无法利用表中已经建立的索引，该查询将会是全表扫描，性能较差。通常建议将列名写在比较运算符左侧，将查询过滤条件放在比较运算符右侧。`,
		},
		"FUN.002": {
			Summary: "The COUNT(*) operation performed poorly when a WHERE condition or non-myisam engine is specified.",
			Content: `COUNT(*)的作用是统计表行数，COUNT(COL)的作用是统计指定列非NULL的行数。MyISAM表对于COUNT(*)统计全表行数进行了特殊的优化，通常情况下非常快。但对于非MyISAM表或指定了某些WHERE条件，COUNT(*)操作需要扫描大量的行才能获取精确的结果，性能也因此不佳。有时候某些业务场景并不需要完全精确的COUNT值，此时可以用近似值来代替。EXPLAIN出来的优化器估算的行数就是一个不错的近似值，执行EXPLAIN并不需要真正去执行查询，所以成本很低。`,
		},
		"FUN.003": {
			Summary: "String concatenation may produce nullable columns.",
			Content: `在一些查询请求中，您需要强制让某一列或者某个表达式返回非NULL的值，从而让查询逻辑变得更简单，担忧不想将这个值存下来。使用COALESCE()函数来构造连接的表达式，这样即使是空值列也不会使整表达式变为NULL。`,
		},
		"FUN.004": {
			Summary: "The use of SYSDATE() functions is not recommended.",
			Content: `SYSDATE()函数可能导致主从数据不一致，请使用NOW()函数替代SYSDATE()。`,
		},
		"FUN.005": {
			Summary: "The use of COUNT(col) or COUNT(const) is not recommended.",
			Content: `不要使用COUNT(col)或COUNT(常量)来替代COUNT(*),COUNT(*)是SQL92定义的标准统计行数的方法，跟数据无关，跟NULL和非NULL也无关。`,
		},
		"FUN.006": {
			Summary: "Caution the NPE exception when using SUM(col).",
			Content: `当某一列的值全是NULL时，COUNT(COL)的返回结果为0,但SUM(COL)的返回结果为NULL,因此使用SUM()时需注意NPE问题。可以使用如下方式来避免SUM的NPE问题: SELECT IF(ISNULL(SUM(COL)), 0, SUM(COL)) FROM tbl`,
		},
		"GRP.001": {
			Summary: "GROUP BY is not recommended for equivalent query columns.",
			Content: `GROUP BY中的列在前面的WHERE条件中使用了等值查询，对这样的列进行GROUP BY意义不大。`,
		},
		"JOI.001": {
			Summary: "JOIN statements are mixed with commas and ANSI patterns.",
			Content: `表连接的时候混用逗号和ANSI JOIN不便于人类理解，并且MySQL不同版本的表连接行为和优先级均有所不同，当MySQL版本变化后可能会引入错误。`,
		},
		"JOI.002": {
			Summary: "The same table join twice.",
			Content: `相同的表在FROM子句中至少出现两次，可以简化为对该表的单次访问。`,
		},
		"JOI.003": {
			Summary: "OUTER JOIN failure",
			Content: `由于WHERE条件错误使得OUTER JOIN的外部表无数据返回，这会将查询隐式转换为 INNER JOIN 。如：select c from L left join R using(c) where L.a=5 and R.b=10。这种SQL逻辑上可能存在错误或程序员对OUTER JOIN如何工作存在误解，因为LEFT/RIGHT JOIN是LEFT/RIGHT OUTER JOIN的缩写。`,
		},
		"JOI.004": {
			Summary: "The use of exclusive JOIN is not recommended.",
			Content: `只在右侧表为NULL的带WHERE子句的LEFT OUTER JOIN语句，有可能是在WHERE子句中使用错误的列，如：“... FROM l LEFT OUTER JOIN r ON l.l = r.r WHERE r.z IS NULL”，这个查询正确的逻辑可能是 WHERE r.r IS NULL。`,
		},
		"JOI.005": {
			Summary: "Reduce the number of JOIN.",
			Content: `太多的JOIN是复杂的裹脚布式查询的症状。考虑将复杂查询分解成许多简单的查询，并减少JOIN的数量。`,
		},
		"JOI.006": {
			Summary: "Rewriting nested queries as joins typically results in more efficient execution and more efficient optimization",
			Content: `一般来说，非嵌套子查询总是用于关联子查询，最多是来自FROM子句中的一个表，这些子查询用于ANY、ALL和EXISTS的谓词。如果可以根据查询语义决定子查询最多返回一个行，那么一个不相关的子查询或来自FROM子句中的多个表的子查询就被压平了。`,
		},
		"JOI.007": {
			Summary: "Federated table updates are not recommended",
			Content: `当需要同时更新多张表时建议使用简单SQL，一条SQL只更新一张表，尽量不要将多张表的更新在同一条SQL中完成。`,
		},
		"JOI.008": {
			Summary: "Don’t use JOIN query in different DB.",
			Content: `一般来说，跨DB的Join查询意味着查询语句跨越了两个不同的子系统，这可能意味着系统耦合度过高或库表结构设计不合理。`,
		},
		// TODO: 跨库事务的检查，目前SOAR未对事务做处理
		"KEY.001": {
			Summary: "Recommend use auto increment column as primary key,please set auto increment column as the first column if you use joint autokey.",
			Content: `建议使用自增列作为主键，如使用联合自增主键时请将自增键作为第一列`,
		},
		"KEY.002": {
			Summary: "Table structures cannot be changed online without primary or unique keys.",
			Content: `无主键或唯一键，无法在线变更表结构`,
		},
		"KEY.003": {
			Summary: "Avoid recursive relationships such as foreign keys.",
			Content: `存在递归关系的数据很常见，数据常会像树或者以层级方式组织。然而，创建一个外键约束来强制执行同一表中两列之间的关系，会导致笨拙的查询。树的每一层对应着另一个连接。您将需要发出递归查询，以获得节点的所有后代或所有祖先。解决方案是构造一个附加的闭包表。它记录了树中所有节点间的关系，而不仅仅是那些具有直接的父子关系。您也可以比较不同层次的数据设计：闭包表，路径枚举，嵌套集。然后根据应用程序的需要选择一个。`,
		},
		// TODO: 新增复合索引，字段按散粒度是否由大到小排序，区分度最高的在最左边
		"KEY.004": {
			Summary: "Reminder: align the index attributes with the query.",
			Content: `如果为列创建复合索引，请确保查询属性与索引属性的顺序相同，以便DBMS在处理查询时使用索引。如果查询和索引属性订单没有对齐，那么DBMS可能无法在查询处理期间使用索引。`,
		},
		"KEY.005": {
			Summary: "Tables build too many indexes.",
			Content: `表建的索引过多`,
		},
		"KEY.006": {
			Summary: "Too many columns in the primary key.",
			Content: `主键中的列过多`,
		},
		"KEY.007": {
			Summary: "No primary key or primary key isn’t int or bigint is specified.",
			Content: `未指定主键或主键非int或bigint，建议将主键设置为int unsigned或bigint unsigned。`,
		},
		"KEY.008": {
			Summary: "The index may not be available for use if the ORDER BY column is not ordered in the same direction.",
			Content: `在MySQL 8.0之前当ORDER BY多个列指定的排序方向不同时将无法使用已经建立的索引。`,
		},
		"KEY.009": {
			Summary: "Check data uniqueness before adding a unique key.",
			Content: `请提前检查添加唯一索引列的数据唯一性，如果数据不唯一在线表结构调整时将有可能自动将重复列删除，这有可能导致数据丢失。`,
		},
		"KWR.001": {
			Summary: "SQL_CALC_FOUND_ROWS is inefficient.",
			Content: `因为SQL_CALC_FOUND_ROWS不能很好地扩展，所以可能导致性能问题; 建议业务使用其他策略来替代SQL_CALC_FOUND_ROWS提供的计数功能，比如：分页结果展示等。`,
		},
		"KWR.002": {
			Summary: "MySQL keywords are not recommended for column or table names.",
			Content: `当使用关键字做为列名或表名时程序需要对列名和表名进行转义，如果疏忽被将导致请求无法执行。`,
		},
		"KWR.003": {
			Summary: "Complex Numbers are not recommended for column or table names.",
			Content: `表名应该仅仅表示表里面的实体内容，不应该表示实体数量，对应于 DO 类名也是单数形式，符合表达习惯。`,
		},
		"LCK.001": {
			Summary: "Note:INSERT INTO xx SELECT will greater lock granularity.",
			Content: `INSERT INTO xx SELECT加锁粒度较大请谨慎`,
		},
		"LCK.002": {
			Summary: "Caution:INSERT ON DUPLICATE KEY UPDATE",
			Content: `当主键为自增键时使用INSERT ON DUPLICATE KEY UPDATE可能会导致主键出现大量不连续快速增长，导致主键快速溢出无法继续写入。极端情况下还有可能导致主从数据不一致。`,
		},
		"LIT.001": {
			Summary: "Store IP addresses with character types.",
			Content: `字符串字面上看起来像IP地址，但不是INET_ATON()的参数，表示数据被存储为字符而不是整数。将IP地址存储为整数更为有效。`,
		},
		"LIT.002": {
			Summary: "Date/time is not enclosed in quotation marks.",
			Content: `诸如“WHERE col <2010-02-12”之类的查询是有效的SQL，但可能是一个错误，因为它将被解释为“WHERE col <1996”; 日期/时间文字应该加引号。`,
		},
		"LIT.003": {
			Summary: "A collection of related data stored in a column.",
			Content: `将ID存储为一个列表，作为VARCHAR/TEXT列，这样能导致性能和数据完整性问题。查询这样的列需要使用模式匹配的表达式。使用逗号分隔的列表来做多表联结查询定位一行数据是极不优雅和耗时的。这将使验证ID更加困难。考虑一下，列表最多支持存放多少数据呢？将ID存储在一张单独的表中，代替使用多值属性，从而每个单独的属性值都可以占据一行。这样交叉表实现了两张表之间的多对多关系。这将更好地简化查询，也更有效地验证ID。`,
		},
		"LIT.004": {
			Summary: "Use a semicolon or the designated DELIMITER end.",
			Content: `USE database, SHOW DATABASES等命令也需要使用使用分号或已设定的DELIMITER结尾。`,
		},
		"RES.001": {
			Summary: "Nondeterministic GROUP BY.",
			Content: `SQL返回的列既不在聚合函数中也不是GROUP BY表达式的列中，因此这些值的结果将是非确定性的。如：select a, b, c from tbl where foo="bar" group by a，该SQL返回的结果就是不确定的。`,
		},
		"RES.002": {
			Summary: "LIMIT query without ORDER BY.",
			Content: `没有ORDER BY的LIMIT会导致非确定性的结果，这取决于查询执行计划。`,
		},
		"RES.003": {
			Summary: "Using LIMIT with UPDATE/DELETE.",
			Content: `UPDATE/DELETE操作使用LIMIT条件和不添加WHERE条件一样危险，它可将会导致主从数据不一致或从库同步中断。`,
		},
		"RES.004": {
			Summary: "Using ORDER BY with UPDATE/DELETE.",
			Content: `UPDATE/DELETE操作不要指定ORDER BY条件。`,
		},
		"RES.005": {
			Summary: "UPDATE may have a logical error that causes data corruption.",
			Content: "",
		},
		"RES.006": {
			Summary: "Compare condition always false.",
			Content: "查询条件永远非真，这将导致查询无匹配到的结果。",
		},
		"RES.007": {
			Summary: "COmpare condition always true.",
			Content: "查询条件永远为真，这将导致WHERE条件失效进行全表查询。",
		},
		"RES.008": {
			Summary: "The use of LOAD DATA/SELECT ... INTO OUTFILE is not recommended.",
			Content: "SELECT INTO OUTFILE需要授予FILE权限，这通过会引入安全问题。LOAD DATA虽然可以提高数据导入速度，但同时也可能导致从库同步延迟过大。",
		},
		"SEC.001": {
			Summary: "Caution:using TRUNCATE operation.",
			Content: `一般来说想清空一张表最快速的做法就是使用TRUNCATE TABLE tbl_name;语句。但TRUNCATE操作也并非是毫无代价的，TRUNCATE TABLE无法返回被删除的准确行数，如果需要返回被删除的行数建议使用DELETE语法。TRUNCATE操作还会重置AUTO_INCREMENT，如果不想重置该值建议使用DELETE FROM tbl_name WHERE 1;替代。TRUNCATE操作会对数据字典添加源数据锁(MDL)，当一次需要TRUNCATE很多表时会影响整个实例的所有请求，因此如果要TRUNCATE多个表建议用DROP+CREATE的方式以减少锁时长。`,
		},
		"SEC.002": {
			Summary: "Don’t stored password in clear text.",
			Content: `使用明文存储密码或者使用明文在网络上传递密码都是不安全的。如果攻击者能够截获您用来插入密码的SQL语句，他们就能直接读到密码。另外，将用户输入的字符串以明文的形式插入到纯SQL语句中，也会让攻击者发现它。如果您能够读取密码，黑客也可以。解决方案是使用单向哈希函数对原始密码进行加密编码。哈希是指将输入字符串转化成另一个新的、不可识别的字符串的函数。对密码加密表达式加点随机串来防御“字典攻击”。不要将明文密码输入到SQL查询语句中。在应用程序代码中计算哈希串，只在SQL查询中使用哈希串。`,
		},
		"SEC.003": {
			Summary: "Back up when using actions such as DELETE/DROP/TRUNCATE.",
			Content: `在执行高危操作之前对数据进行备份是十分有必要的。`,
		},
		"STA.001": {
			Summary: "'!=' isn’t a standard operator.",
			Content: `"<>"才是标准SQL中的不等于运算符。`,
		},
		"STA.002": {
			Summary: "Recommend: Don’t add white space after table or database.",
			Content: `当使用db.table或table.column格式访问表或字段时，请不要在点号后面添加空格，虽然这样语法正确。`,
		},
		"STA.003": {
			Summary: "Index naming is not standard.",
			Content: `建议普通二级索引以idx_为前缀，唯一索引以uk_为前缀。`,
		},
		"STA.004": {
			Summary: "Do not use characters other than letters, Numbers, and underscores when naming names.",
			Content: `以字母或下划线开头，名字只允许使用字母、数字和下划线。请统一大小写，不要使用驼峰命名法。不要在名字中出现连续下划线'__'，这样很难辨认。`,
		},
		"SUB.001": {
			Summary: "MySQL doesn't optimize subqueries very well.",
			Content: `MySQL将外部查询中的每一行作为依赖子查询执行子查询。 这是导致严重性能问题的常见原因。这可能会在 MySQL 5.6版本中得到改善, 但对于5.1及更早版本, 建议将该类查询分别重写为JOIN或LEFT OUTER JOIN。`,
		},
		"SUB.002": {
			Summary: "If you don't care about duplication, use UNION ALL instead of UNION.",
			Content: `与去除重复的UNION不同，UNION ALL允许重复元组。如果您不关心重复元组，那么使用UNION ALL将是一个更快的选项。`,
		},
		"SUB.003": {
			Summary: "Consider using a EXISTS instead of a DISTINCT subquery.",
			Content: `DISTINCT关键字在对元组排序后删除重复。相反，考虑使用一个带有EXISTS关键字的子查询，您可以避免返回整个表。`,
		},
		// TODO: 5.6有了semi join还要把in转成exists么？
		// Use EXISTS instead of IN to check existence of data.
		// http://www.winwire.com/25-tips-to-improve-sql-query-performance/
		"SUB.004": {
			Summary: "The nested connections in the execution plan are too deep.",
			Content: `MySQL对子查询的优化效果不佳,MySQL将外部查询中的每一行作为依赖子查询执行子查询。 这是导致严重性能问题的常见原因。`,
		},
		// SUB.005灵感来自 https://blog.csdn.net/zhuocr/article/details/61192418
		"SUB.005": {
			Summary: "Subqueries do not support limits.",
			Content: `当前MySQL版本不支持在子查询中进行'LIMIT & IN/ALL/ANY/SOME'。`,
		},
		"SUB.006": {
			Summary: "The use of functions in subquery is not recommended.",
			Content: `MySQL将外部查询中的每一行作为依赖子查询执行子查询，如果在子查询中使用函数，即使是semi-join也很难进行高效的查询。可以将子查询重写为OUTER JOIN语句并用连接条件对数据进行过滤。`,
		},
		"TBL.001": {
			Summary: "The use of partition table is not recommended.",
			Content: `不建议使用分区表`,
		},
		"TBL.002": {
			Summary: "Select the appropriate storage engine for the table.",
			Content: `建表或修改表的存储引擎时建议使用推荐的存储引擎，如：` + strings.Join(common.Config.TableAllowEngines, ","),
		},
		"TBL.003": {
			Summary: "A table named DUAL has special meaning in the database.",
			Content: `DUAL表为虚拟表，不需要创建即可使用，也不建议服务以DUAL命名表。`,
		},
		"TBL.004": {
			Summary: "The initial AUTO_INCREMENT value of the table is not 0.",
			Content: `AUTO_INCREMENT不为0会导致数据空洞。`,
		},
		"TBL.005": {
			Summary: "Please use the recommended character set.",
			Content: `表字符集只允许设置为` + strings.Join(common.Config.TableAllowCharsets, ","),
		},
	}
}
