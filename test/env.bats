#!/usr/bin/env bats

load test_helper

@test "Simple Query Optimizer" {
  ${SOAR_BIN_ENV} -query "select * from film where length > 120" | sed "s/ [0-9.]*%/ n%/g" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "$output"
  [ $status -eq 0 ]
}

@test "Run all test cases" {
  ${SOAR_BIN} -list-test-sqls | ${SOAR_BIN_ENV} | sed "s/ [0-9.]*%/ n%/g" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "$output"
  [ $status -eq 0 ]
}

@test "Check dial timeout" {
  run timeout 1 ${SOAR_BIN} -test-dsn "1.1.1.1" -check-config
  echo "$output"
  [ $status -eq 124 ]
}

# 12. 带数据库连接时黑名单功能是否正常
# soar 的日志和黑名单的相对路径都相对于 soar 的二进制文件路径说的
@test "Check Soar With Mysql Connect Blacklist" {
  run ${SOAR_BIN_ENV} -blacklist ../etc/soar.blacklist -query "show processlist;"
  [ $status -eq 0 ]
  [ -z ${output} ]
}
