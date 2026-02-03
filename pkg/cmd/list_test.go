package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smy-101/gskills/internal/add"
	"github.com/smy-101/gskills/internal/types"
)

// createTestRegistry creates a test skills registry in a temporary directory
// and returns the registry file path.
func createTestRegistry(t *testing.T, skills []types.SkillMetadata) string {
	t.Helper()
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "skills.json")

	if err := add.SaveRegistryWithPath(registryPath, skills); err != nil {
		t.Fatalf("failed to write test registry: %v", err)
	}
	return registryPath
}

// setupTestRegistry sets up a test registry by backing up the original
// registry and replacing it with the test registry.
// Returns a cleanup function that restores the original registry.
func setupTestRegistry(t *testing.T, testRegistryPath string) func() {
	t.Helper()

	homeDir, _ := os.UserHomeDir()
	origRegistryPath := filepath.Join(homeDir, ".gskills", "skills.json")
	tmpBackup := filepath.Join(t.TempDir(), "backup-skills.json")

	var originalData []byte
	if data, err := os.ReadFile(origRegistryPath); err == nil {
		if err := os.WriteFile(tmpBackup, data, 0644); err != nil {
			t.Logf("warning: failed to backup registry: %v", err)
		}
		originalData = data
	}

	// Use test registry
	if err := os.MkdirAll(filepath.Dir(origRegistryPath), 0755); err != nil {
		t.Fatalf("failed to create registry dir: %v", err)
	}
	data, err := os.ReadFile(testRegistryPath)
	if err != nil {
		t.Fatalf("failed to read test registry: %v", err)
	}
	if err := os.WriteFile(origRegistryPath, data, 0644); err != nil {
		t.Fatalf("failed to write test registry: %v", err)
	}

	return func() {
		if originalData != nil {
			if err := os.WriteFile(origRegistryPath, originalData, 0644); err != nil {
				t.Logf("warning: failed to restore registry: %v", err)
			}
		}
	}
}

func TestExecuteList(t *testing.T) {
	tests := []struct {
		name         string
		skills       []types.SkillMetadata
		wantErr      bool
		containsText []string
	}{
		{
			name:    "empty registry",
			skills:  []types.SkillMetadata{},
			wantErr: false,
			containsText: []string{
				"No skills installed yet.",
				"Use 'gskills add <url>' to install a skill.",
			},
		},
		{
			name: "single skill",
			skills: []types.SkillMetadata{
				{
					ID:          "test-skill@main",
					Name:        "test-skill",
					SourceURL:   "https://github.com/owner/repo/tree/main/test-skill",
					StorePath:   "/home/user/.gskills/skills/test-skill",
					UpdatedAt:   time.Now(),
					Version:     "main",
					Description: "Test skill",
				},
			},
			wantErr: false,
			containsText: []string{
				"test-skill",
				"Total: 1 skills",
				"https://github.com/owner/repo/tree/main/test-skill",
			},
		},
		{
			name: "multiple skills",
			skills: []types.SkillMetadata{
				{
					ID:          "prompt-engineer@main",
					Name:        "prompt-engineer",
					SourceURL:   "https://github.com/owner/repo/tree/main/prompt-engineer",
					StorePath:   "/home/user/.gskills/skills/prompt-engineer",
					UpdatedAt:   time.Now(),
					Version:     "main",
					Description: "Prompt engineering skill",
				},
				{
					ID:          "react-native-expert@v1.0.0",
					Name:        "react-native-expert",
					SourceURL:   "https://github.com/owner/repo/tree/main/react-native-expert",
					StorePath:   "/home/user/.gskills/skills/react-native-expert",
					UpdatedAt:   time.Now().Add(-time.Hour),
					Version:     "v1.0.0",
					Description: "React Native expert skill",
				},
			},
			wantErr: false,
			containsText: []string{
				"prompt-engineer",
				"react-native-expert",
				"Total: 2 skills",
				"https://github.com/owner/repo/tree/main/prompt-engineer",
				"https://github.com/owner/repo/tree/main/react-native-expert",
			},
		},
		{
			name: "skill with long URL",
			skills: []types.SkillMetadata{
				{
					ID:          "long-url-skill@main",
					Name:        "long-url-skill",
					SourceURL:   "https://github.com/owner/repository/tree/branch-name/very-long-path/to/skill-with-extremely-long-name",
					StorePath:   "/home/user/.gskills/skills/long-url-skill",
					UpdatedAt:   time.Now(),
					Version:     "main",
					Description: "Skill with long URL",
				},
			},
			wantErr: false,
			containsText: []string{
				"long-url-skill",
				"Total: 1 skills",
				"https://github.com/owner/repository/tree/branch-name/very-long-path/to/skill-with-extremely-long-name",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registryPath := createTestRegistry(t, tt.skills)
			cleanup := setupTestRegistry(t, registryPath)
			defer cleanup()

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := executeList()

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			_ = r.Close()

			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("executeList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for _, text := range tt.containsText {
				if !strings.Contains(output, text) {
					t.Errorf("executeList() output should contain %q, got:\n%s", text, output)
				}
			}
		})
	}
}
