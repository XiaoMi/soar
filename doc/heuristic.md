# 启发式规则建议

[toc]

## 建议使用AS关键字显示声明一个别名

* **Item**:ALI.001
* **Severity**:L0
* **Content**:在列或表别名(如"tbl AS alias")中, 明确使用AS关键字比隐含别名(如"tbl alias")更易懂。
* **Case**:

```sql
select name from tbl t1 where id < 1000
```
## 不建议给列通配符'\*'设置别名

* **Item**:ALI.002
* **Severity**:L8
* **Content**:例: "SELECT tbl.\* col1, col2"上面这条SQL给列通配符设置了别名，这样的SQL可能存在逻辑错误。您可能意在查询col1, 但是代替它的是重命名的是tbl的最后一列。
* **Case**:

```sql
select tbl.* as c1,c2,c3 from tbl where id < 1000
```
## 别名不要与表或列的名字相同

* **Item**:ALI.003
* **Severity**:L1
* **Content**:表或列的别名与其真实名称相同, 这样的别名会使得查询更难去分辨。
* **Case**:

```sql
select name from tbl as tbl where id < 1000
```
## 修改表的默认字符集不会改表各个字段的字符集

* **Item**:ALT.001
* **Severity**:L4
* **Content**:很多初学者会将ALTER TABLE tbl\_name [DEFAULT] CHARACTER SET 'UTF8'误认为会修改所有字段的字符集，但实际上它只会影响后续新增的字段不会改表已有字段的字符集。如果想修改整张表所有字段的字符集建议使用ALTER TABLE tbl\_name CONVERT TO CHARACTER SET charset\_name;
* **Case**:

```sql
ALTER TABLE tbl_name CONVERT TO CHARACTER SET charset_name;
```
## 同一张表的多条ALTER请求建议合为一条

* **Item**:ALT.002
* **Severity**:L2
* **Content**:每次表结构变更对线上服务都会产生影响，即使是能够通过在线工具进行调整也请尽量通过合并ALTER请求的试减少操作次数。
* **Case**:

```sql
ALTER TABLE tbl ADD COLUMN col int, ADD INDEX idx_col (`col`);
```
## 删除列为高危操作，操作前请注意检查业务逻辑是否还有依赖

* **Item**:ALT.003
* **Severity**:L0
* **Content**:如业务逻辑依赖未完全消除，列被删除后可能导致数据无法写入或无法查询到已删除列数据导致程序异常的情况。这种情况下即使通过备份数据回滚也会丢失用户请求写入的数据。
* **Case**:

```sql
ALTER TABLE tbl DROP COLUMN col;
```
## 删除主键和外键为高危操作，操作前请与DBA确认影响

* **Item**:ALT.004
* **Severity**:L0
* **Content**:主键和外键为关系型数据库中两种重要约束，删除已有约束会打破已有业务逻辑，操作前请业务开发与DBA确认影响，三思而行。
* **Case**:

```sql
ALTER TABLE tbl DROP PRIMARY KEY;
```
## 不建议使用前项通配符查找

* **Item**:ARG.001
* **Severity**:L4
* **Content**:例如“％foo”，查询参数有一个前项通配符的情况无法使用已有索引。
* **Case**:

```sql
select c1,c2,c3 from tbl where name like '%foo'
```
## 没有通配符的LIKE查询

* **Item**:ARG.002
* **Severity**:L1
* **Content**:不包含通配符的LIKE查询可能存在逻辑错误，因为逻辑上它与等值查询相同。
* **Case**:

```sql
select c1,c2,c3 from tbl where name like 'foo'
```
## 参数比较包含隐式转换，无法使用索引

* **Item**:ARG.003
* **Severity**:L4
* **Content**:隐式类型转换有无法命中索引的风险，在高并发、大数据量的情况下，命不中索引带来的后果非常严重。
* **Case**:

```sql
SELECT * FROM sakila.film WHERE length >= '60';
```
## IN (NULL)/NOT IN (NULL)永远非真

* **Item**:ARG.004
* **Severity**:L4
* **Content**:正确的作法是col IN ('val1', 'val2', 'val3') OR col IS NULL
* **Case**:

```sql
SELECT * FROM tb WHERE col IN (NULL);
```
## IN要慎用，元素过多会导致全表扫描

* **Item**:ARG.005
* **Severity**:L1
* **Content**: 如：select id from t where num in(1,2,3)对于连续的数值，能用BETWEEN就不要用IN了：select id from t where num between 1 and 3。而当IN值过多时MySQL也可能会进入全表扫描导致性能急剧下降。
* **Case**:

```sql
select id from t where num in(1,2,3)
```
## 应尽量避免在WHERE子句中对字段进行NULL值判断

* **Item**:ARG.006
* **Severity**:L1
* **Content**:使用IS NULL或IS NOT NULL将可能导致引擎放弃使用索引而进行全表扫描，如：select id from t where num is null;可以在num上设置默认值0，确保表中num列没有null值，然后这样查询： select id from t where num=0;
* **Case**:

```sql
select id from t where num is null
```
## 避免使用模式匹配

* **Item**:ARG.007
* **Severity**:L3
* **Content**:性能问题是使用模式匹配操作符的最大缺点。使用LIKE或正则表达式进行模式匹配进行查询的另一个问题，是可能会返回意料之外的结果。最好的方案就是使用特殊的搜索引擎技术来替代SQL，比如Apache Lucene。另一个可选方案是将结果保存起来从而减少重复的搜索开销。如果一定要使用SQL，请考虑在MySQL中使用像FULLTEXT索引这样的第三方扩展。但更广泛地说，您不一定要使用SQL来解决所有问题。
* **Case**:

```sql
select c_id,c2,c3 from tbl where c2 like 'test%'
```
## OR查询索引列时请尽量使用IN谓词

* **Item**:ARG.008
* **Severity**:L1
* **Content**:IN-list谓词可以用于索引检索，并且优化器可以对IN-list进行排序，以匹配索引的排序序列，从而获得更有效的检索。请注意，IN-list必须只包含常量，或在查询块执行期间保持常量的值，例如外引用。
* **Case**:

