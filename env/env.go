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
	"strings"

	"github.com/XiaoMi/soar/ast"
	"github.com/XiaoMi/soar/common"
	"github.com/XiaoMi/soar/database"

	"github.com/dchest/uniuri"
	"vitess.io/vitess/go/vt/sqlparser"
)

// VirtualEnv SQL优化评审 测试环境
// DB使用的信息从配置文件中获取
type VirtualEnv struct {
	*database.Connector

	// 保存DB测试环境映射关系，防止vEnv环境冲突。
	DBRef   map[string]string
	hash2Db map[string]string
	// 保存Table创建关系，防止重复创建表
	TableMap map[string]map[string]string
	// 错误
	Error error
}

// NewVirtualEnv 初始化一个新的测试环境
func NewVirtualEnv(vEnv *database.Connector) *VirtualEnv {
	return &VirtualEnv{
		Connector: vEnv,
		DBRef:     make(map[string]string),
		hash2Db:   make(map[string]string),
		TableMap:  make(map[string]map[string]string),
	}
}

// BuildEnv 测试环境初始化&连接线上环境检查
// @output *VirtualEnv	测试环境
// @output *database.Connector 线上环境连接句柄
func BuildEnv() (*VirtualEnv, *database.Connector) {
	// 生成测试环境
	vEnv := NewVirtualEnv(&database.Connector{
		Addr:     common.Config.TestDSN.Addr,
		User:     common.Config.TestDSN.User,
		Pass:     common.Config.TestDSN.Password,
		Database: common.Config.TestDSN.Schema,
		Charset:  common.Config.TestDSN.Charset,
	})

	// 检查测试环境可用性，并记录数据库版本
	vEnvVersion, err := vEnv.Version()
	common.Config.TestDSN.Version = vEnvVersion
	if err != nil {
		common.Log.Warn("BuildEnv TestDSN: %s:********@%s/%s not available , Error: %s",
			vEnv.User, vEnv.Addr, vEnv.Database, err.Error())
		common.Config.TestDSN.Disable = true
	}

	// 连接线上环境
	// 如果未配置线上环境线测试环境配置为线上环境
	if common.Config.OnlineDSN.Addr == "" {
		common.Log.Warn("BuildEnv AllowOnlineAsTest: OnlineDSN not config, use TestDSN： %s:********@%s/%s as OnlineDSN",
			vEnv.User, vEnv.Addr, vEnv.Database)
		common.Config.OnlineDSN = common.Config.TestDSN
	}
	conn := &database.Connector{
		Addr:     common.Config.OnlineDSN.Addr,
		User:     common.Config.OnlineDSN.User,
		Pass:     common.Config.OnlineDSN.Password,
		Database: common.Config.OnlineDSN.Schema,
		Charset:  common.Config.OnlineDSN.Charset,
	}

	// 检查线上环境可用性版本
	rEnvVersion, err := vEnv.Version()
	common.Config.OnlineDSN.Version = rEnvVersion
	if err != nil {
		common.Log.Warn("BuildEnv OnlineDSN: %s:********@%s/%s not available , Error: %s",
			vEnv.User, vEnv.Addr, vEnv.Database, err.Error())
		common.Config.TestDSN.Disable = true
	}

	// 检查是否允许Online和Test一致，防止误操作
	if common.FormatDSN(common.Config.OnlineDSN) == common.FormatDSN(common.Config.TestDSN) &&
		!common.Config.AllowOnlineAsTest {
		common.Log.Warn("BuildEnv AllowOnlineAsTest: %s:********@%s/%s OnlineDSN can't config as TestDSN",
			vEnv.User, vEnv.Addr, vEnv.Database)
		common.Config.TestDSN.Disable = true
		common.Config.OnlineDSN.Disable = true
	}

	// 判断测试环境与remote环境版本是否一致
	if vEnvVersion < rEnvVersion {
		common.Log.Warning("TestDSN MySQL version older than OnlineDSN, TestDSN will not be used", vEnvVersion, rEnvVersion)
		common.Config.TestDSN.Disable = true
	}

	return vEnv, conn
}

// RealDB 从测试环境中获取通过hash后的DB
func (ve VirtualEnv) RealDB(hash string) string {
	if _, ok := ve.hash2Db[hash]; ok {
		return ve.hash2Db[hash]
	}
	return hash
}

