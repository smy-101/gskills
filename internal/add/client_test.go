package add

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/smy-101/gskills/internal/registry"
	"github.com/smy-101/gskills/internal/types"
)

type LogCall struct {
	Msg    string
	Fields []interface{}
}

type MockLogger struct {
	mu         sync.Mutex
	DebugCalls []LogCall
	InfoCalls  []LogCall
	WarnCalls  []LogCall
	ErrorCalls []LogCall
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DebugCalls = append(m.DebugCalls, LogCall{Msg: msg, Fields: fields})
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InfoCalls = append(m.InfoCalls, LogCall{Msg: msg, Fields: fields})
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WarnCalls = append(m.WarnCalls, LogCall{Msg: msg, Fields: fields})
}

func (m *MockLogger) Error(msg string, err error, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorCalls = append(m.ErrorCalls, LogCall{Msg: msg, Fields: append(fields, "error", err)})
}

func (m *MockLogger) HasDebugCall(msg string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, call := range m.DebugCalls {
		if call.Msg == msg {
			return true
		}
	}
	return false
}

func (m *MockLogger) HasInfoCall(msg string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, call := range m.InfoCalls {
		if call.Msg == msg {
			return true
		}
	}
	return false
}

func (m *MockLogger) HasWarnCall(msg string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, call := range m.WarnCalls {
		if call.Msg == msg {
			return true
		}
	}
	return false
}

func (m *MockLogger) HasErrorCall(msg string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, call := range m.ErrorCalls {
		if call.Msg == msg {
			return true
		}
	}
	return false
}

func (m *MockLogger) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DebugCalls = nil
	m.InfoCalls = nil
	m.WarnCalls = nil
	m.ErrorCalls = nil
}

type TestServer struct {
	Server   *httptest.Server
	Handlers map[string]http.HandlerFunc
	mu       sync.Mutex
	CallLog  []string
}

func NewTestServer() *TestServer {
	ts := &TestServer{
		Handlers: make(map[string]http.HandlerFunc),
		CallLog:  make([]string, 0),
	}

	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.mu.Lock()
		ts.CallLog = append(ts.CallLog, r.URL.Path)
		ts.mu.Unlock()

		handler := ts.Handlers[r.URL.Path]
		if handler == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handler(w, r)
	}))

	return ts
}

func (ts *TestServer) Close() {
	ts.Server.Close()
}

func (ts *TestServer) SetHandler(path string, handler http.HandlerFunc) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.Handlers[path] = handler
}

func (ts *TestServer) GetCallCount(path string) int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	count := 0
	for _, p := range ts.CallLog {
		if p == path {
			count++
		}
	}
	return count
}

func (ts *TestServer) URL() string {
	return ts.Server.URL
}

func setupTestEnv(t *testing.T) (homeDir string, cleanup func()) {
	t.Helper()
	homeDir = t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)

	gskillsDir := filepath.Join(homeDir, ".gskills", "skills")
	if err := os.MkdirAll(gskillsDir, 0755); err != nil {
		t.Fatalf("failed to create .gskills/skills directory: %v", err)
	}

	registryDir := filepath.Join(homeDir, ".gskills")
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		t.Fatalf("failed to create .gskills directory: %v", err)
	}

	cleanup = func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		} else {
			os.Unsetenv("HOME")
		}
	}

	return homeDir, cleanup
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		wantToken     bool
		wantBaseURL   string
		wantUserAgent string
	}{
		{
			name:          "client without token",
			token:         "",
			wantToken:     false,
			wantBaseURL:   "https://api.github.com",
			wantUserAgent: "gskills-cli/1.0",
		},
		{
			name:          "client with token",
			token:         "test-token",
			wantToken:     true,
			wantBaseURL:   "https://api.github.com",
			wantUserAgent: "gskills-cli/1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.token)

			if client.token != tt.token {
				t.Errorf("NewClient() token = %v, want %v", client.token, tt.token)
			}

			if client.baseURL != tt.wantBaseURL {
				t.Errorf("NewClient() baseURL = %v, want %v", client.baseURL, tt.wantBaseURL)
			}

			if client.restyClient == nil {
				t.Fatal("NewClient() restyClient is nil")
			}

			userAgent := client.restyClient.Header.Get("User-Agent")
			if userAgent != tt.wantUserAgent {
				t.Errorf("NewClient() User-Agent = %v, want %v", userAgent, tt.wantUserAgent)
			}

			if tt.wantToken {
				authHeader := client.restyClient.Header.Get("Authorization")
				expectedAuth := fmt.Sprintf("Bearer %s", tt.token)
				if authHeader != expectedAuth {
					t.Errorf("NewClient() Authorization = %v, want %v", authHeader, expectedAuth)
				}
			}
		})
	}
}