```sql
SELECT c1,c2,c3 FROM tbl WHERE c1 = 14 OR c1 = 17
```
## 引号中的字符串开头或结尾包含空格

* **Item**:ARG.009
* **Severity**:L1
* **Content**:如果VARCHAR列的前后存在空格将可能引起逻辑问题，如在MySQL 5.5中'a'和'a '可能会在查询中被认为是相同的值。
* **Case**:

```sql
SELECT 'abc '
```
## 不要使用hint，如sql\_no\_cache, force index, ignore key, straight join等

* **Item**:ARG.010
* **Severity**:L1
* **Content**:hint是用来强制SQL按照某个执行计划来执行，但随着数据量变化我们无法保证自己当初的预判是正确的。
* **Case**:

```sql
SELECT * FROM t1 USE INDEX (i1) ORDER BY a;
```
## 不要使用负向查询，如：NOT IN/NOT LIKE

* **Item**:ARG.011
* **Severity**:L3
* **Content**:请尽量不要使用负向查询，这将导致全表扫描，对查询性能影响较大。
* **Case**:

```sql
select id from t where num not in(1,2,3);
```
## 最外层SELECT未指定WHERE条件

* **Item**:CLA.001
* **Severity**:L4
* **Content**:SELECT语句没有WHERE子句，可能检查比预期更多的行(全表扫描)。对于SELECT COUNT(\*)类型的请求如果不要求精度，建议使用SHOW TABLE STATUS或EXPLAIN替代。
* **Case**:

```sql
select id from tbl
```
## 不建议使用ORDER BY RAND()

* **Item**:CLA.002
* **Severity**:L3
* **Content**:ORDER BY RAND()是从结果集中检索随机行的一种非常低效的方法，因为它会对整个结果进行排序并丢弃其大部分数据。
* **Case**:

```sql
select name from tbl where id < 1000 order by rand(number)
```
## 不建议使用带OFFSET的LIMIT查询

* **Item**:CLA.003
* **Severity**:L2
* **Content**:使用LIMIT和OFFSET对结果集分页的复杂度是O(n^2)，并且会随着数据增大而导致性能问题。采用“书签”扫描的方法实现分页效率更高。
* **Case**:

```sql
select c1,c2 from tbl where name=xx order by number limit 1 offset 20
```
## 不建议对常量进行GROUP BY

* **Item**:CLA.004
* **Severity**:L2
* **Content**:GROUP BY 1 表示按第一列进行GROUP BY。如果在GROUP BY子句中使用数字，而不是表达式或列名称，当查询列顺序改变时，可能会导致问题。
* **Case**:

```sql
select col1,col2 from tbl group by 1
```
## ORDER BY常数列没有任何意义

* **Item**:CLA.005
* **Severity**:L2
* **Content**:SQL逻辑上可能存在错误; 最多只是一个无用的操作，不会更改查询结果。
* **Case**:

```sql
select id from test where id=1 order by id
```
## 在不同的表中GROUP BY或ORDER BY

* **Item**:CLA.006
* **Severity**:L4
* **Content**:这将强制使用临时表和filesort，可能产生巨大性能隐患，并且可能消耗大量内存和磁盘上的临时空间。
* **Case**:

```sql
select tb1.col, tb2.col from tb1, tb2 where id=1 group by tb1.col, tb2.col
```
## ORDER BY语句对多个不同条件使用不同方向的排序无法使用索引

* **Item**:CLA.007
* **Severity**:L2
* **Content**:ORDER BY子句中的所有表达式必须按统一的ASC或DESC方向排序，以便利用索引。
* **Case**:

```sql
select c1,c2,c3 from t1 where c1='foo' order by c2 desc, c3 asc
```
## 请为GROUP BY显示添加ORDER BY条件

* **Item**:CLA.008
* **Severity**:L2
* **Content**:默认MySQL会对'GROUP BY col1, col2, ...'请求按如下顺序排序'ORDER BY col1, col2, ...'。如果GROUP BY语句不指定ORDER BY条件会导致无谓的排序产生，如果不需要排序建议添加'ORDER BY NULL'。
* **Case**:

```sql
select c1,c2,c3 from t1 where c1='foo' group by c2
```
## ORDER BY的条件为表达式

* **Item**:CLA.009
* **Severity**:L2
* **Content**:当ORDER BY条件为表达式或函数时会使用到临时表，如果在未指定WHERE或WHERE条件返回的结果集较大时性能会很差。
* **Case**:

```sql
select description from film where title ='ACADEMY DINOSAUR' order by length-language_id;
```
## GROUP BY的条件为表达式

* **Item**:CLA.010
* **Severity**:L2
* **Content**:当GROUP BY条件为表达式或函数时会使用到临时表，如果在未指定WHERE或WHERE条件返回的结果集较大时性能会很差。
* **Case**:

```sql
select description from film where title ='ACADEMY DINOSAUR' GROUP BY length-language_id;
```
## 建议为表添加注释

* **Item**:CLA.011
* **Severity**:L1
* **Content**:为表添加注释能够使得表的意义更明确，从而为日后的维护带来极大的便利。
* **Case**:

```sql
CREATE TABLE `test1` (`ID` bigint(20) NOT NULL AUTO_INCREMENT,`c1` varchar(128) DEFAULT NULL,PRIMARY KEY (`ID`)) ENGINE=InnoDB DEFAULT CHARSET=utf8
```
## 将复杂的裹脚布式查询分解成几个简单的查询

