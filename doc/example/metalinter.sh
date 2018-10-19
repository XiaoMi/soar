#!/bin/bash

METABIN=$(which gometalinter.v1)
PROJECT_PATH=${GOPATH}/src/github.com/XiaoMi/soar/

if [ "x$METABIN" == "x" ]; then
	go get -u gopkg.in/alecthomas/gometalinter.v1
	${GOPATH}/bin/gometalinter.v1 --install
fi

UPDATE=$1

if [ "${UPDATE}X" != "X" ]; then
	${GOPATH}/bin/gometalinter.v1 --config ${PROJECT_PATH}/doc/example/metalinter.json ./... | tr -d [0-9] | sort > ${PROJECT_PATH}/doc/example/metalinter.txt	
else
	cd ${PROJECT_PATH} && diff <(${GOPATH}/bin/gometalinter.v1 --config ${PROJECT_PATH}/doc/example/metalinter.json ./... | tr -d [0-9] | sort) <(cat ${PROJECT_PATH}/doc/example/metalinter.txt)
fi

