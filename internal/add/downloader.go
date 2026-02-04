package add

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/smy-101/gskills/internal/types"
)

func isRateLimitResponse(statusCode int) bool {
	return statusCode == 403 || statusCode == 429
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "403") || strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit exceeded")
}

func (c *Client) getBranchCommitSHA(ctx context.Context, repoInfo *GitHubRepoInfo) (string, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/commits/%s", c.baseURL, repoInfo.Owner, repoInfo.Repo, repoInfo.Branch)

	var lastErr error
	for attempt := range maxRetryAttempts {
		resp, err := c.restyClient.R().SetContext(ctx).Get(apiURL)
		if err != nil {
			if isRateLimitError(err) && attempt < maxRetryAttempts-1 {
				backoff := min(time.Duration(1<<uint(attempt))*time.Second, 16*time.Second)

				c.logger.Warn("Rate limit hit, backing off", "attempt", attempt+1, "backoff", backoff)

				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
			lastErr = err
			continue
		}

		if resp.StatusCode() != 200 {
			if isRateLimitResponse(resp.StatusCode()) && attempt < maxRetryAttempts-1 {
				backoff := min(time.Duration(1<<uint(attempt))*time.Second, 16*time.Second)

				c.logger.Warn("Rate limit hit, backing off", "attempt", attempt+1, "backoff", backoff)

				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
			lastErr = fmt.Errorf("GitHub API returned status %d for commit SHA", resp.StatusCode())
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body(), &result); err != nil {
			return "", fmt.Errorf("failed to unmarshal commit response: %w", err)
		}

		sha, ok := result["sha"].(string)
		if !ok || sha == "" {
			return "", fmt.Errorf("commit SHA not found in response")
		}

		return sha, nil
	}

	return "", lastErr
}

func (c *Client) getGitHubContents(ctx context.Context, repoInfo *GitHubRepoInfo, path string) ([]types.GitHubContent, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", c.baseURL, repoInfo.Owner, repoInfo.Repo, path, repoInfo.Branch)

	var lastErr error
	for attempt := range maxRetryAttempts {
		resp, err := c.restyClient.R().SetContext(ctx).Get(apiURL)
		if err != nil {
			if isRateLimitError(err) && attempt < maxRetryAttempts-1 {
				backoff := min(time.Duration(1<<uint(attempt))*time.Second, 16*time.Second)

				c.logger.Warn("Rate limit hit, backing off", "attempt", attempt+1, "backoff", backoff)

				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			lastErr = err
			continue
		}

		if resp.StatusCode() != 200 {
			if isRateLimitResponse(resp.StatusCode()) && attempt < maxRetryAttempts-1 {
				backoff := min(time.Duration(1<<uint(attempt))*time.Second, 16*time.Second)

				c.logger.Warn("Rate limit hit, backing off", "attempt", attempt+1, "backoff", backoff)

				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			lastErr = fmt.Errorf("GitHub API returned status %d for path %s", resp.StatusCode(), path)
			continue
		}

		var contents []types.GitHubContent
		if err := json.Unmarshal(resp.Body(), &contents); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		return contents, nil
	}

	return nil, lastErr
}

func (c *Client) downloadFile(ctx context.Context, downloadURL string) ([]byte, error) {
	var lastErr error
	for attempt := range maxRetryAttempts {
		resp, err := c.restyClient.R().SetContext(ctx).Get(downloadURL)
		if err != nil {
			if isRateLimitError(err) && attempt < maxRetryAttempts-1 {
				backoff := min(time.Duration(1<<uint(attempt))*time.Second, 16*time.Second)

				c.logger.Warn("Rate limit hit, backing off", "attempt", attempt+1, "backoff", backoff)

				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			lastErr = err
			continue
		}

		if resp.StatusCode() != 200 {
			if isRateLimitResponse(resp.StatusCode()) && attempt < maxRetryAttempts-1 {
				backoff := min(time.Duration(1<<uint(attempt))*time.Second, 16*time.Second)

				c.logger.Warn("Rate limit hit, backing off", "attempt", attempt+1, "backoff", backoff)

				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			lastErr = fmt.Errorf("download failed with status %d", resp.StatusCode())
			continue
		}

		return resp.Body(), nil
	}

	return nil, lastErr
}
