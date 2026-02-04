package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configKeys 定义所有支持的配置项
var configKeys = []string{"github_token", "proxy"}

// validConfigKeys 用于验证配置键的有效性
var validConfigKeys = map[string]bool{}

// configMutex 保护 viper 并发访问的互斥锁（viper 不是并发安全的）
var configMutex sync.Mutex

func init() {
	for _, key := range configKeys {
		validConfigKeys[key] = true
	}
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理 gskills 配置",
	Long:  "管理 gskills 配置文件 (~/.gskills/config.json)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeConfigList()
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "获取指定配置项的值",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeConfigGet(args[0])
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "设置配置项的值",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeConfigSet(args[0], args[1])
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有配置项",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeConfigList()
	},
}

// executeConfigGet 获取并显示指定配置项的值
// 对于敏感配置（如 github_token），显示时会隐藏实际值
// 使用互斥锁保护 viper 并发访问
func executeConfigGet(key string) error {
	if !validConfigKeys[key] {
		return fmt.Errorf("无效的配置项: %s (有效选项: github_token, proxy)", key)
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	value := viper.GetString(key)
	if value == "" {
		fmt.Printf("%s: (未设置)\n", key)
		return nil
	}

	if key == "github_token" {
		fmt.Printf("%s: ***\n", key)
	} else {
		fmt.Printf("%s: %s\n", key, value)
	}
	return nil
}

// executeConfigSet 设置指定配置项的值并持久化到配置文件
// 配置文件权限设置为 0600（仅所有者可读写）以保护敏感信息
// 使用互斥锁保护 viper 并发访问（viper 不是并发安全的）
func executeConfigSet(key, value string) error {
	if !validConfigKeys[key] {
		return fmt.Errorf("无效的配置项: %s (有效选项: github_token, proxy)", key)
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	viper.Set(key, value)

	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("无法获取用户目录: %w", err)
		}
		configPath = filepath.Join(home, ".gskills", "config.json")
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("无法创建配置目录: %w", err)
	}

	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	// 设置配置文件权限为 0600（仅所有者可读写）
	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("设置配置文件权限失败: %w", err)
	}

	fmt.Printf("已设置 %s = %s\n", key, value)
	return nil
}

// executeConfigList 列出所有配置项的当前值
// 对于敏感配置（如 github_token），显示时会隐藏实际值
// 使用互斥锁保护 viper 并发访问
func executeConfigList() error {
	configMutex.Lock()
	defer configMutex.Unlock()

	fmt.Println("当前配置:")
	for _, key := range configKeys {
		value := viper.GetString(key)
		if value == "" {
			fmt.Printf("  %s: (未设置)\n", key)
		} else {
			if key == "github_token" {
				fmt.Printf("  %s: ***\n", key)
			} else {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
	}

	configPath := viper.ConfigFileUsed()
	if configPath != "" {
		fmt.Printf("\n配置文件: %s\n", configPath)
	}

	return nil
}
