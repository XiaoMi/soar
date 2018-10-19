#!/bin/bash

NEEDED_COMMANDS="mysql docker git go govendor retool"

for cmd in ${NEEDED_COMMANDS} ; do
    if ! command -v "${cmd}" &> /dev/null ; then
        echo -e "\033[91m${cmd} missing\033[0m"
        exit 1
    else
        echo "${cmd} found"
    fi
done
