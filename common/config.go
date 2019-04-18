/*
 * Copyright 2018 Xiaomi, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	yaml "gopkg.in/yaml.v2"
)

var (
	// BlackList 黑名单中的SQL不会被评审
	BlackList []string
	// PrintConfig -print-config
	PrintConfig bool
	// PrintVersion -print-config
	PrintVersion bool
	// CheckConfig -check-config
	CheckConfig bool
	// 防止 readCmdFlags 函数重入
	hasParsed bool
)

// Configuration 配置文件定义结构体
type Configuration struct {
	// +++++++++++++++测试环境+++++++++++++++++
	OnlineDSN               *Dsn   `yaml:"online-dsn"`                // 线上环境数据库配置
	TestDSN                 *Dsn   `yaml:"test-dsn"`                  // 测试环境数据库配置
	AllowOnlineAsTest       bool   `yaml:"allow-online-as-test"`      // 允许 Online 环境也可以当作 Test 环境
	DropTestTemporary       bool   `yaml:"drop-test-temporary"`       // 是否清理Test环境产生的临时库表
	CleanupTestDatabase     bool   `yaml:"cleanup-test-database"`     // 清理残余的测试数据库（程序异常退出或未开启drop-test-temporary）  issue #48
	OnlySyntaxCheck         bool   `yaml:"only-syntax-check"`         // 只做语法检查不输出优化建议
	SamplingStatisticTarget int    `yaml:"sampling-statistic-target"` // 数据采样因子，对应 PostgreSQL 的 default_statistics_target
	Sampling                bool   `yaml:"sampling"`                  // 数据采样开关
	SamplingCondition       string `yaml:"sampling-condition"`        // 指定采样条件，如：WHERE xxx LIMIT xxx;
	Profiling               bool   `yaml:"profiling"`                 // 在开启数据采样的情况下，在测试环境执行进行profile
	Trace                   bool   `yaml:"trace"`                     // 在开启数据采样的情况下，在测试环境执行进行Trace
	Explain                 bool   `yaml:"explain"`                   // Explain开关
	Delimiter               string `yaml:"delimiter"`                 // SQL分隔符

	// +++++++++++++++日志相关+++++++++++++++++
	// 日志级别，这里使用了 beego 的 log 包
	// [0:Emergency, 1:Alert, 2:Critical, 3:Error, 4:Warning, 5:Notice, 6:Informational, 7:Debug]
	LogLevel int `yaml:"log-level"`
	// 日志输出位置，默认日志输出到控制台
	// 目前只支持['console', 'file']两种形式，如非console形式这里需要指定文件的路径，可以是相对路径
	LogOutput string `yaml:"log-output"`
	// 优化建议输出格式，目前支持: json, text, markdown格式，如指定其他格式会给 pretty.Println 的输出
	ReportType string `yaml:"report-type"`
	// 当 ReportType 为 html 格式时使用的 css 风格，如不指定会提供一个默认风格。CSS可 以是本地文件，也可以是一个URL
	ReportCSS string `yaml:"report-css"`
	// 当 ReportType 为 html 格式时使用的 javascript 脚本，如不指定默认会加载SQL pretty 使用的 javascript。像CSS一样可以是本地文件，也可以是一个URL
	ReportJavascript string `yaml:"report-javascript"`
	// 当ReportType 为 html 格式时，HTML 的 title
	ReportTitle string `yaml:"report-title"`
	// blackfriday markdown2html config
	MarkdownExtensions int `yaml:"markdown-extensions"` // markdown 转 html 支持的扩展包, 参考blackfriday
	MarkdownHTMLFlags  int `yaml:"markdown-html-flags"` // markdown 转 html 支持的 flag, 参考blackfriday, default 0

	// ++++++++++++++优化建议相关++++++++++++++
	IgnoreRules          []string `yaml:"ignore-rules"`              // 忽略的优化建议规则
	RewriteRules         []string `yaml:"rewrite-rules"`             // 生效的重写规则
	BlackList            string   `yaml:"blacklist"`                 // blacklist 中的 SQL 不会被评审，可以是指纹，也可以是正则
	MaxJoinTableCount    int      `yaml:"max-join-table-count"`      // 单条 SQL 中 JOIN 表的最大数量
	MaxGroupByColsCount  int      `yaml:"max-group-by-cols-count"`   // 单条 SQL 中 GroupBy 包含列的最大数量
	MaxDistinctCount     int      `yaml:"max-distinct-count"`        // 单条 SQL 中 Distinct 的最大数量
	MaxIdxColsCount      int      `yaml:"max-index-cols-count"`      // 复合索引中包含列的最大数量
	MaxTextColsCount     int      `yaml:"max-text-cols-count"`       // 表中含有的 text/blob 列的最大数量
	MaxTotalRows         uint64   `yaml:"max-total-rows"`            // 计算散粒度时，当数据行数大于 MaxTotalRows 即开启数据库保护模式，散粒度返回结果可信度下降
	MaxQueryCost         int64    `yaml:"max-query-cost"`            // last_query_cost 超过该值时将给予警告
	SpaghettiQueryLength int      `yaml:"spaghetti-query-length"`    // SQL最大长度警告，超过该长度会给警告
	AllowDropIndex       bool     `yaml:"allow-drop-index"`          // 允许输出删除重复索引的建议
	MaxInCount           int      `yaml:"max-in-count"`              // IN()最大数量
	MaxIdxBytesPerColumn int      `yaml:"max-index-bytes-percolumn"` // 索引中单列最大字节数，默认767
	MaxIdxBytes          int      `yaml:"max-index-bytes"`           // 索引总长度限制，默认3072
	AllowCharsets        []string `yaml:"allow-charsets"`            // 允许使用的 DEFAULT CHARSET
	AllowCollates        []string `yaml:"allow-collates"`            // 允许使用的 COLLATE
	AllowEngines         []string `yaml:"allow-engines"`             // 允许使用的存储引擎
	MaxIdxCount          int      `yaml:"max-index-count"`           // 单张表允许最多索引数
	MaxColCount          int      `yaml:"max-column-count"`          // 单张表允许最大列数
	MaxValueCount        int      `yaml:"max-value-count"`           // INSERT/REPLACE 单次允许批量写入的行数
	IdxPrefix            string   `yaml:"index-prefix"`              // 普通索引建议使用的前缀
	UkPrefix             string   `yaml:"unique-key-prefix"`         // 唯一键建议使用的前缀
	MaxSubqueryDepth     int      `yaml:"max-subquery-depth"`        // 子查询最大尝试
	MaxVarcharLength     int      `yaml:"max-varchar-length"`        // varchar最大长度
	ColumnNotAllowType   []string `yaml:"column-not-allow-type"`     // 字段不允许使用的数据类型
	MinCardinality       float64  `yaml:"min-cardinality"`           // 添加索引散粒度阈值，范围 0~100

	// ++++++++++++++EXPLAIN检查项+++++++++++++
	ExplainSQLReportType   string   `yaml:"explain-sql-report-type"`  // EXPLAIN markdown 格式输出 SQL 样式，支持 sample, fingerprint, pretty 等
	ExplainType            string   `yaml:"explain-type"`             // EXPLAIN方式 [traditional, extended, partitions]
	ExplainFormat          string   `yaml:"explain-format"`           // FORMAT=[json, traditional]
	ExplainWarnSelectType  []string `yaml:"explain-warn-select-type"` // 哪些 select_type 不建议使用
	ExplainWarnAccessType  []string `yaml:"explain-warn-access-type"` // 哪些 access type 不建议使用
	ExplainMaxKeyLength    int      `yaml:"explain-max-keys"`         // 最大 key_len
	ExplainMinPossibleKeys int      `yaml:"explain-min-keys"`         // 最小 possible_keys 警告
	ExplainMaxRows         int64    `yaml:"explain-max-rows"`         // 最大扫描行数警告
	ExplainWarnExtra       []string `yaml:"explain-warn-extra"`       // 哪些 extra 信息会给警告
	ExplainMaxFiltered     float64  `yaml:"explain-max-filtered"`     // filtered 大于该配置给出警告
	ExplainWarnScalability []string `yaml:"explain-warn-scalability"` // 复杂度警告名单
	ShowWarnings           bool     `yaml:"show-warnings"`            // explain extended with show warnings
	ShowLastQueryCost      bool     `yaml:"show-last-query-cost"`     // switch with show status like 'last_query_cost'
	// ++++++++++++++其他配置项+++++++++++++++
	Query              string `yaml:"query"`                 // 需要进行调优的SQL
	ListHeuristicRules bool   `yaml:"list-heuristic-rules"`  // 打印支持的评审规则列表
	ListRewriteRules   bool   `yaml:"list-rewrite-rules"`    // 打印重写规则
	ListTestSqls       bool   `yaml:"list-test-sqls"`        // 打印测试case用于测试
	ListReportTypes    bool   `yaml:"list-report-types"`     // 打印支持的报告输出类型
	Verbose            bool   `yaml:"verbose"`               // verbose模式，会多输出一些信息
	DryRun             bool   `yaml:"dry-run"`               // 是否在预演环境执行
	MaxPrettySQLLength int    `yaml:"max-pretty-sql-length"` // 超出该长度的SQL会转换成指纹输出
}

// Config 默认设置
var Config = &Configuration{
	OnlineDSN:               newDSN(nil),
	TestDSN:                 newDSN(nil),
	AllowOnlineAsTest:       false,
	DropTestTemporary:       true,
	CleanupTestDatabase:     false,
	DryRun:                  true,
	OnlySyntaxCheck:         false,
	SamplingStatisticTarget: 100,
	Sampling:                false,
	Profiling:               false,
	Trace:                   false,
	Explain:                 true,
	Delimiter:               ";",
	MinCardinality:          0,

	MaxJoinTableCount:    5,
	MaxGroupByColsCount:  5,
	MaxDistinctCount:     5,
	MaxIdxColsCount:      5,
	MaxTextColsCount:     2,
	MaxIdxBytesPerColumn: 767,
	MaxIdxBytes:          3072,
	MaxTotalRows:         9999999,
	MaxQueryCost:         9999,
	SpaghettiQueryLength: 2048,
	AllowDropIndex:       false,
	LogLevel:             3,
	LogOutput:            "soar.log",
	ReportType:           "markdown",
	ReportCSS:            "",
	ReportJavascript:     "",
	ReportTitle:          "SQL优化分析报告",
	BlackList:            "",
	AllowCharsets:        []string{"utf8", "utf8mb4"},
	AllowCollates:        []string{},
	AllowEngines:         []string{"innodb"},
	MaxIdxCount:          10,
	MaxColCount:          40,
	MaxValueCount:        100,
	MaxInCount:           10,
	IdxPrefix:            "idx_",
	UkPrefix:             "uk_",
	MaxSubqueryDepth:     5,
	MaxVarcharLength:     1024,
	ColumnNotAllowType:   []string{"boolean"},

	MarkdownExtensions: 94,
	MarkdownHTMLFlags:  0,

	ExplainSQLReportType:   "pretty",
	ExplainType:            "extended",
	ExplainFormat:          "traditional",
	ExplainWarnSelectType:  []string{""},
	ExplainWarnAccessType:  []string{"ALL"},
	ExplainMaxKeyLength:    3,
	ExplainMinPossibleKeys: 0,
	ExplainMaxRows:         10000,
	ExplainWarnExtra:       []string{"Using temporary", "Using filesort"},
	ExplainMaxFiltered:     100.0,
	ExplainWarnScalability: []string{"O(n)"},
	ShowWarnings:           false,
	ShowLastQueryCost:      false,

	IgnoreRules: []string{
		"COL.011",
	},
	RewriteRules: []string{
		"delimiter",
		"orderbynull",
		"groupbyconst",
		"dmlorderby",
		"having",
		"star2columns",
		"insertcolumns",
		"distinctstar",
	},

	ListHeuristicRules: false,
	ListRewriteRules:   false,
	ListTestSqls:       false,
	ListReportTypes:    false,
	MaxPrettySQLLength: 1024,
}

// Dsn Data source name
type Dsn struct {
	User             string            `yaml:"user"`               // Usernames
	Password         string            `yaml:"password"`           // Password (requires User)
	Net              string            `yaml:"net"`                // Network type
	Addr             string            `yaml:"addr"`               // Network address (requires Net)
	Schema           string            `yaml:"schema"`             // Database name
	Charset          string            `yaml:"charset"`            // SET NAMES charset
	Collation        string            `yaml:"collation"`          // Connection collation
	Loc              string            `yaml:"loc"`                // Location for time.Time values
	TLS              string            `yaml:"tls"`                // TLS configuration name
	ServerPubKey     string            `yaml:"server-public-key"`  // Server public key name
	MaxAllowedPacket int               `ymal:"max-allowed-packet"` // Max packet size allowed
	Params           map[string]string `yaml:"params"`             // Other Connection parameters, `SET param=val`, `SET NAMES charset`
	Timeout          string            `yaml:"timeout"`            // Dial timeout
	ReadTimeout      string            `yaml:"read-timeout"`       // I/O read timeout
	WriteTimeout     string            `yaml:"write-timeout"`      // I/O write timeout

	AllowNativePasswords bool `yaml:"allow-native-passwords"` // Allows the native password authentication method
	AllowOldPasswords    bool `yaml:"allow-old-passwords"`    // Allows the old insecure password method

	Disable bool `yaml:"disable"`
	Version int  `yaml:"-"` // 版本自动检查，不可配置
}

// newDSN create default Dsn struct
func newDSN(cfg *mysql.Config) *Dsn {
	dsn := &Dsn{
		Net:                  "tcp",
		Schema:               "information_schema",
		Charset:              "utf8",
		Timeout:              "3s",
		AllowNativePasswords: true,
		Params:               make(map[string]string),
		MaxAllowedPacket:     4 << 20, // 4 MiB

		// Disable: true,
		Version: 99999,
	}
	if cfg == nil {
		return dsn
	}
	dsn.User = cfg.User
	dsn.Password = cfg.Passwd
	dsn.Net = cfg.Net
	dsn.Addr = cfg.Addr
	dsn.Schema = cfg.DBName
	dsn.Params = make(map[string]string)
	for k, v := range cfg.Params {
		dsn.Params[k] = v
	}
	if _, ok := cfg.Params["charset"]; ok {
		dsn.Charset = cfg.Params["charset"]
	}
	dsn.Collation = cfg.Collation
	dsn.Loc = cfg.Loc.String()
	dsn.MaxAllowedPacket = cfg.MaxAllowedPacket
	dsn.ServerPubKey = cfg.ServerPubKey
	dsn.TLS = cfg.TLSConfig
	dsn.Timeout = cfg.Timeout.String()
	dsn.ReadTimeout = cfg.ReadTimeout.String()
	dsn.WriteTimeout = cfg.WriteTimeout.String()
	dsn.AllowNativePasswords = cfg.AllowNativePasswords
	dsn.AllowOldPasswords = cfg.AllowOldPasswords
	return dsn
}

// newMySQLConfig convert Dsn to go-sql-drive Config
func (env *Dsn) newMySQLConfig() (*mysql.Config, error) {
	var err error
	dsn := mysql.NewConfig()

	dsn.User = env.User
	dsn.Passwd = env.Password
	dsn.Net = env.Net
	dsn.Addr = env.Addr
	dsn.DBName = env.Schema
	dsn.Params = make(map[string]string)
	for k, v := range env.Params {
		dsn.Params[k] = v
	}
	dsn.Params["charset"] = env.Charset
	dsn.Collation = env.Collation
	dsn.Loc, err = time.LoadLocation(env.Loc)
	if err != nil {
		return nil, err
	}
	dsn.MaxAllowedPacket = env.MaxAllowedPacket
	dsn.ServerPubKey = env.ServerPubKey
	dsn.TLSConfig = env.TLS
	if env.Timeout != "" {
		dsn.Timeout, err = time.ParseDuration(env.Timeout)
		LogIfError(err, "timeout: '%s'", env.Timeout)
	}
	if env.WriteTimeout != "" {
		dsn.WriteTimeout, err = time.ParseDuration(env.WriteTimeout)
		LogIfError(err, "writeTimeout: '%s'", env.WriteTimeout)
	}
	if env.ReadTimeout != "" {
		dsn.ReadTimeout, err = time.ParseDuration(env.ReadTimeout)
		LogIfError(err, "readTimeout: '%s'", env.ReadTimeout)
	}
	dsn.AllowNativePasswords = env.AllowNativePasswords
	dsn.AllowOldPasswords = env.AllowOldPasswords
	return dsn, err
}

// 解析命令行DSN输入
func parseDSN(odbc string, d *Dsn) *Dsn {
	dsn := newDSN(nil)
	var addr, user, password, schema, charset, timeout string
	if odbc == FormatDSN(d) {
		return d
	}

	if d != nil {
		// 原来有个判断，后来判断条件被删除了就导致第一次addr无论如何都会被修改。所以这边先注释掉
		// addr = d.Addr
		user = d.User
		password = d.Password
		schema = d.Schema
		charset = d.Charset
		timeout = d.Timeout
	}

	// 设置为空表示禁用环境
	odbc = strings.TrimSpace(odbc)
	if odbc == "" {
		return &Dsn{Disable: true}
	}

	var userInfo, hostInfo, query string

	// DSN 格式匹配
	if res := regexp.MustCompile(`^(.*)@(.*?)/(.*?)($|\?)(.*)`).FindStringSubmatch(odbc); len(res) > 5 {
		// userInfo@hostInfo/database
		userInfo = res[1]
		hostInfo = res[2]
		schema = res[3]
		query = res[5]
	} else if res := regexp.MustCompile(`^(.*)/(.*?)($|\?)(.*)`).FindStringSubmatch(odbc); len(res) > 4 {
		// hostInfo/database
		hostInfo = res[1]
		schema = res[2]
		query = res[4]
	} else if res := regexp.MustCompile(`^(.*)@(.*?)($|\?)(.*)`).FindStringSubmatch(odbc); len(res) > 4 {
		// userInfo@hostInfo
		userInfo = res[1]
		hostInfo = res[2]
		query = res[4]
	} else if res := regexp.MustCompile(`^(.*?)($|\?)(.*)`).FindStringSubmatch(odbc); len(res) > 3 {
		// hostInfo
		hostInfo = res[1]
		query = res[3]
	}

	// 解析用户信息
	if userInfo != "" {
		user = strings.Split(userInfo, ":")[0]
		// 防止密码中含有与用户名相同的字符, 所以用正则替换, 剩下的就是密码
		password = strings.TrimLeft(regexp.MustCompile("^"+user).ReplaceAllString(userInfo, ""), ":")
	}

	// 解析主机信息
	host := strings.Split(hostInfo, ":")[0]
	port := strings.TrimLeft(strings.Replace(hostInfo, host, "", 1), ":")
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "3306"
	}
	addr = host + ":" + port

	// 解析查询字符串
	if query != "" {
		params := strings.Split(query, "&")
		for _, f := range params {
			attr := strings.Split(f, "=")
			if len(attr) > 1 {
				arg := strings.TrimSpace(attr[0])
				val := strings.TrimSpace(attr[1])
				switch arg {
				case "charset":
					charset = val
				case "timeout":
					timeout = val
				default:
				}
			}
		}
	}

	// 默认用information_schema库
	if schema == "" {
		schema = "information_schema"
	}

	// 默认 utf8 使用字符集
	if charset == "" {
		charset = "utf8"
	}

	// 默认连接数据库超时时间 3s
	if timeout == "" {
		timeout = "3s"
	}

	dsn.Addr = addr
	dsn.User = user
	dsn.Password = password
	dsn.Schema = schema
	dsn.Charset = charset
	dsn.Timeout = timeout
	return dsn
}

// ParseDSN compatible with old version soar < 0.11.0
func ParseDSN(odbc string, d *Dsn) *Dsn {
	cfg, err := mysql.ParseDSN(odbc)
	if err != nil {
		// Log.Debug("go-sql-driver/mysql.ParseDSN Error: %s, DSN: %s, try to use old version parseDSN", err.Error(), odbc)
		return parseDSN(odbc, d)
	}
	return newDSN(cfg)
}

// FormatDSN 格式化打印DSN
func FormatDSN(env *Dsn) string {
	if env == nil || env.Disable {
		return ""
	}
	dsn, err := env.newMySQLConfig()
	if err != nil {
		return ""
	}
	return dsn.FormatDSN()
}

// SoarVersion soar version information
func SoarVersion() {
	fmt.Println("Version:", Version)
	fmt.Println("Branch:", Branch)
	fmt.Println("Compile:", Compile)
	fmt.Println("GitDirty:", GitDirty)
}

// 因为vitess sqlparser 使用了 glog 中也会使用 flag，为了不让用户困扰我们单独写一个 usage
func usage() {
	regPwd := regexp.MustCompile(`:.*@`)
	vitessHelp := []string{
		"-alsologtostderr",
		"log to standard error as well as files",
		"-log_backtrace_at value",
		"when logging hits line file:N, emit a stack trace",
		"-log_dir string",
		"If non-empty, write log files in this directory",
		"-logtostderr",
		"log to standard error instead of files",
		"-sql-max-length-errors int",
		"truncate queries in error logs to the given length (default unlimited)",
		"-sql-max-length-ui int",
		"truncate queries in debug UIs to the given length (default 512) (default 512)",
		"-stderrthreshold value",
		"logs at or above this threshold go to stderr",
		"-v value",
		"log level for V logs",
		"-vmodule value",
		"comma-separated list of pattern=N settings for file-filtered logging",
	}

	// io redirect
	restoreStdout := os.Stdout
	restoreStderr := os.Stderr
	stdin, stdout, _ := os.Pipe()
	os.Stderr = stdout
	os.Stdout = stdout

	flag.PrintDefaults()

	// copy the output in a separate goroutine so printing can't block indefinitely
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, stdin)
		if err != nil {
			fmt.Println(err.Error())
		}
		outC <- buf.String()
	}()

	// back to normal state
	stdout.Close()
	os.Stdout = restoreStdout // restoring the real stderr
	os.Stderr = restoreStderr

	fmt.Printf("Usage of %s:\n", os.Args[0])
	// reading our temp stdout
	out := <-outC
	for _, line := range strings.Split(out, "\n") {
		found := false
		for _, ignore := range vitessHelp {
			if strings.TrimSpace(line) == strings.TrimSpace(ignore) {
				found = true
			}
			if regPwd.MatchString(line) && !Config.Verbose {
				line = regPwd.ReplaceAllString(line, ":********@")
			}
		}
		if !found {
			fmt.Println(line)
		}
	}
}

// PrintConfiguration for `-print-config` flag
func PrintConfiguration() {
	// 打印配置的时候密码不显示
	if !Config.Verbose {
		Config.OnlineDSN.Password = "********"
		Config.TestDSN.Password = "********"
	}
	data, _ := yaml.Marshal(Config)
	fmt.Print(string(data))
}

// 加载配置文件
func (conf *Configuration) readConfigFile(path string) error {
	configFile, err := os.Open(path)
	if err != nil {
		Log.Warning("readConfigFile(%s) os.Open failed: %v", path, err)
		return err
	}
	defer configFile.Close()

	content, err := ioutil.ReadAll(configFile)
	if err != nil {
		Log.Warning("readConfigFile(%s) ioutil.ReadAll failed: %v", path, err)
		return err
	}

	err = yaml.Unmarshal(content, Config)
	if err != nil {
		Log.Warning("readConfigFile(%s) yaml.Unmarshal failed: %v", path, err)
		return err
	}
	return nil
}

// 从命令行参数读配置
func readCmdFlags() error {
	if hasParsed {
		Log.Debug("Skip read cmd flags.")
		return nil
	}

	_ = flag.String("config", "", "Config file path")
	// +++++++++++++++测试环境+++++++++++++++++
	onlineDSN := flag.String("online-dsn", FormatDSN(Config.OnlineDSN), "OnlineDSN, 线上环境数据库配置, username:password@tcp(ip:port)/schema")
	testDSN := flag.String("test-dsn", FormatDSN(Config.TestDSN), "TestDSN, 测试环境数据库配置, username:password@tcp(ip:port)/schema")
	allowOnlineAsTest := flag.Bool("allow-online-as-test", Config.AllowOnlineAsTest, "AllowOnlineAsTest, 允许线上环境也可以当作测试环境")
	dropTestTemporary := flag.Bool("drop-test-temporary", Config.DropTestTemporary, "DropTestTemporary, 是否清理测试环境产生的临时库表")
	cleanupTestDatabase := flag.Bool("cleanup-test-database", Config.CleanupTestDatabase, "单次运行清理历史1小时前残余的测试库。")
	onlySyntaxCheck := flag.Bool("only-syntax-check", Config.OnlySyntaxCheck, "OnlySyntaxCheck, 只做语法检查不输出优化建议")
	profiling := flag.Bool("profiling", Config.Profiling, "Profiling, 开启数据采样的情况下在测试环境执行Profile")
	trace := flag.Bool("trace", Config.Trace, "Trace, 开启数据采样的情况下在测试环境执行Trace")
	explain := flag.Bool("explain", Config.Explain, "Explain, 是否开启Explain执行计划分析")
	sampling := flag.Bool("sampling", Config.Sampling, "Sampling, 数据采样开关")
	samplingStatisticTarget := flag.Int("sampling-statistic-target", Config.SamplingStatisticTarget, "SamplingStatisticTarget, 数据采样因子，对应 PostgreSQL 的 default_statistics_target")
	samplingCondition := flag.String("sampling-condition", Config.SamplingCondition, "SamplingCondition, 数据采样条件，如： WHERE xxx LIMIT xxx")
	delimiter := flag.String("delimiter", Config.Delimiter, "Delimiter, SQL分隔符")
	minCardinality := flag.Float64("min-cardinality", Config.MinCardinality, "MinCardinality，索引列散粒度最低阈值，散粒度低于该值的列不添加索引，建议范围0.0 ~ 100.0")
	// +++++++++++++++日志相关+++++++++++++++++
	logLevel := flag.Int("log-level", Config.LogLevel, "LogLevel, 日志级别, [0:Emergency, 1:Alert, 2:Critical, 3:Error, 4:Warning, 5:Notice, 6:Informational, 7:Debug]")
	logOutput := flag.String("log-output", Config.LogOutput, "LogOutput, 日志输出位置")
	reportType := flag.String("report-type", Config.ReportType, "ReportType, 优化建议输出格式，目前支持: json, text, markdown, html等")
	reportCSS := flag.String("report-css", Config.ReportCSS, "ReportCSS, 当 ReportType 为 html 格式时使用的 css 风格，如不指定会提供一个默认风格。CSS可以是本地文件，也可以是一个URL")
	reportJavascript := flag.String("report-javascript", Config.ReportJavascript, "ReportJavascript, 当 ReportType 为 html 格式时使用的javascript脚本，如不指定默认会加载SQL pretty 使用的 javascript。像CSS一样可以是本地文件，也可以是一个URL")
	reportTitle := flag.String("report-title", Config.ReportTitle, "ReportTitle, 当 ReportType 为 html 格式时，HTML 的 title")
	// +++++++++++++++markdown+++++++++++++++++
	markdownExtensions := flag.Int("markdown-extensions", Config.MarkdownExtensions, "MarkdownExtensions, markdown 转 html支持的扩展包, 参考blackfriday")
	markdownHTMLFlags := flag.Int("markdown-html-flags", Config.MarkdownHTMLFlags, "MarkdownHTMLFlags, markdown 转 html 支持的 flag, 参考blackfriday")
	// ++++++++++++++优化建议相关++++++++++++++
	ignoreRules := flag.String("ignore-rules", strings.Join(Config.IgnoreRules, ","), "IgnoreRules, 忽略的优化建议规则")
	rewriteRules := flag.String("rewrite-rules", strings.Join(Config.RewriteRules, ","), "RewriteRules, 生效的重写规则")
	blackList := flag.String("blacklist", Config.BlackList, "指定 blacklist 配置文件的位置，文件中的 SQL 不会被评审。一行一条SQL，可以是指纹，也可以是正则")
	maxJoinTableCount := flag.Int("max-join-table-count", Config.MaxJoinTableCount, "MaxJoinTableCount, 单条 SQL 中 JOIN 表的最大数量")
	maxGroupByColsCount := flag.Int("max-group-by-cols-count", Config.MaxGroupByColsCount, "MaxGroupByColsCount, 单条 SQL 中 GroupBy 包含列的最大数量")
	maxDistinctCount := flag.Int("max-distinct-count", Config.MaxDistinctCount, "MaxDistinctCount, 单条 SQL 中 Distinct 的最大数量")
	maxIdxColsCount := flag.Int("max-index-cols-count", Config.MaxIdxColsCount, "MaxIdxColsCount, 复合索引中包含列的最大数量")
	maxTextColsCount := flag.Int("max-text-cols-count", Config.MaxTextColsCount, "MaxTextColsCount, 表中含有的 text/blob 列的最大数量")
	maxTotalRows := flag.Uint64("max-total-rows", Config.MaxTotalRows, "MaxTotalRows, 计算散粒度时，当数据行数大于MaxTotalRows即开启数据库保护模式，不计算散粒度")
	maxQueryCost := flag.Int64("max-query-cost", Config.MaxQueryCost, "MaxQueryCost, last_query_cost 超过该值时将给予警告")
	spaghettiQueryLength := flag.Int("spaghetti-query-length", Config.SpaghettiQueryLength, "SpaghettiQueryLength, SQL最大长度警告，超过该长度会给警告")
	allowDropIdx := flag.Bool("allow-drop-index", Config.AllowDropIndex, "AllowDropIndex, 允许输出删除重复索引的建议")
	maxInCount := flag.Int("max-in-count", Config.MaxInCount, "MaxInCount, IN()最大数量")
	maxIdxBytesPerColumn := flag.Int("max-index-bytes-percolumn", Config.MaxIdxBytesPerColumn, "MaxIdxBytesPerColumn, 索引中单列最大字节数")
	maxIdxBytes := flag.Int("max-index-bytes", Config.MaxIdxBytes, "MaxIdxBytes, 索引总长度限制")
	allowCharsets := flag.String("allow-charsets", strings.ToLower(strings.Join(Config.AllowCharsets, ",")), "AllowCharsets")
	allowCollates := flag.String("allow-collates", strings.ToLower(strings.Join(Config.AllowCollates, ",")), "AllowCollates")
	allowEngines := flag.String("allow-engines", strings.ToLower(strings.Join(Config.AllowEngines, ",")), "AllowEngines")
	maxIdxCount := flag.Int("max-index-count", Config.MaxIdxCount, "MaxIdxCount, 单表最大索引个数")
	maxColCount := flag.Int("max-column-count", Config.MaxColCount, "MaxColCount, 单表允许的最大列数")
	maxValueCount := flag.Int("max-value-count", Config.MaxValueCount, "MaxValueCount, INSERT/REPLACE 单次批量写入允许的行数")
	idxPrefix := flag.String("index-prefix", Config.IdxPrefix, "IdxPrefix")
	ukPrefix := flag.String("unique-key-prefix", Config.UkPrefix, "UkPrefix")
	maxSubqueryDepth := flag.Int("max-subquery-depth", Config.MaxSubqueryDepth, "MaxSubqueryDepth")
	maxVarcharLength := flag.Int("max-varchar-length", Config.MaxVarcharLength, "MaxVarcharLength")
	columnNotAllowType := flag.String("column-not-allow-type", strings.Join(Config.ColumnNotAllowType, ","), "ColumnNotAllowType")
	// ++++++++++++++EXPLAIN检查项+++++++++++++
	explainSQLReportType := flag.String("explain-sql-report-type", strings.ToLower(Config.ExplainSQLReportType), "ExplainSQLReportType [pretty, sample, fingerprint]")
	explainType := flag.String("explain-type", strings.ToLower(Config.ExplainType), "ExplainType [extended, partitions, traditional]")
	explainFormat := flag.String("explain-format", strings.ToLower(Config.ExplainFormat), "ExplainFormat [json, traditional]")
	explainWarnSelectType := flag.String("explain-warn-select-type", strings.Join(Config.ExplainWarnSelectType, ","), "ExplainWarnSelectType, 哪些select_type不建议使用")
	explainWarnAccessType := flag.String("explain-warn-access-type", strings.Join(Config.ExplainWarnAccessType, ","), "ExplainWarnAccessType, 哪些access type不建议使用")
	explainMaxKeyLength := flag.Int("explain-max-keys", Config.ExplainMaxKeyLength, "ExplainMaxKeyLength, 最大key_len")
	explainMinPossibleKeys := flag.Int("explain-min-keys", Config.ExplainMinPossibleKeys, "ExplainMinPossibleKeys, 最小possible_keys警告")
	explainMaxRows := flag.Int64("explain-max-rows", Config.ExplainMaxRows, "ExplainMaxRows, 最大扫描行数警告")
	explainWarnExtra := flag.String("explain-warn-extra", strings.Join(Config.ExplainWarnExtra, ","), "ExplainWarnExtra, 哪些extra信息会给警告")
	explainMaxFiltered := flag.Float64("explain-max-filtered", Config.ExplainMaxFiltered, "ExplainMaxFiltered, filtered大于该配置给出警告")
	explainWarnScalability := flag.String("explain-warn-scalability", strings.Join(Config.ExplainWarnScalability, ","), "ExplainWarnScalability, 复杂度警告名单, 支持O(n),O(log n),O(1),O(?)")
	showWarnings := flag.Bool("show-warnings", Config.ShowWarnings, "ShowWarnings")
	showLastQueryCost := flag.Bool("show-last-query-cost", Config.ShowLastQueryCost, "ShowLastQueryCost")
	// +++++++++++++++++其他+++++++++++++++++++
	printConfig := flag.Bool("print-config", false, "Print configs")
	checkConfig := flag.Bool("check-config", false, "Check configs")
	printVersion := flag.Bool("version", false, "Print version info")
	query := flag.String("query", Config.Query, "待评审的 SQL 或 SQL 文件，如 SQL 中包含特殊字符建议使用文件名。")
	listHeuristicRules := flag.Bool("list-heuristic-rules", Config.ListHeuristicRules, "ListHeuristicRules, 打印支持的评审规则列表")
	listRewriteRules := flag.Bool("list-rewrite-rules", Config.ListRewriteRules, "ListRewriteRules, 打印支持的重写规则列表")
	listTestSQLs := flag.Bool("list-test-sqls", Config.ListTestSqls, "ListTestSqls, 打印测试case用于测试")
	listReportTypes := flag.Bool("list-report-types", Config.ListReportTypes, "ListReportTypes, 打印支持的报告输出类型")
	verbose := flag.Bool("verbose", Config.Verbose, "Verbose")
	dryrun := flag.Bool("dry-run", Config.DryRun, "是否在预演环境执行")
	maxPrettySQLLength := flag.Int("max-pretty-sql-length", Config.MaxPrettySQLLength, "MaxPrettySQLLength, 超出该长度的SQL会转换成指纹输出")
	// 一个不存在 log-level，用于更新 usage。
	// 因为 vitess 里面也用了 flag，这些 vitess 的参数我们不需要关注
	if !Config.Verbose && runtime.GOOS != "windows" {
		flag.Usage = usage
	}
	flag.Parse()

	Config.OnlineDSN = ParseDSN(*onlineDSN, Config.OnlineDSN)
	Config.TestDSN = ParseDSN(*testDSN, Config.TestDSN)
	Config.AllowOnlineAsTest = *allowOnlineAsTest
	Config.DropTestTemporary = *dropTestTemporary
	Config.CleanupTestDatabase = *cleanupTestDatabase
	Config.OnlySyntaxCheck = *onlySyntaxCheck
	Config.Profiling = *profiling
	Config.Trace = *trace
	Config.Explain = *explain
	Config.Sampling = *sampling
	Config.SamplingStatisticTarget = *samplingStatisticTarget
	Config.SamplingCondition = *samplingCondition

	Config.LogLevel = *logLevel

	if filepath.IsAbs(*logOutput) || *logOutput == "" {
		Config.LogOutput = *logOutput
	} else {
		Config.LogOutput = filepath.Join(BaseDir, *logOutput)
	}

	Config.ReportType = strings.ToLower(*reportType)
	if Config.ReportType == "tables" && Config.TestDSN.Schema == "information_schema" {
		Config.TestDSN.Schema = "unknown"
	}
	Config.ReportCSS = *reportCSS
	Config.ReportJavascript = *reportJavascript
	Config.ReportTitle = *reportTitle
	Config.MarkdownExtensions = *markdownExtensions
	Config.MarkdownHTMLFlags = *markdownHTMLFlags
	Config.IgnoreRules = strings.Split(*ignoreRules, ",")
	Config.RewriteRules = strings.Split(*rewriteRules, ",")
	*blackList = strings.TrimSpace(*blackList)
	Config.MinCardinality = *minCardinality

	if filepath.IsAbs(*blackList) || *blackList == "" {
		Config.BlackList = *blackList
	} else {
		Config.BlackList = filepath.Join(BaseDir, *blackList)
	}

	Config.MaxJoinTableCount = *maxJoinTableCount
	Config.MaxGroupByColsCount = *maxGroupByColsCount
	Config.MaxDistinctCount = *maxDistinctCount

	if *maxIdxColsCount < 16 {
		Config.MaxIdxColsCount = *maxIdxColsCount
	} else {
		Config.MaxIdxColsCount = 16
	}

	Config.MaxTextColsCount = *maxTextColsCount
	Config.MaxIdxBytesPerColumn = *maxIdxBytesPerColumn
	Config.MaxIdxBytes = *maxIdxBytes
	if *allowCharsets != "" {
		Config.AllowCharsets = strings.Split(strings.ToLower(*allowCharsets), ",")
	}
	if *allowCollates != "" {
		Config.AllowCollates = strings.Split(strings.ToLower(*allowCollates), ",")
	}
	if *allowEngines != "" {
		Config.AllowEngines = strings.Split(strings.ToLower(*allowEngines), ",")
	}
	Config.MaxIdxCount = *maxIdxCount
	Config.MaxColCount = *maxColCount
	Config.MaxValueCount = *maxValueCount
	Config.IdxPrefix = *idxPrefix
	Config.UkPrefix = *ukPrefix
	Config.MaxSubqueryDepth = *maxSubqueryDepth
	Config.MaxTotalRows = *maxTotalRows
	Config.MaxQueryCost = *maxQueryCost
	Config.AllowDropIndex = *allowDropIdx
	Config.MaxInCount = *maxInCount
	Config.SpaghettiQueryLength = *spaghettiQueryLength
	Config.Query = *query
	Config.Delimiter = *delimiter

	Config.ExplainSQLReportType = strings.ToLower(*explainSQLReportType)
	Config.ExplainType = strings.ToLower(*explainType)
	Config.ExplainFormat = strings.ToLower(*explainFormat)
	Config.ExplainWarnSelectType = strings.Split(*explainWarnSelectType, ",")
	Config.ExplainWarnAccessType = strings.Split(*explainWarnAccessType, ",")
	Config.ExplainMaxKeyLength = *explainMaxKeyLength
	Config.ExplainMinPossibleKeys = *explainMinPossibleKeys
	Config.ExplainMaxRows = *explainMaxRows
	Config.ExplainWarnExtra = strings.Split(*explainWarnExtra, ",")
	Config.ExplainMaxFiltered = *explainMaxFiltered
	Config.ExplainWarnScalability = strings.Split(*explainWarnScalability, ",")
	Config.ShowWarnings = *showWarnings
	Config.ShowLastQueryCost = *showLastQueryCost
	Config.ListHeuristicRules = *listHeuristicRules
	Config.ListRewriteRules = *listRewriteRules
	Config.ListTestSqls = *listTestSQLs
	Config.ListReportTypes = *listReportTypes
	Config.Verbose = *verbose
	Config.DryRun = *dryrun
	Config.MaxPrettySQLLength = *maxPrettySQLLength
	Config.MaxVarcharLength = *maxVarcharLength
	if *columnNotAllowType != "" {
		Config.ColumnNotAllowType = strings.Split(strings.ToLower(*columnNotAllowType), ",")
	}

	PrintVersion = *printVersion
	PrintConfig = *printConfig
	CheckConfig = *checkConfig

	hasParsed = true
	return nil
}

// ParseConfig 加载配置文件和命令行参数
func ParseConfig(configFile string) error {
	var err error
	var configs []string
	// 指定了配置文件优先读配置文件，未指定配置文件按如下顺序加载，先找到哪个加载哪个
	if configFile == "" {
		configs = []string{
			"/etc/soar.yaml",
			filepath.Join(BaseDir, "etc", "soar.yaml"),
			filepath.Join(BaseDir, "soar.yaml"),
		}
	} else {
		configs = []string{
			configFile,
		}
	}

	for _, config := range configs {
		if _, err = os.Stat(config); err == nil {
			err = Config.readConfigFile(config)
			if err != nil {
				Log.Error("ParseConfig Config.readConfigFile Error: %v", err)
			}
			// LogOutput now is "console", if add Log.Debug here will print into stdout anyway.
			// Log.Debug("ParseConfig use config file: %s", config)
			break
		}
	}

	err = readCmdFlags()
	if err != nil {
		Log.Error("ParseConfig readCmdFlags Error: %v", err)
		return err
	}

	// parse blacklist & ignore blacklist file parse error
	if _, e := os.Stat(Config.BlackList); e == nil {
		var blFd *os.File
		blFd, err = os.Open(Config.BlackList)
		if err == nil {
			bl := bufio.NewReader(blFd)
			for {
				rule, e := bl.ReadString('\n')
				if e != nil {
					break
				}
				rule = strings.TrimSpace(rule)
				if strings.HasPrefix(rule, "#") || rule == "" {
					continue
				}
				BlackList = append(BlackList, rule)
			}
		}
		defer blFd.Close()
	}
	LoggerInit()
	return err
}

// ReportType 元数据结构定义
type ReportType struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Example     string `json:"Example"`
}

// ReportTypes 命令行-report-type支持的形式
var ReportTypes = []ReportType{
	{
		Name:        "lint",
		Description: "参考sqlint格式，以插件形式集成到代码编辑器，显示输出更加友好",
		Example:     `soar -report-type lint -query test.sql`,
	},
	{
		Name:        "markdown",
		Description: "该格式为默认输出格式，以markdown格式展现，可以用网页浏览器插件直接打开，也可以用markdown编辑器打开",
		Example:     `echo "select * from film" | soar`,
	},
	{
		Name:        "rewrite",
		Description: "SQL重写功能，配合-rewrite-rules参数一起使用，可以通过-list-rewrite-rules 查看所有支持的 SQL 重写规则",
		Example:     `echo "select * from film" | soar -rewrite-rules star2columns,delimiter -report-type rewrite`,
	},
	{
		Name:        "ast",
		Description: "输出 SQL 的抽象语法树，主要用于测试",
		Example:     `echo "select * from film" | soar -report-type ast`,
	},
	{
		Name:        "ast-json",
		Description: "以 JSON 格式输出 SQL 的抽象语法树，主要用于测试",
		Example:     `echo "select * from film" | soar -report-type ast-json`,
	},
	{
		Name:        "tiast",
		Description: "输出 SQL 的 TiDB抽象语法树，主要用于测试",
		Example:     `echo "select * from film" | soar -report-type tiast`,
	},
	{
		Name:        "tiast-json",
		Description: "以 JSON 格式输出 SQL 的 TiDB抽象语法树，主要用于测试",
		Example:     `echo "select * from film" | soar -report-type tiast-json`,
	},
	{
		Name:        "tables",
		Description: "以 JSON 格式输出 SQL 使用的库表名",
		Example:     `echo "select * from film" | soar -report-type tables`,
	},
	{
		Name:        "query-type",
		Description: "SQL 语句的请求类型",
		Example:     `echo "select * from film" | soar -report-type query-type`,
	},
	{
		Name:        "fingerprint",
		Description: "输出SQL的指纹",
		Example:     `echo "select * from film where language_id=1" | soar -report-type fingerprint`,
	},
	{
		Name:        "md2html",
		Description: "markdown 格式转 html 格式小工具",
		Example:     `soar -list-heuristic-rules | soar -report-type md2html > heuristic_rules.html`,
	},
	{
		Name:        "explain-digest",
		Description: "输入为EXPLAIN的表格，JSON 或 Vertical格式，对其进行分析，给出分析结果",
		Example: `soar -report-type explain-digest << EOF
+----+-------------+-------+------+---------------+------+---------+------+------+-------+
| id | select_type | table | type | possible_keys | key  | key_len | ref  | rows | Extra |
+----+-------------+-------+------+---------------+------+---------+------+------+-------+
|  1 | SIMPLE      | film  | ALL  | NULL          | NULL | NULL    | NULL | 1131 |       |
+----+-------------+-------+------+---------------+------+---------+------+------+-------+
EOF`,
	},
	{
		Name:        "duplicate-key-checker",
		Description: "对 OnlineDsn 中指定的 database 进行索引重复检查",
		Example:     `soar -report-type duplicate-key-checker -online-dsn user:password@127.0.0.1:3306/db`,
	},
	{
		Name:        "html",
		Description: "以HTML格式输出报表",
		Example:     `echo "select * from film" | soar -report-type html`,
	},
	{
		Name:        "json",
		Description: "输出JSON格式报表，方便应用程序处理",
		Example:     `echo "select * from film" | soar -report-type json`,
	},
	{
		Name:        "tokenize",
		Description: "对SQL进行切词，主要用于测试",
		Example:     `echo "select * from film" | soar -report-type tokenize`,
	},
	{
		Name:        "compress",
		Description: "SQL压缩小工具，使用内置SQL压缩逻辑，测试中的功能",
		Example: `echo "select
*
from
  film" | soar -report-type compress`,
	},
	{
		Name:        "pretty",
		Description: "使用kr/pretty打印报告，主要用于测试",
		Example:     `echo "select * from film" | soar -report-type pretty`,
	},
	{
		Name:        "remove-comment",
		Description: "去除SQL语句中的注释，支持单行多行注释的去除",
		Example:     `echo "select/*comment*/ * from film" | soar -report-type remove-comment`,
	},
	{
		Name:        "chardet",
		Description: "猜测输入的 SQL 使用的字符集",
		Example:     "echo '中文' | soar -report-type chardet",
	},
}

