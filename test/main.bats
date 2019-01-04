#!/usr/bin/env bats

load test_helper

@test "Simple Query Optimizer" {
	${SOAR_DEV_DIRNAME}/bin/soar -config ${SOAR_DEV_DIRNAME}/etc/soar.yaml -query "select * from film where length > 120" | grep -v "散粒度" > ${BATS_TMP_DIRNAME}/${BATS_TEST_NAME}.golden
  run golden_diff ${BATS_TEST_NAME}
	[ $status -eq 0 ]
}

@test "Syntax Check" {
	run ${SOAR_DEV_DIRNAME}/bin/soar -query "select * frm film" -only-syntax-check
	[ $status -eq 1 ]
}
