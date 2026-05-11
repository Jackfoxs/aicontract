package core

import (
	"backend/config"
	"backend/global"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// InitConfig 初始化配置
func InitConfig() {
	var configFile = "settings.yaml"
	// 读取配置文件
	file, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("读取配置文件失败: %s\n", err)
		return
	}

	// 解析配置文件
	c := &config.Config{}
	err = yaml.Unmarshal(file, c)
	if err != nil {
		fmt.Printf("解析配置文件失败: %s\n", err)
		return
	}

	if c.ChatConfig != nil {
		c.ChatConfig.ApplyDefaults()
	}
	if c.Compliance != nil {
		c.Compliance.ApplyDefaults()
	}

	// 设置全局配置
	global.Config = c
}
