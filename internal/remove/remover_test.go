package remove

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

func TestPromptForConfirmation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{
			name:    "confirm with y",
			input:   "y\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "confirm with yes",
			input:   "yes\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "confirm with YES",
			input:   "YES\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "confirm with Y",
			input:   "Y\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "reject with n",
			input:   "n\n",
			want:    false,
			wantErr: false,
		},
		{
			name:    "reject with no",
			input:   "no\n",
			want:    false,
			wantErr: false,
		},
		{
			name:    "reject with empty",
			input:   "\n",
			want:    false,
			wantErr: false,
		},
		{
			name:    "reject with spaces",
			input:   "   \n",
			want:    false,
			wantErr: false,
		},
		{
			name:    "reject with random text",
			input:   "maybe\n",
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r
			defer func() {
				os.Stdin = oldStdin
				r.Close()
				w.Close()
			}()

			// Write input
			w.WriteString(tt.input)
			w.Close()

			got, err := promptForConfirmation("test-skill")

			if (err != nil) != tt.wantErr {
				t.Errorf("promptForConfirmation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("promptForConfirmation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveSkillDirectory(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "remove existing directory",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				skillDir := filepath.Join(tmpDir, "skill")
				os.MkdirAll(filepath.Join(skillDir, "subdir"), 0755)
				os.WriteFile(filepath.Join(skillDir, "file.txt"), []byte("test"), 0644)
				return skillDir
			},
			wantErr: false,
		},
		{
			name: "remove non-existent directory",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "nonexistent")
			},
			wantErr: false, // RemoveAll doesn't error on non-existent path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storePath := tt.setupFunc()

			err := removeSkillDirectory(storePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("removeSkillDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}

			// Verify directory is removed
			if _, err := os.Stat(storePath); !os.IsNotExist(err) {
				t.Errorf("directory still exists after removal: %s", storePath)
			}
		})
	}
}

func TestRemoveSkillByName(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() (string, func())
		input       string
		skillName   string
		wantErr     bool
		errContains string
		verifyDir   bool
	}{
		{
			name: "remove skill with confirmation y",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()
				skillDir := filepath.Join(tmpDir, "skills", "test-skill")
				os.MkdirAll(skillDir, 0755)
				os.WriteFile(filepath.Join(skillDir, "file.txt"), []byte("test"), 0644)

				registryPath := filepath.Join(tmpDir, "skills.json")
				skills := []types.SkillMetadata{
					{
						ID:        "test-skill@main",
						Name:      "test-skill",
						SourceURL: "https://github.com/test/skill",
						StorePath: skillDir,
						Version:   "main",
						UpdatedAt: time.Now(),
					},
				}
				if err := add.SaveRegistryWithPath(registryPath, skills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return registryPath, func() {}
			},
			input:     "y\n",
			skillName: "test-skill",
			wantErr:   false,
			verifyDir: true,
		},
		{
			name: "remove skill with confirmation yes",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()
				skillDir := filepath.Join(tmpDir, "skills", "test-skill")
				os.MkdirAll(skillDir, 0755)

				registryPath := filepath.Join(tmpDir, "skills.json")
				skills := []types.SkillMetadata{
					{
						ID:        "test-skill@main",
						Name:      "test-skill",
						SourceURL: "https://github.com/test/skill",
						StorePath: skillDir,
						Version:   "main",
						UpdatedAt: time.Now(),
					},
				}
				if err := add.SaveRegistryWithPath(registryPath, skills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return registryPath, func() {}
			},
			input:     "yes\n",
			skillName: "test-skill",
			wantErr:   false,
			verifyDir: true,
		},
		{
			name: "cancel removal with n",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()
				skillDir := filepath.Join(tmpDir, "skills", "test-skill")
				os.MkdirAll(skillDir, 0755)

				registryPath := filepath.Join(tmpDir, "skills.json")
				skills := []types.SkillMetadata{
					{
						ID:        "test-skill@main",
						Name:      "test-skill",
						SourceURL: "https://github.com/test/skill",
						StorePath: skillDir,
						Version:   "main",
						UpdatedAt: time.Now(),
					},
				}
				if err := add.SaveRegistryWithPath(registryPath, skills); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return registryPath, func() {}
			},
			input:       "n\n",
			skillName:   "test-skill",
			wantErr:     true,
			errContains: "operation cancelled",
		},
		{
			name: "skill not found",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()
				registryPath := filepath.Join(tmpDir, "skills.json")
				if err := add.SaveRegistryWithPath(registryPath, []types.SkillMetadata{}); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return registryPath, func() {}
			},
			input:       "y\n",
			skillName:   "nonexistent-skill",
			wantErr:     true,
			errContains: "skill 'nonexistent-skill' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registryPath, teardown := tt.setupFunc()
			defer teardown()

			// Backup original registry path
			homeDir, _ := os.UserHomeDir()
			origRegistryPath := filepath.Join(homeDir, ".gskills", "skills.json")
			tmpBackup := filepath.Join(t.TempDir(), "backup-skills.json")
			if data, err := os.ReadFile(origRegistryPath); err == nil {
				os.WriteFile(tmpBackup, data, 0644)
			}

			// Restore original registry after test
			defer func() {
				if _, err := os.Stat(tmpBackup); err == nil {
					data, _ := os.ReadFile(tmpBackup)
					os.MkdirAll(filepath.Dir(origRegistryPath), 0755)
					os.WriteFile(origRegistryPath, data, 0644)
				}
			}()

			// Use test registry
			if err := os.MkdirAll(filepath.Dir(origRegistryPath), 0755); err != nil {
				t.Fatalf("failed to create registry dir: %v", err)
			}
			data, err := os.ReadFile(registryPath)
			if err != nil {
				t.Fatalf("failed to read test registry: %v", err)
			}
			if err := os.WriteFile(origRegistryPath, data, 0644); err != nil {
				t.Fatalf("failed to write test registry: %v", err)
			}

			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r
			defer func() {
				os.Stdin = oldStdin
				r.Close()
				w.Close()
			}()

			// Capture stdout
			oldStdout := os.Stdout
			var buf bytes.Buffer
			rOut, wOut, _ := os.Pipe()
			os.Stdout = wOut

			// Write input
			w.WriteString(tt.input)
			w.Close()

			// Run test in goroutine to close pipes
			done := make(chan error, 1)
			go func() {
				done <- RemoveSkillByName(tt.skillName)
			}()

			// Copy output
			go func() {
				io.Copy(&buf, rOut)
			}()

			err = <-done

			wOut.Close()
			os.Stdout = oldStdout

			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveSkillByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			// Verify skill is removed from registry if confirmed
			if strings.Contains(tt.input, "y\n") || strings.Contains(tt.input, "yes\n") {
				skills, err := add.LoadRegistry()
				if err != nil {
					t.Errorf("failed to load registry after removal: %v", err)
				}

				found := false
				for _, skill := range skills {
					if skill.Name == tt.skillName {
						found = true
						break
					}
				}

				if found {
					t.Errorf("skill still exists in registry after removal")
				}
			}
		})
	}
}
