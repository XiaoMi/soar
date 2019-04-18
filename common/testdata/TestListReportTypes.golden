# 支持的报告类型

[toc]

## lint
* **Description**:参考sqlint格式，以插件形式集成到代码编辑器，显示输出更加友好

* **Example**:

```bash
soar -report-type lint -query test.sql
```
## markdown
* **Description**:该格式为默认输出格式，以markdown格式展现，可以用网页浏览器插件直接打开，也可以用markdown编辑器打开

* **Example**:

```bash
echo "select * from film" | soar
```
## rewrite
* **Description**:SQL重写功能，配合-rewrite-rules参数一起使用，可以通过-list-rewrite-rules 查看所有支持的 SQL 重写规则

* **Example**:

```bash
echo "select * from film" | soar -rewrite-rules star2columns,delimiter -report-type rewrite
```
## ast
* **Description**:输出 SQL 的抽象语法树，主要用于测试

* **Example**:

```bash
echo "select * from film" | soar -report-type ast
```
## ast-json
* **Description**:以 JSON 格式输出 SQL 的抽象语法树，主要用于测试

* **Example**:

```bash
echo "select * from film" | soar -report-type ast-json
```
## tiast
* **Description**:输出 SQL 的 TiDB抽象语法树，主要用于测试

* **Example**:

```bash
echo "select * from film" | soar -report-type tiast
```
## tiast-json
* **Description**:以 JSON 格式输出 SQL 的 TiDB抽象语法树，主要用于测试

* **Example**:

```bash
echo "select * from film" | soar -report-type tiast-json
```
## tables
* **Description**:以 JSON 格式输出 SQL 使用的库表名

* **Example**:

```bash
echo "select * from film" | soar -report-type tables
```
## query-type
* **Description**:SQL 语句的请求类型

* **Example**:

```bash
echo "select * from film" | soar -report-type query-type
```
## fingerprint
* **Description**:输出SQL的指纹

* **Example**:

```bash
echo "select * from film where language_id=1" | soar -report-type fingerprint
```
## md2html
* **Description**:markdown 格式转 html 格式小工具

* **Example**:

```bash
soar -list-heuristic-rules | soar -report-type md2html > heuristic_rules.html
```
## explain-digest
* **Description**:输入为EXPLAIN的表格，JSON 或 Vertical格式，对其进行分析，给出分析结果

* **Example**:

```bash
soar -report-type explain-digest << EOF
+----+-------------+-------+------+---------------+------+---------+------+------+-------+
| id | select_type | table | type | possible_keys | key  | key_len | ref  | rows | Extra |
+----+-------------+-------+------+---------------+------+---------+------+------+-------+
|  1 | SIMPLE      | film  | ALL  | NULL          | NULL | NULL    | NULL | 1131 |       |
+----+-------------+-------+------+---------------+------+---------+------+------+-------+
EOF
```
## duplicate-key-checker
* **Description**:对 OnlineDsn 中指定的 database 进行索引重复检查

* **Example**:

```bash
soar -report-type duplicate-key-checker -online-dsn user:password@127.0.0.1:3306/db
```
## html
* **Description**:以HTML格式输出报表

* **Example**:

```bash
echo "select * from film" | soar -report-type html
```
## json
* **Description**:输出JSON格式报表，方便应用程序处理

* **Example**:

```bash
echo "select * from film" | soar -report-type json
```
## tokenize
* **Description**:对SQL进行切词，主要用于测试

* **Example**:

```bash
echo "select * from film" | soar -report-type tokenize
```
## compress
* **Description**:SQL压缩小工具，使用内置SQL压缩逻辑，测试中的功能

* **Example**:

```bash
echo "select
*
from
  film" | soar -report-type compress
```
## pretty
* **Description**:使用kr/pretty打印报告，主要用于测试

* **Example**:

```bash
echo "select * from film" | soar -report-type pretty
```
## remove-comment
* **Description**:去除SQL语句中的注释，支持单行多行注释的去除

* **Example**:

```bash
echo "select/*comment*/ * from film" | soar -report-type remove-comment
```
## chardet
* **Description**:猜测输入的 SQL 使用的字符集

* **Example**:

```bash
echo '中文' | soar -report-type chardet
```
