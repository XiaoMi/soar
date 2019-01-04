#!/bin/bash

NEEDED_COMMANDS="docker git go govendor retool bats"

for cmd in ${NEEDED_COMMANDS} ; do
    if ! command -v "${cmd}" &> /dev/null ; then
        echo -e "\033[91m${cmd} missing\033[0m"
        exit 1
    else
        echo "${cmd} found"
    fi
done

# Docker
## https://www.docker.com

# Git
## https://git-scm.com/

# Go
## https://golang.org/

# Govendor
## go get github.com/kardianos/govendor

# retool
## go get github.com/twitchtv/retool

# bats https://github.com/sstephenson/bats
## Ubuntu: apt-get install bats
## Mac: brew install bats