func TestCheckSKILLExists(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "SKILL.md exists",
			statusCode: http.StatusOK,
			response:   `{"name":"SKILL.md","path":"SKILL.md","type":"file"}`,
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "SKILL.md not found",
			statusCode: http.StatusNotFound,
			response:   `{"message":"Not Found"}`,
			wantExists: false,
			wantErr:    false,
		},
		{
			name:       "API error",
			statusCode: http.StatusInternalServerError,
			response:   `{"message":"Internal Server Error"}`,
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTestServer()
			defer ts.Close()

			path := "/repos/owner/repo/contents/skills/test/SKILL.md"
			ts.SetHandler(path, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			})

			client := NewClient("")
			client.baseURL = ts.URL()

			repoInfo := &GitHubRepoInfo{
				Owner:  "owner",
				Repo:   "repo",
				Branch: "main",
				Path:   "skills/test",
			}

			ctx := context.Background()
			exists, err := client.checkSKILLExists(ctx, repoInfo)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkSKILLExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if exists != tt.wantExists {
				t.Errorf("checkSKILLExists() = %v, want %v", exists, tt.wantExists)
			}

			if ts.GetCallCount(path) != 1 {
				t.Errorf("checkSKILLExists() called test server %d times, want 1", ts.GetCallCount(path))
			}
		})
	}
}

func TestGetBranchCommitSHA(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantSHA    string
		wantErr    bool
	}{
		{
			name:       "successful fetch",
			statusCode: http.StatusOK,
			response:   `{"sha":"abc123def456","commit":{"message":"test commit"}}`,
			wantSHA:    "abc123def456",
			wantErr:    false,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			response:   `{"message":"Not Found"}`,
			wantSHA:    "",
			wantErr:    true,
		},
		{
			name:       "missing sha in response",
			statusCode: http.StatusOK,
			response:   `{"commit":{"message":"test"}}`,
			wantSHA:    "",
			wantErr:    true,
		},
		{
			name:       "invalid JSON",
			statusCode: http.StatusOK,
			response:   `invalid json`,
			wantSHA:    "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTestServer()
			defer ts.Close()

			path := "/repos/owner/repo/commits/main"
			ts.SetHandler(path, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			})

			client := NewClient("")
			client.baseURL = ts.URL()

			repoInfo := &GitHubRepoInfo{
				Owner:  "owner",
				Repo:   "repo",
				Branch: "main",
				Path:   "skills/test",
			}

			ctx := context.Background()
			sha, err := client.getBranchCommitSHA(ctx, repoInfo)

			if (err != nil) != tt.wantErr {
				t.Errorf("getBranchCommitSHA() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if sha != tt.wantSHA {
				t.Errorf("getBranchCommitSHA() = %v, want %v", sha, tt.wantSHA)
			}

			if !tt.wantErr && ts.GetCallCount(path) != 1 {
				t.Errorf("getBranchCommitSHA() called test server %d times, want 1", ts.GetCallCount(path))
			}
		})
	}
}

