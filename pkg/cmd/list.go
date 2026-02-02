package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有项目",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("列出所有项目命令被调用")
	},
}
