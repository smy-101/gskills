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
	initViper()
	cmd.Execute()
}

func initViper() {
	viper.SetDefault("github_token", "")
	viper.SetDefault("proxy", "")

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

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)

		defaultConfig := map[string]interface{}{
			"github_token": "",
			"proxy":        "",
		}

		data, err := json.MarshalIndent(defaultConfig, "", "  ")
		if err != nil {
			fmt.Printf("Error creating default config: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			os.Exit(1)
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("Error reading config file: %v\n", err)
			os.Exit(1)
		}
	}
}
