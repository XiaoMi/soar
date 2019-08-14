# CHANGELOG

## 2019-08
- Fix RuleImplicitConversion(ARG.003) with INT and DECIMAL
- Fix RuleImplicitConversion duplicate suggest when use IN () operator

## 2019-07
- Fix #213 CLA.001 NO WHERE CONDITION
- Fix PRIMARY key append to multi column index
- fingerprint verbose mode add id

## 2019-05
- Fix issue #208 14c19f4 regression bug
- Add max_execution_time hint for explain query
- Fix #205 create index rewrite error

## 2019-04
- Add test case for STA.004
- RuleSpaceWithQuote add list range check
- Fix #199 -report-type=json add score
- Fix #98 JSON result format
- Fix index col compare case sensitive bug
- Fix ARG.008 cases: col = 1 OR col IS NULL
- Fix tokenize bug with multi type of quote

## 2019-02
- add go.mod for go1.11
- add new -report-type query-type
- add new heuristic rule SEC.004
- fix #196 wrong ip/password will cause soar -check-config hangup

## 2019-01

- add mysql environment verbose info
- add JSONFind function, which support JSON iterate
- add new test database `world_x`
- SplitStatement support optimizer hint `/*+xxx */`
- include [bats](https://github.com/bats-core/bats-core) bash auto test framework
- fix #173 with JSONFind `WHERE col = col = '' and col1 = 'xx'`
- fix #184 table status field datatype overflow
- fix explain result with multi rows error
- fix #178 JSON datatype only support utf8mb4

## 2018-12

- replace mysql database driver mymysql with go-sql-driver
- add new -report-type [ast-json, tiast-json]
- command line dsn args support '@', '/', ':' in password
- add new heuristic rule RES.009, "SELECT * FROM tbl WHERE col = col = 'abc'"
- add new heuristic rule RuleColumnNotAllowType COL.018
- add string escape function for security
- fix #122 single table select * don't auto-complete table name
- fix #171 support socket access type
- fix #58 sampling not deal with NULL able string
- fix #172 compatible with mysql 5.1, which explain has no Index_Comment column
- fix #163 column.Tp may be nil, which may raise panic
- fix #151 bit type not config as int, when two columns compare will give ARG.003 suggestion.
- 
## 2018-11

- add all third-party lib into vendor
- support `-report-type chardet`
- add more heuristic rules: TBL.008, KEY.010, ARG.012, KWR.004
- add -cleanup-test-database command-line arg
- add -check-config parameter
- fix #146 pretty cause syntax error
- fix #140 COL.012, COL.015 NULL type about TEXT/BLOB
- fix #141 empty output when query execute failed on mysql
- fix #89 index advisor give wrong database name, `optimizer_xx`
- fix #121 RemoveSQLComment trim space
- fix #120 trimspace before check single line comment
- fix mac os stdout print buffer truncate
- fix -config arg load file error
- fix #116 SplitStatement check if single comment line is in multi-line sql.
- fix #112 multi-line comment will cause line counter error, when -report-type=lint
- fix #110 remove bom before auditing
- fix #104 case insensitive regex @ CLA.009
- fix #87 RuleImplicitConversion value type mismatch check bug
- fix #38 always true where condition check
- abandon stdin terminal interactive mod, which may seems like hangup

## 2018-10

- Fix SplitStatement multistatement eof bug #66
- Fix pretty func hangup issue #47
- Fix some foolish code spell error
- Use travis for CI
- Fix Go 1.8 default GOPATH compatible issue BUG #5
- 2018-10-20 开源先锋日(OSCAR)对外正式开源发布代码

## 2018-09

- 修复多个启发式建议不准确BUG，优化部分建议文案使得建议更清晰
- 基于 TiDB Parser 完善多个 DDL 类型语句的建议
- 新增lint report-type类型，支持Vim Plugin优化建议输出
- 更新整理项目文档，开源准备
- 2018-09-21 Gdevops SOAR首次对外进行技术分享宣传

## 2018-08

- 利用 docker 临时容器进行 daily 测试
- 添加main_test全功能回归测试
- 修复在测试中发现的问题
- mymysql 合并 MySQL8.0 相关PR，修改vendor依赖
- 改善HeuristicRule中的文案
- 持续集成Vitess Parser的改进
- NewQuery4Audit 结构体中引入 TiDB Parser
- 通过TiAST完成大量与 DDL 相关的TODO
- 修改heuristic rules检查的返回值，提升拓展性
- 建议中引入Position，用于表示建议产生于SQL的位置
- 新增多个HeuristicRule
- Makefile中添加依赖检查，优化Makefile中逻辑，添加新功能
- 优化gometalinter性能，引入新的代码质量检测工具，提升代码质量
- 引入 retool 用于管理依赖的工具
- 优化 doc 文档

## 2018-07

- 补充文档，添加项目LOGO
- 改善代码质量提升测试覆盖度
- mymysql升级，支持MySQL 8.0
- 提供remove-comment小工具
- 提供索引重复检查小工具
- HeuristicRule 新增 RuleSpaceAfterDot
- 支持字符集和Collation不相同时的隐式数据类型转换的检查

## 2018-06

- 支持更多的SQL Rewrite规则
- 添加SQL执行超时限制
- 索引优化建议支持对约束的检查
- 修复数据采样中 NULL 值处理不正确的问题
- Explain 支持 last_query_cost

## 2018-05

- 添加数据采样功能
- 添加语句执行安全检查
- 支持DDL语法检查
- 支持DDL在测试环境的执行
- 支持隐式数据类型转换检查
- 支持索引去重
- 索引优化建议支持前缀索引
- 支持SQL Pretty输出

## 2018-04

- 支持语法检查
- 支持测试环境
- 支持MySQL原数据的获取
- 支持基于数据库环境信息给予索引优化建议
- 支持不依赖数据库原信息的简单索引优化建议
- 添加日志模块
- 引入配置文件

## 2018-03

- 基本架构设计
- 添加大量底层函数用于处理AST
- 添加Insert、Delete、Update 转写成 Select 的基本函数
- 支持MySQL Explain信息输出
