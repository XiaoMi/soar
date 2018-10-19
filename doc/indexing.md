
# 索引优化建议

以下优化算法基于个人当前理解，能力有限，如有偏颇还请斧正。

## 简单查询索引优化

### 等值查询优化

* 单列等值查询，为该等值列加索引
* 多列等值查询，每列求取散粒度，按从大到小排序取前N列添加到索引（N可配置）

```sql
SELECT * FROM tbl WHERE a = 123;
SELECT * FROM tbl WHERE a = 123 AND b = 456;
SELECT * FROM tbl WHERE a IS NULL;
SELECT * FROM tbl WHERE a <=> 123;
SELECT * FROM tbl WHERE a IS TRUE;
SELECT * FROM tbl WHERE a IS FALSE;
SELECT * FROM tbl WHERE a IS NOT TRUE;
SELECT * FROM tbl WHERE a IS NOT FALSE;
SELECT * FROM tbl WHERE a IN ("xxx"); -- IN单值
```

### 非等值查询优化

* 单列非等值查询，为该非等值列加索引
* 多列非等值查询，每列求取散粒度，为散粒度最大的列加索引。

思考：对于多列非等值，为filtered最小列加索引可能比较好。因为输入可变，所以现在只按散粒度排序。对于高版本MySQL如果开启了Index Merge，考虑为非等值列加单列索引可能会比较好。

```sql
SELECT * FROM tbl WHERE a >= 123 -- <, <=, >=, >, !=, <>
SELECT * FROM tbl WHERE a BETWEEN 22 AND 44; -- NOT BETWEEN
SELECT * FROM tbl WHERE a LIKE 'blah%'; -- NOT LIKE
SELECT * FROM tbl WHERE a IS NOT NULL;
SELECT * FROM tbl WHERE a IN ("xxx"); -- IN多值
```

### 等值 & 非等值组合查询优化

1. 先按`等值查询优化`为等值列添加索引
2. 再将`非等值查询优化`的列追加在等值列索引后

```sql
SELECT * FROM tbl WHERE c = 9 AND a > 12 AND b > 345; -- INDEX(c, a)或INDEX(c, b)
```

### OR操作符

如果使用了OR操作符，即使OR两边是简单的查询条件也会对优化器带来很大的困难。一般对OR的优化需要依赖UNION ALL或Index Merge等多索引访问技术来实现。SOAR目前不会对使用OR操作符连接的字段进行索引优化。

### GROUP BY子句

GROUP BY相关字段能否加入索引列表需要依赖WHERE子句中的条件。当查询指定了WHERE条件，在满足WHERE子句只有等值查询时，可以对GROUP BY字段添加索引。当查询未指定WHERE条件，可以直接对GROUP BY字段添加索引。

* 按照GROPU BY的先后顺序添加索引
* GROUP BY字段出现常量，数学运算或函数运算时会给出警告

### ORDER BY子句

ORDER BY相关字段能否加入索引列表需要依赖WHERE子句和GROUP BY子句中的条件。当查询指定了WHERE条件，在满足WHERE子句只有等值查询且无GROUP BY子句时，可以对ORDER BY字段添加索引。当查询未指定WHERE条件，在满足无GROUP BY子句时，可以对ORDER BY字段添加索引。

* 多个字段之间如果指定顺序相同，按照ORDER BY的先后顺序添加索引
* 多个字段之间如果指定顺序不同，所有ORDER BY字段都不添加索引
* ORDER BY字段出现常量，数学运算或函数运算时会给出警告

## 复杂查询索引优化

### JOIN索引优化算法

* LEFT JOIN为右表加索引
* RIGHT JOIN为左表加索引
* INNER JOIN两张表都加索引
* NATURAL的处理方法参考前三条
* STRAIGHT_JOIN为后面的表加索引

### SUBQUERY和UNION的复杂查询

对于使用了IN，EXIST等词的SUBQUERY或UNION类型的SQL，先将其拆成多条独立的SELECT语句。然后基于上面简单查询索引优化算法，对单条SELECT查询进行优化。SUBQUERY的连接列暂不考虑添加索引。


```sql
SELECT * FROM film WHERE language_id = (SELECT language_id FROM language LIMIT 1);

1. SELECT * FROM film;
2. SELECT language_id FROM language LIMIT 1;
```

```sql
SELECT * FROM city a LEFT JOIN country b ON a.country_id=b.country_id
UNION
SELECT * FROM city a RIGHT JOIN country b ON a.country_id=b.country_id;

1. SELECT * FROM city a LEFT JOIN country b ON a.country_id=b.country_id;
2. SELECT * FROM city a RIGHT JOIN country b ON a.country_id=b.country_id;
```

## 无法使用索引的情况