// DBHash 从测试环境中根据DB找到对应的hash值
func (ve VirtualEnv) DBHash(db string) string {
	if _, ok := ve.DBRef[db]; ok {
		return ve.DBRef[db]
	}
	return db
}

// CleanUp 环境清理
func (ve VirtualEnv) CleanUp() bool {
	if !common.Config.TestDSN.Disable && common.Config.DropTestTemporary {
		common.Log.Debug("CleanUp ...")
		for db := range ve.hash2Db {
			ve.Database = db
			_, err := ve.Query("drop database %s", db)
			if err != nil {
				common.Log.Error("CleanUp failed Error: %s", err)
				return false
			}
		}
		common.Log.Debug("CleanUp, done")
	}
	return true
}

// BuildVirtualEnv rEnv为SQL源环境，DB使用的信息从接口获取
// 注意：如果是USE,DDL等语句，执行完第一条就会返回，后面的SQL不会执行
func (ve *VirtualEnv) BuildVirtualEnv(rEnv *database.Connector, SQLs ...string) bool {
	var stmt sqlparser.Statement
	var err error

	// 置空错误信息
	ve.Error = nil
	// 检测是否已经创建初始数据库，如果未创建则创建一个名称hash过的映射数据库
	err = ve.createDatabase(*rEnv, rEnv.Database)
	common.LogIfWarn(err, "")

	// 测试环境检测
	if common.Config.TestDSN.Disable {
		common.Log.Info("BuildVirtualEnv TestDSN not config")
		return true
	}

	// 判断rEnv中是否指定了DB
	if rEnv.Database == "" {
		common.Log.Error("BuildVirtualEnv no database specified, TestDSN init failed")
		return false
	}

	// 库表提取
	meta := make(map[string]*common.DB)
	for _, sql := range SQLs {

		common.Log.Debug("BuildVirtualEnv Database&Table Mapping, SQL: %s", sql)

		stmt, err = sqlparser.Parse(sql)
		if err != nil {
			common.Log.Error("BuildVirtualEnv Error : %v", err)
			return false
		}

		// 语句类型判断
		switch stmt := stmt.(type) {
		case *sqlparser.Use:
			// 如果是use语句，则更改基础环配置
			if _, ok := meta[stmt.DBName.String()]; !ok {
				// 如果USE了一个线上环境不存在的数据库，将创建该数据库，字符集默认utf8mb4
				meta[stmt.DBName.String()] = common.NewDB(stmt.DBName.String())
				rEnv.Database = stmt.DBName.String()

				// use DB 后检查 DB是否已经创建，如果没有创建则创建DB
				err = ve.createDatabase(*rEnv, rEnv.Database)
				common.LogIfWarn(err, "")
			}
			return true
		case *sqlparser.DDL:
			// 如果是DDL，则先获取DDL对应的表结构，然后直接在测试环境接执行SQL
			// 为不影响其他SQL操作，复制一个Connector对象，将数据库切换到对应的DB上直接执行
			tmpDB := *ve.Connector
			tmpDB.Database = ve.DBRef[rEnv.Database]

			// 为了支持并发，需要将DB进行映射，但db.table这种形式无法保证DB的映射是正确的
			// TODO：暂不支持 create db.tableName (id int) 形式的建表语句
			if stmt.Table.Qualifier.String() != "" || stmt.NewName.Qualifier.String() != "" {
				common.Log.Error("BuildVirtualEnv DDL Not support '.'")
				return false
			}

			// 拉取表结构
			table := stmt.Table.Name.String()
			if table != "" {
				err = ve.createTable(*rEnv, rEnv.Database, table)
				if err != nil {
					common.Log.Error("BuildVirtualEnv Error : %v", err)
					return false
				}
			}

			_, err = tmpDB.Query(sql)
			if err != nil {
				switch stmt.Action {
				case "create", "alter":
					// 如果是创建或者修改语句，且报错信息为如重复建表、重复索引等信息，将错误反馈到上一次层输出建议
					ve.Error = err
				default:
					common.Log.Error("BuildVirtualEnv DDL Execute Error : %v", err)
				}
			}
			return true
		}

		meta := ast.GetMeta(stmt, nil)

		// 由于DB环境可能是变的，所以需要每一次都单独的提取库表结构，整体随着rEnv的变动而发生变化
		for db, table := range meta {
			if db == "" {
				db = rEnv.Database
			}
			tmpEnv := *rEnv
			tmpEnv.Database = db

			// 创建数据库环境
			for _, tb := range table.Table {
				if tb.TableName == "" {
					continue
				}

				// 视图检查
				common.Log.Debug("BuildVirtualEnv Checking view -- %s.%s", tmpEnv.Database, tb.TableName)
				tbStatus, err := tmpEnv.ShowTableStatus(tb.TableName)
				if err != nil {
					common.Log.Error("BuildVirtualEnv ShowTableStatus Error : %v", err)
					return false
				}

				// 如果是视图，解析语句
				if len(tbStatus.Rows) > 0 && tbStatus.Rows[0].Comment == "VIEW" {
					tmpEnv.Database = db
					var viewDDL string
					viewDDL, err = tmpEnv.ShowCreateTable(tb.TableName)
					if err != nil {
						common.Log.Error("BuildVirtualEnv create view failed: %v", err)
						return false
					}

					startIdx := strings.Index(viewDDL, "AS")
					viewDDL = viewDDL[startIdx+2:]
					if !ve.BuildVirtualEnv(&tmpEnv, viewDDL) {
						return false
					}
				}

				err = ve.createTable(tmpEnv, db, tb.TableName)
				if err != nil {
					common.Log.Error("BuildVirtualEnv Error : %v", err)
					return false
				}
			}
		}
	}
	return true
}

