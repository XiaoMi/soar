#!/usr/bin/env bats

load test_helper

@test "Test soar version" {
  run ${SOAR_BIN} -version
  [ "$status" -eq 0 ]
  [ "${lines[0]%% *}" == "Version:" ]
  [ "${lines[1]%% *}" == "Branch:" ]
  [ "${lines[2]%% *}" == "Compile:" ]
  [ $(expr "${lines[2]}" : "Compile: $(date +'%Y-%m-%d').*") -ne 0 ]
}

@test "No arguments prints message" {
  run ${SOAR_BIN}
  [ $status -eq 1 ]
  [ "${lines[0]}" == 'Args format error, use --help see how to use it!' ]
}

@test "Run default printconfig cases" {
  ${SOAR_BIN} -print-config -log-output=/dev/null  > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

@test "Check config cases" {
  run ${SOAR_BIN_ENV} -check-config
  [ $status -eq 0 ]
  [ -z ${output} ]
}

@test "Syntax Check OK" {
  run ${SOAR_BIN} -query "select * from film" -only-syntax-check
  [ $status -eq 0 ]
  [ -z $ouput ]
}

@test "Syntax Check Error" {
  run ${SOAR_BIN} -query "select * frm film" -only-syntax-check
  [ $status -eq 1 ]
  [ -n $ouput ]
}
