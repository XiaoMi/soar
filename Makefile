# This how we want to name the binary output
#
# use checkmake linter https://github.com/mrtazz/checkmake
# $ checkmake Makefile
#
BINARY=soar
GOPATH ?= $(shell go env GOPATH)
# Ensure GOPATH is set before running build process.
ifeq "$(GOPATH)" ""
  $(error Please set the environment variable GOPATH before running `make`)
endif
PATH := ${GOPATH}/bin:$(PATH)
GCFLAGS=-gcflags "all=-trimpath=${GOPATH}"
LDFLAGS=-ldflags="-s -w"

# These are the values we want to pass for VERSION  and BUILD
BUILD_TIME=`date +%Y%m%d%H%M`
COMMIT_VERSION=`git rev-parse HEAD`

# colors compatible setting
CRED:=$(shell tput setaf 1 2>/dev/null)
CGREEN:=$(shell tput setaf 2 2>/dev/null)
CYELLOW:=$(shell tput setaf 3 2>/dev/null)
CEND:=$(shell tput sgr0 2>/dev/null)

# Add mysql version for testing `MYSQL_RELEASE=percona MYSQL_VERSION=5.7 make docker`
# MySQL 5.1 `MYSQL_RELEASE=vsamov/mysql-5.1.73 make docker`
# MYSQL_RELEASE: mysql, percona, mariadb ...
# MYSQL_VERSION: latest, 8.0, 5.7, 5.6, 5.5 ...
# use mysql:latest as default
MYSQL_RELEASE := $(or ${MYSQL_RELEASE}, ${MYSQL_RELEASE}, mysql)
MYSQL_VERSION := $(or ${MYSQL_VERSION}, ${MYSQL_VERSION}, latest)

.PHONY: all
all: | fmt build

.PHONY: go_version_check
GO_VERSION_MIN=1.10
# Parse out the x.y or x.y.z version and output a single value x*10000+y*100+z (e.g., 1.9 is 10900)
# that allows the three components to be checked in a single comparison.
VER_TO_INT:=awk '{split(substr($$0, match ($$0, /[0-9\.]+/)), a, "."); print a[1]*10000+a[2]*100+a[3]}'
go_version_check:
	@echo "$(CGREEN)Go version check ...$(CEND)"
	@if test $(shell go version | $(VER_TO_INT) ) -lt \
  	$(shell echo "$(GO_VERSION_MIN)" | $(VER_TO_INT)); \
  	then printf "go version $(GO_VERSION_MIN)+ required, found: "; go version; exit 1; \
		else echo "go version check pass";	fi

# Dependency check
.PHONY: deps
deps:
	@echo "$(CGREEN)Dependency check ...$(CEND)"
	@bash ./deps.sh
	# The retool tools.json is setup from retool-install.sh
	# some packages download need more open internet access
	retool sync
	#retool do gometalinter.v2 --install

