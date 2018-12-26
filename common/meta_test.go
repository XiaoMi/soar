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
	"testing"
)

func TestGetDataTypeLength(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	typeList := map[string][]int{
		"varchar(20)":  {20},
		"int(2)":       {2},
		"int(2000000)": {2000000},
		"DECIMAL(1,2)": {1, 2},
		"int":          {-1},
	}

	for typ, want := range typeList {
		got := GetDataTypeLength(typ)
		for i := 0; i < len(want); i++ {
			if want[i] != got[i] {
				t.Errorf("Not match, want %v, got %v", want, got)
			}
		}
	}
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func TestGetDataTypeBase(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	typeList := map[string]string{
		"varchar(20)":  "varchar",
		"int(2)":       "int",
		"int(2000000)": "int",
	}

	for typ := range typeList {
		if got := GetDataTypeBase(typ); got != typeList[typ] {
			t.Errorf("Not match, want %s, got %s", typeList[typ], got)
		}
	}
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func TestGetDataBytes(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	cols50604 := map[*Column]int{
		// numeric type
		{Name: "col000", DataType: "tinyint", Character: "utf8"}:        1,
		{Name: "col001", DataType: "SMALLINT", Character: "utf8"}:       2,
		{Name: "col002", DataType: "MEDIUMINT", Character: "utf8"}:      3,
		{Name: "col003", DataType: "int(32)", Character: "utf8"}:        4,
		{Name: "col004", DataType: "integer(32)", Character: "utf8"}:    4,
		{Name: "col005", DataType: "bigint(10)", Character: "utf8"}:     8,
		{Name: "col006", DataType: "float(12)", Character: "utf8"}:      4,
		{Name: "col007", DataType: "float(50)", Character: "utf8"}:      8,
		{Name: "col008", DataType: "float(100)", Character: "utf8"}:     8,
		{Name: "col009", DataType: "float", Character: "utf8"}:          4,
		{Name: "col010", DataType: "double", Character: "utf8"}:         8,
		{Name: "col011", DataType: "real", Character: "utf8"}:           8,
		{Name: "col012", DataType: "BIT(32)", Character: "utf8"}:        4,
		{Name: "col013", DataType: "numeric(32,32)", Character: "utf8"}: 30,
		{Name: "col013", DataType: "decimal(2,32)", Character: "utf8"}:  16,
		{Name: "col014", DataType: "BIT(32)", Character: "utf8"}:        4,

		// date & time
		{Name: "col015", DataType: "year(32)", Character: "utf8mb4"}:      1,
		{Name: "col016", DataType: "date", Character: "utf8mb4"}:          3,
		{Name: "col017", DataType: "time", Character: "utf8mb4"}:          3,
		{Name: "col018", DataType: "time(0)", Character: "utf8mb4"}:       3,
		{Name: "col019", DataType: "time(2)", Character: "utf8mb4"}:       4,
		{Name: "col020", DataType: "time(4)", Character: "utf8mb4"}:       5,
		{Name: "col021", DataType: "time(6)", Character: "utf8mb4"}:       6,
		{Name: "col022", DataType: "datetime", Character: "utf8mb4"}:      5,
		{Name: "col023", DataType: "timestamp(32)", Character: "utf8mb4"}: 7,

		// string
		{Name: "col024", DataType: "varchar(255)", Character: "utf8"}:    767,
		{Name: "col025", DataType: "varchar(191)", Character: "utf8mb4"}: 765,
	}

	for col, bytes := range cols50604 {
		if got := col.GetDataBytes(50604); got != bytes {
			t.Errorf("Version 5.6.4, %s Not match, want %d, got %d", col.Name, bytes, got)
		}
	}

	cols50500 := map[*Column]int{
		// numeric type
		{Name: "col000", DataType: "tinyint", Character: "utf8"}:        1,
		{Name: "col001", DataType: "SMALLINT", Character: "utf8"}:       2,
		{Name: "col002", DataType: "MEDIUMINT", Character: "utf8"}:      3,
		{Name: "col003", DataType: "int(32)", Character: "utf8"}:        4,
		{Name: "col004", DataType: "integer(32)", Character: "utf8"}:    4,
		{Name: "col005", DataType: "bigint(10)", Character: "utf8"}:     8,
		{Name: "col006", DataType: "float(12)", Character: "utf8"}:      4,
		{Name: "col007", DataType: "float(50)", Character: "utf8"}:      8,
		{Name: "col008", DataType: "float(100)", Character: "utf8"}:     8,
		{Name: "col009", DataType: "float", Character: "utf8"}:          4,
		{Name: "col010", DataType: "double", Character: "utf8"}:         8,
		{Name: "col011", DataType: "real", Character: "utf8"}:           8,
		{Name: "col012", DataType: "BIT(32)", Character: "utf8"}:        4,
		{Name: "col013", DataType: "numeric(32,32)", Character: "utf8"}: 30,
		{Name: "col013", DataType: "decimal(2,32)", Character: "utf8"}:  16,
		{Name: "col014", DataType: "BIT(32)", Character: "utf8"}:        4,

		// date & time
		{Name: "col015", DataType: "year(32)", Character: "utf8mb4"}:      1,
		{Name: "col016", DataType: "date", Character: "utf8mb4"}:          3,
		{Name: "col017", DataType: "time", Character: "utf8mb4"}:          3,
		{Name: "col018", DataType: "time(0)", Character: "utf8mb4"}:       3,
		{Name: "col019", DataType: "time(2)", Character: "utf8mb4"}:       3,
		{Name: "col020", DataType: "time(4)", Character: "utf8mb4"}:       3,
		{Name: "col021", DataType: "time(6)", Character: "utf8mb4"}:       3,
		{Name: "col022", DataType: "datetime", Character: "utf8mb4"}:      8,
		{Name: "col023", DataType: "timestamp(32)", Character: "utf8mb4"}: 4,

		// string
		{Name: "col024", DataType: "varchar(255)", Character: "utf8"}:    767,
		{Name: "col025", DataType: "varchar(191)", Character: "utf8mb4"}: 765,
	}

	for col, bytes := range cols50500 {
		if got := col.GetDataBytes(50500); got != bytes {
			t.Errorf("Version: 5.5.0, %s Not match, want %d, got %d", col.Name, bytes, got)
		}
	}
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func TestStringStorageReq(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	dataTypes := []string{
		"char(10)",
		"char(256)",
		"binary(10)",
		"binary(256)",
		"varchar(10)",
		"varbinary(10)",
		"enum('G','PG','PG-13','R','NC-17')",
		"set('one', 'two')",
		// wrong case
		"not_exist",
		"char(-1)",
	}
	err := GoldenDiff(func() {
		for name := range CharSets {
			for _, tp := range dataTypes {
				fmt.Println(tp, name, StringStorageReq(tp, name))
			}
		}
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	Log.Debug("Exiting function: %s", GetFunctionName())
}
