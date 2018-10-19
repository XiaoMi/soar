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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

// GoldenDiff 从gofmt学来的测试方法
// https://medium.com/soon-london/testing-with-golden-files-in-go-7fccc71c43d3
func GoldenDiff(f func(), name string, update *bool) error {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	str := captureOutput(f)
	_, err := w.WriteString(str)
	if err != nil {
		Log.Warning(err.Error())
	}
	err = w.Flush()
	if err != nil {
		Log.Warning(err.Error())
	}

	gp := filepath.Join("testdata", name+".golden")
	if *update {
		if err = ioutil.WriteFile(gp, b.Bytes(), 0644); err != nil {
			err = fmt.Errorf("%s failed to update golden file: %s", name, err)
			return err
		}
	}
	g, err := ioutil.ReadFile(gp)
	if err != nil {
		err = fmt.Errorf("%s failed reading .golden: %s", name, err)
	}
	if !bytes.Equal(b.Bytes(), g) {
		err = fmt.Errorf("%s does not match .golden file", name)
	}
	return err
}

// captureOutput 获取函数标准输出
func captureOutput(f func()) string {
	// keep backup of the real stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// execute function
	f()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			Log.Warning(err.Error())
		}
		outC <- buf.String()
	}()

	// back to normal state
	err := w.Close()
	if err != nil {
		Log.Warning(err.Error())
	}
	os.Stdout = oldStdout // restoring the real stdout
	out := <-outC
	os.Stdout = oldStdout
	return out
}

// SortedKey sort map[string]interface{}, use in range clause
func SortedKey(m interface{}) []string {
	var keys []string
	switch reflect.TypeOf(m).Kind() {
	case reflect.Map:
		switch reflect.TypeOf(m).Key().Kind() {
		case reflect.String:
			for _, k := range reflect.ValueOf(m).MapKeys() {
				keys = append(keys, k.String())
			}
		}
	}
	sort.Strings(keys)
	return keys
}
