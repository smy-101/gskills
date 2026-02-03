package add

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/smy-101/gskills/internal/types"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    *GitHubRepoInfo
		wantErr bool
		errMsg  string
	}{
		{
			name:   "valid URL with branch and path",
			rawURL: "https://github.com/owner/repo/tree/main/path/to/skill",
			want: &GitHubRepoInfo{
				Owner:  "owner",
				Repo:   "repo",
				Branch: "main",
				Path:   "path/to/skill",
			},
			wantErr: false,
		},
		{
			name:   "valid URL with branch and single path",
			rawURL: "https://github.com/owner/repo/tree/develop/skill",
			want: &GitHubRepoInfo{
				Owner:  "owner",
				Repo:   "repo",
				Branch: "develop",
				Path:   "skill",
			},
			wantErr: false,
		},
		{
			name:    "invalid URL - missing branch",
			rawURL:  "https://github.com/owner/repo/path",
			wantErr: true,
			errMsg:  "branch must be specified",
		},
		{
			name:    "invalid URL - non-github host",
			rawURL:  "https://gitlab.com/owner/repo/tree/main/path",
			wantErr: true,
			errMsg:  "only GitHub URLs are supported",
		},
		{
			name:    "invalid URL format",
			rawURL:  "not-a-url",
			wantErr: true,
			errMsg:  "only GitHub URLs",
		},
		{
			name:    "invalid URL - missing path",
			rawURL:  "https://github.com/owner/repo/tree/main",
			wantErr: true,
			errMsg:  "path must be specified",
		},
		{
			name:    "invalid URL - empty owner",
			rawURL:  "https://github.com//repo/tree/main/path",
			wantErr: true,
			errMsg:  "branch must be specified",
		},
		{
			name:    "invalid URL - empty repo",
			rawURL:  "https://github.com/owner//tree/main/path",
			wantErr: true,
			errMsg:  "repo cannot be empty",
		},
		{
			name:    "invalid URL - empty branch",
			rawURL:  "https://github.com/owner/repo/tree//path",
			wantErr: true,
			errMsg:  "branch cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGitHubURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGitHubURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseGitHubURL() expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("parseGitHubURL() error = %v, expected to contain %q", err, tt.errMsg)
				}
				return
			}
			if got == nil {
				t.Errorf("parseGitHubURL() got nil, want %+v", tt.want)
				return
			}
			if got.Owner != tt.want.Owner {
				t.Errorf("parseGitHubURL().Owner = %v, want %v", got.Owner, tt.want.Owner)
			}
			if got.Repo != tt.want.Repo {
				t.Errorf("parseGitHubURL().Repo = %v, want %v", got.Repo, tt.want.Repo)
			}
			if got.Branch != tt.want.Branch {
				t.Errorf("parseGitHubURL().Branch = %v, want %v", got.Branch, tt.want.Branch)
			}
			if got.Path != tt.want.Path {
				t.Errorf("parseGitHubURL().Path = %v, want %v", got.Path, tt.want.Path)
			}
		})
	}
}

func TestCheckPathExists(t *testing.T) {
	tmpDir := t.TempDir()

	existingDir := filepath.Join(tmpDir, "existing")
	if err := os.Mkdir(existingDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	existingFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantExist bool
		wantErr   bool
	}{
		{
			name:      "existing directory",
			path:      existingDir,
			wantExist: true,
			wantErr:   false,
		},
		{
			name:      "existing file",
			path:      existingFile,
			wantExist: true,
			wantErr:   false,
		},
		{
			name:      "non-existent path",
			path:      filepath.Join(tmpDir, "nonexistent"),
			wantExist: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPathExists(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPathExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantExist {
				t.Errorf("checkPathExists() = %v, want %v", got, tt.wantExist)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "client with token",
			token: "test-token-123",
		},
		{
			name:  "client without token",
			token: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.token)
			if client == nil {
				t.Fatal("NewClient() returned nil")
			}
			if client.restyClient == nil {
				t.Error("NewClient() restyClient is nil")
			}
			if client.token != tt.token {
				t.Errorf("NewClient() token = %v, want %v", client.token, tt.token)
			}
		})
	}
}