* **Item**:CLA.012
* **Severity**:L2
* **Content**:SQL是一门极具表现力的语言，您可以在单个SQL查询或者单条语句中完成很多事情。但这并不意味着必须强制只使用一行代码，或者认为使用一行代码就搞定每个任务是个好主意。通过一个查询来获得所有结果的常见后果是得到了一个笛卡儿积。当查询中的两张表之间没有条件限制它们的关系时，就会发生这种情况。没有对应的限制而直接使用两张表进行联结查询，就会得到第一张表中的每一行和第二张表中的每一行的一个组合。每一个这样的组合就会成为结果集中的一行，最终您就会得到一个行数很多的结果集。重要的是要考虑这些查询很难编写、难以修改和难以调试。数据库查询请求的日益增加应该是预料之中的事。经理们想要更复杂的报告以及在用户界面上添加更多的字段。如果您的设计很复杂，并且是一个单一查询，要扩展它们就会很费时费力。不论对您还是项目来说，时间花在这些事情上面不值得。将复杂的意大利面条式查询分解成几个简单的查询。当您拆分一个复杂的SQL查询时，得到的结果可能是很多类似的查询，可能仅仅在数据类型上有所不同。编写所有的这些查询是很乏味的，因此，最好能够有个程序自动生成这些代码。SQL代码生成是一个很好的应用。尽管SQL支持用一行代码解决复杂的问题，但也别做不切实际的事情。
* **Case**:

```sql
这是一条很长很长的SQL，案例略。
```
## 不建议使用HAVING子句

* **Item**:CLA.013
* **Severity**:L3
* **Content**:将查询的HAVING子句改写为WHERE中的查询条件，可以在查询处理期间使用索引。
* **Case**:

```sql
SELECT s.c_id,count(s.c_id) FROM s where c = test GROUP BY s.c_id HAVING s.c_id <> '1660' AND s.c_id <> '2' order by s.c_id
```
## 删除全表时建议使用TRUNCATE替代DELETE

* **Item**:CLA.014
* **Severity**:L2
* **Content**:删除全表时建议使用TRUNCATE替代DELETE
* **Case**:

```sql
delete from tbl
```
## UPDATE未指定WHERE条件

* **Item**:CLA.015
* **Severity**:L4
* **Content**:UPDATE不指定WHERE条件一般是致命的，请您三思后行
* **Case**:

```sql
update tbl set col=1
```
## 不要UPDATE主键

* **Item**:CLA.016
* **Severity**:L2
* **Content**:主键是数据表中记录的唯一标识符，不建议频繁更新主键列，这将影响元数据统计信息进而影响正常的查询。
* **Case**:

```sql
update tbl set col=1
```
## 不建议使用存储过程、视图、触发器、临时表等

* **Item**:CLA.017
* **Severity**:L2
* **Content**:这些功能的使用在一定程度上会使得程序难以调试和拓展，更没有移植性，且会极大的增加出现BUG的概率。
* **Case**:

```sql
CREATE VIEW v_today (today) AS SELECT CURRENT_DATE;
```
## 不建议使用SELECT \* 类型查询

* **Item**:COL.001
* **Severity**:L1
* **Content**:当表结构变更时，使用\*通配符选择所有列将导致查询的含义和行为会发生更改，可能导致查询返回更多的数据。
* **Case**:

```sql
select * from tbl where id=1
```
## INSERT未指定列名

* **Item**:COL.002
* **Severity**:L2
* **Content**:当表结构发生变更，如果INSERT或REPLACE请求不明确指定列名，请求的结果将会与预想的不同; 建议使用“INSERT INTO tbl(col1，col2)VALUES ...”代替。
* **Case**:

```sql
insert into tbl values(1,'name')
```
## 建议修改自增ID为无符号类型

* **Item**:COL.003
* **Severity**:L2
* **Content**:建议修改自增ID为无符号类型
* **Case**:

```sql
create table test(`id` int(11) NOT NULL AUTO_INCREMENT)
```
## 请为列添加默认值

* **Item**:COL.004
* **Severity**:L1
* **Content**:请为列添加默认值，如果是ALTER操作，请不要忘记将原字段的默认值写上。字段无默认值，当表较大时无法在线变更表结构。
* **Case**:

```sql
CREATE TABLE tbl (col int) ENGINE=InnoDB;
```
## 列未添加注释

* **Item**:COL.005
* **Severity**:L1
* **Content**:建议对表中每个列添加注释，来明确每个列在表中的含义及作用。
* **Case**:

```sql
CREATE TABLE tbl (col int) ENGINE=InnoDB;
```
## 表中包含有太多的列

* **Item**:COL.006
* **Severity**:L3
* **Content**:表中包含有太多的列
* **Case**:

```sql
CREATE TABLE tbl ( cols ....);
```
## 可使用VARCHAR代替CHAR，VARBINARY代替BINARY

* **Item**:COL.008
* **Severity**:L1
* **Content**:为首先变长字段存储空间小，可以节省存储空间。其次对于查询来说，在一个相对较小的字段内搜索效率显然要高些。
* **Case**:

```sql
create table t1(id int,name char(20),last_time date)
```
## 建议使用精确的数据类型

* **Item**:COL.009
* **Severity**:L2
* **Content**:实际上，任何使用FLOAT、REAL或DOUBLE PRECISION数据类型的设计都有可能是反模式。大多数应用程序使用的浮点数的取值范围并不需要达到IEEE 754标准所定义的最大/最小区间。在计算总量时，非精确浮点数所积累的影响是严重的。使用SQL中的NUMERIC或DECIMAL类型来代替FLOAT及其类似的数据类型进行固定精度的小数存储。这些数据类型精确地根据您定义这一列时指定的精度来存储数据。尽可能不要使用浮点数。
* **Case**:

```sql
CREATE TABLE tab2 (p_id  BIGINT UNSIGNED NOT NULL,a_id  BIGINT UNSIGNED NOT NULL,hours float not null,PRIMARY KEY (p_id, a_id))
```
## 不建议使用ENUM数据类型

