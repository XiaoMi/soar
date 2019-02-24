setup() {
  export SOAR_DEV_DIRNAME="${BATS_TEST_DIRNAME}/../"
  export SOAR_BIN="${SOAR_DEV_DIRNAME}/bin/soar" 
  export SOAR_BIN_ENV="${SOAR_DEV_DIRNAME}/bin/soar -config ${SOAR_DEV_DIRNAME}/etc/soar.yaml" 
  export BATS_TMP_DIRNAME="${BATS_TEST_DIRNAME}/tmp"
  export BATS_FIXTURE_DIRNAME="${BATS_TEST_DIRNAME}/fixture"
  export LC_ALL=C # Linux macOS 下 sort 排序问题
  mkdir -p "${BATS_TMP_DIRNAME}"
}

# golden_diff like gofmt golden file check method, use this function check output different with template
golden_diff() {
  diff "${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden" "${BATS_FIXTURE_DIRNAME}/${BATS_TEST_NAME}.golden"
}
