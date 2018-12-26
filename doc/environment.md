## 集成环境

![集成环境](https://raw.githubusercontent.com/XiaoMi/soar/master/doc/images/env.png)

| 线上环境 | 测试环境 |   场景                          |
| ---      | ---      | ---                             |
|    有    |    有    | 日常优化，完整的建议，推荐      |
|    无    |    有    | 新申请资源，环境初始化测试      |
|    无    |    无    | 盲测，试用，无EXPLAIN和索引建议 |
|    有    |    无    | 用线上环境当测试环境，不推荐    |

## 线上环境

* 数据字典
* 数据采样
* EXPLAIN

## 测试环境

* 库表映射
* 语法检查
* 模拟执行
* 索引建议/去重

## 注意
* 测试环境 MySQL 版本必须高于或等于线上环境
* 测试环境需要所有权限(建议通过[docker](https://hub.docker.com/_/mysql/)启动)，线上环境至少需要只读权限
