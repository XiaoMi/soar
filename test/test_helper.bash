setup() {
  export SOAR_DEV_DIRNAME="${BATS_TEST_DIRNAME}/../"
  export SOAR_BIN="${SOAR_DEV_DIRNAME}/bin/soar" 
  export SOAR_BIN_ENV="${SOAR_DEV_DIRNAME}/bin/soar -config ${SOAR_DEV_DIRNAME}/etc/soar.yaml" 
  export BATS_TMP_DIRNAME="${BATS_TEST_DIRNAME}/tmp"
  export BATS_FIXTURE_DIRNAME="${BATS_TEST_DIRNAME}/fixture"
  mkdir -p "${BATS_TMP_DIRNAME}"
}

golden_diff() {
  FUNC_NAME=$1
  diff "${BATS_TMP_DIRNAME}/${FUNC_NAME}.golden" "${BATS_FIXTURE_DIRNAME}/${FUNC_NAME}.golden" >/dev/null
  return $?
}
