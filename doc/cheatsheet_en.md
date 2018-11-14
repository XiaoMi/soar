
# Useful Commands

[toc]

## Basic suggest

```bash
echo "select title from sakila.film" | ./soar -log-output=soar.log
```

## Analyze SQL with test environment

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

## List supported heuristic rules

```bash
soar -list-heuristic-rules
```

## Ignore Rules

```bash
soar -ignore-rules "ALI.001,IDX.*"
```

## List supported report-type

```bash
soar -list-report-types
```

## Set report-type for output

```bash
soar -report-type json
```

## Syntax Check

```bash
echo "select * from tb" | soar -only-syntax-check
echo $?
0

echo "select * frm tb" | soar -only-syntax-check
At SQL 1 : syntax error at position 13 near 'frm'
$ echo $?
1

```

## Slow log analyzing

```bash
pt-query-digest slow.log > slow.log.digest
# parse pt-query-digest's output which example script
python2.7 doc/example/digest_pt.py slow.log.digest > slow.md
```

## SQL FingerPrint

```bash
echo "select * from film where col='abc'" | soar -report-type=fingerprint
```

Output

```sql
select * from film where col=?
```

## Convert UPDATE/DELETE/INSERT into SELECT

```bash
echo "update film set title = 'abc'" | soar -rewrite-rules dml2select,delimiter  -report-type rewrite
```

Output

```sql
select * from film;
```

## Merge ALTER SQLs

```bash
echo "alter table tb add column a int; alter table tb add column b int;" | soar -report-type rewrite -rewrite-rules mergealter
```

Output

```sql
ALTER TABLE `tb` add column a int, add column b int ;
```

## SQL Pretty

```bash
echo "select * from tbl where col = 'val'" | ./soar -report-type=pretty
```

Output

```sql
SELECT
  *
FROM
  tbl
WHERE
  col  = 'val';
```

## Convert markdown to HTML

md2html comes with other flags, such as `-report-css`, `-report-javascript`, `-markdown-extensions`, `-markdown-html-flags`, you can get more self control HTML report.

```bash
cat test.md | soar -report-type md2html > test.html
```