func TestDownloadFile(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   []byte
		wantErr    bool
	}{
		{
			name:       "successful download",
			statusCode: http.StatusOK,
			response:   []byte("file content"),
			wantErr:    false,
		},
		{
			name:       "rate limit with retry",
			statusCode: http.StatusForbidden,
			response:   []byte(`{"message":"API rate limit exceeded"}`),
			wantErr:    true,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			response:   []byte(`{"message":"Not Found"}`),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTestServer()
			defer ts.Close()

			callCount := 0
			ts.SetHandler("/download", func(w http.ResponseWriter, r *http.Request) {
				callCount++
				if callCount == 1 && tt.statusCode == http.StatusForbidden {
					w.WriteHeader(http.StatusForbidden)
					w.Write(tt.response)
					return
				}
				w.WriteHeader(tt.statusCode)
				w.Write(tt.response)
			})

			client := NewClient("")
			client.baseURL = ts.URL()
			client.logger = &MockLogger{}

			ctx := context.Background()
			data, err := client.downloadFile(ctx, ts.URL()+"/download")

			if (err != nil) != tt.wantErr {
				t.Errorf("downloadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && string(data) != string(tt.response) {
				t.Errorf("downloadFile() = %v, want %v", string(data), string(tt.response))
			}
		})
	}
}

func TestGetGitHubContents(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
		validate   func(*testing.T, []types.GitHubContent)
	}{
		{
			name:       "successful directory listing",
			statusCode: http.StatusOK,
			response: `[
				{"type":"file","name":"test.txt","path":"test.txt","sha":"abc123","size":100,"url":"https://api.github.com/...","html_url":"https://github.com/...","download_url":"https://raw.githubusercontent.com/..."},
				{"type":"dir","name":"subdir","path":"subdir","sha":"def456","size":0,"url":"https://api.github.com/...","html_url":"https://github.com/...","download_url":""}
			]`,
			wantErr: false,
			validate: func(t *testing.T, contents []types.GitHubContent) {
				if len(contents) != 2 {
					t.Errorf("got %d items, want 2", len(contents))
				}
				if contents[0].Name != "test.txt" {
					t.Errorf("first item name = %s, want test.txt", contents[0].Name)
				}
				if contents[1].Name != "subdir" {
					t.Errorf("second item name = %s, want subdir", contents[1].Name)
				}
			},
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			response:   `{"message":"Not Found"}`,
			wantErr:    true,
		},
		{
			name:       "invalid JSON",
			statusCode: http.StatusOK,
			response:   `invalid json`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTestServer()
			defer ts.Close()

			path := "/repos/owner/repo/contents/test"
			ts.SetHandler(path, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			})

			client := NewClient("")
			client.baseURL = ts.URL()

			repoInfo := &GitHubRepoInfo{
				Owner:  "owner",
				Repo:   "repo",
				Branch: "main",
				Path:   "test",
			}

			ctx := context.Background()
			contents, err := client.getGitHubContents(ctx, repoInfo, "test")

			if (err != nil) != tt.wantErr {
				t.Errorf("getGitHubContents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, contents)
			}
		})
	}
}

