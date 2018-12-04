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
	"github.com/kr/pretty"
	"github.com/saintfish/chardet"
)

// Chardet get best match charset
func Chardet(buf []byte) string {
	// check character set by file BOM
	charset := CheckCharsetByBOM(buf)
	if charset != "" {
		return charset
	}

	// use chardet pkg check file charset
	charset = "unknown"
	var confidence int
	detector := chardet.NewTextDetector()

	// detector.DetectBest is unstable
	// when the confidence value are equally, the best detect charset will be random
	result, err := detector.DetectAll(buf)
	if err != nil {
		return charset
	}
	Log.Debug("Chardet DetectAll Result: %s", pretty.Sprint(result))

	// SOAR's main user speak Chinese, GB-18030, UTF-8 are higher suggested
	for _, r := range result {
		if confidence > r.Confidence && r.Confidence != 0 {
			return charset
		}
		confidence = r.Confidence
		if r.Charset == "GB-18030" || r.Charset == "UTF-8" {
			return r.Charset
		}
		charset = r.Charset
	}
	return charset
}

// CheckCharsetByBOM ref: https://en.wikipedia.org/wiki/Byte_order_mark
func CheckCharsetByBOM(buf []byte) string {
	// TODO: There are many kind of BOM
	// UTF-8	EF BB BF
	if len(buf) >= 3 {
		if buf[0] == 0xef && buf[1] == 0xbb && buf[2] == 0xbf {
			return "UTF-8"
		}
	}
	// GB-18030	84 31 95 33
	if len(buf) >= 4 {
		if buf[0] == 0x84 && buf[1] == 0x31 && buf[2] == 0x95 && buf[3] == 0x33 {
			return "GB-18030"
		}
	}
	return ""
}

// RemoveBOM remove bom from file
func RemoveBOM(buf []byte) (string, []byte) {
	// ef bb bf, UTF-8 BOM
	if len(buf) > 3 {
		if buf[0] == 0xef && buf[1] == 0xbb && buf[2] == 0xbf {
			return string(buf[3:]), buf[:3]
		}
	}
	// ff fe, UTF-16 (LE) BOM
	if len(buf) > 2 {
		if buf[0] == 0xff && buf[1] == 0xfe {
			return string(buf[2:]), buf[:2]
		}
	}
	return string(buf), []byte{}
}
