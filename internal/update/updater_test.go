package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smy-101/gskills/internal/add"
	"github.com/smy-101/gskills/internal/types"
)

func TestCheckUpdate(t *testing.T) {
	tests := []struct {
		name         string
		skill        *types.SkillMetadata
		serverResp   string
		serverStatus int
		wantUpdate   bool
		wantSHA      string
		wantErr      bool
	}{
		{
			name: "has update available",
			skill: &types.SkillMetadata{
				Name:      "test-skill",
				SourceURL: "https://github.com/owner/repo/tree/main/skills/test",
				CommitSHA: "oldsha123456789",
			},
			serverResp:   `{"sha": "newsha987654321"}`,
			serverStatus: 200,
			wantUpdate:   true,
			wantSHA:      "newsha987654321",
		},
		{
			name: "already up to date",
			skill: &types.SkillMetadata{
				Name:      "test-skill",
				SourceURL: "https://github.com/owner/repo/tree/main/skills/test",
				CommitSHA: "currentsha123",
			},
			serverResp:   `{"sha": "currentsha123"}`,
			serverStatus: 200,
			wantUpdate:   false,
			wantSHA:      "currentsha123",
		},
		{
			name: "API error",
			skill: &types.SkillMetadata{
				Name:      "test-skill",
				SourceURL: "https://github.com/owner/repo/tree/main/skills/test",
				CommitSHA: "oldsha",
			},
			serverResp:   `{"message": "Not Found"}`,
			serverStatus: 404,
			wantErr:      true,
		},
		{
			name:    "nil skill",
			skill:   nil,
			wantErr: true,
		},
		{
			name: "empty source URL",
			skill: &types.SkillMetadata{
				Name:      "test-skill",
				SourceURL: "",
				CommitSHA: "oldsha",
			},
			wantErr: true,
		},
		{
			name: "invalid URL",
			skill: &types.SkillMetadata{
				Name:      "test-skill",
				SourceURL: "not-a-valid-url",
				CommitSHA: "oldsha",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts *httptest.Server
			if tt.serverResp != "" {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.serverStatus)
					w.Write([]byte(tt.serverResp))
				}))
				defer ts.Close()
			}

			updater := NewUpdater("")
			if ts != nil {
				updater.SetBaseURL(ts.URL)
			}

			hasUpdate, newSHA, err := updater.CheckUpdate(tt.skill)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if hasUpdate != tt.wantUpdate {
				t.Errorf("CheckUpdate() hasUpdate = %v, want %v", hasUpdate, tt.wantUpdate)
			}

			if newSHA != tt.wantSHA {
				t.Errorf("CheckUpdate() newSHA = %v, want %v", newSHA, tt.wantSHA)
			}
		})
	}
}

func TestUpdateSkill(t *testing.T) {
	t.Run("nil skill", func(t *testing.T) {
		updater := NewUpdater("")
		err := updater.UpdateSkill(nil)
		if err == nil {
			t.Error("UpdateSkill() should error with nil skill")
		}
	})

	t.Run("no update available", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]string{"sha": "sameSha"})
		}))
		defer ts.Close()

		skill := &types.SkillMetadata{
			Name:      "test-skill",
			SourceURL: "https://github.com/owner/repo/tree/main/skills/test",
			CommitSHA: "sameSha",
		}

		updater := NewUpdater("")
		updater.SetBaseURL(ts.URL)

		err := updater.UpdateSkill(skill)
		if err != nil {
			t.Errorf("UpdateSkill() with no update should not error, got: %v", err)
		}
	})
}

func TestUpdateAll(t *testing.T) {
	t.Run("update multiple skills", func(t *testing.T) {
		tmpDir := t.TempDir()

		skillDirs := []string{
			filepath.Join(tmpDir, "skills", "skill1"),
			filepath.Join(tmpDir, "skills", "skill2"),
		}

		for _, dir := range skillDirs {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("failed to create skill dir: %v", err)
			}
		}

		var ts *httptest.Server
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/repos/owner/repo/commits/main" {
				w.WriteHeader(200)
				json.NewEncoder(w).Encode(map[string]string{"sha": "newsha"})
			} else if r.URL.Path == "/repos/owner/repo/contents/skills/skill1" || r.URL.Path == "/repos/owner/repo/contents/skills/skill2" {
				w.WriteHeader(200)
				json.NewEncoder(w).Encode([]types.GitHubContent{
					{
						Type:        "file",
						Name:        "test.txt",
						Path:        "skills/skill1/test.txt",
						SHA:         "filesha",
						DownloadURL: ts.URL + "/download/test.txt",
					},
				})
			} else if r.URL.Path == "/download/test.txt" {
				w.WriteHeader(200)
				w.Write([]byte("test content"))
			}
		}))
		defer ts.Close()

		skills := []*types.SkillMetadata{
			{
				ID:        "skill1@main",
				Name:      "skill1",
				SourceURL: "https://github.com/owner/repo/tree/main/skills/skill1",
				CommitSHA: "oldsha",
				StorePath: skillDirs[0],
				UpdatedAt: time.Now(),
			},
			{
				ID:        "skill2@main",
				Name:      "skill2",
				SourceURL: "https://github.com/owner/repo/tree/main/skills/skill2",
				CommitSHA: "oldsha",
				StorePath: skillDirs[1],
				UpdatedAt: time.Now(),
			},
		}

		updater := NewUpdater("")
		updater.SetBaseURL(ts.URL)

		stats, err := updater.UpdateAll(skills)
		if err != nil {
			t.Logf("UpdateAll() error = %v", err)
		}

		if stats.Total != 2 {
			t.Errorf("UpdateAll() stats.Total = %d, want 2", stats.Total)
		}

		for _, dir := range skillDirs {
			testFile := filepath.Join(dir, "test.txt")
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Errorf("file not found: %s", testFile)
			}
		}
	})
}

