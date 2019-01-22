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
	Log.Debug("Entering function: %s", GetFunctionName())
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
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func TestJSONFind(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	jsons := []string{
		`
	{
		"Collate":{
			"Collate":{
				"Collate":{
					"key":"value"
				}
			}
		}
	}
`,
		`{
  "programmers": [
    {
      "firstName": "Janet",
      "Collate": "McLaughlin",
    }, {
      "firstName": "Elliotte",
      "Collate": "Hunter",
    }, {
      "firstName": "Jason",
      "Collate": "Harold",
    }
  ]
}`,
		`
{
  "widget": {
    "debug": "on",
    "Collate": {
      "title": "Sample Konfabulator Widget",
      "name": "main_window",
      "width": 500,
      "height": 500
    },
    "image": {
      "src": "Images/Sun.png",
      "hOffset": 250,
      "vOffset": 250,
      "alignment": "center"
    },
    "text": {
      "data": "Click Here",
      "size": 36,
      "style": "bold",
      "vOffset": 100,
      "alignment": "center",
      "onMouseUp": "sun1.opacity = (sun1.opacity / 100) * 90;"
    }
  }
}
`,
		`
[
  {
    "SQLCache": true,
    "CalcFoundRows": false,
    "StraightJoin": false,
    "Priority": 0,
    "Distinct": false,
    "From": {
      "TableRefs": {
        "Left": {
          "Source": {
            "Schema": {
              "O": "",
              "L": ""
            },
            "Name": {
              "O": "tb",
              "L": "tb"
            },
            "DBInfo": null,
            "TableInfo": null,
            "IndexHints": null
          },
          "AsName": {
            "O": "",
            "L": ""
          }
        },
        "Right": null,
        "Tp": 0,
        "On": null,
        "Using": null,
        "NaturalJoin": false,
        "StraightJoin": false
      }
    },
    "Where": {
      "Type": {
        "Tp": 0,
        "Flag": 0,
        "Flen": 0,
        "Decimal": 0,
        "Charset": "",
        "Collate": "",
        "Elems": null
      },
      "Op": 4,
      "L": {
        "Type": {
          "Tp": 0,
          "Flag": 0,
          "Flen": 0,
          "Decimal": 0,
          "Charset": "",
          "Collate": "",
          "Elems": null
        },
        "Op": 7,
        "L": {
          "Type": {
            "Tp": 0,
            "Flag": 0,
            "Flen": 0,
            "Decimal": 0,
            "Charset": "",
            "Collate": "",
            "Elems": null
          },
          "Name": {
            "Schema": {
              "O": "",
              "L": ""
            },
            "Table": {
              "O": "",
              "L": ""
            },
            "Name": {
              "O": "col3",
              "L": "col3"
            }
          },
          "Refer": null
        },
        "R": {
          "Type": {
            "Tp": 8,
            "Flag": 128,
            "Flen": 1,
            "Decimal": 0,
            "Charset": "binary",
            "Collate": "binary",
            "Elems": null
          }
        }
      },
      "R": {
        "Type": {
          "Tp": 0,
          "Flag": 0,
          "Flen": 0,
          "Decimal": 0,
          "Charset": "",
          "Collate": "",
          "Elems": null
        },
        "Op": 1,
        "L": {
          "Type": {
            "Tp": 0,
            "Flag": 0,
            "Flen": 0,
            "Decimal": 0,
            "Charset": "",
            "Collate": "",
            "Elems": null
          },
          "Op": 7,
          "L": {
            "Type": {
              "Tp": 0,
              "Flag": 0,
              "Flen": 0,
              "Decimal": 0,
              "Charset": "",
              "Collate": "",
              "Elems": null
            },
            "Name": {
              "Schema": {
                "O": "",
                "L": ""
              },
              "Table": {
                "O": "",
                "L": ""
              },
              "Name": {
                "O": "col3",
                "L": "col3"
              }
            },
            "Refer": null
          },
          "R": {
            "Type": {
              "Tp": 8,
              "Flag": 128,
              "Flen": 1,
              "Decimal": 0,
              "Charset": "binary",
              "Collate": "binary",
              "Elems": null
            }
          }
        },
        "R": {
          "Type": {
            "Tp": 0,
            "Flag": 0,
            "Flen": 0,
            "Decimal": 0,
            "Charset": "",
            "Collate": "",
            "Elems": null
          },
          "Op": 7,
          "L": {
            "Type": {
              "Tp": 0,
              "Flag": 0,
              "Flen": 0,
              "Decimal": 0,
              "Charset": "",
              "Collate": "",
              "Elems": null
            },
            "Op": 7,
            "L": {
              "Type": {
                "Tp": 0,
                "Flag": 0,
                "Flen": 0,
                "Decimal": 0,
                "Charset": "",
                "Collate": "",
                "Elems": null
              },
              "Name": {
                "Schema": {
                  "O": "",
                  "L": ""
                },
                "Table": {
                  "O": "",
                  "L": ""
                },
                "Name": {
                  "O": "col1",
                  "L": "col1"
                }
              },
              "Refer": null
            },
            "R": {
              "Type": {
                "Tp": 0,
                "Flag": 0,
                "Flen": 0,
                "Decimal": 0,
                "Charset": "",
                "Collate": "",
                "Elems": null
              },
              "Name": {
                "Schema": {
                  "O": "",
                  "L": ""
                },
                "Table": {
                  "O": "",
                  "L": ""
                },
                "Name": {
                  "O": "col2",
                  "L": "col2"
                }
              },
              "Refer": null
            }
          },
          "R": {
            "Type": {
              "Tp": 253,
              "Flag": 0,
              "Flen": 3,
              "Decimal": -1,
              "Charset": "utf8mb4",
              "Collate": "utf8mb4_bin",
              "Elems": null
            }
          }
        }
      }
    },
    "Fields": {
      "Fields": [
        {
          "Offset": 7,
          "WildCard": {
            "Table": {
              "O": "",
              "L": ""
            },
            "Schema": {
              "O": "",
              "L": ""
            }
          },
          "Expr": null,
          "AsName": {
            "O": "",
            "L": ""
          },
          "Auxiliary": false
        }
      ]
    },
    "GroupBy": null,
    "Having": null,
    "WindowSpecs": null,
    "OrderBy": null,
    "Limit": null,
    "LockTp": 0,
    "TableHints": null,
    "IsAfterUnionDistinct": false,
    "IsInBraces": false
  }
]
`,
	}
	err := GoldenDiff(func() {
		for _, json := range jsons {
			result := JSONFind(json, "Collate")
			fmt.Println(result)
		}
	}, t.Name(), update)
	if err != nil {
		t.Error(err)
	}
	Log.Debug("Exiting function: %s", GetFunctionName())
}

func TestRemoveDuplicatesItem(t *testing.T) {
	Log.Debug("Entering function: %s", GetFunctionName())
	unique := RemoveDuplicatesItem([]string{"a", "a", "b", "c"})
	if len(unique) != 3 {
		t.Error("string list length should 3", unique)
	}
	Log.Debug("Exiting function: %s", GetFunctionName())
}
