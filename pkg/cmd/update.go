package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/smy-101/gskills/internal/registry"
	"github.com/smy-101/gskills/internal/types"
	"github.com/smy-101/gskills/internal/update"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update [skill-name]",
	Short: "更新已安装的技能",
	Long:  `更新已安装的技能。如果不指定技能名称，则检查并更新所有技能。`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("用法: gskills update [skill-name]")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		token := viper.GetString("github_token")
		return executeUpdate(token, args)
	},
}

func executeUpdate(token string, args []string) error {
	updater := update.NewUpdater(token)

	if len(args) == 0 {
		return updateAllSkills(updater)
	}

	return updateSingleSkill(updater, args[0])
}

func updateSingleSkill(updater *update.Updater, skillName string) error {
	skill, err := registry.FindSkillByName(skillName)
	if err != nil {
		return fmt.Errorf("技能 '%s' 未找到: %w", skillName, err)
	}

	fmt.Printf("检查更新: %s...\n", skillName)

	hasUpdate, newSHA, err := updater.CheckUpdate(skill)
	if err != nil {
		return fmt.Errorf("检查更新失败: %w", err)
	}

	if !hasUpdate {
		fmt.Printf("  ✓ %s 已是最新版本 (commit: %s)\n", skillName, shortSHA(skill.CommitSHA))
		return nil
	}

	fmt.Printf("  → 发现更新: %s → %s\n", shortSHA(skill.CommitSHA), shortSHA(newSHA))
	fmt.Printf("更新 '%s'? [y/N]: ", skillName)

	response, err := readUserInput()
	if err != nil && err != io.EOF {
		return fmt.Errorf("读取输入失败: %w", err)
	}

	if !isYesResponse(response) {
		fmt.Println("更新已取消")
		return nil
	}

	fmt.Printf("正在更新 %s...\n", skillName)
	if err := updater.UpdateSkill(skill); err != nil {
		return fmt.Errorf("更新失败: %w", err)
	}

	fmt.Printf("  ✓ %s 更新成功\n", skillName)
	return nil
}

func updateAllSkills(updater *update.Updater) error {
	fmt.Println("检查所有技能的更新...")

	updates, err := updater.CheckAllUpdates()
	if err != nil {
		return fmt.Errorf("检查更新失败: %w", err)
	}

	if len(updates) == 0 {
		fmt.Println("没有安装任何技能")
		fmt.Println("使用 'gskills add <url>' 来安装技能")
		return nil
	}

	var availableUpdates []*types.SkillMetadata
	for _, info := range updates {
		if info.Status == update.UpdateStatusAvailable {
			availableUpdates = append(availableUpdates, info.Skill)
			fmt.Printf("  → %s: %s → %s\n", info.Skill.Name, shortSHA(info.Skill.CommitSHA), shortSHA(info.NewCommitSHA))
		} else if info.Status == update.UpdateStatusUpToDate {
			fmt.Printf("  ✓ %s: 已是最新\n", info.Skill.Name)
		} else if info.Status == update.UpdateStatusFailed {
			fmt.Printf("  ✗ %s: 检查失败 - %v\n", info.Skill.Name, info.Error)
		}
	}

	if len(availableUpdates) == 0 {
		fmt.Println("\n所有技能都是最新版本")
		return nil
	}

	fmt.Printf("\n发现 %d 个技能有更新\n", len(availableUpdates))
	fmt.Print("更新这些技能? [y/N]: ")

	response, err := readUserInput()
	if err != nil && err != io.EOF {
		return fmt.Errorf("读取输入失败: %w", err)
	}

	if !isYesResponse(response) {
		fmt.Println("更新已取消")
		return nil
	}

	fmt.Println("\n正在更新技能...")
	stats, err := updater.UpdateAll(availableUpdates)
	if err != nil {
		return fmt.Errorf("更新失败: %w", err)
	}

	fmt.Printf("\n更新完成:\n")
	fmt.Printf("  成功: %d\n", stats.Updated)
	fmt.Printf("  失败: %d\n", stats.Failed)
	fmt.Printf("  耗时: %v\n", stats.Duration)

	if stats.Failed > 0 {
		return fmt.Errorf("部分技能更新失败")
	}

	return nil
}

func shortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

// readUserInput reads a line of input from the user.
// Returns io.EOF if no input is available (e.g., user pressed Ctrl+D).
func readUserInput() (string, error) {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response), nil
}

// isYesResponse checks if the user response indicates agreement ("y" or "yes").
func isYesResponse(response string) bool {
	response = strings.ToLower(response)
	return response == "y" || response == "yes"
}
