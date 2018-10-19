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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestMarkdownEscape(_ *testing.T) {
	var strs = []string{
		"a`bc",
		"abc",
		"a'bc",
		"a\"bc",
	}
	for _, str := range strs {
		fmt.Println(MarkdownEscape(str))
	}
}

func TestMarkdown2Html(t *testing.T) {
	md := filepath.Join("testdata", t.Name()+".md")
	buf, err := ioutil.ReadFile(md)
	if err != nil {
		t.Error(err.Error())
	}
	err = GoldenDiff(func() {
		fmt.Println(Markdown2HTML(string(buf)))
	}, t.Name(), update)
	if nil != err {
		t.Fatal(err)
	}

	// golden文件拷贝成html文件，这步是给人看的
	gd, err := os.OpenFile("testdata/"+t.Name()+".golden", os.O_RDONLY, 0666)
	if nil != err {
		t.Fatal(err)
	}
	html, err := os.OpenFile("testdata/"+t.Name()+".html", os.O_CREATE|os.O_RDWR, 0666)
	if nil != err {
		t.Fatal(err)
	}
	io.Copy(html, gd)
}

func TestScore(t *testing.T) {
	score := Score(50)
	if score != "★ ★ ☆ ☆ ☆ 50分" {
		t.Error(score)
	}
}

func TestLoadExternalResource(t *testing.T) {
	buf := loadExternalResource("../doc/themes/github.css")
	if buf == "" {
		t.Error("loadExternalResource local error")
	}
	buf = loadExternalResource("http://www.baidu.com")
	if buf == "" {
		t.Error("loadExternalResource http error")
	}
}

func TestMarkdownHTMLHeader(t *testing.T) {
	err := GoldenDiff(func() {
		MarkdownHTMLHeader()
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
}
