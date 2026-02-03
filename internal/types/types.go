package types

import "time"

// SkillMetadata 技能元数据
type SkillMetadata struct {
	ID             string                       `json:"id"`
	Name           string                       `json:"name"`
	SourceURL      string                       `json:"source_url"`
	StorePath      string                       `json:"store_path"`
	UpdatedAt      time.Time                    `json:"updated_at"`
	Version        string                       `json:"version,omitempty"`
	Description    string                       `json:"description,omitempty"`
	LinkedProjects map[string]LinkedProjectInfo `json:"linked_projects,omitempty"`
}

// LinkedProjectInfo tracks where a skill is linked
type LinkedProjectInfo struct {
	SymlinkPath string    `json:"symlink_path"`
	LinkedAt    time.Time `json:"linked_at"`
}

// GitHubContent GitHub API返回的内容项
type GitHubContent struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
}
