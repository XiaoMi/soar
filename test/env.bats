#!/usr/bin/env bats

load test_helper

@test "Simple Query Optimizer" {
  ${SOAR_BIN_ENV} -query "select * from film where length > 120" | grep -v "散粒度" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "$output"
  [ $status -eq 0 ]
}

@test "Run all test cases" {
  ${SOAR_BIN} -list-test-sqls | ${SOAR_BIN_ENV} | grep -v "散粒度" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "$output"
  [ $status -eq 0 ]
}

@test "Check dial timeout" {
  run timeout 1 ${SOAR_BIN} -test-dsn "1.1.1.1" -check-config
  [ $status -eq 124 ]
}
