package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/smy-101/gskills/internal/add"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "从 GitHub 的 skills 仓库下载并添加 skills",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("用法:gskills add <github_url>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		if err := executeAdd(url); err != nil {
			fmt.Printf("Error adding skill: %v\n", err)
			os.Exit(1)
		}
		return nil
	},
}

func executeAdd(rawURL string) error {
	token := viper.GetString("github_token")
	client := add.NewClient(token)

	err := client.Download(rawURL)
	if err != nil {
		return err
	}
	return nil
}
