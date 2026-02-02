package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(unlinkCmd)
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "unlink一个新项目",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("unlink一个新项目命令被调用")
	},
}