func (ve VirtualEnv) createDatabase(rEnv database.Connector, dbName string) error {
	// 生成映射关系
	if _, ok := ve.DBRef[dbName]; ok {
		common.Log.Debug("createDatabase, Database `%s` created", dbName)
		return nil
	}

	dbHash := "optimizer_" + uniuri.New()
	common.Log.Debug("createDatabase, mapping `%s` :`%s`-->`%s`", dbName, dbName, dbHash)
	ddl, err := rEnv.ShowCreateDatabase(dbName)
	if err != nil {
		common.Log.Warning("createDatabase, rEnv.ShowCreateDatabase Error : %v", err)
		ddl = fmt.Sprintf("create database `%s` character set utf8mb4", dbName)
	}

	ddl = strings.Replace(ddl, dbName, dbHash, -1)
	_, err = ve.Query(ddl)
	if err != nil {
		common.Log.Warning("createDatabase, Error : %v", err)
		return err
	}

	// 创建成功，添加映射记录
	ve.DBRef[dbName] = dbHash
	ve.hash2Db[dbHash] = dbName
	return nil
}

/*
	@input:
		database.Connector 为一个线上环境数据库连接句柄的复制，因为在处理SQL时需要对上下文进行关联处理，
		所以存在修改DB连接参数（主要是数据库名称变更）的可能性，为了不影响整体上下文的环境，所以需要一个镜像句柄来做当前环境的操作。

		dbName, tbName: 需要在环境中操作的库表名称，

	@output:
		return 执行过程中的错误

	NOTE:
		该函数会将线上环境中使用到的库表结构复制到测试环境中，为后续操作提供基础环境。
		传入的库表名称均来自于对AST的解析，库表名称的获取遵循以下原则：
			如果未在SQL中指定数据库名称，则数据库一定是配置文件（或命令行参数传入DSN）中指定的数据库
			如果一个SQL中存在多个数据库，则只能有一个数据库是没有在SQL中被显示指定的（即DSN中指定的数据库）
	TODO:
		在一些可能的情况下，由于数据库配置的不一致（如SQL_MODE不同）导致remote环境的库表无法正确的在测试环境进行同步，
		soar能够做出判断并进行session级别的修改，但是这一阶段可用性保证应该是由用户提供两个完全相同（或测试环境兼容线上环境）
		的数据库环境来实现的。
*/
func (ve VirtualEnv) createTable(rEnv database.Connector, dbName, tbName string) error {

	if dbName == "" {
		dbName = rEnv.Database
	}
	// 如果 dbName 不为空，说明指定了DB，临时修改rEnv中DB参数，来确保执行正确性
	rEnv.Database = dbName

	if ve.TableMap[dbName] == nil {
		ve.TableMap[dbName] = make(map[string]string)
	}

	if strings.ToLower(tbName) == "dual" {
		common.Log.Debug("createTable, %s no need create", tbName)
		return nil
	}

	if ve.TableMap[dbName][tbName] != "" {
		common.Log.Debug("createTable, `%s`.`%s` created", dbName, tbName)
		return nil
	}

	common.Log.Debug("createTable, Database: %s, Table: %s", dbName, tbName)

	//  TODO：查看是否有外键关联（done），对外键的支持 (未解决循环依赖的问题)

	// 判断数据库是否已经创建
	if ve.DBRef[dbName] == "" {
		// 若没创建，则创建数据库
		err := ve.createDatabase(rEnv, dbName)
		if err != nil {
			return err
		}
	}

	// 记录Table创建信息
	ve.TableMap[dbName][tbName] = tbName

	// 生成建表语句
	common.Log.Debug("createTable DSN(%s/%s): generate ddl", rEnv.Addr, rEnv.Database)

	ddl, err := rEnv.ShowCreateTable(tbName)
	if err != nil {
		// 有可能是用户新建表，因此线上环境查不到
		common.Log.Error("createTable, %s DDL Error : %v", tbName, err)
		return err
	}

	// 改变数据环境
	ve.Database = ve.DBRef[dbName]
	_, err = ve.Query(ddl)
	if err != nil {
		// 有可能是用户新建表，因此线上环境查不到
		common.Log.Error("createTable, %s Error : %v", tbName, err)
		return err
	}

	// 泵取数据
	if common.Config.Sampling {
		common.Log.Debug("createTable, Start Sampling data from %s.%s to %s.%s ...", dbName, tbName, ve.DBRef[dbName], tbName)
		err := ve.SamplingData(rEnv, tbName)
		if err != nil {
			common.Log.Error(" (ve VirtualEnv) createTable SamplingData Error: %v", err)
			return err
		}
	}
	return nil
}