* **Item**:COL.010
* **Severity**:L2
* **Content**:ENUM定义了列中值的类型，使用字符串表示ENUM里的值时，实际存储在列中的数据是这些值在定义时的序数。因此，这列的数据是字节对齐的，当您进行一次排序查询时，结果是按照实际存储的序数值排序的，而不是按字符串值的字母顺序排序的。这可能不是您所希望的。没有什么语法支持从ENUM或者check约束中添加或删除一个值；您只能使用一个新的集合重新定义这一列。如果您打算废弃一个选项，您可能会为历史数据而烦恼。作为一种策略，改变元数据——也就是说，改变表和列的定义——应该是不常见的，并且要注意测试和质量保证。有一个更好的解决方案来约束一列中的可选值:创建一张检查表，每一行包含一个允许在列中出现的候选值；然后在引用新表的旧表上声明一个外键约束。
* **Case**:

```sql
create table tab1(status ENUM('new','in progress','fixed'))
```
## 当需要唯一约束时才使用NULL，仅当列不能有缺失值时才使用NOT NULL

* **Item**:COL.011
* **Severity**:L0
* **Content**:NULL和0是不同的，10乘以NULL还是NULL。NULL和空字符串是不一样的。将一个字符串和标准SQL中的NULL联合起来的结果还是NULL。NULL和FALSE也是不同的。AND、OR和NOT这三个布尔操作如果涉及NULL，其结果也让很多人感到困惑。当您将一列声明为NOT NULL时，也就是说这列中的每一个值都必须存在且是有意义的。使用NULL来表示任意类型不存在的空值。 当您将一列声明为NOT NULL时，也就是说这列中的每一个值都必须存在且是有意义的。
* **Case**:

```sql
select c1,c2,c3 from tbl where c4 is null or c4 <> 1
```
## BLOB和TEXT类型的字段不可设置为NULL

* **Item**:COL.012
* **Severity**:L5
* **Content**:BLOB和TEXT类型的字段不可设置为NULL
* **Case**:

