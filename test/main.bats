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
  echo "${output}"
  [ $status -eq 0 ]
}

# 5. soar 使用 config 配置文件路径是否正确
# 13. soar -check-config 数据库连接配置检查 *
# soar 数据库测试（线上、线下、-allow-online-as-test）
@test "Check config cases" {
  run ${SOAR_BIN_ENV} -check-config
  [ $status -eq 0 ]
  [ -z ${output} ]
}

# 6. soar 使用配置文件修改默认参数是否正确
# 注意：不启用的配置为默认配置项目
@test "Check the default config of the changes" {
  ${SOAR_BIN} -config ${BATS_FIXTURE_DIRNAME}/${BATS_TEST_NAME}.golden -print-config  -log-output=/dev/null > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}

# 8. 执行 soar -query 为文件时是否正常
@test "Check soar query for input file" {
  ${SOAR_BIN} -query <(${SOAR_BIN} -list-test-sqls) > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}

# 9. 管道输入 SQL 是否正常
@test "Check soar for pipe input" {
  ${SOAR_BIN} -list-test-sqls |${SOAR_BIN} > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}
# 10. report 为 json 格式是否正常
@test "Check soar report for json" {
  ${SOAR_BIN} -query "select * from film" \
    -report-type json > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}

# 10. report 为 markdown 格式是否正常
@test "Check soar report for markdown" {
  ${SOAR_BIN} -query "select * from film" \
    -report-type markdown > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}

# 11. report 格式 html 检查
@test "Check soar report for html" {
  ${SOAR_BIN} -query "select * from film" \
    -report-title "soar report check" \
    -report-type html > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}

# 12. 黑名单功能是否正常
# soar 的日志和黑名单的相对路径都相对于 soar 的二进制文件路径说的
@test "Check soar blacklist" {
  run ${SOAR_BIN} -blacklist ../etc/soar.blacklist -query "show processlist;"
  [ $status -eq 0 ]
  [ -z ${output} ]
}

# 13. soar -check-config 数据库连接配置检查 *
# 参见 5

# 14. soar -help 检查
@test "Check soar help" {
  run ${SOAR_BIN} -help
  [ $status -eq 2 ]
  [ "${#lines[@]}" -gt 30 ]
}

# 15. soar 数据库测试（线上、线下、-allow-online-as-test）
# 参见 5

# 16. 语法检查（正确）
@test "Syntax Check OK" {
  run ${SOAR_BIN} -query "select * from film" -only-syntax-check
  [ $status -eq 0 ]
  [ -z $ouput ]
}
# 16. 语法检查（错误）
@test "Syntax Check Error" {
  run ${SOAR_BIN} -query "select * frm film" -only-syntax-check
  [ $status -eq 1 ]
  [ -n $ouput ]
}

# 17. dsn 检查
@test "Check soar test dsn root:passwd@host:port/db" {
  run ${SOAR_BIN} -online-dsn="root:pase@D@192.168.12.11:3306/testDB" -print-config
  [ $(expr "$output" : ".*user: root") -ne 0 ]
  [ $(expr "$output" : ".*addr: 192.168.12.11:3306") -ne 0 ]
  [ $(expr "$output" : ".*schema: testDB") -ne 0 ]
  [ $(expr "$output" : ".*charset: utf8") -ne 0 ]
}

# 18. 日志中是否含有密码
@test "Check log has password" {
  ${SOAR_BIN_ENV} -query "select * from film" -log-level=7
  run grep "1tIsB1g3rt" ${SOAR_BIN}.log
  [ ${status} -eq 1 ]
}

# 18. 输出中是否含有密码
@test "Check stdout has password" {
  run ${SOAR_BIN_ENV} -query "select * from film" -log-level=7
  [ $(expr "$output" : ".*1tIsB1g3rt.*") -eq 0 ]
  [ ${status} -eq 0 ]
}

# 20. 单条 SQL 中 JOIN 表的最大数量超过 2
@test "Check Max Join Table Count Overflow" {
  ${SOAR_BIN}  -max-join-table-count 2 -query="select a from b join c join d" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}

# 21. 单条 SQL 中 JOIN 表未超过时是否正常默认为 5
@test "Check Max Join Table Count Default" {
  ${SOAR_BIN} -query="select a from b join c join d" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff
  echo "${output}"
  [ $status -eq 0 ]
}
