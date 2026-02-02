package add

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/smy-101/gskills/internal/types"
)

// Downloader 文件下载器
type Downloader struct {
	githubClient *Client
	validator    *PathValidator
}

// NewDownloader 创建下载器
func NewDownloader(token string, storePath string) *Downloader {
	return &Downloader{
		githubClient: NewClient(token),
		validator:    NewPathValidator(storePath),
	}
}

// DownloadSkill 下载单个技能
func (d *Downloader) DownloadSkill(detection *DetectionResult) (*types.SkillMetadata, error) {
	// 生成存储路径
	storePath, err := d.validator.ValidateAndSanitize(
		filepath.Join(d.validator.BaseStorePath, detection.Owner, detection.Repo),
		detection.SkillName,
	)
	if err != nil {
		return nil, err
	}

	// 创建目录
	if err := os.MkdirAll(storePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	// 获取技能目录内容
	var skillPath string
	if detection.Path != "" {
		skillPath = detection.Path
	} else {
		skillPath = fmt.Sprintf("skills/%s", detection.SkillName)
	}

	contents, err := d.githubClient.GetRepositoryContents(
		detection.Owner,
		detection.Repo,
		skillPath,
		detection.Branch,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill contents: %v", err)
	}

	// 递归下载所有文件
	if err := d.downloadContentsRecursive(contents, storePath, detection); err != nil {
		return nil, err
	}

	// 构建元数据
	metadata := &types.SkillMetadata{
		ID:        GenerateUniqueID(detection.Owner, detection.Repo, detection.Path),
		Name:      detection.SkillName,
		SourceURL: detection.RawURL,
		StorePath: storePath,
		UpdatedAt: time.Now(),
	}

	return metadata, nil
}

// downloadContentsRecursive 递归下载内容
func (d *Downloader) downloadContentsRecursive(contents []types.GitHubContent, destPath string, detection *DetectionResult) error {
	for _, content := range contents {
		targetPath := filepath.Join(destPath, content.Name)

		// 安全校验
		if _, err := d.validator.ValidateAndSanitize(targetPath, ""); err != nil {
			return fmt.Errorf("invalid path %s: %v", targetPath, err)
		}

		if content.Type == "dir" {
			// 递归处理子目录
			subContents, err := d.githubClient.GetRepositoryContents(
				detection.Owner,
				detection.Repo,
				content.Path,
				detection.Branch,
			)
			if err != nil {
				return err
			}

			// 创建目录
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}

			if err := d.downloadContentsRecursive(subContents, targetPath, detection); err != nil {
				return err
			}
		} else if content.Type == "file" {
			// 下载文件
			if err := d.downloadFile(content.DownloadURL, targetPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// downloadFile 下载单个文件
func (d *Downloader) downloadFile(downloadURL, destPath string) error {
	data, err := d.githubClient.DownloadRawContent(downloadURL)
	if err != nil {
		return err
	}

	// 写入文件
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %v", destPath, err)
	}

	return nil
}

// DownloadBatch 批量下载技能
func (d *Downloader) DownloadBatch(detection *DetectionResult) ([]*types.SkillMetadata, error) {
	// 列出仓库中的所有技能
	skillNames, err := d.githubClient.ListSkillsInRepo(
		detection.Owner,
		detection.Repo,
		detection.Branch,
	)
	if err != nil {
		return nil, err
	}

	var results []*types.SkillMetadata
	for _, skillName := range skillNames {
		// 为每个技能创建检测结果
		skillDetection := &DetectionResult{
			Type:      URLTypeSkillDir,
			Owner:     detection.Owner,
			Repo:      detection.Repo,
			Branch:    detection.Branch,
			Path:      fmt.Sprintf("skills/%s", skillName),
			SkillName: skillName,
			IsGitHub:  detection.IsGitHub,
			RawURL: fmt.Sprintf("%s/tree/%s/skills/%s",
				detection.RawURL, detection.Branch, skillName),
		}

		// 下载技能
		metadata, err := d.DownloadSkill(skillDetection)
		if err != nil {
			fmt.Printf("Warning: Failed to download skill %s: %v\n", skillName, err)
			continue
		}

		results = append(results, metadata)
	}

	return results, nil
}
