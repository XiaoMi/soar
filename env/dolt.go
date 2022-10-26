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

package env

import (
	"fmt"
	"net"
	"time"

	"github.com/XiaoMi/soar/common"

	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/server"
	"github.com/dolthub/go-mysql-server/sql/information_schema"
	"github.com/sirupsen/logrus"
)

func NewDoltDB() {
	// go-mysql-server set log
	logrus.SetLevel(logrus.FatalLevel)

	// random unused port
	// https://www.lifewire.com/port-0-in-tcp-and-udp-818145
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		common.Log.Warn(err.Error())
		return
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err = listener.Close(); err != nil {
		common.Log.Error(err.Error())
		return
	}

	// new in memory database server
	var db string
	switch common.Config.TestDSN.Schema {
	case "", "information_schema", "mysql":
		db = "test"
	default:
		db = common.Config.TestDSN.Schema
	}
	engine := sqle.NewDefault(
		memory.NewMemoryDBProvider(
			memory.NewDatabase(db),
			information_schema.NewInformationSchemaDatabase(),
		),
	)
	pass := fmt.Sprint(time.Now().Nanosecond())
	engine.Analyzer.Catalog.MySQLDb.AddSuperUser("soar", "localhost", pass)
	config := server.Config{
		Protocol: "tcp",
		Address:  fmt.Sprintf("localhost:%d", port),
	}
	s, err := server.NewDefaultServer(config, engine)
	if err != nil {
		common.Log.Error(err.Error())
		return
	}
	common.Config.TestDSN.User = "soar"
	common.Config.TestDSN.Password = pass
	common.Config.TestDSN.Net = "tcp"
	common.Config.TestDSN.Addr = fmt.Sprintf("localhost:%d", port)
	common.Config.Sampling = false // disable sampling, limit memory usage

	common.Log.Info("starting built-in dolt database, address: localhost:%d", port)
	err = s.Start()
	common.LogIfError(err, "")
}
