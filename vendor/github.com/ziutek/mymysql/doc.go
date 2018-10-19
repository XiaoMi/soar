// Package mymysql provides MySQL client API and database/sql driver.
//
// It can be used as a library or as a database/sql driver.
//
// Using as a library
//
// Import native or thrsafe engine. Optionally import autorc for autoreconnect connections.
//
//	import (
//		"github.com/ziutek/mymysql/mysql"
//		_ "github.com/ziutek/mymysql/thrsafe" // OR native
//		// _ "github.com/ziutek/mymysql/native"
//		"github.com/ziutek/mymysql/autorc" // for autoreconnect
//	)
//
//
//
// Using as a Go sql driver
//
// Import Go standard sql package and godrv driver.
//
//	import (
//		"database/sql"
//		_ "github.com/ziutek/mymysql/godrv"
//	)
//
//
//
package mymysql
