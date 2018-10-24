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

# These are the values we want to pass for VERSION  and BUILD
BUILD_TIME=`date +%Y%m%d%H%M`
COMMIT_VERSION=`git rev-parse HEAD`
GO_VERSION_MIN=1.10

# Add mysql version for testing `MYSQL_VERSION=5.7 make docker`
# use mysql:latest as default
MYSQL_VERSION := $(or ${MYSQL_VERSION}, ${MYSQL_VERSION}, latest)

.PHONY: all
all: | fmt build

# Dependency check
.PHONY: deps
deps:
	@echo "\033[92mDependency check\033[0m"
	@bash ./deps.sh
	# The retool tools.json is setup from retool-install.sh
	retool sync
	retool do gometalinter.v2 intall

# Code format
.PHONY: fmt
fmt:
	@echo "\033[92mRun gofmt on all source files ...\033[0m"
	@echo "gofmt -l -s -w ..."
	@ret=0 && for d in $$(go list -f '{{.Dir}}' ./... | grep -v /vendor/); do \
		gofmt -l -s -w $$d/*.go || ret=$$? ; \
	done ; exit $$ret

# Run golang test cases
.PHONY: test
test:
	@echo "\033[92mRun all test cases ...\033[0m"
	go test ./...
	@echo "test Success!"

# Code Coverage
# colorful coverage numerical >=90% GREEN, <80% RED, Other YELLOW
.PHONY: cover
cover: test
	@echo "\033[92mRun test cover check ...\033[0m"
	go test -coverpkg=./... -coverprofile=coverage.data ./... | column -t
	go tool cover -html=coverage.data -o coverage.html
	go tool cover -func=coverage.data -o coverage.txt
	@tail -n 1 coverage.txt | awk '{sub(/%/, "", $$NF); \
		if($$NF < 80) \
			{print "\033[91m"$$0"%\033[0m"} \
		else if ($$NF >= 90) \
			{print "\033[92m"$$0"%\033[0m"} \
		else \
			{print "\033[93m"$$0"%\033[0m"}}'

# Builds the project
build: fmt tidb-parser
	@echo "\033[92mBuilding ...\033[0m"
	@bash ./genver.sh $(GO_VERSION_MIN)
	@ret=0 && for d in $$(go list -f '{{if (eq .Name "main")}}{{.ImportPath}}{{end}}' ./...); do \
		go build $$d || ret=$$? ; \
	done ; exit $$ret
	@echo "build Success!"

.PHONY: fast
fast:
	@echo "\033[92mBuilding ...\033[0m"
	@bash ./genver.sh $(GO_VERSION_MIN)
	@ret=0 && for d in $$(go list -f '{{if (eq .Name "main")}}{{.ImportPath}}{{end}}' ./...); do \
		go build $$d || ret=$$? ; \
	done ; exit $$ret
	@echo "build Success!"

# Installs our project: copies binaries
install: build
	@echo "\033[92mInstall ...\033[0m"
	go install ./...
	@echo "install Success!"

# Generate doc use -list* command
.PHONY: doc
doc: fast
	@echo "\033[92mAuto generate doc ...\033[0m"
	./soar -list-heuristic-rules > doc/heuristic.md
	./soar -list-rewrite-rules > doc/rewrite.md
	./soar -list-report-types > doc/report_type.md

# Add or change a heuristic rule
.PHONY: heuristic
heuristic: doc docker
	@echo "\033[92mUpdate Heuristic rule golden files ...\033[0m"
	go test github.com/XiaoMi/soar/advisor -v -update -run TestListHeuristicRules
	go test github.com/XiaoMi/soar/advisor -v -update -run TestMergeConflictHeuristicRules
	docker stop soar-mysql 2>/dev/null || true

# Update vitess vendor
.PHONY: vitess
vitess:
	@echo "\033[92mUpdate vitess deps ...\033[0m"
	govendor fetch -v vitess.io/vitess/...

# Update tidb vendor
.PHONY: tidb
tidb:
	@echo "\033[92mUpdate tidb deps ...\033[0m"
	@echo -n "Current TiDB commit hash: "
	@(cd ${GOPATH}/src/github.com/pingcap/tidb/ 2>/dev/null && git checkout master && git rev-parse HEAD) || echo "(init)"
	go get -v -u github.com/pingcap/tidb/store/tikv
	@echo -n "TiDB update to: "
	@cd ${GOPATH}/src/github.com/pingcap/tidb/ && git rev-parse HEAD

# Update all vendor
.PHONY: vendor
vendor: vitess tidb

# make tidb parser
.PHONY: tidb-parser
tidb-parser: tidb
	@echo "\033[92mimporting tidb sql parser ...\033[0m"
	@cd ${GOPATH}/src/github.com/pingcap/tidb && git checkout ec9672cea6612481b1da845dbab620b7a5581ca4  && make parser

# gometalinter
# 如果有不想改的lint问题可以使用metalinter.sh加黑名单
#@bash doc/example/metalinter.sh
.PHONY: lint
lint: build
	@echo "\033[92mRun linter check ...\033[0m"
	CGO_ENABLED=0 retool do gometalinter.v2 -j 1 --config doc/example/metalinter.json ./...
	retool do revive -formatter friendly --exclude vendor/... -config doc/example/revive.toml ./...
	retool do golangci-lint --tests=false run
	@echo "gometalinter check your code is pretty good"

.PHONY: release
release: deps build
	@echo "\033[92mCross platform building for release ...\033[0m"
	@for GOOS in darwin linux windows; do \
		for GOARCH in amd64; do \
			for d in $$(go list -f '{{if (eq .Name "main")}}{{.ImportPath}}{{end}}' ./...); do \
				b=$$(basename $${d}) ; \
				echo "Building $${b}.$${GOOS}-$${GOARCH} ..."; \
				GOOS=$${GOOS} GOARCH=$${GOARCH} go build -ldflags="-s -w" -v -o $${b}.$${GOOS}-$${GOARCH} $$d 2>/dev/null ; \
			done ; \
		done ;\
	done

.PHONY: docker
docker:
	@echo "\033[92mBuild mysql test enviorment\033[0m"
	@docker stop soar-mysql 2>/dev/null || true
	@echo "docker run --name soar-mysql mysql:$(MYSQL_VERSION)"
	@docker run --name soar-mysql --rm -d \
	-e MYSQL_ROOT_PASSWORD=1tIsB1g3rt \
	-e MYSQL_DATABASE=sakila \
	-p 3306:3306 \
	-v `pwd`/doc/example/sakila.sql.gz:/docker-entrypoint-initdb.d/sakila.sql.gz \
	mysql:$(MYSQL_VERSION)

	@echo -n "waiting for sakila database initializing "
	@while ! mysql -h 127.0.0.1 -u root sakila -p1tIsB1g3rt -NBe "do 1;" 2>/dev/null; do \
	printf '.' ; \
	sleep 1 ; \
	done ; \
	echo '.'
	@echo "mysql test enviorment is ready!"

.PHONY: connect
connect:
	mysql -h 127.0.0.1 -u root -p1tIsB1g3rt

.PHONY: main_test
main_test: install
	@echo "\033[92mrunning main_test\033[0m"
	@echo "soar -list-test-sqls | soar"
	@./doc/example/main_test.sh
	@echo "main_test Success!"

.PHONY: daily
daily: | deps fmt vendor tidb-parser docker cover doc lint release install main_test clean logo
	@echo "\033[92mdaily build finished\033[0m"

.PHONY: logo
logo:
	@echo "\033[93m"
	@cat doc/images/logo.ascii
	@echo "\033[m"

# Cleans our projects: deletes binaries
.PHONY: clean
clean:
	@echo "\033[92mCleanup ...\033[0m"
	go clean
	@for GOOS in darwin linux windows; do \
	    for GOARCH in 386 amd64; do \
			rm -f ${BINARY}.$${GOOS}-$${GOARCH} ;\
		done ;\
	done
	rm -f ${BINARY} coverage.*
	find . -name "*.log" -delete
	git clean -fi
	docker stop soar-mysql 2>/dev/null || true
