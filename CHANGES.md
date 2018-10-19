# 更新日志

## 2018-10
- 2018-10-20 开源先锋日(OSCAR)对外正式开源发布代码

## 2018-09
- 修复多个启发式建议不准确BUG，优化部分建议文案使得建议更清晰
- 基于TiDB Parser完善多个DDL类型语句的建议
- 新增lint report-type类型，支持Vim Plugin优化建议输出
- 更新整理项目文档，开源准备
- 2018-09-21 Gdevops SOAR首次对外进行技术分享宣传

## 2018-08
- 利用docker临时容器进行daily测试
- 添加main_test全功能回归测试
- 修复在测试中发现的问题
- mymysql合并MySQL8.0相关PR，修改vendor依赖
- 改善HeuristicRule中的文案
- 持续集成Vitess Parser的改进
- NewQuery4Audit结构体中引入TiDB Parser
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
- HeuristicRule新增RuleSpaceAfterDot
- 支持字符集和Collation不相同时的隐式数据类型转换的检查

## 2018-06
- 支持更多的SQL Rewrite规则
- 添加SQL执行超时限制
- 索引优化建议支持对约束的检查
- 修复数据采样中null值处理不正确的问题
- Explain支持last_query_cost

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
- 添加Insert、Delete、Update转写成Select的基本函数
- 支持MySQL Explain信息输出