func TestDownloadRecursive(t *testing.T) {
	t.Run("successful directory download", func(t *testing.T) {
		ts := NewTestServer()
		defer ts.Close()

		ts.SetHandler("/repos/owner/repo/contents/skill", func(w http.ResponseWriter, r *http.Request) {
			contents := []types.GitHubContent{
				{
					Type:        "file",
					Name:        "file1.txt",
					Path:        "skill/file1.txt",
					SHA:         "abc123",
					Size:        10,
					DownloadURL: ts.URL() + "/file1",
				},
				{
					Type:        "dir",
					Name:        "subdir",
					Path:        "skill/subdir",
					SHA:         "def456",
					DownloadURL: "",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(contents)
		})

		ts.SetHandler("/repos/owner/repo/contents/skill/subdir", func(w http.ResponseWriter, r *http.Request) {
			contents := []types.GitHubContent{
				{
					Type:        "file",
					Name:        "file2.txt",
					Path:        "skill/subdir/file2.txt",
					SHA:         "ghi789",
					Size:        15,
					DownloadURL: ts.URL() + "/file2",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(contents)
		})

		ts.SetHandler("/file1", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("content1"))
		})

		ts.SetHandler("/file2", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("content2"))
		})

		client := NewClient("")
		client.baseURL = ts.URL()
		mockLogger := &MockLogger{}
		client.logger = mockLogger

		repoInfo := &GitHubRepoInfo{
			Owner:  "owner",
			Repo:   "repo",
			Branch: "main",
			Path:   "skill",
		}

		tmpDir := t.TempDir()
		ctx := context.Background()

		stats, err := client.downloadRecursive(ctx, repoInfo, tmpDir, "skill")

		if err != nil {
			t.Fatalf("downloadRecursive() error = %v", err)
		}

		if stats.FilesDownloaded != 2 {
			t.Errorf("FilesDownloaded = %d, want 2", stats.FilesDownloaded)
		}

		if stats.DirsCreated != 1 {
			t.Errorf("DirsCreated = %d, want 1", stats.DirsCreated)
		}

		if stats.BytesDownloaded != 16 {
			t.Errorf("BytesDownloaded = %d, want 16", stats.BytesDownloaded)
		}

		file1Path := filepath.Join(tmpDir, "file1.txt")
		content1, err := os.ReadFile(file1Path)
		if err != nil {
			t.Fatalf("failed to read file1.txt: %v", err)
		}
		if string(content1) != "content1" {
			t.Errorf("file1.txt content = %s, want 'content1'", string(content1))
		}

		file2Path := filepath.Join(tmpDir, "subdir", "file2.txt")
		content2, err := os.ReadFile(file2Path)
		if err != nil {
			t.Fatalf("failed to read file2.txt: %v", err)
		}
		if string(content2) != "content2" {
			t.Errorf("file2.txt content = %s, want 'content2'", string(content2))
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ts := NewTestServer()
		defer ts.Close()

		requestCount := 0
		ts.SetHandler("/repos/owner/repo/contents/skill", func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			contents := []types.GitHubContent{
				{
					Type:        "file",
					Name:        "file.txt",
					Path:        "skill/file.txt",
					SHA:         "abc123",
					Size:        10,
					DownloadURL: ts.URL() + "/file",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(contents)
		})

		client := NewClient("")
		client.baseURL = ts.URL()

		repoInfo := &GitHubRepoInfo{
			Owner:  "owner",
			Repo:   "repo",
			Branch: "main",
			Path:   "skill",
		}

		tmpDir := t.TempDir()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		_, err := client.downloadRecursive(ctx, repoInfo, tmpDir, "skill")

		if err == nil {
			t.Error("downloadRecursive() expected error on timeout, got nil")
		}
	})
}

func TestDownload(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	tests := []struct {
		name                string
		url                 string
		setupServer         func(*TestServer)
		mockPromptOverwrite func() (bool, error)
		wantErr             bool
		validate            func(*testing.T, string)
		errorType           ErrorType
	}{
		{
			name: "successful download",
			url:  "https://github.com/owner/repo/tree/main/skill",
			setupServer: func(ts *TestServer) {
				ts.SetHandler("/repos/owner/repo/contents/skill/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"name": "SKILL.md",
						"type": "file",
					})
				})

				ts.SetHandler("/repos/owner/repo/commits/main", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"sha": "abc123def456",
						"commit": map[string]interface{}{
							"message": "test commit",
						},
					})
				})

				ts.SetHandler("/repos/owner/repo/contents/skill", func(w http.ResponseWriter, r *http.Request) {
					contents := []types.GitHubContent{
						{
							Type:        "file",
							Name:        "SKILL.md",
							Path:        "skill/SKILL.md",
							SHA:         "abc123",
							Size:        50,
							DownloadURL: ts.URL() + "/skillmd",
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(contents)
				})

				ts.SetHandler("/skillmd", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("# Test Skill"))
				})
			},
			wantErr: false,
			validate: func(t *testing.T, homeDir string) {
				skillPath := filepath.Join(homeDir, ".gskills", "skills", "skill")
				if _, err := os.Stat(skillPath); os.IsNotExist(err) {
					t.Errorf("skill directory not created at %s", skillPath)
				}

				skillMDPath := filepath.Join(skillPath, "SKILL.md")
				content, err := os.ReadFile(skillMDPath)
				if err != nil {
					t.Fatalf("failed to read SKILL.md: %v", err)
				}
				if string(content) != "# Test Skill" {
					t.Errorf("SKILL.md content = %s, want '# Test Skill'", string(content))
				}

				skills, err := registry.LoadRegistry()
				if err != nil {
					t.Fatalf("failed to load registry: %v", err)
				}

				found := false
				for _, s := range skills {
					if s.ID == "skill@main" {
						found = true
						if s.Name != "skill" {
							t.Errorf("skill name = %s, want 'skill'", s.Name)
						}
						if s.CommitSHA != "abc123def456" {
							t.Errorf("skill commit SHA = %s, want 'abc123def456'", s.CommitSHA)
						}
						break
					}
				}
				if !found {
					t.Error("skill not found in registry")
				}
			},
		},
		{
			name:        "invalid URL format",
			url:         "not-a-url",
			setupServer: func(ts *TestServer) {},
			wantErr:     true,
			errorType:   ErrorTypeInvalidURL,
		},
		{
			name: "SKILL.md not found",
			url:  "https://github.com/owner/repo/tree/main/noskill",
			setupServer: func(ts *TestServer) {
				ts.SetHandler("/repos/owner/repo/contents/noskill/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
			},
			wantErr:   true,
			errorType: ErrorTypeValidation,
		},
		{
			name: "commit SHA fetch fails",
			url:  "https://github.com/owner/repo/tree/main/nocommit",
			setupServer: func(ts *TestServer) {
				ts.SetHandler("/repos/owner/repo/contents/nocommit/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"name": "SKILL.md",
						"type": "file",
					})
				})

				ts.SetHandler("/repos/owner/repo/commits/main", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
			},
			wantErr:   true,
			errorType: ErrorTypeAPI,
		},
		{
			name: "directory already exists - overwrite",
			url:  "https://github.com/owner/repo/tree/main/skill2",
			setupServer: func(ts *TestServer) {
				homeDir, _ := os.UserHomeDir()
				skillPath := filepath.Join(homeDir, ".gskills", "skills", "skill2")
				os.MkdirAll(skillPath, 0755)

				ts.SetHandler("/repos/owner/repo/contents/skill2/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"name": "SKILL.md",
						"type": "file",
					})
				})

				ts.SetHandler("/repos/owner/repo/commits/main", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"sha": "xyz789",
						"commit": map[string]interface{}{
							"message": "overwrite test",
						},
					})
				})

				ts.SetHandler("/repos/owner/repo/contents/skill2", func(w http.ResponseWriter, r *http.Request) {
					contents := []types.GitHubContent{
						{
							Type:        "file",
							Name:        "file.txt",
							Path:        "skill2/file.txt",
							SHA:         "abc123",
							Size:        10,
							DownloadURL: ts.URL() + "/file",
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(contents)
				})

				ts.SetHandler("/file", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("test"))
				})
			},
			mockPromptOverwrite: func() (bool, error) { return true, nil },
			wantErr:             false,
			validate: func(t *testing.T, homeDir string) {
				skillPath := filepath.Join(homeDir, ".gskills", "skills", "skill2")
				if _, err := os.Stat(skillPath); os.IsNotExist(err) {
					t.Errorf("skill directory not found at %s", skillPath)
				}
			},
		},
		{
			name: "directory already exists - cancel",
			url:  "https://github.com/owner/repo/tree/main/skill3",
			setupServer: func(ts *TestServer) {
				homeDir, _ := os.UserHomeDir()
				skillPath := filepath.Join(homeDir, ".gskills", "skills", "skill3")
				os.MkdirAll(skillPath, 0755)

				ts.SetHandler("/repos/owner/repo/contents/skill3/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"name": "SKILL.md",
						"type": "file",
					})
				})

				ts.SetHandler("/repos/owner/repo/commits/main", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"sha": "cancel123",
						"commit": map[string]interface{}{
							"message": "cancel test",
						},
					})
				})
			},
			mockPromptOverwrite: func() (bool, error) { return false, nil },
			wantErr:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTestServer()
			defer ts.Close()

			tt.setupServer(ts)

			client := NewClient("")
			client.baseURL = ts.URL()

			if tt.mockPromptOverwrite != nil {
				oldPromptOverwrite := promptOverwrite
				promptOverwrite = tt.mockPromptOverwrite
				defer func() { promptOverwrite = oldPromptOverwrite }()
			}

			homeDir, _ := os.UserHomeDir()

			err := client.Download(tt.url)

			if (err != nil) != tt.wantErr {
				t.Errorf("Download() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errorType != 0 {
				var downloadErr *DownloadError
				if !err.(*DownloadError).Is(&DownloadError{Type: tt.errorType}) {
					t.Errorf("Download() error type = %v, want %v", err, tt.errorType)
				}
				_ = downloadErr
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, homeDir)
			}
		})
	}
}

