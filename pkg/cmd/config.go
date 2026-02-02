package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "显示当前配置",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("config file:", viper.ConfigFileUsed())
		fmt.Println("github_token:", viper.GetString("github_token"))
		fmt.Println("proxy:", viper.GetString("proxy"))
	},
}
