## 下载二进制安装包

```bash
wget https://github.com/XiaoMi/soar/releases/download/${tag}/soar.linux-amd64 -O soar
chmod a+x soar
如：
wget https://github.com/XiaoMi/soar/releases/download/v0.8.1/soar.linux-amd64 -O soar
chmod a+x soar
```

## 源码安装

### 依赖软件

一般依赖

* Go 1.10+
* git

高级依赖（仅面向开发人员）

* [mysql](https://dev.mysql.com/doc/refman/8.0/en/mysql.html) 客户端版本需要与容器中MySQL版本相同，避免出现由于认证原因导致无法连接问题
* [docker](https://docs.docker.com/engine/reference/commandline/cli/) MySQL Server测试容器管理
* [govendor](https://github.com/kardianos/govendor) Go包管理
* [retool](https://github.com/twitchtv/retool) 依赖外部代码质量静态检查工具二进制文件管理

### 生成二进制文件

```bash
go get -d github.com/XiaoMi/soar
cd ${GOPATH}/src/github.com/XiaoMi/soar && make
```

### 开发调试

如下指令如果您没有精力参与SOAR的开发可以跳过。

* make deps 依赖检查
* make vitess 升级Vitess Parser依赖
* make tidb 升级TiDB Parser依赖
* make fmt 代码格式化，统一风格
* make lint 代码质量检查
* make docker 启动一个MySQL测试容器，可用于测试依赖元数据检查的功能或不同版本MySQL差异
* make test 运行所有的测试用例
* make cover 代码测试覆盖度检查
* make doc 自动生成命令行参数中-list-XX相关文档
* make daily 每日构建，时刻跟进Vitess, TiDB依赖变化
* make release 生成Linux, Windows, Mac发布版本

## 安装验证

```bash
echo 'select * from film' | ./soar
```
