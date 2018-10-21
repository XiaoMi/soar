## FAQ

### Dependency Tools

* [git](https://git-scm.co): clone code from git repository
* [go](https://golang.org/): build source
* [govendor](https://github.com/kardianos/govendor): manager third party dependency
* [docker](https://www.docker.com): manager test environment
* [mysql](https://www.mysql.com/): connect test environment
* [retool](https://github.com/twitchtv/retool): manager test tools such as `gometalinter.v2`, `revive`, `golangci-lint`

### Syntax Error

* Unexpected quote, like `, ", '
* vitess syntax not supported yet

### Program running slowly

* SOAR will use online-dsn, test-dsn for data sampling and testing if they are on a different host to access these instance will cost much time. This may cause analyze slowly, especially when you are optimizing lots of queries.
* As mentioned above, if you set `-sampling=true`(by default), data sampling will take some time for more accurate suggestions.

## build test env

```bash
# create test database
wget http://downloads.mysql.com/doc/sakila-db.tar.gz
tar zxf sakila-db.tar.gz && cd sakila-db
mysql -u root -p -f < sakila-schema.sql
mysql -u root -p -f < sakila-data.sql

# create test user
CREATE USER root@'hostname' IDENTIFIED BY "1t'sB1g3rt";
GRANT ALL ON  *.* TO root@'hostname';
```

## update vitess in vendor

`govendor fetch` or `git clone` [vitess](https://github.com/vitessio/vitess) in somewhere maybe very slow or be blocked, so we add vitess source code in vendor directory.

If you what to update vitess package, you should bypass that block using yourself method.

```bash
$ make vitess
```

## HTML Format Report

```bash
#!/bin/bash

soar -query "select * from film" > ./index.html

(
  echo To: youmail@example.com
  echo From: robot@example.com
  echo "Content-Type: text/html; "
  echo Subject: SQL Analyze Report
  echo
  cat ./index.html
) | sendmail -t

```

## Add a new heuristic rule

```bash
advisor/rules.go HeuristicRules add a new item
advisor/heuristic.go add a new rule function
advisor/heuristic_test.go add a new test function
make doc
go test github.com/XiaoMi/soar/advisor -v -update -run TestListHeuristicRules
go test github.com/XiaoMi/soar/advisor -v -update -run TestMergeConflictHeuristicRules
make daily
```
