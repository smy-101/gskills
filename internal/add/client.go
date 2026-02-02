package add

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/smy-101/gskills/internal/types"
)

// Client GitHub API客户端
type Client struct {
	restyClient *resty.Client
	token       string
}

// NewClient 创建客户端
func NewClient(token string) *Client {
	client := resty.New()

	// 配置
	client.SetTimeout(30 * time.Second)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(2 * time.Second)

	// 设置认证头
	if token != "" {
		client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	// 设置User-Agent
	client.SetHeader("User-Agent", "gskills-cli/1.0")

	return &Client{
		restyClient: client,
		token:       token,
	}
}

// Download 根据URL下载资源
func (c *Client) Download(rawURL string) error {
	//1.先获取到仓库文件列表 使用GetRepositoryContents方法
	detectRes, err := DetectURL(rawURL)
	if err != nil {
		return fmt.Errorf("failed to detect URL: %v", err)
	}
	githubContent, err := c.GetRepositoryContents(detectRes.Owner, detectRes.Repo, detectRes.Path, detectRes.Branch)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", githubContent)

	// c.GetRepositoryContents(rawURL)
	//2.根据获取的文件列表 来判断是单一的skill文件夹还是整个仓库  选择不同的下载逻辑

	//todo 下载操作
	return nil
}

// GetRepositoryContents 获取仓库内容
func (c *Client) GetRepositoryContents(owner, repo, path, ref string) ([]types.GitHubContent, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)

	if ref != "" {
		url += "?ref=" + ref
	}

	var contents []types.GitHubContent
	resp, err := c.restyClient.R().
		SetResult(&contents).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("API request failed: %v", err)
	}

	if resp.StatusCode() == http.StatusForbidden {
		// API限流
		if strings.Contains(resp.String(), "API rate limit exceeded") {
			return nil, fmt.Errorf("API rate limit exceeded. Please configure a GitHub Token via 'gskills config set github_token <token>'")
		}
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode(), resp.String())
	}

	return contents, nil
}

// DownloadRawContent 下载原始内容
func (c *Client) DownloadRawContent(downloadURL string) ([]byte, error) {
	resp, err := c.restyClient.R().
		Get(downloadURL)

	if err != nil {
		return nil, fmt.Errorf("download failed: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("download returned %d", resp.StatusCode())
	}

	return resp.Body(), nil
}

// ListSkillsInRepo 列出仓库中的技能
func (c *Client) ListSkillsInRepo(owner, repo, ref string) ([]string, error) {
	// 检查是否存在skills目录
	contents, err := c.GetRepositoryContents(owner, repo, "skills", ref)
	if err != nil {
		// 可能没有skills目录
		return nil, fmt.Errorf("no skills directory found: %v", err)
	}

	var skillDirs []string
	for _, content := range contents {
		if content.Type == "dir" {
			// 进一步检查是否为技能目录
			skillContents, err := c.GetRepositoryContents(owner, repo, content.Path, ref)
			if err == nil {
				var skillNames []string
				for _, skillContent := range skillContents {
					skillNames = append(skillNames, skillContent.Name)
				}
				if IsSkillDirectory(skillNames) {
					skillDirs = append(skillDirs, content.Name)
				}
			}
		}
	}

	return skillDirs, nil
}