func TestDownloadRecursive_Race(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race test in short mode")
	}

	ts := NewTestServer()
	defer ts.Close()

	ts.SetHandler("/repos/owner/repo/contents/skill", func(w http.ResponseWriter, r *http.Request) {
		contents := make([]types.GitHubContent, 10)
		for i := 0; i < 10; i++ {
			contents[i] = types.GitHubContent{
				Type:        "file",
				Name:        fmt.Sprintf("file%d.txt", i),
				Path:        fmt.Sprintf("skill/file%d.txt", i),
				SHA:         fmt.Sprintf("sha%d", i),
				Size:        100,
				DownloadURL: ts.URL() + fmt.Sprintf("/file%d", i),
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(contents)
	})

	for i := 0; i < 10; i++ {
		ts.SetHandler(fmt.Sprintf("/file%d", i), func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(strings.Repeat("x", 100)))
		})
	}

	client := NewClient("")
	client.baseURL = ts.URL()

	repoInfo := &GitHubRepoInfo{
		Owner:  "owner",
		Repo:   "repo",
		Branch: "main",
		Path:   "skill",
	}

	tmpDir := t.TempDir()
	ctx := context.Background()

	stats, err := client.downloadRecursive(ctx, repoInfo, tmpDir, "skill")

	if err != nil {
		t.Fatalf("downloadRecursive() error = %v", err)
	}

	if stats.FilesDownloaded != 10 {
		t.Errorf("FilesDownloaded = %d, want 10", stats.FilesDownloaded)
	}

	if stats.BytesDownloaded != 1000 {
		t.Errorf("BytesDownloaded = %d, want 1000", stats.BytesDownloaded)
	}
}

func TestIsRateLimitResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"403 forbidden", http.StatusForbidden, true},
		{"429 too many requests", 429, true},
		{"200 OK", http.StatusOK, false},
		{"404 not found", http.StatusNotFound, false},
		{"500 internal server error", http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRateLimitResponse(tt.statusCode); got != tt.want {
				t.Errorf("isRateLimitResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"403 error", fmt.Errorf("API rate limit exceeded: 403"), true},
		{"429 error", fmt.Errorf("too many requests: 429"), true},
		{"other error", fmt.Errorf("some other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRateLimitError(tt.err); got != tt.want {
				t.Errorf("isRateLimitError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDownloadStats(t *testing.T) {
	stats := &DownloadStats{
		FilesDownloaded: 5,
		DirsCreated:     2,
		BytesDownloaded: 1024,
	}

	if stats.FilesDownloaded != 5 {
		t.Errorf("FilesDownloaded = %d, want 5", stats.FilesDownloaded)
	}

	if stats.DirsCreated != 2 {
		t.Errorf("DirsCreated = %d, want 2", stats.DirsCreated)
	}

	if stats.BytesDownloaded != 1024 {
		t.Errorf("BytesDownloaded = %d, want 1024", stats.BytesDownloaded)
	}
}
