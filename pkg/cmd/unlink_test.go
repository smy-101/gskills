package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smy-101/gskills/internal/add"
	"github.com/smy-101/gskills/internal/link"
	"github.com/smy-101/gskills/internal/types"
)

func TestUnlinkCmd_Args(t *testing.T) {
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
			errContains: "用法: gskills unlink <skill_name> [project_path]",
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
			errContains: "用法: gskills unlink <skill_name> [project_path]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := unlinkCmd.Args(nil, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("unlinkCmd.Args() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("unlinkCmd.Args() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestExecuteUnlink(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	homeDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	skillsDir := filepath.Join(homeDir, ".gskills", "skills", "unlink-test-skill")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create test skill directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("test skill"), 0644); err != nil {
		t.Fatalf("failed to create SKILL.md: %v", err)
	}

	testSkill := &types.SkillMetadata{
		ID:        "unlink-test-skill@main",
		Name:      "unlink-test-skill",
		Version:   "main",
		SourceURL: "https://example.com/test",
		StorePath: skillsDir,
		UpdatedAt: time.Now(),
	}

	if err := add.AddOrUpdateSkill(testSkill); err != nil {
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
			name:        "unlink from current directory",
			skillName:   "unlink-test-skill",
			projectPath: ".",
			setupFunc: func() (string, func()) {
				projectDir := t.TempDir()
				originalWd, _ := os.Getwd()
				os.Chdir(projectDir)

				linker := link.NewLinker()
				ctx := context.Background()
				if err := linker.LinkSkill(ctx, "unlink-test-skill", "."); err != nil {
					t.Fatalf("failed to setup link: %v", err)
				}

				return projectDir, func() {
					os.Chdir(originalWd)
				}
			},
			wantErr: false,
			validateFunc: func(projectDir string) error {
				targetPath := filepath.Join(projectDir, ".opencode", "skills", "unlink-test-skill")
				if _, err := os.Lstat(targetPath); err == nil {
					return err
				}
				return nil
			},
		},
		{
			name:        "unlink with explicit project path",
			skillName:   "unlink-test-skill",
			projectPath: "",
			setupFunc: func() (string, func()) {
				projectDir := t.TempDir()

				linker := link.NewLinker()
				ctx := context.Background()
				if err := linker.LinkSkill(ctx, "unlink-test-skill", projectDir); err != nil {
					t.Fatalf("failed to setup link: %v", err)
				}

				return projectDir, func() {}
			},
			wantErr: false,
			validateFunc: func(projectDir string) error {
				targetPath := filepath.Join(projectDir, ".opencode", "skills", "unlink-test-skill")
				if _, err := os.Lstat(targetPath); err == nil {
					return err
				}
				return nil
			},
		},
		{
			name:        "unlink non-existent skill",
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
			name:        "unlink skill not linked to project",
			skillName:   "unlink-test-skill",
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
			errorContains: "not linked to any projects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir, cleanup := tt.setupFunc()
			defer cleanup()

			var err error
			if tt.projectPath == "" {
				err = executeUnlink(tt.skillName, projectDir)
			} else {
				err = executeUnlink(tt.skillName, tt.projectPath)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("executeUnlink() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("executeUnlink() error = %v, want error containing %q", err, tt.errorContains)
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

func TestExecuteUnlink_DefaultToCurrentDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	homeDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	skillsDir := filepath.Join(homeDir, ".gskills", "skills", "default-unlink-test-skill")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create test skill directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("test skill"), 0644); err != nil {
		t.Fatalf("failed to create SKILL.md: %v", err)
	}

	testSkill := &types.SkillMetadata{
		ID:        "default-unlink-test-skill@main",
		Name:      "default-unlink-test-skill",
		Version:   "main",
		SourceURL: "https://example.com/test",
		StorePath: skillsDir,
		UpdatedAt: time.Now(),
	}

	if err := add.AddOrUpdateSkill(testSkill); err != nil {
		t.Fatalf("failed to add test skill to registry: %v", err)
	}

	projectDir := t.TempDir()
	originalWd, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(originalWd)

	linker := link.NewLinker()
	ctx := context.Background()
	if err := linker.LinkSkill(ctx, "default-unlink-test-skill", "."); err != nil {
		t.Fatalf("failed to setup link: %v", err)
	}

	targetPath := filepath.Join(projectDir, ".opencode", "skills", "default-unlink-test-skill")
	if _, err := os.Lstat(targetPath); err != nil {
		t.Fatalf("link not created: %v", err)
	}

	if err := executeUnlink("default-unlink-test-skill", "."); err != nil {
		t.Fatalf("executeUnlink() failed: %v", err)
	}

	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		t.Error("symlink was not removed")
	}

	skill, err := add.FindSkillByName("default-unlink-test-skill")
	if err != nil {
		t.Fatalf("failed to find skill: %v", err)
	}

	if len(skill.LinkedProjects) > 0 {
		t.Error("project was not removed from LinkedProjects")
	}
}
