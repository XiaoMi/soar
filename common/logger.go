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
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/astaxie/beego/logs"
)

// Log 使用 beego 的 log 库
var Log *logs.BeeLogger

// BaseDir 日志打印在binary的根路径
var BaseDir string

func init() {
	Log = logs.NewLogger(0)
	Log.EnableFuncCallDepth(true)
}

// LoggerInit Log配置初始化
func LoggerInit() {
	Log.SetLevel(Config.LogLevel)
	func() { _ = Log.DelLogger(logs.AdapterFile) }()
	logConfig := map[string]interface{}{
		"filename": Config.LogOutput,
		"level":    7,
		"maxlines": 0,
		"maxsize":  0,
		"daily":    false,
		"maxdays":  0,
	}
	logConfigJSON, _ := json.Marshal(logConfig)
	err := Log.SetLogger(logs.AdapterFile, string(logConfigJSON))
	if err != nil {
		fmt.Println(err.Error())
	}
}

// Caller returns the caller of the function that called it :)
// https://stackoverflow.com/questions/35212985/is-it-possible-get-information-about-caller-function-in-golang
func Caller() string {
	// we get the callers as uintptrs - but we just need 1
	fpcs := make([]uintptr, 1)

	// skip 3 levels to get to the caller of whoever called Caller()
	n := runtime.Callers(3, fpcs)
	if n == 0 {
		return "n/a" // proper error her would be better
	}

	// get the info of the actual function that's in the pointer
	fun := runtime.FuncForPC(fpcs[0] - 1)
	if fun == nil {
		return "n/a"
	}

	// return its name
	return fun.Name()
}

// GetFunctionName 获取调当前函数名
func GetFunctionName() string {
	// Skip this function, and fetch the PC and file for its parent
	pc, _, _, _ := runtime.Caller(1)
	// Retrieve a Function object this functions parent
	functionObject := runtime.FuncForPC(pc)
	// Regex to extract just the function name (and not the module path)
	extractFnName := regexp.MustCompile(`^.*\.(.*)$`)
	fnName := extractFnName.ReplaceAllString(functionObject.Name(), "$1")
	return fnName
}

// fileName get filename from path
func fileName(original string) string {
	i := strings.LastIndex(original, "/")
	if i == -1 {
		return original
	}
	return original[i+1:]
}

// LogIfError 简化if err != nil 打 Error 日志代码长度
func LogIfError(err error, format string, v ...interface{}) {
	if err != nil {
		_, fn, line, _ := runtime.Caller(1)
		if format == "" {
			format = "[%s:%d] %s"
			Log.Error(format, fileName(fn), line, err.Error())
		} else {
			format = "[%s:%d] " + format + " Error: %s"
			Log.Error(format, fileName(fn), line, v, err.Error())
		}
	}
}

// LogIfWarn 简化if err != nil 打 Warn 日志代码长度
func LogIfWarn(err error, format string, v ...interface{}) {
	if err != nil {
		_, fn, line, _ := runtime.Caller(1)
		if format == "" {
			format = "[%s:%d] %s"
			Log.Warn(format, fileName(fn), line, err.Error())
		} else {
			format = "[%s:%d] " + format + " Error: %s"
			Log.Warn(format, fileName(fn), line, v, err.Error())
		}
	}
}
