package cmd

import (
	"errors"
	"fmt"

	"github.com/smy-101/gskills/internal/link"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(unlinkCmd)
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink <skill_name> [project_path]",
	Short: "移除项目中的技能链接",
	Long: `移除指定项目中的技能符号链接，并更新注册表。

命令格式: gskills unlink <skill_name> [project_path]

示例:
  gskills unlink prompt-engineer
  gskills unlink prompt-engineer /home/user/myproject

当不提供project_path时，默认使用当前目录。`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 || len(args) > 2 {
			return errors.New("用法: gskills unlink <skill_name> [project_path]")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		projectPath := "."
		if len(args) == 2 {
			projectPath = args[1]
		}
		return executeUnlink(skillName, projectPath)
	},
}

func executeUnlink(skillName, projectPath string) error {
	linker := link.NewLinker()

	fmt.Printf("Unlinking skill '%s' from project '%s'...\n", skillName, projectPath)

	if err := linker.UnlinkSkill(skillName, projectPath); err != nil {
		return err
	}

	fmt.Printf("Successfully unlinked skill '%s' from project '%s'\n", skillName, projectPath)
	return nil
}
