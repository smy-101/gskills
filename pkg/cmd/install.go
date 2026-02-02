package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "安装一个新项目",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("安装新项目命令被调用")
	},
}
