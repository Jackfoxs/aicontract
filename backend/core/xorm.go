package core

import (
	"context"
	"fmt"
	"backend/global"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

// InitXorm 初始化xorm
func InitXorm() *xorm.Engine {
	Dsn := global.Config.Mysql.Dsn()
	if Dsn == "" {
		global.Log.Warn("未配置mysql，取消xorm连接")
		return nil
	}

	DBengine, err := xorm.NewEngine("mysql", Dsn)
	if err != nil {
		global.Log.Error("mysql连接失败", "dsn", Dsn, "error", err)
		panic(fmt.Sprintf("[%s]mysql连接失败: %v", Dsn, err))
	} else {
		PingErr := DBengine.Ping()
		if PingErr != nil {
			global.Log.Error("mysql连接失败", "dsn", Dsn, "error", PingErr)
			panic(fmt.Sprintf("[%s]mysql连接失败: %v", Dsn, PingErr))
		} else {
			fmt.Println("mysql连接成功")
		}
	}

	DBengine.SetLogLevel(2)
	// DBengine.ShowSQL(true)

	SqlDB := DBengine.DB()
	if SqlDB == nil {
		global.Log.Error("无法获取数据库连接池")
		panic("无法获取数据库连接池")
	}
	SqlDB.SetMaxIdleConns(10)               //最大空闲容量
	SqlDB.SetMaxOpenConns(100)              //最多可容纳
	SqlDB.SetConnMaxLifetime(time.Hour * 4) //连接最大服用时间，不能超过mysql的wait_timeout

	return DBengine
}

// CheckTableChanges 检查表是否需要创建或修改
// 返回 changed（是否变更）, created（是否创建）, err
func CheckTableChanges(engine *xorm.Engine, model interface{}) (bool, bool, error) {
	tableName := engine.TableName(model)
	dialect := engine.Dialect()
	ctx := context.Background()
	// 检查表是否存在

	exists, err := dialect.IsTableExist(global.DB.DB(), ctx, tableName)
	if err != nil {
		return false, false, err
	}

	// 如果表不存在，Sync2 会创建表
	if !exists {
		return true, true, nil
	}

	// 获取数据库表的字段信息
	columnNames, _, err := dialect.GetColumns(global.DB.DB(), ctx, tableName)
	if err != nil {
		return false, false, err
	}

	// 获取模型定义的字段信息
	schema, err := engine.TableInfo(model)
	if err != nil {
		return false, false, err
	}
	modelColumns := schema.Columns()

	// 检查是否有新增字段
	modelColumnMap := make(map[string]bool)
	for _, col := range modelColumns {
		modelColumnMap[col.Name] = true
	}

	for _, colName := range columnNames {
		delete(modelColumnMap, colName)
	}

	// 如果 model 中有字段不在数据库中，说明需要新增字段
	changed := len(modelColumnMap) > 0
	return changed, false, nil
}

// SyncWithLog 同步表结构并记录日志
func SyncWithLog(engine *xorm.Engine, models ...interface{}) error {
	for _, model := range models {
		changed, created, err := CheckTableChanges(engine, model)
		if err != nil {
			return err
		}

		tableName := engine.TableName(model)
		if created {
			fmt.Printf("表 %s 被创建\n", tableName)
		} else if changed {
			fmt.Printf("表 %s 结构被修改\n", tableName)
		} else {
			fmt.Printf("表 %s 无变化\n", tableName) // 可选：不打印此行则“无变化”静默
		}
	}

	err := global.DB.Sync2(models...)
	if err != nil {
		return err
	}
	global.DB.SetLogLevel(global.Config.Mysql.LogLevel)
	global.DB.ShowSQL(global.Config.Mysql.ShowSql)
	return nil
}
