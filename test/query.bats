#!/usr/bin/env bats

load test_helper

@test "Check Query Optimizer" {
  run ${SOAR_BIN} -query "select * from film where length > 120"
  [ $status -eq 0 ]
}

#
@test "Check get tables from SQL" {
  ${SOAR_BIN} -list-test-sqls | ${SOAR_BIN} -report-type tables > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  ${SOAR_BIN} -list-test-sqls | ${SOAR_BIN} -report-type tables -test-dsn "/sakila" >> ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}

# SQL 语法检查

# 1. soar SQL 分隔符是否正常 (-delimiter)
@test "Check Soar Delimiter" {
  ${SOAR_BIN} -delimiter "@" -query "select b from c @ select a from b" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 2. max-column-count （ddl 中才会有提示）
@test "Check Soar Max Column Count" {
  ${SOAR_BIN} -query "create table a (a int,b int,c int,d int)" --max-column-count 3 > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# SQL 指纹检查

# 1. 检测各种类型的 SQL 语句，以及多条 SQL 语句的情况下，指纹是否正确
@test "Check Soar SQL Fingerprint" {
  ${SOAR_BIN} -list-test-sqls | ${SOAR_BIN} -report-type "fingerprint" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 1. 检测各种类型的 SQL 语句，以及多条SQL语句的情况下，压缩是否正确
# 1. 检测各种类型的 SQL 语句，以及多条SQL语句的情况下，美化是否正确
@test "Check Soar SQL pretty And Compress " {
  ${SOAR_BIN} -list-test-sqls |${SOAR_BIN} -report-type "pretty"| ${SOAR_BIN} -report-type "compress" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# SQL 改写检查

# 1. 检查 SQL 改写 dml2select
@test "Check Soar SQL Rewrite Dml2select " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "dml2select" \
  -query "DELETE FROM film WHERE length > 100" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 2. 检查 SQL 改写 Star2columns
@test "Check Soar SQL Rewrite Star2columns " {
  ${SOAR_BIN_ENV} -report-type "rewrite" -rewrite-rules "star2columns" \
  -query "select * from film where length > 120" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 4. 检查 SQL 改写 Having
@test "Check Soar SQL Rewrite Having " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "having" \
  -query "SELECT state, COUNT(*) FROM Drivers GROUP BY state HAVING state IN ('GA', 'TX') ORDER BY state" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 5. 检查 SQL 改写 orderbynull
@test "Check Soar SQL Rewrite Orderbynull " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "orderbynull" \
  -query "SELECT sum(col1) FROM tbl GROUP BY col" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 6. 检查 SQL 改写 unionall
@test "Check Soar SQL Rewrite Unionall " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "unionall" \
  -query "select country_id from city union select country_id from country" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 7. 检查 SQL 改写 or2in
@test "Check Soar SQL Rewrite Or2in " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "or2in" \
  -query "select country_id from city where col1 = 1 or (col2 = 1 or col2 = 2 ) or col1 = 3;" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 7. 检查 SQL 改写 dmlorderby
@test "Check Soar SQL Rewrite Dmlorderby " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "dmlorderby" \
  -query "DELETE FROM tbl WHERE col1=1 ORDER BY col" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 8. 检查 SQL 改写 distinctstar
@test "Check Soar SQL Rewrite Distinctstar " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "distinctstar" \
  -query "SELECT DISTINCT * FROM film;" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 9. 检查 SQL 改写 standard
@test "Check Soar SQL Rewrite Standard " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "standard" \
  -query "SELECT sum(col1) FROM tbl GROUP BY 1;" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 10. 检查 SQL 改写 mergealter
# Linux macOS sort 排序不一致 https://unix.stackexchange.com/questions/362728/why-does-gnu-sort-sort-differently-on-my-osx-machine-and-linux-machine
@test "Check Soar SQL Rewrite Mergealter " {
  ${SOAR_BIN} -list-test-sqls |${SOAR_BIN} -report-type rewrite -rewrite-rules mergealter | sort -bdfi > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}

# 11. 检查 SQL 改写 alwaystrue
@test "Check Soar SQL Rewrite Alwaystrue " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "alwaystrue" \
  -query "SELECT count(col) FROM tbl where 'a'= 'a' or ('b' = 'b' and a = 'b');" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 12. 检查 SQL 改写 countstar
@test "Check Soar SQL Rewrite Countstar " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "countstar" \
  -query "SELECT count(col) FROM tbl GROUP BY 1;" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}


# 16. 检查 SQL 改写 truncate
@test "Check Soar SQL Rewrite Truncate " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "truncate" \
  -query "DELETE FROM tbl" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 17. 检查 SQL 改写 rmparenthesis
@test "Check Soar SQL Rewrite Rmparenthesis " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "rmparenthesis" \
  -query "select col from a where (col = 1)" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 18. 检查 SQL 改写 delimiter
@test "Check Soar SQL Rewrite Delimiter " {
  ${SOAR_BIN} -report-type "rewrite" -rewrite-rules "delimiter" \
  -query "use sakila" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}
