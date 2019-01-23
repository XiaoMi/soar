#!/usr/bin/env bats

load test_helper

@test "Check Query Optimizer" {
  run ${SOAR_BIN} -query "select * from film where length > 120"
  [ $status -eq 0 ]
}

@test "Check get tables from SQL" {
  ${SOAR_BIN} -list-test-sqls | ${SOAR_BIN} -report-type tables > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  ${SOAR_BIN} -list-test-sqls | ${SOAR_BIN} -report-type tables -test-dsn "/sakila" >> ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}

