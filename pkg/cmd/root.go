package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gskills",
	Short: "gskills CLI",
	Long:  "gskills CLI 工具入口",

	// 可选：关闭默认的 completion 子命令（你现在看到的 completion 就是 Cobra 自动加的）
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},

	// 可选：让直接运行 `gskills` 时总是打印 help（显式行为）
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
