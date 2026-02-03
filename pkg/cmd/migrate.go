package cmd

import (
	"fmt"
	"os"

	"github.com/smy-101/gskills/internal/link"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "迁移旧版本的链接数据到新格式",
	Long:  `将旧版本的链接数据（linked- 前缀的技能条目）迁移到新格式（在原技能的 LinkedProjects 字段中跟踪链接）。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeMigrate()
	},
}

func executeMigrate() error {
	fmt.Println("Migrating legacy links to new format...")

	if err := link.MigrateLegacyLinks(); err != nil {
		fmt.Printf("Migration failed: %v\n", err)
		os.Exit(1)
	}

	return nil
}
