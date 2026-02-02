package add

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	defaultTimeout         = 30 * time.Second
	maxRetries             = 3
	retryWaitTime          = 2 * time.Second
	maxConcurrentDownloads = 3
	downloadTimeout        = 5 * time.Minute
	maxRetryAttempts       = 5
)

// DownloadStats contains statistics about download operation.
type DownloadStats struct {
	FilesDownloaded int
	DirsCreated     int
	BytesDownloaded int64
}

// Client is a GitHub API client for downloading skill packages.
type Client struct {
	restyClient *resty.Client
	token       string
	baseURL     string
	logger      Logger
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
		logger:      NoOpLogger{},
	}
}

// Download downloads a skill package from the specified GitHub URL.
// The URL must be in format: https://github.com/owner/repo/tree/branch/path
//
// The function performs the following steps:
// 1. Parses and validates the GitHub URL
// 2. Checks that SKILL.md exists in the target directory
// 3. Prompts the user for confirmation if the download directory already exists
// 4. Downloads all files and directories recursively to a temporary location
// 5. Atomically moves the download to the final location
// 6. Displays download statistics
//
// Returns an error if any step fails, nil on success.
func (c *Client) Download(rawURL string) error {
	repoInfo, err := parseGitHubURL(rawURL)
	if err != nil {
		return &DownloadError{
			Type:    ErrorTypeInvalidURL,
			Message: "failed to parse URL",
			Err:     err,
		}
	}

	c.logger.Debug("Parsed GitHub URL", "owner", repoInfo.Owner, "repo", repoInfo.Repo, "branch", repoInfo.Branch, "path", repoInfo.Path)

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	hasSkillMD, err := c.checkSKILLExists(ctx, repoInfo)
	if err != nil {
		return &DownloadError{
			Type:    ErrorTypeAPI,
			Message: "failed to check SKILL.md",
			Err:     err,
		}
	}
	if !hasSkillMD {
		return &DownloadError{
			Type:    ErrorTypeValidation,
			Message: "SKILL.md not found in the target directory. This is not a valid skill package.",
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &DownloadError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to get home directory",
			Err:     err,
		}
	}

	skillName := filepath.Base(repoInfo.Path)
	if skillName == "." || skillName == "" {
		return &DownloadError{
			Type:    ErrorTypeInvalidURL,
			Message: fmt.Sprintf("invalid skill path: %s", repoInfo.Path),
		}
	}
	localPath := filepath.Join(homeDir, ".gskills", "skills", skillName)

	exists, err := checkPathExists(localPath)
	if err != nil {
		return &DownloadError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to check path existence",
			Err:     err,
		}
	}

	if exists {
		overwrite, err := promptOverwrite()
		if err != nil {
			return &DownloadError{
				Type:    ErrorTypeFilesystem,
				Message: "failed to read user input",
				Err:     err,
			}
		}
		if !overwrite {
			fmt.Println("Download cancelled.")
			c.logger.Info("Download cancelled by user")
			return nil
		}
		if err := os.RemoveAll(localPath); err != nil {
			return &DownloadError{
				Type:    ErrorTypeFilesystem,
				Message: "failed to remove existing directory",
				Err:     err,
			}
		}
	}

	tmpDir := filepath.Join(filepath.Dir(localPath), ".tmp."+filepath.Base(localPath)+fmt.Sprintf(".%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return &DownloadError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to create temporary directory",
			Err:     err,
		}
	}

	c.logger.Debug("Created temporary directory", "path", tmpDir)

	defer func() {
		if err != nil {
			c.logger.Error("Cleaning up temporary directory", err, "path", tmpDir)
			os.RemoveAll(tmpDir)
		}
	}()

	c.logger.Info("Starting download", "url", rawURL, "target", tmpDir)
	fmt.Printf("Downloading skill from %s...\n", rawURL)

	stats, err := c.downloadRecursive(ctx, repoInfo, tmpDir, repoInfo.Path)
	if err != nil {
		return &DownloadError{
			Type:    ErrorTypeAPI,
			Message: "failed to download",
			Err:     err,
		}
	}

	if err := os.RemoveAll(localPath); err != nil {
		return &DownloadError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to remove existing directory for atomic move",
			Err:     err,
		}
	}

	if err := os.Rename(tmpDir, localPath); err != nil {
		return &DownloadError{
			Type:    ErrorTypeFilesystem,
			Message: "failed to move download to final location",
			Err:     err,
		}
	}

	c.logger.Info("Download complete", "files", stats.FilesDownloaded, "bytes", stats.BytesDownloaded)

	fmt.Printf("\nDownload complete!\n")
	fmt.Printf("  Files downloaded: %d\n", stats.FilesDownloaded)
	fmt.Printf("  Directories created: %d\n", stats.DirsCreated)
	fmt.Printf("  Total size: %d bytes\n", stats.BytesDownloaded)
	fmt.Printf("  Location: %s\n", localPath)

	return nil
}

func (c *Client) downloadRecursive(ctx context.Context, repoInfo *GitHubRepoInfo, localPath string, downloadPath string) (*DownloadStats, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stats := &DownloadStats{
		FilesDownloaded: 0,
		DirsCreated:     0,
		BytesDownloaded: 0,
	}

	sem := make(chan struct{}, maxConcurrentDownloads)
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
			cancel()
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
					cancel()
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
					cancel()
					return
				}

				if err := os.WriteFile(itemLocalPath, data, 0644); err != nil {
					mu.Lock()
					downloadErr = fmt.Errorf("failed to write file %s: %w", itemLocalPath, err)
					mu.Unlock()
					cancel()
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
