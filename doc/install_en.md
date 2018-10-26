## Get Released Binary

```bash
wget https://github.com/XiaoMi/soar/releases/download/${tag}/soar.linux-amd64 -O soar
chmod a+x soar
eg.
wget https://github.com/XiaoMi/soar/releases/download/v0.8.0/soar.linux-amd64 -O soar
chmod a+x soar
```

## Build From Source

```bash
go get -d github.com/XiaoMi/soar
cd ${GOPATH}/src/github.com/XiaoMi/soar && make
```

## Simple Test Case

```bash
echo 'select * from film' | ./soar
```