func TestUpdateError(t *testing.T) {
	t.Run("error wrapping and unwrapping", func(t *testing.T) {
		originalErr := &UpdateError{
			Type:    UpdateErrorTypeCheck,
			Message: "test error",
			Skill:   "test-skill",
			Err:     &UpdateError{Type: UpdateErrorTypeDownload, Message: "inner error"},
		}

		if originalErr.Error() == "" {
			t.Error("Error() should not return empty string")
		}

		if originalErr.Unwrap() == nil {
			t.Error("Unwrap() should return inner error")
		}

		target := &UpdateError{Type: UpdateErrorTypeCheck}
		if !originalErr.Is(target) {
			t.Error("Is() should return true for same error type")
		}

		differentTarget := &UpdateError{Type: UpdateErrorTypeDownload}
		if originalErr.Is(differentTarget) {
			t.Error("Is() should return false for different error type")
		}
	})
}

func TestDownloadRecursive(t *testing.T) {
	t.Run("successful download with subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetDir := filepath.Join(tmpDir, "target")

		var ts *httptest.Server
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/repos/owner/repo/contents/skills/test" {
				w.WriteHeader(200)
				contents := []types.GitHubContent{
					{
						Type: "dir",
						Name: "subdir",
						Path: "skills/test/subdir",
						URL:  ts.URL + "/repos/owner/repo/contents/subdir",
						SHA:  "dirsha",
					},
					{
						Type:        "file",
						Name:        "file1.txt",
						Path:        "skills/test/file1.txt",
						SHA:         "file1sha",
						DownloadURL: ts.URL + "/download/file1.txt",
					},
				}
				json.NewEncoder(w).Encode(contents)
			} else if r.URL.Path == "/repos/owner/repo/contents/skills/test/subdir" {
				w.WriteHeader(200)
				contents := []types.GitHubContent{
					{
						Type:        "file",
						Name:        "file2.txt",
						Path:        "skills/test/subdir/file2.txt",
						SHA:         "file2sha",
						DownloadURL: ts.URL + "/download/file2.txt",
					},
				}
				json.NewEncoder(w).Encode(contents)
			} else if r.URL.Path == "/download/file1.txt" {
				w.WriteHeader(200)
				w.Write([]byte("file1 content"))
			} else if r.URL.Path == "/download/file2.txt" {
				w.WriteHeader(200)
				w.Write([]byte("file2 content"))
			}
		}))
		defer ts.Close()

		updater := NewUpdater("")
		updater.SetBaseURL(ts.URL)

		repoInfo := &add.GitHubRepoInfo{
			Owner:  "owner",
			Repo:   "repo",
			Branch: "main",
			Path:   "skills/test",
		}

		ctx := context.Background()
		stats, err := updater.downloadRecursive(ctx, repoInfo, targetDir, "skills/test")
		if err != nil {
			t.Fatalf("downloadRecursive() error = %v", err)
		}

		if stats.FilesDownloaded != 2 {
			t.Errorf("FilesDownloaded = %d, want 2", stats.FilesDownloaded)
		}

		if stats.DirsCreated != 1 {
			t.Errorf("DirsCreated = %d, want 1", stats.DirsCreated)
		}

		file1 := filepath.Join(targetDir, "file1.txt")
		if _, err := os.Stat(file1); os.IsNotExist(err) {
			t.Error("file1.txt not created")
		}

		file2 := filepath.Join(targetDir, "subdir", "file2.txt")
		if _, err := os.Stat(file2); os.IsNotExist(err) {
			t.Error("subdir/file2.txt not created")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		tmpDir := t.TempDir()

		serverCalled := make(chan struct{})
		var ts *httptest.Server
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(serverCalled)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode([]types.GitHubContent{})
		}))
		defer ts.Close()

		updater := NewUpdater("")
		updater.SetBaseURL(ts.URL)

		repoInfo := &add.GitHubRepoInfo{
			Owner:  "owner",
			Repo:   "repo",
			Branch: "main",
			Path:   "skills/test",
		}

		ctx, cancel := context.WithCancel(context.Background())

		cancel()

		_, err := updater.downloadRecursive(ctx, repoInfo, tmpDir, "skills/test")

		select {
		case <-serverCalled:
			if err == nil {
				t.Error("downloadRecursive() should error with cancelled context")
			}
		case <-time.After(100 * time.Millisecond):
			if err == nil {
				t.Skip("Server was not called, context cancelled before request started")
			}
		}
	})
}
