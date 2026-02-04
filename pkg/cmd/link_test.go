package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smy-101/gskills/internal/link"
	"github.com/smy-101/gskills/internal/registry"
	"github.com/smy-101/gskills/internal/types"
)

func TestLinkCmd_Args(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "no arguments",
			args:        []string{},
			wantErr:     true,
			errContains: "用法: gskills link <skill_name> [path_to_project]",
		},
		{
			name:    "one argument - skill name only",
			args:    []string{"test-skill"},
			wantErr: false,
		},
		{
			name:    "two arguments - skill name and path",
			args:    []string{"test-skill", "/tmp/test"},
			wantErr: false,
		},
		{
			name:        "three arguments - too many",
			args:        []string{"test-skill", "/tmp/test", "extra"},
			wantErr:     true,
			errContains: "用法: gskills link <skill_name> [path_to_project]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := linkCmd.Args(nil, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("linkCmd.Args() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("linkCmd.Args() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestExecuteLink(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	homeDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	skillsDir := filepath.Join(homeDir, ".gskills", "skills", "test-skill")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create test skill directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("test skill"), 0644); err != nil {
		t.Fatalf("failed to create SKILL.md: %v", err)
	}

	testSkill := &types.SkillMetadata{
		ID:        "test-skill@main",
		Name:      "test-skill",
		Version:   "main",
		CommitSHA: "abc123",
		SourceURL: "https://example.com/test",
		StorePath: skillsDir,
		UpdatedAt: time.Now(),
	}

	if err := registry.AddOrUpdateSkill(testSkill); err != nil {
		t.Fatalf("failed to add test skill to registry: %v", err)
	}

	tests := []struct {
		name          string
		skillName     string
		projectPath   string
		setupFunc     func() (string, func())
		wantErr       bool
		errorContains string
		validateFunc  func(projectDir string) error
	}{
		{
			name:        "link with current directory",
			skillName:   "test-skill",
			projectPath: ".",
			setupFunc: func() (string, func()) {
				projectDir := t.TempDir()
				originalWd, _ := os.Getwd()
				os.Chdir(projectDir)
				return projectDir, func() {
					os.Chdir(originalWd)
				}
			},
			wantErr: false,
			validateFunc: func(projectDir string) error {
				targetPath := filepath.Join(projectDir, ".opencode", "skills", "test-skill")
				if _, err := os.Lstat(targetPath); err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:        "link with explicit project path",
			skillName:   "test-skill",
			projectPath: "",
			setupFunc: func() (string, func()) {
				projectDir := t.TempDir()
				return projectDir, func() {}
			},
			wantErr: false,
			validateFunc: func(projectDir string) error {
				targetPath := filepath.Join(projectDir, ".opencode", "skills", "test-skill")
				if _, err := os.Lstat(targetPath); err != nil {
					return err
				}
				return nil
			},
		},
		{
			name:        "link to non-existent skill",
			skillName:   "nonexistent-skill",
			projectPath: ".",
			setupFunc: func() (string, func()) {
				projectDir := t.TempDir()
				originalWd, _ := os.Getwd()
				os.Chdir(projectDir)
				return projectDir, func() {
					os.Chdir(originalWd)
				}
			},
			wantErr:       true,
			errorContains: "not found",
		},
		{
			name:        "link already linked skill",
			skillName:   "test-skill",
			projectPath: ".",
			setupFunc: func() (string, func()) {
				projectDir := t.TempDir()
				originalWd, _ := os.Getwd()
				os.Chdir(projectDir)

				linker := link.NewLinker()
				ctx := context.Background()
				_ = linker.LinkSkill(ctx, "test-skill", ".")

				return projectDir, func() {
					os.Chdir(originalWd)
				}
			},
			wantErr:       true,
			errorContains: "already linked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir, cleanup := tt.setupFunc()
			defer cleanup()

			var err error
			if tt.projectPath == "" {
				err = executeLink(tt.skillName, projectDir)
			} else {
				err = executeLink(tt.skillName, tt.projectPath)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("executeLink() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("executeLink() error = %v, want error containing %q", err, tt.errorContains)
				}
			}

			if !tt.wantErr && tt.validateFunc != nil {
				if err := tt.validateFunc(projectDir); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

func TestExecuteLink_DefaultToCurrentDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	homeDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	skillsDir := filepath.Join(homeDir, ".gskills", "skills", "default-test-skill")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create test skill directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("test skill"), 0644); err != nil {
		t.Fatalf("failed to create SKILL.md: %v", err)
	}

	testSkill := &types.SkillMetadata{
		ID:        "default-test-skill@main",
		Name:      "default-test-skill",
		Version:   "main",
		CommitSHA: "abc123",
		SourceURL: "https://example.com/test",
		StorePath: skillsDir,
		UpdatedAt: time.Now(),
	}

	if err := registry.AddOrUpdateSkill(testSkill); err != nil {
		t.Fatalf("failed to add test skill to registry: %v", err)
	}

	projectDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(originalWd)

	if err := executeLink("default-test-skill", "."); err != nil {
		t.Fatalf("executeLink() failed: %v", err)
	}

	targetPath := filepath.Join(projectDir, ".opencode", "skills", "default-test-skill")
	if info, err := os.Lstat(targetPath); err != nil {
		t.Errorf("symlink not created: %v", err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Error("target is not a symlink")
	}
}