func TestCheckSKILLExists(t *testing.T) {
	tests := []struct {
		name       string
		mockStatus int
		mockBody   string
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "SKILL.md exists",
			mockStatus: 200,
			mockBody:   `{"name":"SKILL.md","type":"file","path":"SKILL.md"}`,
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "SKILL.md does not exist",
			mockStatus: 404,
			mockBody:   `{"message":"Not Found"}`,
			wantExists: false,
			wantErr:    false,
		},
		{
			name:       "API error",
			mockStatus: 500,
			mockBody:   `{"message":"Internal Server Error"}`,
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			client := NewClient("")
			client.baseURL = server.URL

			repoInfo := &GitHubRepoInfo{
				Owner:  "test-owner",
				Repo:   "test-repo",
				Branch: "main",
				Path:   "skill",
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			got, err := client.checkSKILLExists(ctx, repoInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSKILLExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantExists {
				t.Errorf("checkSKILLExists() = %v, want %v", got, tt.wantExists)
			}
		})
	}
}

func TestGetGitHubContents(t *testing.T) {
	mockContents := []types.GitHubContent{
		{
			Type:        "file",
			Name:        "file1.txt",
			Path:        "file1.txt",
			DownloadURL: "https://example.com/file1.txt",
		},
		{
			Type: "dir",
			Name: "subdir",
			Path: "subdir",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)

		contentsJSON := `[
			{"type":"file","name":"file1.txt","path":"file1.txt","download_url":"https://example.com/file1.txt"},
			{"type":"dir","name":"subdir","path":"subdir"}
		]`
		w.Write([]byte(contentsJSON))
	}))
	defer server.Close()

	client := NewClient("")
	client.baseURL = server.URL

	repoInfo := &GitHubRepoInfo{
		Owner:  "test-owner",
		Repo:   "test-repo",
		Branch: "main",
		Path:   "skill",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := client.getGitHubContents(ctx, repoInfo, "skill")
	if err != nil {
		t.Fatalf("getGitHubContents() error = %v", err)
	}

	if len(got) != len(mockContents) {
		t.Errorf("getGitHubContents() returned %d items, want %d", len(got), len(mockContents))
	}

	for i, item := range got {
		if item.Type != mockContents[i].Type {
			t.Errorf("getGitHubContents()[%d].Type = %v, want %v", i, item.Type, mockContents[i].Type)
		}
		if item.Name != mockContents[i].Name {
			t.Errorf("getGitHubContents()[%d].Name = %v, want %v", i, item.Name, mockContents[i].Name)
		}
	}
}

func TestDownloadFile(t *testing.T) {
	testData := "test file content"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(testData))
	}))
	defer server.Close()

	client := NewClient("")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := client.downloadFile(ctx, server.URL)
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}

	if string(got) != testData {
		t.Errorf("downloadFile() = %v, want %v", string(got), testData)
	}
}

func TestDownloadFileError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewClient("")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.downloadFile(ctx, server.URL)
	if err == nil {
		t.Error("downloadFile() expected error, got nil")
	}
}