如下类型的查询条件无法使用索引或SOAR无法给出正确的索引建议。

```sql
-- MySQL无法使用索引
SELECT * FROM tbl WHERE a LIKE '%blah%';
SELECT * FROM tbl WHERE a IN (SELECT...)
SELECT * FROM tbl WHERE DATE(dt) = 'xxx'
SELECT * FROM tbl WHERE LOWER(s) = 'xxx'
SELECT * FROM tbl WHERE CAST(s …) = 'xxx'
SELECT * FROM tbl where a NOT IN()
-- SOAR不支持的索引建议
SELECT * FROM tbl WHERE a = 'xxx' COLLATE xxx -- vitess语法暂不支持
SELECT * FROM tbl ORDER BY a ASC, b DESC -- 8.0+支持
SELECT * FROM tbl WHERE `date` LIKE '2016-12%' -- 时间数据类型隐式类型转换
```

## 索引长度限制

由于索引长度受数据库版本及不同配置参数影响，参考[InnoDB限制](https://dev.mysql.com/doc/refman/8.0/en/innodb-restrictions.html)。这里将索引长度限制定义为可配置值，用户可以根据实际情况进行设置。

* 通过-max-index-bytes配置每列索引最大长度，默认为767 Bytes
* 超过单列索引最大长度限制后程序会自动添加该列的前缀索引（max-index-bytes/CHARSET_Maxlen）
* 通过-max-index-bytes-percolumn配置多列索引加各最大长度，默认为3072 Bytes
* 超过多列索引最大长度限制后，由程序生成的ALTER语句会将每列前缀索引长度指定为N，用户自行调整

```sql
ALTER TABLE `sakila`.`film_text` add index `idx_description` (`description`(255)) ;

```

## 更新语句转换为只读查询

SOAR支持将DELETE， UPDATE， INSERT， REPLACE四种类型语句转换为SELECT查询。对转换后的SELECT查询进行索引优化。以下为转换示例。

```sql
UPDATE film SET length = 10 WHERE language_id = 20;

SELECT * FROM film WHERE language_id = 20;
```

```sql
DELETE FROM film WHERE length > 100;

SELECT * FROM film WHERE length > 100;
```

```sql
INSERT INTO city (country_id) SELECT country_id FROM country;

SELECT country_id FROM country;
```

```sql
REPLACE INTO city (country_id) SELECT country_id FROM country;

SELECT country_id FROM country;
```

## 散粒度计算

### 计算公式

`Cardinality = ColumnDistinctCount/TableTotalRows * 100%`

由于直接对线上表进行COUNT(DISTINCT)操作会影响数据库请求执行效率，因此默认各列的散粒度均为1。用户可以通过指定`-sampling`参数开启数据采样。SOAR会将线上数据随机采样至测试环境求取散粒度。

### 数据采样算法

以下说明摘抄自PostgreSQL数据直方图采样算法。默认k(-sampling-statistic-target)设置为100，即最多采样3万行记录。

```text
 The following choice of minrows is based on the paper
 "Random sampling for histogram construction: how much is enough?"
 by Surajit Chaudhuri, Rajeev Motwani and Vivek Narasayya, in
 Proceedings of ACM SIGMOD International Conference on Management
 of Data, 1998, Pages 436-447.  Their Corollary 1 to Theorem 5
 says that for table size n, histogram size k, maximum relative
 error in bin size f, and error probability gamma, the minimum
 random sample size is
      r = 4 * k * ln(2*n/gamma) / f^2
 Taking f = 0.5, gamma = 0.01, n = 10^6 rows, we obtain
      r = 305.82 * k
 Note that because of the log function, the dependence on n is
 quite weak; even at n = 10^12, a 300*k sample gives <= 0.66
 bin size error with probability 0.99.  So there's no real need to
 scale for n, which is a good thing because we don't necessarily
 know it at this point.
```

### 随机采样

随机采样使用的SQL如下，其中变量`r`, `n`的含义见上面的说明。

```sql
SELECT * FROM `tbl` WHERE RAND() < r LIMIT n;
```

## 索引去重

### 检查步骤
1. 为查询语句可能使用索引的字段添加索引
2. 枚举用到的所有库表的已知索引
3. 判断所有新加的索引是否与已知索引重复
4. 判断所有新加的索引之间是否存在索引重复


### 检查规则

* PRIMARY > UNIQUE > KEY
* 索引名称相同，即: idxA == idxA
* (a, b) > (a)
* (a, b), (b, a) 会给出警告，用户自行判断是否重复

## 不足

* 目前只支持针对InnoDB引擎添加索引建议，不支持FULLTEXT, SPATIAL等其他类型索引
* 暂不支持索引覆盖（Covering）
* 暂不支持Index Merge情况下的索引建议
