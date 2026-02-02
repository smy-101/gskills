package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新一个项目",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("更新项目命令被调用")
	},
}
