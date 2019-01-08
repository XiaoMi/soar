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
	"strings"
	"testing"
	"time"
)

func TestCaptureOutput(t *testing.T) {
	c1 := make(chan string, 1)
	// test output buf large than 65535
	length := 1<<16 + 1
	go func() {
		str := captureOutput(
			func() {
				var str []string
				for i := 0; i < length; i++ {
					str = append(str, "a")
				}
				fmt.Println(strings.Join(str, ""))
			},
		)
		c1 <- str
	}()

	select {
	case res := <-c1:
		if len(res) <= length {
			t.Errorf("want %d, got %d", length, len(res))
		}
	case <-time.After(1 * time.Second):
		t.Error("capture timeout, pipe read hangup")
	}
}
