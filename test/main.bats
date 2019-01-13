#!/usr/bin/env bats

load test_helper

# 1. 检查版本输出格式是否正确
# 2. 检查版本是否为当天编译的
@test "Test soar version" {
  run ${SOAR_BIN} -version
  [ "$status" -eq 0 ]
  [ "${lines[0]%% *}" == "Version:" ]
  [ "${lines[1]%% *}" == "Branch:" ]
  [ "${lines[2]%% *}" == "Compile:" ]
  [ $(expr "${lines[2]}" : "Compile: $(date +'%Y-%m-%d').*") -ne 0 ]   # 检查当前版本是否为今日编译的
}

# 3. 无参数执行是否正确
@test "No arguments prints message" {
  run ${SOAR_BIN}
  [ $status -eq 1 ]
  [ "${lines[0]}" == 'Args format error, use --help see how to use it!' ]
}

# 4. 检查输出的默认值是否改变 soar -print-config  加log-outpt 是因为日志默认是相对路径 
@test "Run default printconfig cases" {
  ${SOAR_BIN} -print-config -log-output=/dev/null  > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  [ $status -eq 0 ]
}

# 5. soar 使用 config 配置文件路径是否正确
@test "Check config cases" {
  run ${SOAR_BIN_ENV} -check-config
  [ $status -eq 0 ]
  [ -z ${output} ]
}

# 6. soar 使用配置文件修改默认参数是否正确
# 注意  不启用的配置为默认配置项目
@test "check config change default config" {
      ${SOAR_BIN} -config ${BATS_FIXTURE_DIRNAME}/${BATS_TEST_NAME}.golden -print-config  -log-output=/dev/null > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
      # 去掉 2019/01/12 05:45:14.922 [D] [config.go:429]  go-sql-driver/mysql.ParseDSN Error: invalid value / unknown server pub 
      sed  -n '3,$p' ${BATS_TMP_DIRNAME}/test_check_config_change_default_config.golden > ${BATS_TMP_DIRNAME}/test_check_config_change_default_config.golden1
      mv ${BATS_TMP_DIRNAME}/test_check_config_change_default_config.golden1  ${BATS_TMP_DIRNAME}/test_check_config_change_default_config.golden
  run golden_diff
  [ $status -eq 0 ]
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
