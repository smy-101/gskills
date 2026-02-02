package add

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
