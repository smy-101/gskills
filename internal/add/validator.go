package add

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathValidator 路径校验器
type PathValidator struct {
	BaseStorePath string
}

// NewPathValidator 创建校验器
func NewPathValidator(basePath string) *PathValidator {
	return &PathValidator{
		BaseStorePath: basePath,
	}
}

// ValidateAndSanitize 验证并清理路径
func (v *PathValidator) ValidateAndSanitize(targetPath, skillName string) (string, error) {
	// 1. 清理路径
	cleanPath := filepath.Clean(targetPath)

	// 2. 检查是否包含路径遍历攻击
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", targetPath)
	}

	// 3. 确保路径在存储目录内
	absTarget, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %v", err)
	}

	absBase, err := filepath.Abs(v.BaseStorePath)
	if err != nil {
		return "", fmt.Errorf("failed to get base absolute path: %v", err)
	}

	// 4. 检查是否在base目录内
	if !strings.HasPrefix(absTarget, absBase) {
		return "", fmt.Errorf("target path outside of store directory: %s", absTarget)
	}

	// 5. 为技能创建专用目录
	skillDir := filepath.Join(absTarget, skillName)

	return skillDir, nil
}

// GenerateUniqueID 生成唯一ID
func GenerateUniqueID(owner, repo, path string) string {
	if path == "" {
		return fmt.Sprintf("%s/%s", owner, repo)
	}
	return fmt.Sprintf("%s/%s/%s", owner, repo, path)
}

// IsSkillDirectory 检查目录是否为技能目录
func IsSkillDirectory(files []string) bool {
	// 技能目录应包含特定文件
	requiredFiles := []string{"SKILL.md", "skill.md", "README.md", "manifest.json"}

	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[strings.ToLower(file)] = true
	}

	// 至少包含一个技能标识文件
	for _, reqFile := range requiredFiles {
		if fileSet[strings.ToLower(reqFile)] {
			return true
		}
	}

	return false
}
