package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/smy-101/gskills/internal/link"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(linkCmd)
}

var linkCmd = &cobra.Command{
	Use:   "link <skill_name> <path_to_project>",
	Short: "通过symlink的方式将skill链接到指定项目",
	Long: `通过symlink的方式将skill链接到指定项目的.opencode/skill/目录下。

命令格式: gskills link <skill_name> <path_to_project>

示例:
  gskills link prompt-engineer /home/user/myproject

这将在/path/to/project/.opencode/skill/prompt-engineer创建一个符号链接，指向~/.gskills/skills/prompt-engineer。`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("用法: gskills link <skill_name> <path_to_project>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeLink(args[0], args[1])
	},
}

func executeLink(skillName, projectPath string) error {
	linker := link.NewLinker()
	ctx := context.Background()

	fmt.Printf("Linking skill '%s' to project '%s'...\n", skillName, projectPath)

	if err := linker.LinkSkill(ctx, skillName, projectPath); err != nil {
		fmt.Printf("Error linking skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully linked skill '%s' to project '%s'\n", skillName, projectPath)
	fmt.Printf("Skill symlink created at: %s/.opencode/skills/%s\n", projectPath, skillName)
	return nil
}
