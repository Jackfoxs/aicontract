package config

import (
	"strconv"
	"xorm.io/xorm/log"
)

type Mysql struct {
	Host     string       `yaml:"host"`
	Port     int          `yaml:"port"`
	Config   string       `yaml:"config"`
	DB       string       `yaml:"db"`
	User     string       `yaml:"user"`
	Password string       `yaml:"password"`
	LogLevel log.LogLevel `yaml:"log_level"` //日志等级,LOG_DEBUG 0 LOG_INFO 1 LOG_WARNING 2 LOG_ERR 3 LOG_OFF  4 LOG_UNKNOWN 5
	ShowSql  bool         `yaml:"showSql"`
}

func (m *Mysql) Dsn() string {
	// 修改为适用于 XORM 的 DSN 格式
	return m.User + ":" + m.Password + "@tcp(" + m.Host + ":" + strconv.Itoa(m.Port) + ")/" + m.DB + "?" + m.Config
}