# Code format
.PHONY: fmt
fmt: go_version_check
	@echo "$(CGREEN)Run gofmt on all source files ...$(CEND)"
	@echo "gofmt -l -s -w ..."
	@ret=0 && for d in $$(go list -f '{{.Dir}}' ./... | grep -v /vendor/); do \
		gofmt -l -s -w $$d/*.go || ret=$$? ; \
	done ; exit $$ret

# Run golang test cases
.PHONY: test
test:
	@echo "$(CGREEN)Run all test cases ...$(CEND)"
	go test -timeout 10m -race ./...
	@echo "test Success!"

# Rule golang test cases with `-update` flag
.PHONY: test-update
test-update:
	@echo "$(CGREEN)Run all test cases with -update flag ...$(CEND)"
	go test ./... -update
	@echo "test-update Success!"

# Using bats test framework run all cli test cases
# https://github.com/sstephenson/bats
.PHONY: test-cli
test-cli: build
	@echo "$(CGREEN)Run all cli test cases ...$(CEND)"
	bats ./test
	@echo "test-cli Success!"

# Code Coverage
# colorful coverage numerical >=90% GREEN, <80% RED, Other YELLOW
.PHONY: cover
cover: test
	@echo "$(CGREEN)Run test cover check ...$(CEND)"
	go test -coverpkg=./... -coverprofile=coverage.data ./... | column -t
	go tool cover -html=coverage.data -o coverage.html
	go tool cover -func=coverage.data -o coverage.txt
	@tail -n 1 coverage.txt | awk '{sub(/%/, "", $$NF); \
		if($$NF < 80) \
			{print "$(CRED)"$$0"%$(CEND)"} \
		else if ($$NF >= 90) \
			{print "$(CGREEN)"$$0"%$(CEND)"} \
		else \
			{print "$(CYELLOW)"$$0"%$(CEND)"}}'

# Builds the project
build: fmt
	@echo "$(CGREEN)Building ...$(CEND)"
	@mkdir -p bin
	@bash ./genver.sh
	@ret=0 && for d in $$(go list -f '{{if (eq .Name "main")}}{{.ImportPath}}{{end}}' ./...); do \
		b=$$(basename $${d}) ; \
		go build ${GCFLAGS} -o bin/$${b} $$d || ret=$$? ; \
	done ; exit $$ret
	@echo "build Success!"

# Installs our project: copies binaries
install: build
	@echo "$(CGREEN)Install ...$(CEND)"
	go install ./...
	@echo "install Success!"

# Generate doc use -list* command
.PHONY: doc
doc: build
	@echo "$(CGREEN)Auto generate doc ...$(CEND)"
	./bin/soar -list-heuristic-rules > doc/heuristic.md
	./bin/soar -list-rewrite-rules > doc/rewrite.md
	./bin/soar -list-report-types > doc/report_type.md

# Add or change a heuristic rule
.PHONY: heuristic
heuristic: doc
	@echo "$(CGREEN)Update Heuristic rule golden files ...$(CEND)"
	go test github.com/XiaoMi/soar/advisor -v -update -run TestListHeuristicRules
	go test github.com/XiaoMi/soar/advisor -v -update -run TestMergeConflictHeuristicRules
	docker stop soar-mysql 2>/dev/null || true

# Update vitess vendor
.PHONY: vitess
vitess:
	@echo "$(CGREEN)Update vitess deps ...$(CEND)"
	govendor fetch -v vitess.io/vitess/...

# Update tidb vendor
.PHONY: tidb
tidb:
	@echo "$(CGREEN)Update tidb deps ...$(CEND)"
	govendor fetch -v github.com/pingcap/tidb/...

# make pingcap parser
.PHONY: pingcap-parser
pingcap-parser: tidb
	@echo "$(CGREEN)Update pingcap parser deps ...$(CEND)"
	govendor fetch -v github.com/pingcap/parser/...

# Update all vendor
.PHONY: vendor
vendor: vitess pingcap-parser
# gometalinter
.PHONY: lint
lint: build
	@echo "$(CGREEN)Run linter check ...$(CEND)"
	CGO_ENABLED=0 GOMODULE111=off retool do gometalinter.v2 -j 1 --config doc/example/metalinter.json ./...
	GOMODULE111=off retool do revive -formatter friendly --exclude vendor/... -config doc/example/revive.toml ./...
	GOMODULE111=off retool do golangci-lint --tests=false run
	@echo "gometalinter check your code is pretty good"

.PHONY: release
release: build
	@echo "$(CGREEN)Cross platform building for release ...$(CEND)"
	@mkdir -p release
	@for GOOS in darwin linux windows; do \
		for GOARCH in amd64; do \
			for d in $$(go list -f '{{if (eq .Name "main")}}{{.ImportPath}}{{end}}' ./...); do \
				b=$$(basename $${d}) ; \
				echo "Building $${b}.$${GOOS}-$${GOARCH} ..."; \
				GOOS=$${GOOS} GOARCH=$${GOARCH} go build ${GCFLAGS} ${LDFLAGS} -v -o release/$${b}.$${GOOS}-$${GOARCH} $$d 2>/dev/null ; \
			done ; \
		done ;\
	done

.PHONY: docker
docker:
	@echo "$(CGREEN)Build mysql test environment ...$(CEND)"
	@docker stop soar-mysql 2>/dev/null || true
	@docker wait soar-mysql 2>/dev/null >/dev/null || true
	@echo "docker run --name soar-mysql $(MYSQL_RELEASE):$(MYSQL_VERSION)"
	@docker run --name soar-mysql --rm -d \
	-e MYSQL_ROOT_PASSWORD=1tIsB1g3rt \
	-e MYSQL_DATABASE=sakila \
	-p 3306:3306 \
	-v `pwd`/test/sql/init.sql.gz:/docker-entrypoint-initdb.d/init.sql.gz \
	$(MYSQL_RELEASE):$(MYSQL_VERSION) \
	--sql-mode ""

	@echo "waiting for sakila database initializing "
	@timeout=180; while [ $${timeout} -gt 0 ] ; do \
		if ! docker exec soar-mysql mysql --user=root --password=1tIsB1g3rt --host "127.0.0.1" --silent -NBe "do 1" >/dev/null 2>&1 ; then \
			timeout=`expr $$timeout - 1`; \
			printf '.' ;  sleep 1 ; \
		else \
			echo "." ; echo "mysql test environment is ready!" ; break ; \
		fi ; \
		if [ $$timeout = 0 ] ; then \
			echo "." ; echo "$(CRED)docker soar-mysql start timeout(180 s)!$(CEND)" ; exit 1 ; \
		fi ; \
	done

.PHONY: docker-connect
docker-connect:
	@docker exec -it soar-mysql mysql --user=root --password=1tIsB1g3rt --host "127.0.0.1" sakila

# attach docker container with bash interactive mode
.PHONY: docker-it
docker-it:
	docker exec -it soar-mysql /bin/bash

.PHONY: daily
daily: | deps fmt vendor docker cover doc lint release install test-cli clean logo
	@echo "$(CGREEN)daily build finished ...$(CEND)"

# vendor, docker will cost long time, if all those are ready, daily-quick will much more fast.
.PHONY: daily-quick
daily-quick: | deps fmt cover test-cli doc lint logo
	@echo "$(CGREEN)daily-quick build finished ...$(CEND)"

.PHONY: logo
logo:
	@echo "$(CYELLOW)"
	@cat doc/images/logo.ascii
	@echo "$(CEND)"

# Cleans our projects: deletes binaries
.PHONY: clean
clean:
	@echo "$(CGREEN)Cleanup ...$(CEND)"
	go clean
	@for GOOS in darwin linux windows; do \
	    for GOARCH in 386 amd64; do \
			rm -f ${BINARY}.$${GOOS}-$${GOARCH} ;\
		done ;\
	done
	rm -f ${BINARY} coverage.* test/tmp/*
	find . -name "*.log" -delete
	git clean -fi
	docker stop soar-mysql 2>/dev/null || true
