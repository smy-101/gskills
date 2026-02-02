// Package add provides functionality to download skill packages from GitHub repositories.
// It supports downloading individual skill directories that contain a SKILL.md file.
package add

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/smy-101/gskills/internal/types"
)

const (
	defaultTimeout         = 30 * time.Second
	maxRetries             = 3
	retryWaitTime          = 2 * time.Second
	maxConcurrentDownloads = 3
	downloadTimeout        = 5 * time.Minute
)

// Client is a GitHub API client for downloading skill packages.
type Client struct {
	restyClient *resty.Client
	token       string
	baseURL     string
}

// NewClient creates a new GitHub API client with the given authentication token.
// The token can be empty for public repositories.
// The client is configured with a 30-second timeout, 3 retries, and 2-second retry wait time.
func NewClient(token string) *Client {
	client := resty.New()

	client.SetTimeout(defaultTimeout)
	client.SetRetryCount(maxRetries)
	client.SetRetryWaitTime(retryWaitTime)

	if token != "" {
		client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client.SetHeader("User-Agent", "gskills-cli/1.0")

	return &Client{
		restyClient: client,
		token:       token,
		baseURL:     "https://api.github.com",
	}
}

// GitHubRepoInfo contains parsed information from a GitHub repository URL.
type GitHubRepoInfo struct {
	Owner  string
	Repo   string
	Branch string
	Path   string
}

// DownloadStats contains statistics about the download operation.
type DownloadStats struct {
	FilesDownloaded int
	DirsCreated     int
	BytesDownloaded int64
}

// Download downloads a skill package from the specified GitHub URL.
// The URL must be in the format: https://github.com/owner/repo/tree/branch/path
//
// The function performs the following steps:
// 1. Parses and validates the GitHub URL
// 2. Checks that SKILL.md exists in the target directory
// 3. Prompts user for confirmation if the download directory already exists
// 4. Downloads all files and directories recursively
// 5. Displays download statistics
//
// Returns an error if any step fails, nil on success.
func (c *Client) Download(rawURL string) error {
	repoInfo, err := parseGitHubURL(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	hasSkillMD, err := c.checkSKILLExists(ctx, repoInfo)
	if err != nil {
		return fmt.Errorf("failed to check SKILL.md: %w", err)
	}
	if !hasSkillMD {
		return fmt.Errorf("SKILL.md not found in the target directory. This is not a valid skill package.")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	localPath := filepath.Join(homeDir, ".gskills", "skills", filepath.Base(repoInfo.Path))

	exists, err := checkPathExists(localPath)
	if err != nil {
		return fmt.Errorf("failed to check path existence: %w", err)
	}
	if exists {
		overwrite, err := promptOverwrite()
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}
		if !overwrite {
			fmt.Println("Download cancelled.")
			return nil
		}
		if err := os.RemoveAll(localPath); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	fmt.Printf("Downloading skill from %s...\n", rawURL)
	stats, err := c.downloadRecursive(ctx, repoInfo, localPath, repoInfo.Path)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	fmt.Printf("\nDownload complete!\n")
	fmt.Printf("  Files downloaded: %d\n", stats.FilesDownloaded)
	fmt.Printf("  Directories created: %d\n", stats.DirsCreated)
	fmt.Printf("  Total size: %d bytes\n", stats.BytesDownloaded)
	fmt.Printf("  Location: %s\n", localPath)

	return nil
}

func parseGitHubURL(rawURL string) (*GitHubRepoInfo, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Host != "github.com" {
		return nil, fmt.Errorf("only GitHub URLs are supported")
	}

	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL format")
	}

	owner := pathParts[0]
	repo := pathParts[1]

	if owner == "" {
		return nil, fmt.Errorf("owner cannot be empty in URL")
	}
	if repo == "" {
		return nil, fmt.Errorf("repo cannot be empty in URL")
	}

	var branch, path string

	if len(pathParts) >= 4 && pathParts[2] == "tree" {
		branch = pathParts[3]
		if len(pathParts) > 4 {
			path = filepath.Join(pathParts[4:]...)
		}
	} else if len(pathParts) >= 3 {
		return nil, fmt.Errorf("branch must be specified in URL (use format: https://github.com/owner/repo/tree/branch/path)")
	} else {
		return nil, fmt.Errorf("branch must be specified in URL (use format: https://github.com/owner/repo/tree/branch/path)")
	}

	if path == "" {
		return nil, fmt.Errorf("path must be specified in URL")
	}
	if branch == "" {
		return nil, fmt.Errorf("branch cannot be empty in URL")
	}

	return &GitHubRepoInfo{
		Owner:  owner,
		Repo:   repo,
		Branch: branch,
		Path:   path,
	}, nil
}

func (c *Client) checkSKILLExists(ctx context.Context, repoInfo *GitHubRepoInfo) (bool, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", c.baseURL, repoInfo.Owner, repoInfo.Repo, filepath.Join(repoInfo.Path, "SKILL.md"), repoInfo.Branch)

	resp, err := c.restyClient.R().SetContext(ctx).Get(apiURL)
	if err != nil {
		return false, fmt.Errorf("failed to check SKILL.md: %w", err)
	}

	if resp.StatusCode() == 404 {
		return false, nil
	}

	if resp.StatusCode() != 200 {
		return false, fmt.Errorf("GitHub API returned status %d", resp.StatusCode())
	}

	return true, nil
}

func checkPathExists(localPath string) (bool, error) {
	_, err := os.Stat(localPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func promptOverwrite() (bool, error) {
	fmt.Print("Target path already exists. Overwrite? [y/N]: ")

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

func (c *Client) downloadRecursive(ctx context.Context, repoInfo *GitHubRepoInfo, localPath string, downloadPath string) (*DownloadStats, error) {
	stats := &DownloadStats{
		FilesDownloaded: 0,
		DirsCreated:     0,
		BytesDownloaded: 0,
	}

	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var downloadErr error

	var downloadTask func(string, string)
	downloadTask = func(remotePath, localTarget string) {
		defer wg.Done()

		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
			defer func() { <-sem }()
		}

		contents, err := c.getGitHubContents(ctx, repoInfo, remotePath)
		if err != nil {
			mu.Lock()
			downloadErr = fmt.Errorf("failed to get contents for %s: %w", remotePath, err)
			mu.Unlock()
			return
		}

		for _, item := range contents {
			itemLocalPath := filepath.Join(localTarget, item.Name)

			switch item.Type {
			case "dir":
				if err := os.MkdirAll(itemLocalPath, 0755); err != nil {
					mu.Lock()
					downloadErr = fmt.Errorf("failed to create directory %s: %w", itemLocalPath, err)
					mu.Unlock()
					return
				}

				mu.Lock()
				stats.DirsCreated++
				mu.Unlock()

				wg.Add(1)
				go downloadTask(filepath.Join(remotePath, item.Name), itemLocalPath)
			case "file":
				data, err := c.downloadFile(ctx, item.DownloadURL)
				if err != nil {
					mu.Lock()
					downloadErr = fmt.Errorf("failed to download file %s: %w", item.Name, err)
					mu.Unlock()
					return
				}

				if err := os.WriteFile(itemLocalPath, data, 0644); err != nil {
					mu.Lock()
					downloadErr = fmt.Errorf("failed to write file %s: %w", itemLocalPath, err)
					mu.Unlock()
					return
				}

				mu.Lock()
				stats.FilesDownloaded++
				stats.BytesDownloaded += int64(len(data))
				mu.Unlock()
			}
		}
	}

	wg.Add(1)
	go downloadTask(downloadPath, localPath)
	wg.Wait()

	if downloadErr != nil {
		return nil, downloadErr
	}

	return stats, nil
}

func (c *Client) getGitHubContents(ctx context.Context, repoInfo *GitHubRepoInfo, path string) ([]types.GitHubContent, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", c.baseURL, repoInfo.Owner, repoInfo.Repo, path, repoInfo.Branch)

	resp, err := c.restyClient.R().SetContext(ctx).Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get contents: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d for path %s", resp.StatusCode(), path)
	}

	var contents []types.GitHubContent
	if err := json.Unmarshal(resp.Body(), &contents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return contents, nil
}

func (c *Client) downloadFile(ctx context.Context, downloadURL string) ([]byte, error) {
	resp, err := c.restyClient.R().SetContext(ctx).Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode())
	}

	return resp.Body(), nil
}