// GenTableColumns 为Rewrite提供的结构体初始化
func (ve *VirtualEnv) GenTableColumns(meta common.Meta) common.TableColumns {
	tableColumns := make(common.TableColumns)
	for dbName, db := range meta {
		for _, tb := range db.Table {
			// 防止传入非预期值
			if tb == nil {
				break
			}
			td, err := ve.Connector.ShowColumns(tb.TableName)
			if err != nil {
				common.Log.Warn("GenTableColumns, ShowColumns Error: " + err.Error())
				break
			}

			// tableColumns 初始化
			if dbName == "" {
				dbName = ve.RealDB(ve.Connector.Database)
			}

			if _, ok := tableColumns[dbName]; !ok {
				tableColumns[dbName] = make(map[string][]*common.Column)
			}

			if _, ok := tableColumns[dbName][tb.TableName]; !ok {
				tableColumns[dbName][tb.TableName] = make([]*common.Column, 0)
			}

			if len(tb.Column) == 0 {
				// tb.column为空说明SQL里这个表是用的*来查询
				if err != nil {
					common.Log.Error("ast.Rewrite ShowColumns, Error: %v", err)
					break
				}

				for _, colInfo := range td.DescValues {
					tableColumns[dbName][tb.TableName] = append(tableColumns[dbName][tb.TableName], &common.Column{
						Name:       colInfo.Field,
						DB:         dbName,
						Table:      tb.TableName,
						DataType:   colInfo.Type,
						Character:  colInfo.Collation,
						Key:        colInfo.Key,
						Default:    colInfo.Default,
						Extra:      colInfo.Extra,
						Comment:    colInfo.Comment,
						Privileges: colInfo.Privileges,
						Null:       colInfo.Null,
					})
				}
			} else {
				// tb.column如果不为空则需要把使用到的列填写进去
				var columns []*common.Column
				for _, col := range tb.Column {
					for _, colInfo := range td.DescValues {
						if col.Name == colInfo.Field {
							// 根据获取的信息将列的信息补全
							col.DB = dbName
							col.Table = tb.TableName
							col.DataType = colInfo.Type
							col.Character = colInfo.Collation
							col.Key = colInfo.Key
							col.Default = colInfo.Default
							col.Extra = colInfo.Extra
							col.Comment = colInfo.Comment
							col.Privileges = colInfo.Privileges
							col.Null = colInfo.Null

							columns = append(columns, col)
							break
						}
					}
				}
				tableColumns[dbName][tb.TableName] = columns
			}
		}
	}
	return tableColumns
}
