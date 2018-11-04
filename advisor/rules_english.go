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
			Content: `It will be hard to distinguish Alias of table or column when the alias is the same as it's real name.`,
		},
		"ALT.001": {
			Summary: "modify the default character set will not modify the character set of each field in the table.",
			Content: `Many beginner would think that "ALTER TABLE tbl_name [DEFAULT] CHARACTER SET 'UTF8' " will modify character set of all fields,but actually it can only influence new fileds but not that already exits.You may use "ALTER TABLE tbl_name CONVERT TO CHARACTER SET charset_name;" if you want to modify character set of all fields.`,
		},
		"ALT.002": {
			Summary: "recommend to merge multiple ALTER request if it's in the same table.",
			Content: `Once the structure of table changed,the online service will be influenced.Please merge requests of ALTER to reduce the number of operations even though you can adjust it by online tools.`,
		},
		"ALT.003": {
			Summary: "It's danger to delete column,please check if the business logic is still dependent before operation.",
			Content: `If the business logic dosen't eliminated compeletly, you may can't write data or query the data of column begin deleted after column deleted,in which case you may lost data requested by users even by backup and recover.`,
		},
		"ALT.004": {
			Summary: "It's danger to delete primary key and foreign key ,please check with DBA before operation.",
			Content: `Primary key and foreign key are two important constraint in relational database,delete existing constraints may broken existing business logic,you may check the influence with DBA.`,
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
			Content: `The engine may scan the whole table instead of use index when you use "IS NULL" or "IS NOT NULL",such as "select id from t where num is null".You can set num as 0 by default to make sure there is no null value in the column of num,then you can use "select id from t where num=0"`,
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
			Content: `No WHERE clause in SELECT statement will check more rows than that you predict(whole table scanned).It is recommended to use "SHOW TABLE STATUS" or "EXPLAIN" when precision is not required in the type of request like "SELECT COUNT(*)".`,
		},
		"CLA.002": {
			Summary: "Use ORDER BY RAND() isn't recommended.",
			Content: `It's inefficiency to search random rows from result concentrically by using "ORDER BY RAND()",which may sort all of result and discard most of data.`,
		},

		"CLA.003": {
			Summary: "The use of LIMIT query with OFFSET is not recommended.",
			Content: `The complexity is O(n^2) when using LIMIT and OFFSET to paginate to result set,which will cause performance probelm as data increases.Use "bookmark" to scan make it more efficiency to realize pagination.`,
		},
		"CLA.004": {
			Summary: "GROUP BY is not recommended for constants.",
			Content: `GROUP BY 1 means GROUP BY according to the 1st column.If use number clause in GROUP BY instead od expression or column name,it may cause problem when the query order changes.`,
		},
		"CLA.005": {
			Summary: "ORDER BY constant column does not make any sense",
			Content: `SQL may have logic error,which is an unuse operation at most and can't change query result.`,
		},
		"CLA.006": {
			Summary: "Using GROUP BY or ORDER BY in different table.",
			Content: `This will use temporary table and filesort compulsively,cause huge performance hazard,and comsume plenty of memory and temporary space of disk.`,
		},
		"CLA.007": {
			Summary: "Can't use index,because of ORDER BY different directions for multiple different conditions.",
			Content: `All expression in ORDER BY clause must sort by ASC or DESC uniformly to use index.`,
		},
		"CLA.008": {
			Summary: "Please add ORDER BY condition explicitly for GROUP BY.",
			Content: `By default,MySQL will sort 'GROUP BY col1, col2, ...' request in order of 'ORDER BY col1, col2, ...'.If the  GROUP BY statement does not specify an ORDER BY condition,it will result in unnecessary sorting.It's recommended to add 'ORDER BY NULL' if you don't need sorting suggestions.`,
		},
		"CLA.009": {
			Summary: "The condition of ORDER BY is an expression.",
			Content: `Temporary tables are used when the ORDER BY condition is an expression or function, it will cause poor performance if not specifying a WHERE or WHERE condition when the result set is large.`,
		},
		"CLA.010": {
			Summary: "The condition of GROUP BY is an expression.",
			Content: `Temporary tables are used when the GROUP BY condition is an expression or function, it will cause poor performance if not specifying a WHERE or the return result set is large of WHERE condition .`,
		},
		"CLA.011": {
			Summary: "Recommend add comments to the table",
			Content: `Adding comments to the table can make the it clearer to the meaning of the table, which will bring great convenience for future maintenance.`,
		},
		"CLA.012": {
			Summary: "Decompose complex query into several simple queries",
			Content: `SQL is a very expressive language, and you can do a lot of things in a single SQL query or a single statement. But that doesn't mean you have to force a single line of code, or think it's a good idea to get each task with a single line of code.Generally,you get a Cartesian product though a single query.This happens when there is no condition between the two tables in the query to restrict their relationship.Without a corresponding restriction and directly using two tables for the join query,a combination of each row in the first table and each row in the second table is obtained.Each such combination becomes a row in the result set,and you will get a result set with a large number of rows.It is important to consider that these queries are difficult to write,difficult to modify,and difficult to debug.The increasing number of database query requests should be expected.Managers want more complex reports and add more fields to the user interface.If your design is complex and a single query, it can be time and labor consuming to extend them.It's not worth the time to spend on these things whether for you or the project.Break up complex spaghetti queries into a few simple queries.When you split a complex SQL query, the result may be a lot of similar queries, and may only differ in data type.It's uninteresting to write all of these queries,so it's best to have a program to generate the code automatically.SQL code generation is a good application.Although solving complex problems with a single line of code is supported by SQL,but don't do anything unrealistic.`,
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
			Content: `Rewrite the HAVING clause of the query to the query condition in WHERE, which can make index used during query processing.`,
		},
		"CLA.014": {
			Summary: "Recommend to use TRUNCATE instead of DELETE.",
			Content: `It is recommended to use TRUNCATE instead of DELETE when deleting the whole table.`,
		},
		"CLA.015": {
			Summary: "UPDATE without WHERE condition.",
			Content: `UPDATE without specifying a WHERE condition is generally fatal, please think twice.`,
		},
		"CLA.016": {
			Summary: "Don't UPDATE primary key.",
			Content: `The primary key is a unique identifier record in a data table. It is not recommended to update the primary key column frequently, which will affect statistics of the metadata and thus affect the normal query.`,
		},
		"CLA.017": {
			Summary: "The use of stored procedure,view,trigger,temporary table is not recommended.",
			Content: `The use of these functions will make the program difficult to debug and expand,and have no portability,and will increase the probability of occurrence of BUG greatly.`,
		},
		"COL.001": {
			Summary: "SELECT * is not good",
			Content: `When the table structure changes,selecting all the columns with the * wildcard will cause the meaning and behavior of the query to change,which may cause the query to return more data.`,
		},
		"COL.002": {
			Summary: "INSERT without specify a column name.",
			Content: `When the table structure changes,if the INSERT or REPLACE request does not explicitly specify the column name,the result of the request will be different than expected; it is recommended to use "INSERT INTO tbl(col1,col2)VALUES ..." instead.`,
		},
		"COL.003": {
			Summary: "Recommend to modify auto increment Id to unsigned type.",
			Content: `It is recommended to modify the auto_increment ID to be an unsigned type.`,
		},
		"COL.004": {
			Summary: "Please set default values to the column.",
			Content: `Please add a default value for the column. If it is an ALTER operation, please don't forget to write the default value of the original field. , The table structure can't be changed online when the table is large if there is no default value for the field.`,
		},
		"COL.005": {
			Summary: "column without comments.",
			Content: `It is recommended to add a comment to each column in the table to clarify the meaning and role of each column in the table.`,
		},
		"COL.006": {
			Summary: "table contains too many columns.",
			Content: `The table contains too many columns.`,
		},
		"COL.008": {
			Summary: "use VARCHAR instead of CHAR，VARBINARY instead of BINARY",
			Content: `First, the variable-length field has a small storage space, which saves storage space. Secondly, the search efficiency is obviously higher in a relatively small field for queries.`,
		},
		"COL.009": {
			Summary: "Precise data types are recommended.",
			Content: `Acctually, any design uses the FLOAT, REAL, or DOUBLE PRECISION data types may be anti-pattern.The range of values ​​used by most programmers does not need to meet the maximum/minimum range defined by the IEEE 754 standard. The cumulative effect of inexact floating point numbers is severe when calculating totals. Use NUMERIC or DECIMAL types in SQL instead of FLOAT and its similar data types for fractional storage of fixed-precision.These data types store data precisely based on the precision you specify when you define this column. Don't use floating point numbers as much as possible.`,
		},
		"COL.010": {
			Summary: "The use of enum Type is not recommend.",
			Content: `ENUM defines the type of the value in the column. The data actually stored in the column is the ordinal of these values ​​when they are defined when a string is used to represent the value in ENUM.Therefore, the data in this column is byte-aligned. When you do a sort query, the results are sorted by the actual stored ordinal value, not by the alphabetical order of the string values.This may not be what you want. There is no grammatical support to add or remove a value from an ENUM or check constraint; you can only redefine this column with a new collection.If you plan to discard an option, you may be annoyed with history data. As a strategy, changing metadata—that is, changing the definition of tables and columns—should be uncommon and pay attention to testing and quality assurance.There is a better solution to constrain the optional values ​​in a column: create a checklist with each row containing a candidate value that is allowed to appear in the column; then declare a foreign key constraint on the old table that references the new table.`,
		},
		// 这个建议从sqlcheck迁移来的，实际生产环境每条建表SQL都会给这条建议，看多了会不开心。
		"COL.011": {
			Summary: "NULL is used only when a unique constraint is required, and NOT NULL is used only when the column cannot have a missing value.",
			Content: `NULL and 0 are different,10 is multiplied by NULL is equal to NULL.NULL and empty strings are different.The result of combining a string with NULL in standard SQL is still NULL. NULL and FALSE are also different. The three Boolean operations of AND, OR, and NOT also confuse many people if involving NULL.When you declare a column as NOT NULL, that is, each value in this column must exist and be meaningful. Use NULL to represent a null value that does not exist for any type.`,
		},
		"COL.012": {
			Summary: "Never set BLOB and TEXT to NULL.",
			Content: `Fields of BLOB and TEXT type can't be set to NULL.`,
		},
		"COL.013": {
			Summary: "TIMESTAMP type without default value.",
			Content: `TIMESTAMP type doesn't set default value.`,
		},
		"COL.014": {
			Summary: "specify character set for columns.",
			Content: `It's recommended that the column and table use the same character set. Don't specify the character set of the column separately.`,
		},
		"COL.015": {
			Summary: "The default value cannot be specified for a field of BLOB type.",
			Content: `field of BLOB type can't specify default value.`,
		},
		"COL.016": {
			Summary: "Recommend to use INT(10) or BIGINT(20) specify the integer definition.",
			Content: `INT(M) in data type of integer,M means maximum display width.In INT(M),the value of M has nothing to do with how much storage space INT(M) occupies. INT(3),INT(4),INT(8) occupy 4 bytes of storage space on the disk.`,
		},
		"COL.017": {
			Summary: "Define varchar's length too long.",
			Content: fmt.Sprintf(`Varchar is a variable-length string. It's storage space is not pre-allocated. The length should not exceed %d. If the storage length is too long, MySQL will define the field type as text,separate a table and use the primary key to avoid the impact of other fields.`, common.Config.MaxVarcharLength),
		},
		"DIS.001": {
			Summary: "remove unnecessary DISTINCT condition.",
			Content: `Too many DISTINCT conditions are symptoms of complex queries. Try to break a complex query into many simple queries and reduce the number of DISTINCT conditions. If the primary key column is part of a column's result set, the DISTINCT condition may have no effect.`,
		},
		"DIS.002": {
			Summary: "COUNT(DISTINCT) multiple column may get the result unexpected.",
			Content: `COUNT(DISTINCT col) calculates the number of non-repeating rows in the column except NULL. Note that COUNT(DISTINCT col, col2) returns 0 if one of the columns is all NULL even if the other column has a different value.`,
		},
		// DIS.003灵感来源于如下链接
		// http://www.ijstr.org/final-print/oct2015/Query-Optimization-Techniques-Tips-For-Writing-Efficient-And-Faster-Sql-Queries.pdf
		"DIS.003": {
			Summary: "DISTINCT * dosen’t make sense for a table with a primary key.",
			Content: `When the table already has a primary key, the result of operation with DISTINCT for all columns is the same as that without DISTINCT.Please don't add extras.`,
		},
		"FUN.001": {
			Summary: "Avoid using functions or other operation in WHERE conditions.",
			Content: `Although using functions in SQL can simplify many complex queries, which can't take advantage of the indexes already established in the table. The query will make full table scanned with poor performance. Generally, it's recommended to write the column name to the left of the comparison operator while the query filter to the right of it.`,
		},
		"FUN.002": {
			Summary: "The COUNT(*) operation performed poorly when a WHERE condition or non-myisam engine is specified.",
			Content: `The role of COUNT(*) is to count the number of rows in the table while that of COUNT(COL) is to count the number of rows of the specified column that are not NULL.the MyISAM table is specially optimized for statistics  of COUNT(*) for full table rows, which is usually very fast.But for non-MyISAM tables or for certain WHERE conditions, the COUNT(*) operation needs to scan a large number of rows to get accurate results, which result pool performance.Sometimes some business scenarios don't require an accurate COUNT value, in which case an approximation can be used instead.The number of rows estimated by the optimizer from EXPLAIN is a good approximation. Executing EXPLAIN doesn't require real execution of the query,which costs very low.`,
		},
		"FUN.003": {
			Summary: "String concatenation may produce nullable columns.",
			Content: `In some query requests, you need to force a column or an expression to return a non-NULL value to make the query logic simpler and to worry about not saving this value. Use the COALESCE() function to construct a concatenated expression so that even a null column does not make the whole expression NULL.`,
		},
		"FUN.004": {
			Summary: "The use of SYSDATE() functions is not recommended.",
			Content: `The SYSDATE() function may cause inconsistence between master and slave data. Use the NOW() function instead of SYSDATE().`,
		},
		"FUN.005": {
			Summary: "The use of COUNT(col) or COUNT(const) is not recommended.",
			Content: `Don't use COUNT(col) or COUNT(constant) instead of COUNT(*). COUNT(*) is the standard method to count number of rows defined by SQL92, independent of data, NULL and non-NULL.`,
		},
		"FUN.006": {
			Summary: "Caution the NPE exception when using SUM(col).",
			Content: `When the value of a column is all NULL, the result of COUNT(COL) is 0, but the result of SUM(COL) is NULL, so you need to pay attention to the NPE problem when using SUM(). You can use the following methods to avoid the NPE problem of SUM: SELECT IF(ISNULL(SUM(COL)), 0, SUM(COL)) FROM tbl.`,
		},
		"GRP.001": {
			Summary: "GROUP BY is not recommended for equivalent query columns.",
			Content: `Columns in GROUP BY statement use equivalent queries in the preceding WHERE condition,  it doesn't make much sense to use GROUP BY on such columns.`,
		},
		"JOI.001": {
			Summary: "JOIN statements are mixed with commas and ANSI patterns.",
			Content: `The use of mixing commas and ANSI JOINs when joinning tables is not easy for human to understand,and different versions of MySQL have different  behaviors and priorities of joinning tables,which may introduce error when MySQL version changes.`,
		},
		"JOI.002": {
			Summary: "The same table join twice.",
			Content: `It can be simplified to a single access to the table when the same table appears at least twice in the FROM clause.`,
		},
		"JOI.003": {
			Summary: "OUTER JOIN failure",
			Content: `No value returned from outer table of OUTER JOIN due to error of WHERE condition will implicitly convert the query to an INNER JOIN.such as: select c from L left join R using(c) where L.a=5 and R.b=10.This kind of SQL may have logic error or may occur due to the programmer's misunderstanding to OUTER JOIN  because LEFT/RIGHT JOIN is an abbreviation for LEFT/RIGHT OUTER JOIN.`,
		},
		"JOI.004": {
			Summary: "The use of exclusive JOIN is not recommended.",
			Content: `A LEFT OUTER JOIN statement with a WHERE clause only on the right side of the table may be the reason of using wrong column in the WHERE clause,such as:"... FROM l LEFT OUTER JOIN r ON l.l = r.r WHERE r.z IS NULL",the correct logic of which may be "WHERE rr IS NULL". `,
		},
		"JOI.005": {
			Summary: "Reduce the number of JOIN.",
			Content: `Too many JOINs are symptoms of complex queries. Try to break a complex query into many simple queries and reducing the number of JOINs.`,
		},
		"JOI.006": {
			Summary: "Rewriting nested queries as joins typically results in more efficient execution and more efficient optimization",
			Content: `Generally,non-nested subqueries are always used to associate subqueries, up to one table from the FROM clause, which are used for predicates of ANY, ALL, and EXISTS. If a subquery can return up to one row according to query semantics, then an unrelated subquery or subquery of multiple tables from the FROM clause is flattened.`,
		},
		"JOI.007": {
			Summary: "Federated table updates are not recommended",
			Content: `It's recommended to use simple SQL when you need to update multiple tables at the same time.One SQL only updates one table. Try not to update multiple tables in the same SQL.`,
		},
		"JOI.008": {
			Summary: "Don’t use JOIN query in different DB.",
			Content: `Generally, a Join query across DB means that the query across two different subsystems, which may mean that the system has a high degree of coupling or the table structure of library is designed unproperly.`,
		},
		// TODO: 跨库事务的检查，目前SOAR未对事务做处理
		"KEY.001": {
			Summary: "Recommend use auto increment column as primary key,please set auto increment column as the first column if you use joint autokey.",
			Content: `It's recommended to use the self-incrementing column as the primary key. If you use the union auto-increment primary key, please set the auto-increment key as the first column.`,
		},
		"KEY.002": {
			Summary: "Table structures cannot be changed online without primary or unique keys.",
			Content: `Can't change the table structure online when there is no primary key or unique key.`,
		},
		"KEY.003": {
			Summary: "Avoid recursive relationships such as foreign keys.",
			Content: `It's common that data has recursive relationships,and data is often organized like a tree or hierarchically. However, creating a foreign key constraint to enforce the relationship between two columns in the same table will lead clumsy queries.Each layer of the tree corresponds to another connection. You will need to issue a recursive query to get all descendants or all ancestors of the node. The solution is to construct an additional closure table. It records the relationships between all the nodes in the tree, but not just those with direct parent-child relationships. You can also compare data design of different levels: closure tables, path enumerations, nested sets. Then choose one according to the needs of the application.`,
		},
		// TODO: 新增复合索引，字段按散粒度是否由大到小排序，区分度最高的在最左边
		"KEY.004": {
			Summary: "Reminder: align the index attributes with the query.",
			Content: `Please make sure that the query attributes are in the same order as the index attributes if you create a composite index for a column so that the DBMS uses indexes when processing the queries.If queries and index attributes are not aligned, the DBMS may not be able to use indexes when processing the queries.`,
		},
		"KEY.005": {
			Summary: "Tables build too many indexes.",
			Content: `Too many indexes built in the table`,
		},
		"KEY.006": {
			Summary: "Too many columns in the primary key.",
			Content: `Too many columns in the primary key.`,
		},
		"KEY.007": {
			Summary: "No primary key or primary key isn’t int or bigint is specified.",
			Content: `The primary key isn't specified or the primary key isn't int or bigint. It's recommended to set the primary key to int unsigned or bigint unsigned.`,
		},
		"KEY.008": {
			Summary: "The index may not be available for use if the ORDER BY column is not ordered in the same direction.",
			Content: `Indexes already established couldn't be used when the ORDER BY of multiple columns specified different sort directions before MySQL 8.0.`,
		},
		"KEY.009": {
			Summary: "Check data uniqueness before adding a unique key.",
			Content: `Please check the uniqueness of the data added in the unique index column in advance. If the data is not unique, duplicate column will be deleted automatically when the online table structure begin adjusted, which may result in data loss.`,
		},
		"KWR.001": {
			Summary: "SQL_CALC_FOUND_ROWS is inefficient.",
			Content: `It may result performance problems because that SQL_CALC_FOUND_ROWS doesn't scale well.it's recommended that the business use other strategies to replace the counting functions provided by SQL_CALC_FOUND_ROWS, such as: display paging results.`,
		},
		"KWR.002": {
			Summary: "MySQL keywords are not recommended for column or table names.",
			Content: `The program needs to escape the column name and table name when using a keyword as name of column or table. Otherwise,the request will be unexecutable.`,
		},
		"KWR.003": {
			Summary: "Complex Numbers are not recommended for column or table names.",
			Content: `The table name should only represent the entity content in the table and should not indicate the number of entities ,corresponding to the name of DO class is also singular to meet the expression habit.`,
		},
		"LCK.001": {
			Summary: "Note:INSERT INTO xx SELECT will greater lock granularity.",
			Content: `Please be cautious that INSERT INTO xx SELECT will greater lock granularity.`,
		},
		"LCK.002": {
			Summary: "Caution:INSERT ON DUPLICATE KEY UPDATE",
			Content: `Using INSERT ON DUPLICATE KEY UPDATE may cause a large number of discontinuous rapid growth of the primary key when the primary key is an auto-increment key , causing the primary key to quickly overflow and unable to continue writing. It is also possible to cause inconsistence of master and slave data in some extreme cases.`,
		},
		"LIT.001": {
			Summary: "Store IP addresses with character types.",
			Content: `The string literally looks like an IP address, but is not a parameter of INET_ATON(), indicating that the data is stored as a character rather than an integer. It's more efficient to store the IP address as integer.`,
		},
		"LIT.002": {
			Summary: "Date/time is not enclosed in quotation marks.",
			Content: `A query such as "WHERE col <2010-02-12" is a valid SQL, but it may be an error because it will be interpreted as "WHERE col <1996"; text of date/time should be quoted.`,
		},
		"LIT.003": {
			Summary: "A collection of related data stored in a column.",
			Content: `Storing ID as a list as a VARCHAR/TEXT column can result problem of performance and data integrity. Querying such a column requires an expression that uses pattern matching. Using a comma-separated list to do multi-table join queries to locate a row of data is extremely inelegant and time consuming. This will make it more difficult to verify the ID . Consider how much data does the list support at most? Store the ID in a separate table instead of using a multi-valued attribute so that each single attribute value can occupy one row. This across table implements a many-to-many relationship between two tables which will make it simplify to query and efficiency to validate the ID.`,
		},
		"LIT.004": {
			Summary: "Use a semicolon or the designated DELIMITER end.",
			Content: `Commands such as USE database, SHOW DATABASES, etc. also need to end with semicolon or DELIMITER already set.`,
		},
		"RES.001": {
			Summary: "Nondeterministic GROUP BY.",
			Content: `The columns returned by SQL are neither in the aggregate function nor in the columns of the GROUP BY expression, so the result of these values ​​will be uncertain. Such as: "select a, b, c from tbl where foo="bar" group by a", the result returned by the SQL is uncertain.`,
		},
		"RES.002": {
			Summary: "LIMIT query without ORDER BY.",
			Content: `LIMIT without ORDER BY results in uncertain results, which depends on the query execution plan.`,
		},
		"RES.003": {
			Summary: "Using LIMIT with UPDATE/DELETE.",
			Content: `UPDATE/DELETE operations using the LIMIT condition are as dangerous as not adding a WHERE condition, which can cause inconsistence of master and slave data or interrupt synchronization of slaves.`,
		},
		"RES.004": {
			Summary: "Using ORDER BY with UPDATE/DELETE.",
			Content: `Don't specify an ORDER BY condition for UPDATE/DELETE operations.`,
		},
		"RES.005": {
			Summary: "UPDATE may have a logical error that causes data corruption.",
			Content: "",
		},
		"RES.006": {
			Summary: "Compare condition always false.",
			Content: "Query condition never being true will result in the query having no matching results.",
		},
		"RES.007": {
			Summary: "Compare condition always true.",
			Content: "Query condition always being true will result in WHERE condition failing to a full table query.",
		},
		"RES.008": {
			Summary: "The use of LOAD DATA/SELECT ... INTO OUTFILE is not recommended.",
			Content: "SELECT INTO OUTFILE needs to grant FILE permission, which will introduce security problems. Although LOAD DATA can increase the import speed of data, it can also cause the delay of synchronization delay of the slave to be too large.",
		},
		"SEC.001": {
			Summary: "Caution:using TRUNCATE operation.",
			Content: `Generally,the quickest way to clear a table is to use "TRUNCATE TABLE tbl_name". However, the TRUNCATE operation is not without costs. TRUNCATE TABLE can't return the exact number of deleted rows. If you need to return the number of deleted rows, it's recommended to use DELETE. The TRUNCATE operation will also reset AUTO_INCREMENT, and if you don't want to reset the value, it is recommended to use "DELETE FROM tbl_name WHERE 1" instead.The TRUNCATE operation adds a source data lock (MDL) to the data dictionary.It affects all requests for the entire instance When a large number of TRUNCATE tables are needed at once. Therefore, if you want to TRUNCATE multiple tables, use DROP+CREATE to reduce the duration of locking.`,
		},
		"SEC.002": {
			Summary: "Don’t stored password in clear text.",
			Content: `It's not safe to store passwords in plain text or pass passwords on the network in plain text. If the attacker can intercept the SQL statement that you used to insert the password, they can read the password directly. In addition, inserting a string entered by the user into a plain SQL statement in plain text will also allow the attacker to discover it. If you can read the password, the hacker can,too. The solution is to encrypt the original password using a one-way hash function. A hash is a function that converts an input string into another new, unrecognizable string. Add a random string to encryption expression of the password to defend against "dictionary attacks." Do not enter the clear text password into the SQL query statement. Hash strings are computed in code of application, and only used in SQL queries.`,
		},
		"SEC.003": {
			Summary: "Back up when using actions such as DELETE/DROP/TRUNCATE.",
			Content: `It's necessary to back up your data before performing high-risk operations.`,
		},
		"STA.001": {
			Summary: "'!=' isn’t a standard operator.",
			Content: `"<>" is the unequal operator in standard SQL.`,
		},
		"STA.002": {
			Summary: "Recommend: Don’t add white space after table or database.",
			Content: `Don't add spaces after the dot when accessing a table or field using db.table or table.column format,although the syntax is correct.`,
		},
		"STA.003": {
			Summary: "Index naming is not standard.",
			Content: `It's recommended that the normal secondary index be prefixed with idx_ and the unique index be prefixed with uk_.`,
		},
		"STA.004": {
			Summary: "Do not use characters other than letters, Numbers, and underscores when naming names.",
			Content: `Begin with a letter or an underscore, the name is only allowed to use letters, numbers, and underscores. Please be all lowercase or all uppercase, do not use Camel-Case. Don't have a continuous underscore '__' in your name, which is hard to read.`,
		},
		"SUB.001": {
			Summary: "MySQL doesn't optimize subqueries very well.",
			Content: `MySQL executes subquery for each row in outer query as dependent subquery. This is a common cause of serious performance problems. This may be improved in MySQL 5.6, but for 5.1 and earlier, it is recommended to rewrite the query to JOIN or LEFT OUTER JOIN.`,
		},
		"SUB.002": {
			Summary: "If you don't care about duplication, use UNION ALL instead of UNION.",
			Content: `Unlike the UNION with duplicates removed, UNION ALL allows duplicate tuples. If you don't care about repeating tuples, using UNION ALL will be a faster option.`,
		},
		"SUB.003": {
			Summary: "Consider using a EXISTS instead of a DISTINCT subquery.",
			Content: `The DISTINCT keyword removes duplicates after sorting tuples. Instead, consider using a subquery with the EXISTS keyword to avoid returning the entire table.`,
		},
		// TODO: 5.6有了semi join还要把in转成exists么？
		// Use EXISTS instead of IN to check existence of data.
		// http://www.winwire.com/25-tips-to-improve-sql-query-performance/
		"SUB.004": {
			Summary: "The nested connections in the execution plan are too deep.",
			Content: `MySQL doesn't perform well on subqueries, and MySQL executes subqueries for each row in the query as dependent subqueries. This is a common cause of serious performance problems.`,
		},
		// SUB.005灵感来自 https://blog.csdn.net/zhuocr/article/details/61192418
		"SUB.005": {
			Summary: "Subqueries do not support limits.",
			Content: `The current MySQL version doesn't support 'LIMIT & IN/ALL/ANY/SOME' in subqueries.`,
		},
		"SUB.006": {
			Summary: "The use of functions in subquery is not recommended.",
			Content: `MySQL executes subqueries for each row in the query as dependent subqueries. If a function is used in a subquery, even semi-join is difficult to perform efficient queries. You can rewrite the subquery to an OUTER JOIN statement and filter the data with join conditions.`,
		},
		"TBL.001": {
			Summary: "The use of partition table is not recommended.",
			Content: `Partition table is not recommended.`,
		},
		"TBL.002": {
			Summary: "Select the appropriate storage engine for the table.",
			Content: `It's recommended to use the recommended storage engine when building or modifying the storage engine of the table, such as:` + strings.Join(common.Config.TableAllowEngines, ","),
		},
		"TBL.003": {
			Summary: "A table named DUAL has special meaning in the database.",
			Content: `DUAL table is a virtual table that can be used without creation. It's also not recommended that the service name the table with DUAL.`,
		},
		"TBL.004": {
			Summary: "The initial AUTO_INCREMENT value of the table is not 0.",
			Content: `It will result in data holes if AUTO_INCREMENT is not 0.`,
		},
		"TBL.005": {
			Summary: "Please use the recommended character set.",
			Content: `The table character set is only allowed to be set to` + strings.Join(common.Config.TableAllowCharsets, ","),
		},
	}
}
