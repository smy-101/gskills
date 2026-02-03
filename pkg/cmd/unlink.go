package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/smy-101/gskills/internal/link"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(unlinkCmd)
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink <skill_name> <project_path>",
	Short: "移除项目中的技能链接",
	Long:  `移除指定项目中的技能符号链接，并更新注册表。`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("用法: gskills unlink <skill_name> <project_path>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeUnlink(args[0], args[1])
	},
}

func executeUnlink(skillName, projectPath string) error {
	linker := link.NewLinker()

	fmt.Printf("Unlinking skill '%s' from project '%s'...\n", skillName, projectPath)

	if err := linker.UnlinkSkill(skillName, projectPath); err != nil {
		fmt.Printf("Error unlinking skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully unlinked skill '%s' from project '%s'\n", skillName, projectPath)
	return nil
}
