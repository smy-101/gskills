package cmd

import (
	"context"
	"fmt"

	"github.com/smy-101/gskills/internal/tidy"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(tidyCmd)
}

var tidyCmd = &cobra.Command{
	Use:   "tidy",
	Short: "清理无用的技能链接",
	Long: `清理无用的技能链接和注册表项。

此命令执行两个清理操作：
  1. 移除注册表中指向不存在符号链接的项目条目
  2. 删除指向已删除技能的孤立符号链接

示例:
  gskills tidy`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeTidy()
	},
}

func executeTidy() error {
	tidier := tidy.NewTidier()
	ctx := context.Background()

	fmt.Println("正在清理无用的技能链接...")

	report, err := tidier.Tidy(ctx)
	if err != nil {
		return fmt.Errorf("清理失败: %w", err)
	}

	fmt.Println("\n清理完成！")

	if report.StaleRegistryEntries > 0 {
		fmt.Printf("• 移除了 %d 个无效的注册表项\n", report.StaleRegistryEntries)
	}

	if report.OrphanedSymlinks > 0 {
		fmt.Printf("• 删除了 %d 个孤立的符号链接\n", report.OrphanedSymlinks)
	}

	if report.StaleRegistryEntries == 0 && report.OrphanedSymlinks == 0 {
		fmt.Println("• 没有发现需要清理的项目")
	}

	fmt.Printf("\n已检查 %d 个技能，扫描了 %d 个项目目录\n", report.SkillsChecked, report.ProjectsScanned)

	return nil
}
