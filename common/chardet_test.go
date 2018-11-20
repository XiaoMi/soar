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
	"io/ioutil"
	"testing"
)

func TestChardet(t *testing.T) {
	charsets := []string{
		"GB-18030",
		"UTF-8",
	}
	for _, c := range charsets {
		fileName := DevPath + "/common/testdata/chardet_" + c + ".txt"
		buf, err := ioutil.ReadFile(fileName)
		if err != nil {
			t.Errorf("ioutil.ReadFile %s, Error: %s", fileName, err.Error())
		}
		name := Chardet(buf)
		if name != c {
			t.Errorf("file: %s, Want: %s, Get: %s", fileName, c, name)
		}
	}
}
