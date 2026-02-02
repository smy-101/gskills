package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/smy-101/gskills/pkg/cmd"
	"github.com/spf13/viper"
)

func main() {
	// 初始化viper配置
	initViper()

	// 执行cobra命令
	cmd.Execute()
}

func initViper() {
	// 设置默认配置
	viper.SetDefault("github_token", "")
	viper.SetDefault("proxy", "")

	// 设置配置文件路径
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	configDir := filepath.Join(home, ".gskills")
	configPath := filepath.Join(configDir, "config.json")

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(configDir)

	// 如果配置文件不存在，先创建目录
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)

		// 写入默认配置
		defaultConfig := map[string]interface{}{
			"github_token": "",
			"proxy":        "",
		}

		// 直接写入文件，viper会稍后读取
		data, _ := json.MarshalIndent(defaultConfig, "", "  ")
		os.WriteFile(configPath, data, 0644)
	}

	// 读取配置
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件不存在，使用默认值
		} else {
			fmt.Printf("Error reading config file: %v\n", err)
		}
	}
}
