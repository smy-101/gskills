// Package update provides functionality for checking and updating
// installed skills from their GitHub sources.
//
// The update process works by:
//  1. Comparing commit SHAs to detect available updates
//  2. Downloading updated files to a temporary location
//  3. Atomically replacing the old files with new ones
//  4. Updating the registry with new metadata
//
// Features:
//   - Concurrent update checking with configurable limits
//   - SHA-based update detection to avoid unnecessary downloads
//   - Automatic retry with exponential backoff for rate limits
//   - Preserves linked projects during updates
package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/smy-101/gskills/internal/add"
	"github.com/smy-101/gskills/internal/registry"
	"github.com/smy-101/gskills/internal/types"
)

const (
	checkTimeout           = 30 * time.Second
	updateTimeout          = 5 * time.Minute
	maxRetryAttempt        = 5
	maxConcurrentChecks    = 5 // Limit concurrent API calls to avoid rate limits
	maxConcurrentUpdates   = 3 // Limit concurrent downloads to avoid resource exhaustion
	maxConcurrentDownloads = 3 // Limit concurrent file downloads per skill
)

type UpdateStatus int

const (
	UpdateStatusUpToDate UpdateStatus = iota
	UpdateStatusAvailable
	UpdateStatusFailed
)

type SkillUpdateInfo struct {
	Skill        *types.SkillMetadata
	Status       UpdateStatus
	NewCommitSHA string
	Error        error
}

type Updater struct {
	client *add.Client
	logger add.Logger
}

// UpdateStats contains statistics about bulk update operations.
type UpdateStats struct {
	Total    int
	Updated  int
	Skipped  int
	Failed   int
	Duration time.Duration
}

// NewUpdater creates a new Updater instance with the given GitHub token.
// The token can be empty for public repositories. The updater is configured
// with a 30-second timeout for update checks and 5-minute timeout for downloads.
func NewUpdater(token string) *Updater {
	return &Updater{
		client: add.NewClient(token),
		logger: add.NoOpLogger{},
	}
}

// SetLogger sets the logger for the updater. If no logger is set,
// a NoOpLogger is used which suppresses all log output.
func (u *Updater) SetLogger(logger add.Logger) {
	u.logger = logger
}

// SetBaseURL sets the base URL for GitHub API requests.
// This method is intended for testing purposes only.
func (u *Updater) SetBaseURL(url string) {
	u.client.SetBaseURL(url)
}

