#!/usr/bin/env bats

load test_helper

@test "Simple Query Optimizer" {
  run ${SOAR_BIN} -query "select * from film where length > 120"
  [ $status -eq 0 ]
}
