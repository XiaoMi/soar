#!/bin/bash


PROJECT_PATH=${GOPATH}/src/github.com/XiaoMi/soar/

if [ "$1x" == "-updatex" ]; then
	cd "${PROJECT_PATH}" && ./soar -list-test-sqls | ./soar -config=./etc/soar.yaml > ./doc/example/main_test.md
else
	cd "${PROJECT_PATH}" && ./soar -list-test-sqls | ./soar -config=./etc/soar.yaml > ./doc/example/main_test.log
    # optimizer_XXX 库名，散粒度，以及索引先后顺序每次可能会不一致
    DIFF_LINES=$(cat ./doc/example/main_test.log ./doc/example/main_test.md | grep -v "optimizer\|散粒度" | sort | uniq -u | wc -l)
    if [ "${DIFF_LINES}" -gt 0 ]; then
    	git diff ./doc/example/main_test.log ./doc/example/main_test.md
    fi
fi