func TestDownloadRecursive(t *testing.T) {
	var server *httptest.Server

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "SKILL.md") {
			w.WriteHeader(200)
			w.Write([]byte(`{"name":"SKILL.md","type":"file","path":"SKILL.md","download_url":"` + server.URL + `/skill/SKILL.md"}`))
			return
		}

		if r.URL.Path == "/repos/test-owner/test-repo/contents/skill" {
			w.WriteHeader(200)
			contentsJSON := `[
				{"type":"file","name":"file1.txt","path":"file1.txt","download_url":"` + server.URL + `/skill/file1.txt"},
				{"type":"file","name":"SKILL.md","path":"SKILL.md","download_url":"` + server.URL + `/skill/SKILL.md"},
				{"type":"dir","name":"subdir","path":"subdir"}
			]`
			w.Write([]byte(contentsJSON))
			return
		}

		if strings.Contains(r.URL.Path, "subdir") {
			w.WriteHeader(200)
			contentsJSON := `[
				{"type":"file","name":"file2.txt","path":"subdir/file2.txt","download_url":"` + server.URL + `/skill/subdir/file2.txt"}
			]`
			w.Write([]byte(contentsJSON))
			return
		}

		if strings.HasSuffix(r.URL.Path, "file1.txt") || strings.HasSuffix(r.URL.Path, "file2.txt") || strings.HasSuffix(r.URL.Path, "SKILL.md") {
			w.WriteHeader(200)
			w.Write([]byte(fmt.Sprintf("content of %s", r.URL.Path)))
			return
		}

		w.WriteHeader(404)
	})

	server = httptest.NewServer(handler)
	defer server.Close()

	tmpDir := t.TempDir()

	client := NewClient("")
	client.baseURL = server.URL

	repoInfo := &GitHubRepoInfo{
		Owner:  "test-owner",
		Repo:   "test-repo",
		Branch: "main",
		Path:   "skill",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := client.downloadRecursive(ctx, repoInfo, tmpDir, "skill")
	if err != nil {
		t.Fatalf("downloadRecursive() error = %v", err)
	}

	if stats.FilesDownloaded != 3 {
		t.Errorf("downloadRecursive() FilesDownloaded = %d, want 3", stats.FilesDownloaded)
	}

	if stats.DirsCreated != 1 {
		t.Errorf("downloadRecursive() DirsCreated = %d, want 1", stats.DirsCreated)
	}

	if stats.BytesDownloaded == 0 {
		t.Errorf("downloadRecursive() BytesDownloaded = 0, want > 0")
	}

	file1Path := filepath.Join(tmpDir, "file1.txt")
	if _, err := os.Stat(file1Path); os.IsNotExist(err) {
		t.Errorf("downloadRecursive() file1.txt was not downloaded")
	}

	skillMDPath := filepath.Join(tmpDir, "SKILL.md")
	if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
		t.Errorf("downloadRecursive() SKILL.md was not downloaded")
	}

	subdirPath := filepath.Join(tmpDir, "subdir", "file2.txt")
	if _, err := os.Stat(subdirPath); os.IsNotExist(err) {
		t.Errorf("downloadRecursive() subdir/file2.txt was not downloaded")
	}
}

func BenchmarkParseGitHubURL(b *testing.B) {
	rawURL := "https://github.com/owner/repo/tree/main/path/to/skill"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseGitHubURL(rawURL)
	}
}

func BenchmarkCheckPathExists(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = checkPathExists(tmpDir)
	}
}

func TestGetRegistryPath(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "get registry path",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRegistryPath()
			if (err != nil) != tt.wantErr {
				t.Errorf("getRegistryPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == "" {
					t.Error("getRegistryPath() returned empty path")
				}
				if !filepath.IsAbs(got) {
					t.Errorf("getRegistryPath() returned relative path: %s", got)
				}
			}
		})
	}
}

