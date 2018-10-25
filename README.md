![SOAR](https://raw.githubusercontent.com/XiaoMi/soar/master/doc/images/logo.png)

[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/xiaomi-dba/soar) 
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](http://github.com/XiaoMi/soar/blob/master/LICENSE) 
[![Go Report Card](https://goreportcard.com/badge/github.com/XiaoMi/soar)](https://goreportcard.com/report/github.com/XiaoMi/soar) 
[![Build Status](https://travis-ci.org/XiaoMi/soar.svg?branch=master)](https://travis-ci.org/XiaoMi/soar)
[![GoDoc](https://godoc.org/github.com/XiaoMi/soar?status.svg)](https://godoc.org/github.com/XiaoMi/soar)


[文档](http://github.com/XiaoMi/soar/tree/master/doc) | [FAQ](http://github.com/XiaoMi/soar/blob/master/doc/FAQ.md) | [变更记录](http://github.com/XiaoMi/soar/blob/master/CHANGES.md) | [路线图](http://github.com/XiaoMi/soar/blob/master/doc/roadmap.md) | [English](http://github.com/XiaoMi/soar/blob/master/README_EN.md)

## SOAR

SOAR(SQL Optimizer And Rewriter)是一个对SQL进行优化和改写的自动化工具。 由小米人工智能与云平台的数据库团队开发与维护。

## 功能特点
* 跨平台支持（支持Linux, Mac环境，Windows环境理论上也支持，不过未全面测试）
* 目前只支持MySQL语法族协议的SQL优化
* 支持基于启发式算法的语句优化
* 支持复杂查询的多列索引优化（UPDATE, INSERT, DELETE, SELECT）
* 支持EXPLAIN信息丰富解读
* 支持SQL指纹、压缩和美化
* 支持同一张表多条ALTER请求合并
* 支持自定义规则的SQL改写

## 快速入门

* [安装使用](http://github.com/XiaoMi/soar/blob/master/doc/install.md)
* [体系架构](http://github.com/XiaoMi/soar/blob/master/doc/structure.md)
* [配置文件](http://github.com/XiaoMi/soar/blob/master/doc/config.md)
* [常用命令](http://github.com/XiaoMi/soar/blob/master/doc/cheatsheet.md)
* [产品对比](http://github.com/XiaoMi/soar/blob/master/doc/comparison.md)
* [路线图](http://github.com/XiaoMi/soar/blob/master/doc/roadmap.md)

## 交流与反馈

* 欢迎通过Github Issues提交问题报告与建议
* QQ群: 779359816
* [Gitter](https://gitter.im/xiaomi-dba/soar) 推荐

 <img src="https://raw.githubusercontent.com/XiaoMi/soar/master/doc/images/qq.png" width="180" hegiht="200" alt="soar-qq" />  <img src="https://raw.githubusercontent.com/XiaoMi/soar/master/doc/images/miops.jpeg" width="260" hegiht="200" alt="XiaoMI-SA" />


## License

[Apache License 2.0](http://github.com/XiaoMi/soar/blob/master/LICENSE).

