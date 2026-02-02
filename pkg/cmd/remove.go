package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "删除一个项目",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("删除项目命令被调用")
	},
}
