package add

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// URLType URL类型
type URLType int

const (
	URLTypeUnknown   URLType = iota
	URLTypeRepoRoot          // 仓库根目录
	URLTypeSkillDir          // 单技能目录
	URLTypeSkillFile         // 单技能文件
)

// DetectionResult 检测结果
type DetectionResult struct {
	Type      URLType
	Owner     string
	Repo      string
	Branch    string
	Path      string
	SkillName string
	IsGitHub  bool
	RawURL    string
}

// DetectURL 智能识别URL类型
func DetectURL(rawURL string) (*DetectionResult, error) {
	result := &DetectionResult{
		RawURL: rawURL,
	}

	// 解析URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}

	// 检查是否GitHub
	if strings.Contains(parsed.Host, "github.com") {
		result.IsGitHub = true

		// 提取路径部分
		path := strings.TrimPrefix(parsed.Path, "/")
		parts := strings.Split(path, "/")

		if len(parts) >= 2 {
			result.Owner = parts[0]
			result.Repo = parts[1]

			// GitHub URL模式匹配
			if strings.Contains(rawURL, "/tree/") {
				// 带分支的路径
				treeIndex := -1
				for i, part := range parts {
					if part == "tree" {
						treeIndex = i
						break
					}
				}

				if treeIndex != -1 && len(parts) > treeIndex+1 {
					result.Branch = parts[treeIndex+1]
					result.Path = strings.Join(parts[treeIndex+2:], "/")

					// 判断是否指向skills目录
					if strings.HasPrefix(result.Path, "skills/") {
						skillPath := strings.TrimPrefix(result.Path, "skills/")
						skillParts := strings.Split(skillPath, "/")

						if len(skillParts) > 0 {
							result.SkillName = skillParts[0]
							if len(skillParts) == 1 || (len(skillParts) == 2 && skillParts[1] == "") {
								result.Type = URLTypeSkillDir
							} else {
								// 指向具体文件
								result.Type = URLTypeSkillFile
							}
						}
					} else if result.Path == "" {
						// 仓库根目录
						result.Type = URLTypeRepoRoot
					}
				}
			} else if len(parts) == 2 {
				// 只有owner/repo，仓库根目录
				result.Type = URLTypeRepoRoot
			}
		}
	}

	// 如果未识别出类型，尝试其他解析
	if result.Type == URLTypeUnknown {
		// 可以添加其他Git托管平台的支持
		result.Type = URLTypeSkillDir
		result.SkillName = filepath.Base(rawURL)
	}

	return result, nil
}

// ShouldBatchProcess 是否需要批量处理
func (d *DetectionResult) ShouldBatchProcess() bool {
	return d.Type == URLTypeRepoRoot
}
