package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(linkCmd)
}

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "link一个新项目",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("link新项目命令被调用")
	},
}
