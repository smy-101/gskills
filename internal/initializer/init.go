package initializer

import (
	"io"
	"os"
	"path/filepath"
)

type Initializer struct {
	binDir    string
	configDir string
}

func New() *Initializer {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}

	configDir := filepath.Join(home, ".gskills")
	binDir := filepath.Join(configDir, "bin")

	return &Initializer{
		binDir:    binDir,
		configDir: configDir,
	}
}

func (i *Initializer) GetBinDir() string {
	return i.binDir
}

func (i *Initializer) InstallBinary(sourcePath string) error {
	if err := os.MkdirAll(i.binDir, 0755); err != nil {
		return &InitError{
			Type:    ErrTypeDirCreate,
			Message: "无法创建 bin 目录",
			Err:     err,
		}
	}

	destPath := filepath.Join(i.binDir, "gskills")

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return &InitError{
			Type:    ErrTypeBinaryCopy,
			Message: "无法打开源二进制文件",
			Err:     err,
		}
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return &InitError{
			Type:    ErrTypeBinaryCopy,
			Message: "无法获取源文件信息",
			Err:     err,
		}
	}

	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return &InitError{
			Type:    ErrTypeBinaryCopy,
			Message: "无法创建目标二进制文件",
			Err:     err,
		}
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return &InitError{
			Type:    ErrTypeBinaryCopy,
			Message: "无法复制二进制文件",
			Err:     err,
		}
	}

	if err := destFile.Close(); err != nil {
		return &InitError{
			Type:    ErrTypeBinaryCopy,
			Message: "无法关闭目标文件",
			Err:     err,
		}
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return &InitError{
			Type:    ErrTypeBinaryCopy,
			Message: "无法设置二进制文件权限",
			Err:     err,
		}
	}

	_ = sourceInfo

	return nil
}

func (i *Initializer) DetectShell() (Shell, string, error) {
	return DetectShell()
}

func (i *Initializer) UpdatePATH(binPath, configPath string, shell Shell) error {
	hasExport, err := HasExportInConfig(configPath, binPath)
	if err != nil {
		return err
	}

	if hasExport {
		return nil
	}

	exportLine := GeneratePATHExport(binPath, shell)
	if exportLine == "" {
		return &InitError{
			Type:    ErrTypeConfigWrite,
			Message: "无法生成 PATH 导出语句：不支持的 shell 类型",
		}
	}

	return AppendToConfig(configPath, exportLine)
}

func (i *Initializer) IsInPATH(binPath string) bool {
	return IsInPATH(binPath)
}

func GetExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", &InitError{
			Type:    ErrTypePathResolution,
			Message: "无法获取可执行文件路径",
			Err:     err,
		}
	}

	resolvedPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", &InitError{
			Type:    ErrTypePathResolution,
			Message: "无法解析符号链接",
			Err:     err,
		}
	}

	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", &InitError{
			Type:    ErrTypePathResolution,
			Message: "无法获取绝对路径",
			Err:     err,
		}
	}

	return absPath, nil
}
