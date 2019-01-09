#!/usr/bin/env bats

load test_helper

@test "Simple Query Optimizer" {
  ${SOAR_BIN_ENV} -query "select * from film where length > 120" | grep -v "散粒度" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

@test "Run all test cases" {
  ${SOAR_BIN} -list-test-sqls | ${SOAR_BIN_ENV} | grep -v "散粒度" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

