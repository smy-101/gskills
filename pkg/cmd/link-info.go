package cmd

import (
	"fmt"
	"os"

	"github.com/smy-101/gskills/internal/registry"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(linkInfoCmd)
}

var linkInfoCmd = &cobra.Command{
	Use:   "info <skill_name>",
	Short: "显示技能的详细链接信息",
	Long:  `显示指定技能的详细链接信息，包括链接到的所有项目路径。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeLinkInfo(args[0])
	},
}

func executeLinkInfo(skillName string) error {
	skill, err := registry.FindSkillByName(skillName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Skill: %s\n", skill.Name)
	fmt.Printf("Version: %s\n", skill.Version)
	fmt.Printf("Source: %s\n", skill.SourceURL)
	fmt.Printf("Store Path: %s\n", skill.StorePath)
	fmt.Printf("\n")

	if len(skill.LinkedProjects) == 0 {
		fmt.Println("This skill is not linked to any projects.")
		return nil
	}

	fmt.Printf("Linked to %d project(s):\n", len(skill.LinkedProjects))
	for projectPath, linkInfo := range skill.LinkedProjects {
		fmt.Printf("  • %s\n", projectPath)
		fmt.Printf("    Symlink: %s\n", linkInfo.SymlinkPath)
		fmt.Printf("    Linked: %s\n", linkInfo.LinkedAt.Format("2006-01-02 15:04"))
		fmt.Printf("\n")
	}

	return nil
}
