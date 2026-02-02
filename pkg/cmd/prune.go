package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pruneCmd)
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "清理无用项目",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("清理无用项目命令被调用")
	},
}