// ListReportTypes 查看所有支持的report-type
func ListReportTypes() {
	switch Config.ReportType {
	case "json":
		js, err := json.MarshalIndent(ReportTypes, "", "  ")
		if err == nil {
			fmt.Println(string(js))
		}
	default:
		fmt.Print("# 支持的报告类型\n\n[toc]\n\n")
		for _, r := range ReportTypes {
			fmt.Print("## ", MarkdownEscape(r.Name),
				"\n* **Description**:", r.Description+"\n",
				"\n* **Example**:\n\n```bash\n", r.Example, "\n```\n")
		}
	}
}

// ArgConfig get -config arg value from cli
func ArgConfig() string {
	var configFile string
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "-config") {
		if os.Args[1] == "-config" && len(os.Args) > 2 {
			if os.Args[2] == "=" && len(os.Args) > 3 {
				// -config = soar.yaml not support
				fmt.Println("wrong format, no space between '=', eg: -config=soar.yaml")
			} else {
				// -config soar.yaml
				configFile = os.Args[2]
			}
			if strings.HasPrefix(configFile, "=") {
				// -config =soar.yaml
				configFile = strings.Split(configFile, "=")[1]
			}
		}
		if strings.Contains(os.Args[1], "=") {
			// -config=soar.yaml
			configFile = strings.Split(os.Args[1], "=")[1]
		}
	} else {
		for i, c := range os.Args {
			if strings.HasPrefix(c, "-config") && i != 1 {
				fmt.Println("-config must be the first arg")
			}
		}
	}
	return configFile
}
