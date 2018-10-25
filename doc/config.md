## 配置文件说明

配置文件为[yaml](https://en.wikipedia.org/wiki/YAML)格式。一般情况下只需要配置online-dsn, test-dsn, log-output等少数几个参数。即使不创建配置文件SOAR仍然会给出基本的启发式建议。

默认文件会按照`/etc/soar.yaml`, `./etc/soar.yaml`, `./soar.yaml`顺序加载，找到第一个后不再继续加载后面的配置文件。如需指定其他配置文件可以通过`-config`参数指定。

关于数据库权限`online-dsn`需要相应库表的SELECT权限，`test-dsn`需要root最高权限。

```text
# 线上环境配置
online-dsn:
  addr: 127.0.0.1:3306
  schema: sakila
  user: root
  password: 1t'sB1g3rt
  disable: false
# 测试环境配置
test-dsn:
  addr: 127.0.0.1:3307
  schema: test
  user: root
  password: 1t'sB1g3rt
  disable: false
# 是否允许测试环境与线上环境配置相同
allow-online-as-test: true
# 是否清理测试时产生的临时文件
drop-test-temporary: true
# 语法检查小工具
only-syntax-check: false
sampling-data-factor: 100
sampling: true
# 日志级别，[0:Emergency, 1:Alert, 2:Critical, 3:Error, 4:Warning, 5:Notice, 6:Informational, 7:Debug]
log-level: 7
log-output: ${BASE_DIR}/soar.log
# 优化建议输出格式
report-type: markdown
ignore-rules:
- ""
blacklist: ${BASE_DIR}/soar.blacklist
# 启发式算法相关配置
max-join-table-count: 5
max-group-by-cols-count: 5
max-distinct-count: 5
max-index-cols-count: 5
max-total-rows: 9999999
spaghetti-query-length: 2048
allow-drop-index: false
# EXPLAIN相关配置
explain-sql-report-type: pretty
explain-type: extended
explain-format: traditional
explain-warn-select-type:
- ""
explain-warn-access-type:
- ALL
explain-max-keys: 3
explain-min-keys: 0
explain-max-rows: 10000
explain-warn-extra:
- ""
explain-max-filtered: 100
explain-warn-scalability:
- O(n)
query: ""
list-heuristic-rules: false
list-test-sqls: false
verbose: true
```

## 命令行参数

几乎所有配置文件中指定的参数都通通过命令行参数进行修改，且命令行参数优先级较配置文件优先级高。

```bash
$ soar -h
```

### 命令行参数配置DSN

> 账号密码中如包含特殊符号(如：'@',':','/'等)可在配置文件中设置，存在特殊字符的情况不适合在命令行中使用。目前`soar`只支持tcp协议的MySQL数据库连接方式，如需要配置本机MySQL环境建议将`localhost`修改为'127.0.0.1'，并检查对应的'user'@'127.0.0.1'账号是否存在。

```bash
$ soar -online-dsn "user:password@ip:port/database"

$ soar -test-dsn "user:password@ip:port/database"
```

#### DSN格式支持
* "user:password@127.0.0.1:3307/database"
* "user:password@127.0.0.1:3307"
* "user:password@127.0.0.1:/database"
* "user:password@:3307/database"
* "user:password@"
* "127.0.0.1:3307/database"
* "@127.0.0.1:3307/database"
* "@127.0.0.1"
* "127.0.0.1"
* "@/database"
* "@127.0.0.1:3307"
* "@:3307/database"
* ":3307/database"
* "/database"

### SQL评分

不同类型的建议指定的Severity不同，严重程度数字由低到高依次排序。满分100分，扣到0分为止。L0不扣分只给出建议，L1扣5分，L2扣10分，每级多扣5分以此类推。当由时给出L1, L2两要建议时扣分叠加，即扣15分。

如果您想给出不同的扣分建议或者对指引中的文字内容不满意可以为在git中提ISSUE，也可直接修改rules.go的相应配置然后重新编译自己的版本。

注意：目前只有`markdown`和`html`两种`-report-type`支持评分输出显示，其他输出格式如有评分需求可以按上述规则自行计算。