func TestLoadRegistry(t *testing.T) {
	tests := []struct {
		name       string
		setupFile  func(t *testing.T, registryPath string)
		wantErr    bool
		wantCount  int
		wantSkills map[string]types.SkillMetadata
	}{
		{
			name: "file doesn't exist",
			setupFile: func(t *testing.T, registryPath string) {
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "file exists with valid data",
			setupFile: func(t *testing.T, registryPath string) {
				skills := []types.SkillMetadata{
					{
						ID:        "test-skill@main",
						Name:      "test-skill",
						Version:   "main",
						SourceURL: "https://github.com/test/repo/tree/main/test-skill",
						StorePath: "/home/test/.gskills/skills/test-skill",
					},
				}
				if err := SaveRegistryWithPath(registryPath, skills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			wantErr:   false,
			wantCount: 1,
			wantSkills: map[string]types.SkillMetadata{
				"test-skill@main": {
					ID:        "test-skill@main",
					Name:      "test-skill",
					Version:   "main",
					SourceURL: "https://github.com/test/repo/tree/main/test-skill",
					StorePath: "/home/test/.gskills/skills/test-skill",
				},
			},
		},
		{
			name: "file exists with multiple skills",
			setupFile: func(t *testing.T, registryPath string) {
				skills := []types.SkillMetadata{
					{
						ID:        "skill-a@main",
						Name:      "skill-a",
						Version:   "main",
						SourceURL: "https://github.com/test/repo/tree/main/skill-a",
						StorePath: "/home/test/.gskills/skills/skill-a",
					},
					{
						ID:        "skill-b@v1.0.0",
						Name:      "skill-b",
						Version:   "v1.0.0",
						SourceURL: "https://github.com/test/repo/tree/v1.0.0/skill-b",
						StorePath: "/home/test/.gskills/skills/skill-b",
					},
				}
				if err := SaveRegistryWithPath(registryPath, skills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "file exists with malformed JSON",
			setupFile: func(t *testing.T, registryPath string) {
				registryDir := filepath.Dir(registryPath)
				if err := os.MkdirAll(registryDir, 0755); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				if err := os.WriteFile(registryPath, []byte("{invalid json"), 0644); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			registryPath := filepath.Join(tmpDir, "skills.json")
			tt.setupFile(t, registryPath)

			skills, err := loadRegistryWithPath(registryPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadRegistryWithPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(skills) != tt.wantCount {
					t.Errorf("loadRegistryWithPath() returned %d skills, want %d", len(skills), tt.wantCount)
				}
				for id, wantSkill := range tt.wantSkills {
					found := false
					for _, skill := range skills {
						if skill.ID == id {
							found = true
							if skill.Name != wantSkill.Name {
								t.Errorf("loadRegistryWithPath()[%s].Name = %v, want %v", id, skill.Name, wantSkill.Name)
							}
							if skill.Version != wantSkill.Version {
								t.Errorf("loadRegistryWithPath()[%s].Version = %v, want %v", id, skill.Version, wantSkill.Version)
							}
							if skill.SourceURL != wantSkill.SourceURL {
								t.Errorf("loadRegistryWithPath()[%s].SourceURL = %v, want %v", id, skill.SourceURL, wantSkill.SourceURL)
							}
							if skill.StorePath != wantSkill.StorePath {
								t.Errorf("loadRegistryWithPath()[%s].StorePath = %v, want %v", id, skill.StorePath, wantSkill.StorePath)
							}
							break
						}
					}
					if !found {
						t.Errorf("loadRegistryWithPath() skill %s not found", id)
					}
				}
			}
		})
	}
}

func TestSaveRegistry(t *testing.T) {
	tests := []struct {
		name      string
		skills    []types.SkillMetadata
		wantErr   bool
		setupFile func(t *testing.T, registryPath string)
	}{
		{
			name:    "save empty registry",
			skills:  []types.SkillMetadata{},
			wantErr: false,
		},
		{
			name: "save single skill",
			skills: []types.SkillMetadata{
				{
					ID:        "test-skill@main",
					Name:      "test-skill",
					Version:   "main",
					SourceURL: "https://github.com/test/repo/tree/main/test-skill",
					StorePath: "/home/test/.gskills/skills/test-skill",
				},
			},
			wantErr: false,
		},
		{
			name: "save multiple skills",
			skills: []types.SkillMetadata{
				{
					ID:        "skill-a@main",
					Name:      "skill-a",
					Version:   "main",
					SourceURL: "https://github.com/test/repo/tree/main/skill-a",
					StorePath: "/home/test/.gskills/skills/skill-a",
				},
				{
					ID:        "skill-b@v1.0.0",
					Name:      "skill-b",
					Version:   "v1.0.0",
					SourceURL: "https://github.com/test/repo/tree/v1.0.0/skill-b",
					StorePath: "/home/test/.gskills/skills/skill-b",
				},
			},
			wantErr: false,
		},
		{
			name: "overwrite existing file",
			skills: []types.SkillMetadata{
				{
					ID:        "new-skill@main",
					Name:      "new-skill",
					Version:   "main",
					SourceURL: "https://github.com/test/repo/tree/main/new-skill",
					StorePath: "/home/test/.gskills/skills/new-skill",
				},
			},
			wantErr: false,
			setupFile: func(t *testing.T, registryPath string) {
				existingSkills := []types.SkillMetadata{
					{
						ID:        "old-skill@main",
						Name:      "old-skill",
						Version:   "main",
						SourceURL: "https://github.com/test/repo/tree/main/old-skill",
						StorePath: "/home/test/.gskills/skills/old-skill",
					},
				}
				if err := SaveRegistryWithPath(registryPath, existingSkills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			registryPath := filepath.Join(tmpDir, "skills.json")

			if tt.setupFile != nil {
				tt.setupFile(t, registryPath)
			}

			err := SaveRegistryWithPath(registryPath, tt.skills)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveRegistryWithPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				loadedSkills, err := loadRegistryWithPath(registryPath)
				if err != nil {
					t.Errorf("SaveRegistryWithPath() failed to load saved data: %v", err)
					return
				}
				if len(loadedSkills) != len(tt.skills) {
					t.Errorf("SaveRegistryWithPath() saved %d skills, want %d", len(loadedSkills), len(tt.skills))
				}
				for i, skill := range loadedSkills {
					if skill.ID != tt.skills[i].ID {
						t.Errorf("SaveRegistryWithPath()[%d].ID = %v, want %v", i, skill.ID, tt.skills[i].ID)
					}
					if skill.Name != tt.skills[i].Name {
						t.Errorf("SaveRegistryWithPath()[%d].Name = %v, want %v", i, skill.Name, tt.skills[i].Name)
					}
					if skill.Version != tt.skills[i].Version {
						t.Errorf("SaveRegistryWithPath()[%d].Version = %v, want %v", i, skill.Version, tt.skills[i].Version)
					}
				}
			}
		})
	}
}

func TestSaveRegistryAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "skills.json")

	skills := []types.SkillMetadata{
		{
			ID:        "test-skill@main",
			Name:      "test-skill",
			Version:   "main",
			SourceURL: "https://github.com/test/repo/tree/main/test-skill",
			StorePath: "/home/test/.gskills/skills/test-skill",
		},
	}

	tmpPath := registryPath + ".tmp"
	initialData := []byte("initial content")
	if err := os.WriteFile(registryPath, initialData, 0644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	err := SaveRegistryWithPath(registryPath, skills)
	if err != nil {
		t.Fatalf("SaveRegistryWithPath() error = %v", err)
	}

	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("SaveRegistryWithPath() temporary file was not cleaned up")
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(data) == string(initialData) {
		t.Error("SaveRegistryWithPath() did not update the file")
	}
}

func TestAddOrUpdateSkill(t *testing.T) {
	tests := []struct {
		name       string
		setupFile  func(t *testing.T, registryPath string)
		skillToAdd *types.SkillMetadata
		wantErr    bool
		wantCount  int
		expectedID string
	}{
		{
			name: "add new skill",
			setupFile: func(t *testing.T, registryPath string) {
			},
			skillToAdd: &types.SkillMetadata{
				ID:        "new-skill@main",
				Name:      "new-skill",
				Version:   "main",
				SourceURL: "https://github.com/test/repo/tree/main/new-skill",
				StorePath: "/home/test/.gskills/skills/new-skill",
			},
			wantErr:    false,
			wantCount:  1,
			expectedID: "new-skill@main",
		},
		{
			name: "update existing skill",
			setupFile: func(t *testing.T, registryPath string) {
				existingSkills := []types.SkillMetadata{
					{
						ID:        "existing-skill@main",
						Name:      "existing-skill",
						Version:   "main",
						SourceURL: "https://github.com/test/repo/tree/main/existing-skill",
						StorePath: "/home/test/.gskills/skills/existing-skill",
					},
				}
				if err := SaveRegistryWithPath(registryPath, existingSkills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			skillToAdd: &types.SkillMetadata{
				ID:        "existing-skill@main",
				Name:      "existing-skill",
				Version:   "main",
				SourceURL: "https://github.com/test/repo/tree/main/existing-skill",
				StorePath: "/home/test/.gskills/skills/existing-skill",
			},
			wantErr:    false,
			wantCount:  1,
			expectedID: "existing-skill@main",
		},
		{
			name: "add different version of same skill",
			setupFile: func(t *testing.T, registryPath string) {
				existingSkills := []types.SkillMetadata{
					{
						ID:        "skill@main",
						Name:      "skill",
						Version:   "main",
						SourceURL: "https://github.com/test/repo/tree/main/skill",
						StorePath: "/home/test/.gskills/skills/skill",
					},
				}
				if err := SaveRegistryWithPath(registryPath, existingSkills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			skillToAdd: &types.SkillMetadata{
				ID:        "skill@v1.0.0",
				Name:      "skill",
				Version:   "v1.0.0",
				SourceURL: "https://github.com/test/repo/tree/v1.0.0/skill",
				StorePath: "/home/test/.gskills/skills/skill-v1.0.0",
			},
			wantErr:    false,
			wantCount:  2,
			expectedID: "skill@v1.0.0",
		},
		{
			name: "add multiple skills",
			setupFile: func(t *testing.T, registryPath string) {
				existingSkills := []types.SkillMetadata{
					{
						ID:        "skill-a@main",
						Name:      "skill-a",
						Version:   "main",
						SourceURL: "https://github.com/test/repo/tree/main/skill-a",
						StorePath: "/home/test/.gskills/skills/skill-a",
					},
				}
				if err := SaveRegistryWithPath(registryPath, existingSkills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			skillToAdd: &types.SkillMetadata{
				ID:        "skill-b@main",
				Name:      "skill-b",
				Version:   "main",
				SourceURL: "https://github.com/test/repo/tree/main/skill-b",
				StorePath: "/home/test/.gskills/skills/skill-b",
			},
			wantErr:    false,
			wantCount:  2,
			expectedID: "skill-b@main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			registryPath := filepath.Join(tmpDir, "skills.json")

			if tt.setupFile != nil {
				tt.setupFile(t, registryPath)
			}

			err := addOrUpdateSkillWithPath(registryPath, tt.skillToAdd)
			if (err != nil) != tt.wantErr {
				t.Errorf("addOrUpdateSkillWithPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			loadedSkills, err := loadRegistryWithPath(registryPath)
			if err != nil {
				t.Fatalf("failed to load registry after add/update: %v", err)
			}
			if len(loadedSkills) != tt.wantCount {
				t.Errorf("registry contains %d skills, want %d", len(loadedSkills), tt.wantCount)
			}
			found := false
			for _, skill := range loadedSkills {
				if skill.ID == tt.expectedID {
					found = true
					if skill.Name != tt.skillToAdd.Name {
						t.Errorf("skill.Name = %v, want %v", skill.Name, tt.skillToAdd.Name)
					}
					if skill.Version != tt.skillToAdd.Version {
						t.Errorf("skill.Version = %v, want %v", skill.Version, tt.skillToAdd.Version)
					}
					break
				}
			}
			if !found {
				t.Errorf("registry does not contain expected skill ID: %s", tt.expectedID)
			}
		})
	}
}

func TestRemoveSkill(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T, registryPath string)
		skillID     string
		wantErr     bool
		wantCount   int
		expectedIDs map[string]bool
	}{
		{
			name: "remove existing skill",
			setupFile: func(t *testing.T, registryPath string) {
				skills := []types.SkillMetadata{
					{
						ID:        "skill-a@main",
						Name:      "skill-a",
						Version:   "main",
						SourceURL: "https://github.com/test/repo/tree/main/skill-a",
						StorePath: "/home/test/.gskills/skills/skill-a",
					},
					{
						ID:        "skill-b@main",
						Name:      "skill-b",
						Version:   "main",
						SourceURL: "https://github.com/test/repo/tree/main/skill-b",
						StorePath: "/home/test/.gskills/skills/skill-b",
					},
				}
				if err := SaveRegistryWithPath(registryPath, skills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			skillID:   "skill-a@main",
			wantErr:   false,
			wantCount: 1,
			expectedIDs: map[string]bool{
				"skill-a@main": false,
				"skill-b@main": true,
			},
		},
		{
			name: "remove non-existent skill",
			setupFile: func(t *testing.T, registryPath string) {
				skills := []types.SkillMetadata{
					{
						ID:        "skill-a@main",
						Name:      "skill-a",
						Version:   "main",
						SourceURL: "https://github.com/test/repo/tree/main/skill-a",
						StorePath: "/home/test/.gskills/skills/skill-a",
					},
				}
				if err := SaveRegistryWithPath(registryPath, skills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			skillID:   "skill-b@main",
			wantErr:   false,
			wantCount: 1,
			expectedIDs: map[string]bool{
				"skill-a@main": true,
			},
		},
		{
			name: "remove from empty registry",
			setupFile: func(t *testing.T, registryPath string) {
			},
			skillID:     "skill-a@main",
			wantErr:     false,
			wantCount:   0,
			expectedIDs: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			registryPath := filepath.Join(tmpDir, "skills.json")

			if tt.setupFile != nil {
				tt.setupFile(t, registryPath)
			}

			err := removeSkillWithPath(registryPath, tt.skillID)
			if (err != nil) != tt.wantErr {
				t.Errorf("removeSkillWithPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				finalSkills, err := loadRegistryWithPath(registryPath)
				if err != nil {
					t.Fatalf("failed to load registry after remove: %v", err)
				}
				if len(finalSkills) != tt.wantCount {
					t.Errorf("registry contains %d skills, want %d", len(finalSkills), tt.wantCount)
				}
				for id, shouldExist := range tt.expectedIDs {
					found := false
					for _, skill := range finalSkills {
						if skill.ID == id {
							found = true
							break
						}
					}
					if found != shouldExist {
						t.Errorf("skill %s existence = %v, want %v", id, found, shouldExist)
					}
				}
			}
		})
	}
}

func TestValidateSkillMetadata(t *testing.T) {
	tests := []struct {
		name    string
		skill   *types.SkillMetadata
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid skill",
			skill: &types.SkillMetadata{
				ID:        "test-skill@main",
				Name:      "test-skill",
				Version:   "main",
				SourceURL: "https://github.com/test/repo/tree/main/test-skill",
				StorePath: "/home/test/.gskills/skills/test-skill",
			},
			wantErr: false,
		},
		{
			name:    "nil skill",
			skill:   nil,
			wantErr: true,
			errMsg:  "cannot be nil",
		},
		{
			name: "empty ID",
			skill: &types.SkillMetadata{
				Name:      "test-skill",
				Version:   "main",
				SourceURL: "https://github.com/test/repo/tree/main/test-skill",
				StorePath: "/home/test/.gskills/skills/test-skill",
			},
			wantErr: true,
			errMsg:  "ID cannot be empty",
		},
		{
			name: "empty name",
			skill: &types.SkillMetadata{
				ID:        "test-skill@main",
				Version:   "main",
				SourceURL: "https://github.com/test/repo/tree/main/test-skill",
				StorePath: "/home/test/.gskills/skills/test-skill",
			},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name: "empty version",
			skill: &types.SkillMetadata{
				ID:        "test-skill@main",
				Name:      "test-skill",
				SourceURL: "https://github.com/test/repo/tree/main/test-skill",
				StorePath: "/home/test/.gskills/skills/test-skill",
			},
			wantErr: true,
			errMsg:  "version cannot be empty",
		},
		{
			name: "empty source URL",
			skill: &types.SkillMetadata{
				ID:        "test-skill@main",
				Name:      "test-skill",
				Version:   "main",
				StorePath: "/home/test/.gskills/skills/test-skill",
			},
			wantErr: true,
			errMsg:  "source URL cannot be empty",
		},
		{
			name: "empty store path",
			skill: &types.SkillMetadata{
				ID:        "test-skill@main",
				Name:      "test-skill",
				Version:   "main",
				SourceURL: "https://github.com/test/repo/tree/main/test-skill",
			},
			wantErr: true,
			errMsg:  "store path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSkillMetadata(tt.skill)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSkillMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateSkillMetadata() error = %v, expected to contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestAddOrUpdateSkillConcurrent(t *testing.T) {
	t.Run("concurrent adds", func(t *testing.T) {
		tmpDir := t.TempDir()
		registryPath := filepath.Join(tmpDir, "skills.json")

		goroutines := 10
		var wg sync.WaitGroup
		wg.Add(goroutines)
		errors := make(chan error, goroutines)

		for i := 0; i < goroutines; i++ {
			go func(index int) {
				defer wg.Done()
				skill := &types.SkillMetadata{
					ID:        fmt.Sprintf("skill-%d@main", index),
					Name:      fmt.Sprintf("skill-%d", index),
					Version:   "main",
					SourceURL: fmt.Sprintf("https://github.com/test/repo/tree/main/skill-%d", index),
					StorePath: fmt.Sprintf("/home/test/.gskills/skills/skill-%d", index),
				}
				err := addOrUpdateSkillWithPath(registryPath, skill)
				errors <- err
			}(i)
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			if err != nil {
				errorCount++
				t.Logf("Error from goroutine: %v", err)
			}
		}

		if errorCount > 0 {
			t.Errorf("concurrent adds had %d errors", errorCount)
		}

		loaded, err := loadRegistryWithPath(registryPath)
		if err != nil {
			t.Fatalf("failed to load registry after concurrent adds: %v", err)
		}

		if len(loaded) != goroutines {
			t.Errorf("expected %d skills, got %d", goroutines, len(loaded))
		}
	})
}

func BenchmarkAddOrUpdateSkill(b *testing.B) {
	tmpDir := b.TempDir()
	registryPath := filepath.Join(tmpDir, "skills.json")

	skill := &types.SkillMetadata{
		ID:        "benchmark-skill@main",
		Name:      "benchmark-skill",
		Version:   "main",
		SourceURL: "https://github.com/test/repo/tree/main/benchmark-skill",
		StorePath: "/home/test/.gskills/skills/benchmark-skill",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		skill.ID = fmt.Sprintf("benchmark-skill-%d@main", i)
		_ = addOrUpdateSkillWithPath(registryPath, skill)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		len(s) >= len(substr) && (s == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
