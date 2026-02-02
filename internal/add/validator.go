package add

import (
	"context"
	"fmt"
	"path/filepath"
)

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
