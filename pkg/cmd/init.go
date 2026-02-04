package cmd

import (
	"fmt"
	"os"

	"github.com/smy-101/gskills/internal/initializer"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化 gskills，将二进制文件安装到 ~/.gskills/bin 并添加到 PATH",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeInit()
	},
}

func executeInit() error {
	execPath, err := initializer.GetExecutablePath()
	if err != nil {
		return fmt.Errorf("无法获取 gskills 可执行文件路径: %w", err)
	}

	init := initializer.New()
	binDir := init.GetBinDir()

	if init.IsInPATH(binDir) {
		fmt.Println("✓ gskills 已经在 PATH 中，无需重复初始化")
		return nil
	}

	fmt.Printf("✓ 检测到源路径: %s\n", execPath)

	if err := init.InstallBinary(execPath); err != nil {
		return fmt.Errorf("无法安装二进制文件: %w", err)
	}

	fmt.Printf("✓ 复制二进制文件: %s/gskills\n", binDir)

	shell, configPath, err := init.DetectShell()
	if err != nil {
		return fmt.Errorf("无法检测 shell: %w", err)
	}

	fmt.Printf("✓ 检测到 shell: %s\n", shell)

	if err := init.UpdatePATH(binDir, configPath, shell); err != nil {
		return fmt.Errorf("无法更新 PATH: %w", err)
	}

	fmt.Printf("✓ 更新配置文件: %s\n", configPath)

	fmt.Println("\ngskills 已成功初始化！")
	fmt.Println("\n请执行以下命令使配置生效:")

	switch shell {
	case initializer.ShellZsh:
		fmt.Println("  source ~/.zshrc")
	case initializer.ShellBash:
		home, _ := os.UserHomeDir()
		if _, err := os.Stat("/Users/" + home); err == nil {
			fmt.Println("  source ~/.bash_profile")
		} else {
			fmt.Println("  source ~/.bashrc")
		}
	case initializer.ShellFish:
		fmt.Println("  source ~/.config/fish/config.fish")
	}

	fmt.Println("\n或重新打开终端窗口。")
	return nil
}