```sql
CREATE TABLE `tbl` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` longblob, PRIMARY KEY (`id`));
```
## TIMESTAMP类型未设置默认值

* **Item**:COL.013
* **Severity**:L4
* **Content**:TIMESTAMP类型未设置默认值
* **Case**:

```sql
CREATE TABLE tbl( `id` bigint not null, `create_time` timestamp);
```
## 为列指定了字符集

* **Item**:COL.014
* **Severity**:L5
* **Content**:建议列与表使用同一个字符集，不要单独指定列的字符集。
* **Case**:

```sql
CREATE TABLE `tb2` ( `id` int(11) DEFAULT NULL, `col` char(10) CHARACTER SET utf8 DEFAULT NULL)
```
## BLOB类型的字段不可指定默认值

* **Item**:COL.015
* **Severity**:L4
* **Content**:BLOB类型的字段不可指定默认值
* **Case**:

```sql
CREATE TABLE `tbl` ( `id` int(10) unsigned NOT NULL AUTO_INCREMENT, `c` blob NOT NULL DEFAULT '', PRIMARY KEY (`id`));
```
## 整型定义建议采用INT(10)或BIGINT(20)

* **Item**:COL.016
* **Severity**:L1
* **Content**:INT(M) 在 integer 数据类型中，M 表示最大显示宽度。 在 INT(M) 中，M 的值跟 INT(M) 所占多少存储空间并无任何关系。 INT(3)、INT(4)、INT(8) 在磁盘上都是占用 4 bytes 的存储空间。
* **Case**:

```sql
CREATE TABLE tab (a INT(1));
```
## varchar定义长度过长

* **Item**:COL.017
* **Severity**:L2
* **Content**:varchar 是可变长字符串，不预先分配存储空间，长度不要超过1024，如果存储长度过长MySQL将定义字段类型为text，独立出来一张表，用主键来对应，避免影响其它字段索引效率。
* **Case**:

```sql
CREATE TABLE tab (a varchar(3500));
```
## 消除不必要的DISTINCT条件

* **Item**:DIS.001
* **Severity**:L1
* **Content**:太多DISTINCT条件是复杂的裹脚布式查询的症状。考虑将复杂查询分解成许多简单的查询，并减少DISTINCT条件的数量。如果主键列是列的结果集的一部分，则DISTINCT条件可能没有影响。
* **Case**:

```sql
SELECT DISTINCT c.c_id,count(DISTINCT c.c_name),count(DISTINCT c.c_e),count(DISTINCT c.c_n),count(DISTINCT c.c_me),c.c_d FROM (select distinct xing, name from B) as e WHERE e.country_id = c.country_id
```
## COUNT(DISTINCT)多列时结果可能和你预想的不同

* **Item**:DIS.002
* **Severity**:L3
* **Content**:COUNT(DISTINCT col)计算该列除NULL之外的不重复行数，注意COUNT(DISTINCT col, col2)如果其中一列全为NULL那么即使另一列有不同的值，也返回0。
* **Case**:

```sql
SELECT COUNT(DISTINCT col, col2) FROM tbl;
```
## DISTINCT \*对有主键的表没有意义

* **Item**:DIS.003
* **Severity**:L3
* **Content**:当表已经有主键时，对所有列进行DISTINCT的输出结果与不进行DISTINCT操作的结果相同，请不要画蛇添足。
* **Case**:

```sql
SELECT DISTINCT * FROM film;
```
## 避免在WHERE条件中使用函数或其他运算符

* **Item**:FUN.001
* **Severity**:L2
* **Content**:虽然在SQL中使用函数可以简化很多复杂的查询，但使用了函数的查询无法利用表中已经建立的索引，该查询将会是全表扫描，性能较差。通常建议将列名写在比较运算符左侧，将查询过滤条件放在比较运算符右侧。
* **Case**:

```sql
select id from t where substring(name,1,3)='abc'
```
## 指定了WHERE条件或非MyISAM引擎时使用COUNT(\*)操作性能不佳

* **Item**:FUN.002
* **Severity**:L1
* **Content**:COUNT(\*)的作用是统计表行数，COUNT(COL)的作用是统计指定列非NULL的行数。MyISAM表对于COUNT(\*)统计全表行数进行了特殊的优化，通常情况下非常快。但对于非MyISAM表或指定了某些WHERE条件，COUNT(\*)操作需要扫描大量的行才能获取精确的结果，性能也因此不佳。有时候某些业务场景并不需要完全精确的COUNT值，此时可以用近似值来代替。EXPLAIN出来的优化器估算的行数就是一个不错的近似值，执行EXPLAIN并不需要真正去执行查询，所以成本很低。
* **Case**:

```sql
SELECT c3, COUNT(*) AS accounts FROM tab where c2 < 10000 GROUP BY c3 ORDER BY num
```
## 使用了合并为可空列的字符串连接

* **Item**:FUN.003
* **Severity**:L3
* **Content**:在一些查询请求中，您需要强制让某一列或者某个表达式返回非NULL的值，从而让查询逻辑变得更简单，担忧不想将这个值存下来。使用COALESCE()函数来构造连接的表达式，这样即使是空值列也不会使整表达式变为NULL。
* **Case**:

```sql
select c1 || coalesce(' ' || c2 || ' ', ' ') || c3 as c from tbl
```
## 不建议使用SYSDATE()函数

* **Item**:FUN.004
* **Severity**:L4
* **Content**:SYSDATE()函数可能导致主从数据不一致，请使用NOW()函数替代SYSDATE()。
* **Case**:

```sql
SELECT SYSDATE();
```
## 不建议使用COUNT(col)或COUNT(常量)

* **Item**:FUN.005
* **Severity**:L1
* **Content**:不要使用COUNT(col)或COUNT(常量)来替代COUNT(\*),COUNT(\*)是SQL92定义的标准统计行数的方法，跟数据无关，跟NULL和非NULL也无关。
* **Case**:

```sql
SELECT COUNT(1) FROM tbl;
```
## 使用SUM(COL)时需注意NPE问题

* **Item**:FUN.006
* **Severity**:L1
* **Content**:当某一列的值全是NULL时，COUNT(COL)的返回结果为0,但SUM(COL)的返回结果为NULL,因此使用SUM()时需注意NPE问题。可以使用如下方式来避免SUM的NPE问题: SELECT IF(ISNULL(SUM(COL)), 0, SUM(COL)) FROM tbl
* **Case**:

```sql
SELECT SUM(COL) FROM tbl;
```
## 不建议对等值查询列使用GROUP BY

* **Item**:GRP.001
* **Severity**:L2
* **Content**:GROUP BY中的列在前面的WHERE条件中使用了等值查询，对这样的列进行GROUP BY意义不大。
* **Case**:

```sql
select film_id, title from film where release_year='2006' group by release_year
```
## JOIN语句混用逗号和ANSI模式

* **Item**:JOI.001
* **Severity**:L2
* **Content**:表连接的时候混用逗号和ANSI JOIN不便于人类理解，并且MySQL不同版本的表连接行为和优先级均有所不同，当MySQL版本变化后可能会引入错误。
* **Case**:

```sql
select c1,c2,c3 from t1,t2 join t3 on t1.c1=t2.c1,t1.c3=t3,c1 where id>1000
```
## 同一张表被连接两次

* **Item**:JOI.002
* **Severity**:L4
* **Content**:相同的表在FROM子句中至少出现两次，可以简化为对该表的单次访问。
* **Case**:

```sql
select tb1.col from (tb1, tb2) join tb2 on tb1.id=tb.id where tb1.id=1
```
## OUTER JOIN失效

* **Item**:JOI.003
* **Severity**:L4
* **Content**:由于WHERE条件错误使得OUTER JOIN的外部表无数据返回，这会将查询隐式转换为 INNER JOIN 。如：select c from L left join R using(c) where L.a=5 and R.b=10。这种SQL逻辑上可能存在错误或程序员对OUTER JOIN如何工作存在误解，因为LEFT/RIGHT JOIN是LEFT/RIGHT OUTER JOIN的缩写。
* **Case**:

```sql
select c1,c2,c3 from t1 left outer join t2 using(c1) where t1.c2=2 and t2.c3=4
```
## 不建议使用排它JOIN

* **Item**:JOI.004
* **Severity**:L4
* **Content**:只在右侧表为NULL的带WHERE子句的LEFT OUTER JOIN语句，有可能是在WHERE子句中使用错误的列，如：“... FROM l LEFT OUTER JOIN r ON l.l = r.r WHERE r.z IS NULL”，这个查询正确的逻辑可能是 WHERE r.r IS NULL。
* **Case**:

```sql
select c1,c2,c3 from t1 left outer join t2 on t1.c1=t2.c1 where t2.c2 is null
```
## 减少JOIN的数量

* **Item**:JOI.005
* **Severity**:L2
* **Content**:太多的JOIN是复杂的裹脚布式查询的症状。考虑将复杂查询分解成许多简单的查询，并减少JOIN的数量。
* **Case**:

```sql
select bp1.p_id, b1.d_d as l, b1.b_id from b1 join bp1 on (b1.b_id = bp1.b_id) left outer join (b1 as b2 join bp2 on (b2.b_id = bp2.b_id)) on (bp1.p_id = bp2.p_id ) join bp21 on (b1.b_id = bp1.b_id) join bp31 on (b1.b_id = bp1.b_id) join bp41 on (b1.b_id = bp1.b_id) where b2.b_id = 0
```
## 将嵌套查询重写为JOIN通常会导致更高效的执行和更有效的优化

* **Item**:JOI.006
* **Severity**:L4
* **Content**:一般来说，非嵌套子查询总是用于关联子查询，最多是来自FROM子句中的一个表，这些子查询用于ANY、ALL和EXISTS的谓词。如果可以根据查询语义决定子查询最多返回一个行，那么一个不相关的子查询或来自FROM子句中的多个表的子查询就被压平了。
* **Case**:

```sql
SELECT s,p,d FROM tbl WHERE p.p_id = (SELECT s.p_id FROM tbl WHERE s.c_id = 100996 AND s.q = 1 )
```
## 不建议使用联表更新

* **Item**:JOI.007
* **Severity**:L4
* **Content**:当需要同时更新多张表时建议使用简单SQL，一条SQL只更新一张表，尽量不要将多张表的更新在同一条SQL中完成。
* **Case**:

```sql
UPDATE users u LEFT JOIN hobby h ON u.id = h.uid SET u.name = 'pianoboy' WHERE h.hobby = 'piano';
```
## 不要使用跨DB的Join查询

* **Item**:JOI.008
* **Severity**:L4
* **Content**:一般来说，跨DB的Join查询意味着查询语句跨越了两个不同的子系统，这可能意味着系统耦合度过高或库表结构设计不合理。
* **Case**:

```sql
SELECT s,p,d FROM tbl WHERE p.p_id = (SELECT s.p_id FROM tbl WHERE s.c_id = 100996 AND s.q = 1 )
```
## 建议使用自增列作为主键，如使用联合自增主键时请将自增键作为第一列

* **Item**:KEY.001
* **Severity**:L2
* **Content**:建议使用自增列作为主键，如使用联合自增主键时请将自增键作为第一列
* **Case**:

```sql
create table test(`id` int(11) NOT NULL PRIMARY KEY (`id`))
```
## 无主键或唯一键，无法在线变更表结构

* **Item**:KEY.002
* **Severity**:L4
* **Content**:无主键或唯一键，无法在线变更表结构
* **Case**:

```sql
create table test(col varchar(5000))
```
## 避免外键等递归关系

* **Item**:KEY.003
* **Severity**:L4
* **Content**:存在递归关系的数据很常见，数据常会像树或者以层级方式组织。然而，创建一个外键约束来强制执行同一表中两列之间的关系，会导致笨拙的查询。树的每一层对应着另一个连接。您将需要发出递归查询，以获得节点的所有后代或所有祖先。解决方案是构造一个附加的闭包表。它记录了树中所有节点间的关系，而不仅仅是那些具有直接的父子关系。您也可以比较不同层次的数据设计：闭包表，路径枚举，嵌套集。然后根据应用程序的需要选择一个。
* **Case**:

```sql
CREATE TABLE tab2 (p_id  BIGINT UNSIGNED NOT NULL,a_id  BIGINT UNSIGNED NOT NULL,PRIMARY KEY (p_id, a_id),FOREIGN KEY (p_id) REFERENCES tab1(p_id),FOREIGN KEY (a_id) REFERENCES tab3(a_id))
```
## 提醒：请将索引属性顺序与查询对齐

* **Item**:KEY.004
* **Severity**:L0
* **Content**:如果为列创建复合索引，请确保查询属性与索引属性的顺序相同，以便DBMS在处理查询时使用索引。如果查询和索引属性订单没有对齐，那么DBMS可能无法在查询处理期间使用索引。
* **Case**:

```sql
create index idx1 on tbl (last_name,first_name)
```
## 表建的索引过多

* **Item**:KEY.005
* **Severity**:L2
* **Content**:表建的索引过多
* **Case**:

```sql
CREATE TABLE tbl ( a int, b int, c int, KEY idx_a (`a`),KEY idx_b(`b`),KEY idx_c(`c`));
```
## 主键中的列过多

* **Item**:KEY.006
* **Severity**:L4
* **Content**:主键中的列过多
* **Case**:

```sql
CREATE TABLE tbl ( a int, b int, c int, PRIMARY KEY(`a`,`b`,`c`));
```
## 未指定主键或主键非int或bigint

* **Item**:KEY.007
* **Severity**:L4
* **Content**:未指定主键或主键非int或bigint，建议将主键设置为int unsigned或bigint unsigned。
* **Case**:

```sql
CREATE TABLE tbl (a int);
```
## ORDER BY多个列但排序方向不同时可能无法使用索引

* **Item**:KEY.008
* **Severity**:L4
* **Content**:在MySQL 8.0之前当ORDER BY多个列指定的排序方向不同时将无法使用已经建立的索引。
* **Case**:

```sql
SELECT * FROM tbl ORDER BY a DESC, b ASC;
```
## 添加唯一索引前请注意检查数据唯一性

* **Item**:KEY.009
* **Severity**:L0
* **Content**:请提前检查添加唯一索引列的数据唯一性，如果数据不唯一在线表结构调整时将有可能自动将重复列删除，这有可能导致数据丢失。
* **Case**:

```sql
CREATE UNIQUE INDEX part_of_name ON customer (name(10));
```
## SQL\_CALC\_FOUND\_ROWS效率低下

* **Item**:KWR.001
* **Severity**:L2
* **Content**:因为SQL\_CALC\_FOUND\_ROWS不能很好地扩展，所以可能导致性能问题; 建议业务使用其他策略来替代SQL\_CALC\_FOUND\_ROWS提供的计数功能，比如：分页结果展示等。
* **Case**:

```sql
select SQL_CALC_FOUND_ROWS col from tbl where id>1000
```
## 不建议使用MySQL关键字做列名或表名

* **Item**:KWR.002
* **Severity**:L2
* **Content**:当使用关键字做为列名或表名时程序需要对列名和表名进行转义，如果疏忽被将导致请求无法执行。
* **Case**:

```sql
CREATE TABLE tbl ( `select` int )
```
## 不建议使用复数做列名或表名

* **Item**:KWR.003
* **Severity**:L1
* **Content**:表名应该仅仅表示表里面的实体内容，不应该表示实体数量，对应于 DO 类名也是单数形式，符合表达习惯。
* **Case**:

```sql
CREATE TABLE tbl ( `books` int )
```
## INSERT INTO xx SELECT加锁粒度较大请谨慎

* **Item**:LCK.001
* **Severity**:L3
* **Content**:INSERT INTO xx SELECT加锁粒度较大请谨慎
* **Case**:

```sql
INSERT INTO tbl SELECT * FROM tbl2;
```
## 请慎用INSERT ON DUPLICATE KEY UPDATE

* **Item**:LCK.002
* **Severity**:L3
* **Content**:当主键为自增键时使用INSERT ON DUPLICATE KEY UPDATE可能会导致主键出现大量不连续快速增长，导致主键快速溢出无法继续写入。极端情况下还有可能导致主从数据不一致。
* **Case**:

```sql
INSERT INTO t1(a,b,c) VALUES (1,2,3) ON DUPLICATE KEY UPDATE c=c+1;
```
## 用字符类型存储IP地址

* **Item**:LIT.001
* **Severity**:L2
* **Content**:字符串字面上看起来像IP地址，但不是INET\_ATON()的参数，表示数据被存储为字符而不是整数。将IP地址存储为整数更为有效。
* **Case**:

```sql
insert into tbl (IP,name) values('10.20.306.122','test')
```
## 日期/时间未使用引号括起

* **Item**:LIT.002
* **Severity**:L4
* **Content**:诸如“WHERE col <2010-02-12”之类的查询是有效的SQL，但可能是一个错误，因为它将被解释为“WHERE col <1996”; 日期/时间文字应该加引号。
* **Case**:

```sql
select col1,col2 from tbl where time < 2018-01-10
```
## 一列中存储一系列相关数据的集合

* **Item**:LIT.003
* **Severity**:L3
* **Content**:将ID存储为一个列表，作为VARCHAR/TEXT列，这样能导致性能和数据完整性问题。查询这样的列需要使用模式匹配的表达式。使用逗号分隔的列表来做多表联结查询定位一行数据是极不优雅和耗时的。这将使验证ID更加困难。考虑一下，列表最多支持存放多少数据呢？将ID存储在一张单独的表中，代替使用多值属性，从而每个单独的属性值都可以占据一行。这样交叉表实现了两张表之间的多对多关系。这将更好地简化查询，也更有效地验证ID。
* **Case**:

```sql
select c1,c2,c3,c4 from tab1 where col_id REGEXP '[[:<:]]12[[:>:]]'
```
## 请使用分号或已设定的DELIMITER结尾

* **Item**:LIT.004
* **Severity**:L1
* **Content**:USE database, SHOW DATABASES等命令也需要使用使用分号或已设定的DELIMITER结尾。
* **Case**:

```sql
USE db
```
## 非确定性的GROUP BY

* **Item**:RES.001
* **Severity**:L4
* **Content**:SQL返回的列既不在聚合函数中也不是GROUP BY表达式的列中，因此这些值的结果将是非确定性的。如：select a, b, c from tbl where foo="bar" group by a，该SQL返回的结果就是不确定的。
* **Case**:

```sql
select c1,c2,c3 from t1 where c2='foo' group by c2
```
## 未使用ORDER BY的LIMIT查询

* **Item**:RES.002
* **Severity**:L4
* **Content**:没有ORDER BY的LIMIT会导致非确定性的结果，这取决于查询执行计划。
* **Case**:

```sql
select col1,col2 from tbl where name=xx limit 10
```
## UPDATE/DELETE操作使用了LIMIT条件

* **Item**:RES.003
* **Severity**:L4
* **Content**:UPDATE/DELETE操作使用LIMIT条件和不添加WHERE条件一样危险，它可将会导致主从数据不一致或从库同步中断。
* **Case**:

```sql
UPDATE film SET length = 120 WHERE title = 'abc' LIMIT 1;
```
## UPDATE/DELETE操作指定了ORDER BY条件

* **Item**:RES.004
* **Severity**:L4
* **Content**:UPDATE/DELETE操作不要指定ORDER BY条件。
* **Case**:

```sql
UPDATE film SET length = 120 WHERE title = 'abc' ORDER BY title
```
## UPDATE可能存在逻辑错误，导致数据损坏

* **Item**:RES.005
* **Severity**:L4
* **Content**:
* **Case**:

```sql
update tbl set col = 1 and cl = 2 where col=3;
```
## 永远不真的比较条件

* **Item**:RES.006
* **Severity**:L4
* **Content**:查询条件永远非真，这将导致查询无匹配到的结果。
* **Case**:

```sql
select * from tbl where 1 != 1;
```
## 永远为真的比较条件

* **Item**:RES.007
* **Severity**:L4
* **Content**:查询条件永远为真，这将导致WHERE条件失效进行全表查询。
* **Case**:

```sql
select * from tbl where 1 = 1;
```
## 不建议使用LOAD DATA/SELECT ... INTO OUTFILE

* **Item**:RES.008
* **Severity**:L2
* **Content**:SELECT INTO OUTFILE需要授予FILE权限，这通过会引入安全问题。LOAD DATA虽然可以提高数据导入速度，但同时也可能导致从库同步延迟过大。
* **Case**:

```sql
LOAD DATA INFILE 'data.txt' INTO TABLE db2.my_table;
```
## 请谨慎使用TRUNCATE操作

* **Item**:SEC.001
* **Severity**:L0
* **Content**:一般来说想清空一张表最快速的做法就是使用TRUNCATE TABLE tbl\_name;语句。但TRUNCATE操作也并非是毫无代价的，TRUNCATE TABLE无法返回被删除的准确行数，如果需要返回被删除的行数建议使用DELETE语法。TRUNCATE操作还会重置AUTO\_INCREMENT，如果不想重置该值建议使用DELETE FROM tbl\_name WHERE 1;替代。TRUNCATE操作会对数据字典添加源数据锁(MDL)，当一次需要TRUNCATE很多表时会影响整个实例的所有请求，因此如果要TRUNCATE多个表建议用DROP+CREATE的方式以减少锁时长。
* **Case**:

```sql
TRUNCATE TABLE tbl_name
```
## 不使用明文存储密码

* **Item**:SEC.002
* **Severity**:L0
* **Content**:使用明文存储密码或者使用明文在网络上传递密码都是不安全的。如果攻击者能够截获您用来插入密码的SQL语句，他们就能直接读到密码。另外，将用户输入的字符串以明文的形式插入到纯SQL语句中，也会让攻击者发现它。如果您能够读取密码，黑客也可以。解决方案是使用单向哈希函数对原始密码进行加密编码。哈希是指将输入字符串转化成另一个新的、不可识别的字符串的函数。对密码加密表达式加点随机串来防御“字典攻击”。不要将明文密码输入到SQL查询语句中。在应用程序代码中计算哈希串，只在SQL查询中使用哈希串。
* **Case**:

```sql
create table test(id int,name varchar(20) not null,password varchar(200)not null)
```
## 使用DELETE/DROP/TRUNCATE等操作时注意备份

* **Item**:SEC.003
* **Severity**:L0
* **Content**:在执行高危操作之前对数据进行备份是十分有必要的。
* **Case**:

```sql
delete from table where col = 'condition'
```
## '!=' 运算符是非标准的

* **Item**:STA.001
* **Severity**:L0
* **Content**:"<>"才是标准SQL中的不等于运算符。
* **Case**:

```sql
select col1,col2 from tbl where type!=0
```
## 库名或表名点后建议不要加空格

* **Item**:STA.002
* **Severity**:L1
* **Content**:当使用db.table或table.column格式访问表或字段时，请不要在点号后面添加空格，虽然这样语法正确。
* **Case**:

```sql
select col from sakila. film
```
## 索引起名不规范

* **Item**:STA.003
* **Severity**:L1
* **Content**:建议普通二级索引以idx\_为前缀，唯一索引以uk\_为前缀。
* **Case**:

```sql
select col from now where type!=0
```
## 起名时请不要使用字母、数字和下划线之外的字符

* **Item**:STA.004
* **Severity**:L1
* **Content**:以字母或下划线开头，名字只允许使用字母、数字和下划线。请统一大小写，不要使用驼峰命名法。不要在名字中出现连续下划线'\_\_'，这样很难辨认。
* **Case**:

```sql
CREATE TABLE ` abc` (a int);
```
## MySQL对子查询的优化效果不佳

* **Item**:SUB.001
* **Severity**:L4
* **Content**:MySQL将外部查询中的每一行作为依赖子查询执行子查询。 这是导致严重性能问题的常见原因。这可能会在 MySQL 5.6版本中得到改善, 但对于5.1及更早版本, 建议将该类查询分别重写为JOIN或LEFT OUTER JOIN。
* **Case**:

```sql
select col1,col2,col3 from table1 where col2 in(select col from table2)
```
## 如果您不在乎重复的话，建议使用UNION ALL替代UNION

* **Item**:SUB.002
* **Severity**:L2
* **Content**:与去除重复的UNION不同，UNION ALL允许重复元组。如果您不关心重复元组，那么使用UNION ALL将是一个更快的选项。
* **Case**:

```sql
select teacher_id as id,people_name as name from t1,t2 where t1.teacher_id=t2.people_id union select student_id as id,people_name as name from t1,t2 where t1.student_id=t2.people_id
```
## 考虑使用EXISTS而不是DISTINCT子查询

* **Item**:SUB.003
* **Severity**:L3
* **Content**:DISTINCT关键字在对元组排序后删除重复。相反，考虑使用一个带有EXISTS关键字的子查询，您可以避免返回整个表。
* **Case**:

```sql
SELECT DISTINCT c.c_id, c.c_name FROM c,e WHERE e.c_id = c.c_id
```
## 执行计划中嵌套连接深度过深

* **Item**:SUB.004
* **Severity**:L3
* **Content**:MySQL对子查询的优化效果不佳,MySQL将外部查询中的每一行作为依赖子查询执行子查询。 这是导致严重性能问题的常见原因。
* **Case**:

```sql
SELECT * from tb where id in (select id from (select id from tb))
```
## 子查询不支持LIMIT

* **Item**:SUB.005
* **Severity**:L8
* **Content**:当前MySQL版本不支持在子查询中进行'LIMIT & IN/ALL/ANY/SOME'。
* **Case**:

```sql
SELECT * FROM staff WHERE name IN (SELECT NAME FROM customer ORDER BY name LIMIT 1)
```
## 不建议在子查询中使用函数

* **Item**:SUB.006
* **Severity**:L2
* **Content**:MySQL将外部查询中的每一行作为依赖子查询执行子查询，如果在子查询中使用函数，即使是semi-join也很难进行高效的查询。可以将子查询重写为OUTER JOIN语句并用连接条件对数据进行过滤。
* **Case**:

```sql
SELECT * FROM staff WHERE name IN (SELECT max(NAME) FROM customer)
```
## 不建议使用分区表

* **Item**:TBL.001
* **Severity**:L4
* **Content**:不建议使用分区表
* **Case**:

```sql
CREATE TABLE trb3(id INT, name VARCHAR(50), purchased DATE) PARTITION BY RANGE(YEAR(purchased)) (PARTITION p0 VALUES LESS THAN (1990), PARTITION p1 VALUES LESS THAN (1995), PARTITION p2 VALUES LESS THAN (2000), PARTITION p3 VALUES LESS THAN (2005) );
```
## 请为表选择合适的存储引擎

* **Item**:TBL.002
* **Severity**:L4
* **Content**:建表或修改表的存储引擎时建议使用推荐的存储引擎，如：innodb
* **Case**:

```sql
create table test(`id` int(11) NOT NULL AUTO_INCREMENT)
```
## 以DUAL命名的表在数据库中有特殊含义

* **Item**:TBL.003
* **Severity**:L8
* **Content**:DUAL表为虚拟表，不需要创建即可使用，也不建议服务以DUAL命名表。
* **Case**:

```sql
create table dual(id int, primary key (id));
```
## 表的初始AUTO\_INCREMENT值不为0

* **Item**:TBL.004
* **Severity**:L2
* **Content**:AUTO\_INCREMENT不为0会导致数据空洞。
* **Case**:

```sql
CREATE TABLE tbl (a int) AUTO_INCREMENT = 10;
```
## 请使用推荐的字符集

* **Item**:TBL.005
* **Severity**:L4
* **Content**:表字符集只允许设置为utf8,utf8mb4
* **Case**:

```sql
CREATE TABLE tbl (a int) DEFAULT CHARSET = latin1;
```