// CheckUpdate checks if a skill has an available update by comparing
// the current commit SHA with the latest commit SHA from GitHub.
//
// Returns:
//   - hasUpdate: true if the skill has an update available
//   - newSHA: the latest commit SHA from GitHub
//   - err: any error that occurred during the check
func (u *Updater) CheckUpdate(skill *types.SkillMetadata) (hasUpdate bool, newSHA string, err error) {
	if skill == nil {
		return false, "", fmt.Errorf("skill metadata cannot be nil")
	}
	if skill.SourceURL == "" {
		return false, "", fmt.Errorf("skill source URL cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	repoInfo, err := add.ParseGitHubURL(skill.SourceURL)
	if err != nil {
		return false, "", &UpdateError{
			Type:    UpdateErrorTypeCheck,
			Message: "failed to parse source URL",
			Err:     err,
			Skill:   skill.Name,
		}
	}

	newSHA, err = u.getCommitSHAWithRetry(ctx, repoInfo)
	if err != nil {
		return false, "", &UpdateError{
			Type:    UpdateErrorTypeCheck,
			Message: "failed to fetch latest commit SHA",
			Err:     err,
			Skill:   skill.Name,
		}
	}

	if newSHA == skill.CommitSHA {
		return false, newSHA, nil
	}

	return true, newSHA, nil
}

// getCommitSHAWithRetry fetches the latest commit SHA from GitHub with retry logic
// for handling rate limits. Uses exponential backoff with a maximum of 16 seconds.
func (u *Updater) getCommitSHAWithRetry(ctx context.Context, repoInfo *add.GitHubRepoInfo) (string, error) {
	var lastErr error
	for attempt := range maxRetryAttempt {
		sha, err := u.client.GetBranchCommitSHA(ctx, repoInfo)
		if err == nil {
			return sha, nil
		}

		lastErr = err
		if isRateLimitError(err) && attempt < maxRetryAttempt-1 {
			backoff := min(time.Duration(1<<uint(attempt))*time.Second, 16*time.Second)
			u.logger.Warn("Rate limit hit, backing off", "attempt", attempt+1, "backoff", backoff)

			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}

	return "", lastErr
}

// UpdateSkill checks for updates to a single skill and downloads the latest version
// if an update is available. The update process:
//  1. Checks if an update is available by comparing commit SHAs
//  2. Downloads new files to a temporary directory
//  3. Atomically replaces the old skill directory
//  4. Updates the registry with new metadata
//
// Returns nil if the skill is up to date or if the update succeeds.
func (u *Updater) UpdateSkill(skill *types.SkillMetadata) error {
	if skill == nil {
		return fmt.Errorf("skill metadata cannot be nil")
	}

	hasUpdate, newSHA, err := u.CheckUpdate(skill)
	if err != nil {
		return err
	}

	if !hasUpdate {
		return nil
	}

	return u.downloadAndUpdate(skill, newSHA)
}

// downloadAndUpdate performs the actual download and update of a skill.
// Downloads files to a temporary directory, then atomically moves them
// to the final location.
func (u *Updater) downloadAndUpdate(skill *types.SkillMetadata, newSHA string) error {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()

	repoInfo, err := add.ParseGitHubURL(skill.SourceURL)
	if err != nil {
		return &UpdateError{
			Type:    UpdateErrorTypeDownload,
			Message: "failed to parse source URL",
			Err:     err,
			Skill:   skill.Name,
		}
	}

	localPath := skill.StorePath
	if localPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return &UpdateError{
				Type:    UpdateErrorTypeDownload,
				Message: "failed to get home directory",
				Err:     err,
				Skill:   skill.Name,
			}
		}
		skillName := filepath.Base(repoInfo.Path)
		localPath = filepath.Join(homeDir, ".gskills", "skills", skillName)
	}

	tmpDir := filepath.Join(filepath.Dir(localPath), ".tmp."+filepath.Base(localPath)+fmt.Sprintf(".%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return &UpdateError{
			Type:    UpdateErrorTypeDownload,
			Message: "failed to create temporary directory",
			Err:     err,
			Skill:   skill.Name,
		}
	}

	u.logger.Debug("Created temporary directory", "path", tmpDir)

	defer func() {
		if err != nil {
			u.logger.Error("Cleaning up temporary directory", err, "path", tmpDir)
			os.RemoveAll(tmpDir)
		}
	}()

	u.logger.Info("Starting update", "skill", skill.Name, "target", tmpDir)

	stats, err := u.downloadRecursive(ctx, repoInfo, tmpDir, repoInfo.Path)
	if err != nil {
		return &UpdateError{
			Type:    UpdateErrorTypeDownload,
			Message: "failed to download files",
			Err:     err,
			Skill:   skill.Name,
		}
	}

	if err := os.RemoveAll(localPath); err != nil {
		return &UpdateError{
			Type:    UpdateErrorTypeDownload,
			Message: "failed to remove existing directory",
			Err:     err,
			Skill:   skill.Name,
		}
	}

	if err := os.Rename(tmpDir, localPath); err != nil {
		return &UpdateError{
			Type:    UpdateErrorTypeDownload,
			Message: "failed to move files to final location",
			Err:     err,
			Skill:   skill.Name,
		}
	}

	u.logger.Info("Update complete", "skill", skill.Name, "files", stats.FilesDownloaded)

	updatedSkill := *skill
	updatedSkill.CommitSHA = newSHA
	updatedSkill.UpdatedAt = time.Now()

	if err := registry.UpdateSkill(&updatedSkill); err != nil {
		return &UpdateError{
			Type:    UpdateErrorTypeRegistry,
			Message: "failed to update registry",
			Err:     err,
			Skill:   skill.Name,
		}
	}

	return nil
}

// CheckAllUpdates checks all installed skills for available updates concurrently.
// Returns a slice of SkillUpdateInfo with the status of each skill.
//
// The function uses concurrency to check multiple skills simultaneously,
// with a limit of maxConcurrentChecks (5) concurrent operations.
func (u *Updater) CheckAllUpdates() ([]SkillUpdateInfo, error) {
	skills, err := registry.LoadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	if len(skills) == 0 {
		return []SkillUpdateInfo{}, nil
	}

	results := make([]SkillUpdateInfo, len(skills))
	var wg sync.WaitGroup
	var mu sync.Mutex

	sem := make(chan struct{}, maxConcurrentChecks)

	for i, skill := range skills {
		wg.Add(1)
		go func(idx int, s *types.SkillMetadata) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			hasUpdate, newSHA, err := u.CheckUpdate(s)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results[idx] = SkillUpdateInfo{
					Skill:  s,
					Status: UpdateStatusFailed,
					Error:  err,
				}
				return
			}

			if hasUpdate {
				results[idx] = SkillUpdateInfo{
					Skill:        s,
					Status:       UpdateStatusAvailable,
					NewCommitSHA: newSHA,
				}
			} else {
				results[idx] = SkillUpdateInfo{
					Skill:  s,
					Status: UpdateStatusUpToDate,
				}
			}
		}(i, &skill)
	}

	wg.Wait()
	return results, nil
}

