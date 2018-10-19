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
	"errors"
	"testing"
)

func init() {
	BaseDir = DevPath
}

func TestLogger(t *testing.T) {
	Log.Info("info")
	Log.Debug("debug")
	Log.Warning("warning")
	Log.Error("error")
}

func TestCaller(t *testing.T) {
	caller := Caller()
	if caller != "testing.tRunner" {
		t.Error("get caller failer")
	}
}

func TestGetFunctionName(t *testing.T) {
	f := GetFunctionName()
	if f != "TestGetFunctionName" {
		t.Error("get functionname failer")
	}
}

func TestIfError(t *testing.T) {
	err := errors.New("test")
	LogIfError(err, "")
	LogIfError(err, "func %s", "func_test")
}

func TestIfWarn(t *testing.T) {
	err := errors.New("test")
	LogIfWarn(err, "")
	LogIfWarn(err, "func %s", "func_test")
}
