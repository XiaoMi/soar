# 常用命令

[toc]

## 基本用法

```bash
echo "select title from sakila.film" | ./soar -log-output=soar.log
```

## 指定输入源

```bash
# 从文件读取SQL
./soar -query file.sql

# 从管道读取SQL
cat file.sql | ./soar
```

## 指定配置文件

```bash
vi soar.yaml
# yaml format config file
online-dsn:
    addr:     127.0.0.1:3306
    schema:   sakila
    user:     root
    password: "1t'sB1g3rt"
    disable:  false

test-dsn:
    addr:     127.0.0.1:3306
    schema:   sakila
    user:     root
    password: "1t'sB1g3rt"
    disable:  false
```

```bash
echo "select title from sakila.film" | ./soar -test-dsn="root:1t'sB1g3rt@127.0.0.1:3306/sakila" -allow-online-as-test -log-output=soar.log
```

## 打印所有的启发式规则

```bash
soar -list-heuristic-rules
```

## 忽略某些规则

```bash
soar -ignore-rules "ALI.001,IDX.*"
```

## 打印支持的报告格式

```bash
soar -list-report-types
```

## 以指定格式输出报告

```bash
soar -report-type json
```

## 语法检查工具

```bash
echo "select * from tb" | soar -only-syntax-check
echo $?
0

echo "select * frm tb" | soar -only-syntax-check
At SQL 1 : syntax error at position 13 near 'frm'
echo $?
1
```

## 慢日志进行分析示例

```bash
pt-query-digest slow.log > slow.log.digest
# parse pt-query-digest's output which example script
python2.7 doc/example/digest_pt.py slow.log.digest > slow.md
```

## SQL指纹

```bash
echo "select * from film where col='abc'" | soar -report-type=fingerprint
```

输出

```sql
select * from film where col=?
```

## 将 UPDATE/DELETE/INSERT 语法转为 SELECT

```bash
echo "update film set title = 'abc'" | soar -rewrite-rules dml2select,delimiter  -report-type rewrite
```

输出

```sql
select * from film;
```

## 合并多条ALTER语句

```bash
echo "alter table tb add column a int; alter table tb add column b int;" | soar -report-type rewrite -rewrite-rules mergealter
```

输出

```sql
ALTER TABLE `tb` add column a int, add column b int ;
```

## SQL美化

```bash
echo "select * from tbl where col = 'val'" | ./soar -report-type=pretty
```

输出

```sql
SELECT
  *
FROM
  tbl
WHERE
  col  = 'val';
```

## EXPLAIN信息分析报告

```bash
soar -report-type explain-digest << EOF
+----+-------------+-------+------+---------------+------+---------+------+------+-------+
| id | select_type | table | type | possible_keys | key  | key_len | ref  | rows | Extra |
+----+-------------+-------+------+---------------+------+---------+------+------+-------+
|  1 | SIMPLE      | film  | ALL  | NULL          | NULL | NULL    | NULL | 1131 |       |
+----+-------------+-------+------+---------------+------+---------+------+------+-------+
EOF
```

```text
##  Explain信息

| id | select\_type | table | partitions | type | possible_keys | key | key\_len | ref | rows | filtered | scalability | Extra |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1  | SIMPLE | *film* | NULL | ALL | NULL | NULL | NULL | NULL | 0 | 0.00% | ☠️ **O(n)** |  |


### Explain信息解读

#### SelectType信息解读

* **SIMPLE**: 简单SELECT(不使用UNION或子查询等).

#### Type信息解读

* ☠️ **ALL**: 最坏的情况, 从头到尾全表扫描.
```

## markdown 转 HTML

通过指定-report-css, -report-javascript, -markdown-extensions, -markdown-html-flags这些参数，你还可以控制HTML的显示格式。

```bash
cat test.md | soar -report-type md2html > test.html
```

## 清理测试环境残余的临时库表

如配置了`-drop-test-temporary=false`或`soar`异常中止，`-test-dsn`中会残余以`optimizer_`为前缀的临时库表。手工清理这些库表可以使用如下命令。

注意：为了不影响正在进行的其他SQL评审，`-cleanup-test-database`中会删除1小时前生成的临时库表。

```bash
./soar -cleanup-test-database
```
