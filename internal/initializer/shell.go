package initializer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Shell string

const (
	ShellZsh     Shell = "zsh"
	ShellBash    Shell = "bash"
	ShellFish    Shell = "fish"
	ShellUnknown Shell = ""
)

func DetectShell() (Shell, string, error) {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return ShellUnknown, "", &InitError{
			Type:    ErrTypeShellDetection,
			Message: "无法检测 shell 类型：SHELL 环境变量未设置",
		}
	}

	shellName := filepath.Base(shellPath)
	switch shellName {
	case "zsh":
		return ShellZsh, getConfigPath(ShellZsh), nil
	case "bash":
		return ShellBash, getConfigPath(ShellBash), nil
	case "fish":
		return ShellFish, getConfigPath(ShellFish), nil
	default:
		return ShellUnknown, "", &InitError{
			Type:    ErrTypeShellDetection,
			Message: fmt.Sprintf("不支持的 shell 类型: %s", shellName),
		}
	}
}

func getConfigPath(shell Shell) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch shell {
	case ShellZsh:
		return filepath.Join(home, ".zshrc")
	case ShellBash:
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, ".bash_profile")
		}
		return filepath.Join(home, ".bashrc")
	case ShellFish:
		return filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return ""
	}
}

func GeneratePATHExport(binPath string, shell Shell) string {
	switch shell {
	case ShellZsh, ShellBash:
		return fmt.Sprintf("\n# gskills PATH export\nexport PATH=\"%s:$PATH\"\n", binPath)
	case ShellFish:
		return fmt.Sprintf("\n# gskills PATH export\nfish_add_path %s\n", binPath)
	default:
		return ""
	}
}

func IsInPATH(binPath string) bool {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return false
	}

	paths := strings.Split(pathEnv, ":")
	for _, p := range paths {
		if p == binPath {
			return true
		}
	}
	return false
}

func HasExportInConfig(configPath, binPath string) (bool, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &InitError{
			Type:    ErrTypeConfigWrite,
			Message: "无法读取配置文件",
			Err:     err,
		}
	}

	configContent := string(content)
	return strings.Contains(configContent, binPath) ||
		strings.Contains(configContent, ".gskills/bin"), nil
}

func AppendToConfig(configPath, exportLine string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &InitError{
			Type:    ErrTypeDirCreate,
			Message: "无法创建配置文件目录",
			Err:     err,
		}
	}

	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return &InitError{
			Type:    ErrTypeConfigWrite,
			Message: "无法打开配置文件",
			Err:     err,
		}
	}
	defer file.Close()

	if _, err := file.WriteString(exportLine); err != nil {
		return &InitError{
			Type:    ErrTypeConfigWrite,
			Message: "无法写入配置文件",
			Err:     err,
		}
	}

	return nil
}
