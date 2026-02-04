package add

import (
	"fmt"
	"net/url"
	pathpkg "path"
	"strings"
)

// GitHubRepoInfo contains parsed information from a GitHub repository URL.
type GitHubRepoInfo struct {
	Owner  string
	Repo   string
	Branch string
	Path   string
}

func ParseGitHubURL(rawURL string) (*GitHubRepoInfo, error) {
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
			path = pathpkg.Join(pathParts[4:]...)
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