// UpdateAll updates multiple skills concurrently and returns statistics
// about the operation. Skills are updated with a limit of maxConcurrentUpdates (3)
// concurrent operations to avoid resource exhaustion.
//
// Parameters:
//   - skillsToUpdate: slice of skill metadata to update
//
// Returns:
//   - UpdateStats: statistics about the update operation
//   - error: any error that occurred during the update process
func (u *Updater) UpdateAll(skillsToUpdate []*types.SkillMetadata) (*UpdateStats, error) {
	if skillsToUpdate == nil {
		return &UpdateStats{}, nil
	}
	startTime := time.Now()
	stats := &UpdateStats{
		Total: len(skillsToUpdate),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	sem := make(chan struct{}, maxConcurrentUpdates)

	for _, skill := range skillsToUpdate {
		wg.Add(1)
		go func(s *types.SkillMetadata) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			err := u.UpdateSkill(s)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				stats.Failed++
				u.logger.Error("Failed to update skill", err, "skill", s.Name)
			} else {
				stats.Updated++
			}
		}(skill)
	}

	wg.Wait()
	stats.Duration = time.Since(startTime)

	return stats, nil
}

// downloadRecursive recursively downloads files and directories from GitHub.
// Uses a worker pool pattern with maxConcurrentDownloads (3) concurrent downloads.
func (u *Updater) downloadRecursive(ctx context.Context, repoInfo *add.GitHubRepoInfo, localPath string, downloadPath string) (*add.DownloadStats, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stats := &add.DownloadStats{
		FilesDownloaded: 0,
		DirsCreated:     0,
		BytesDownloaded: 0,
	}

	sem := make(chan struct{}, maxConcurrentDownloads)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var downloadErr error

	var downloadTaskFunc func(string, string)
	downloadTaskFunc = func(remotePath, localTarget string) {
		defer wg.Done()

		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
			defer func() { <-sem }()
		}

		contents, err := u.client.GetGitHubContents(ctx, repoInfo, remotePath)
		if err != nil {
			mu.Lock()
			downloadErr = fmt.Errorf("failed to get contents for %s: %w", remotePath, err)
			mu.Unlock()
			cancel()
			return
		}

		for _, item := range contents {
			itemLocalPath := filepath.Join(localTarget, item.Name)

			if item.Type == "dir" {
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
				go downloadTaskFunc(item.Path, itemLocalPath)
			} else if item.Type == "file" {
				data, err := u.client.DownloadFile(ctx, item.DownloadURL)
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
	go downloadTaskFunc(downloadPath, localPath)
	wg.Wait()

	if downloadErr != nil {
		return nil, downloadErr
	}

	return stats, nil
}

// isRateLimitError checks if an error is related to GitHub API rate limiting.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "rate limit exceeded")
}
