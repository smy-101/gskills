package cmd

import (
	"errors"
	"fmt"

	"github.com/smy-101/gskills/internal/remove"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove <skill-name>",
	Short: "删除指定技能",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("用法: gskills remove <skill-name>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]
		if err := remove.RemoveSkillByName(skillName); err != nil {
			if err.Error() == "operation cancelled" {
				fmt.Println("Operation cancelled")
				return nil
			}
			return err
		}
		fmt.Printf("Successfully removed skill '%s'\n", skillName)
		return nil
	},
}
