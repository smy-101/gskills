package link

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smy-101/gskills/internal/registry"
	"github.com/smy-101/gskills/internal/types"
)

func TestLinker_LinkSkill(t *testing.T) {
	tests := []struct {
		name          string
		skillName     string
		projectPath   string
		setupFunc     func() func()
		wantErr       bool
		errorContains string
	}{
		{
			name:          "empty skill name",
			skillName:     "",
			projectPath:   "/tmp/test",
			setupFunc:     func() func() { return func() {} },
			wantErr:       true,
			errorContains: "skill name cannot be empty",
		},
		{
			name:          "empty project path",
			skillName:     "test-skill",
			projectPath:   "",
			setupFunc:     func() func() { return func() {} },
			wantErr:       true,
			errorContains: "project path cannot be empty",
		},
		{
			name:          "skill not found",
			skillName:     "test-skill",
			projectPath:   t.TempDir(),
			setupFunc:     func() func() { return func() {} },
			wantErr:       true,
			errorContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teardown := tt.setupFunc()
			defer teardown()

			linker := NewLinker()
			err := linker.LinkSkill(context.Background(), tt.skillName, tt.projectPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("LinkSkill() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("LinkSkill() error = %v, want error containing %v", err, tt.errorContains)
				}
			}
		})
	}
}

func TestLinker_getSkillPath(t *testing.T) {
	homeDir := t.TempDir()

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	skillsDir := filepath.Join(homeDir, ".gskills", "skills")
	testSkillDir := filepath.Join(skillsDir, "test-skill")
	if err := os.MkdirAll(testSkillDir, 0755); err != nil {
		t.Fatalf("failed to create test skill directory: %v", err)
	}

	tests := []struct {
		name          string
		skillName     string
		setupFunc     func() func()
		wantErr       bool
		errorContains string
	}{
		{
			name:      "existing skill",
			skillName: "test-skill",
			setupFunc: func() func() { return func() {} },
			wantErr:   false,
		},
		{
			name:          "nonexistent skill",
			skillName:     "nonexistent-skill",
			setupFunc:     func() func() { return func() {} },
			wantErr:       true,
			errorContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teardown := tt.setupFunc()
			defer teardown()

			linker := NewLinker()
			path, err := linker.getSkillPath(tt.skillName)

			if (err != nil) != tt.wantErr {
				t.Errorf("getSkillPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && !filepath.IsAbs(path) {
				t.Errorf("getSkillPath() returned non-absolute path: %s", path)
			}
		})
	}
}

func TestLinker_validateProjectPath(t *testing.T) {
	tests := []struct {
		name          string
		projectPath   string
		setupFunc     func() func()
		wantErr       bool
		errorContains string
	}{
		{
			name:        "existing directory",
			projectPath: t.TempDir(),
			setupFunc:   func() func() { return func() {} },
			wantErr:     false,
		},
		{
			name:          "nonexistent path",
			projectPath:   "/tmp/nonexistent-path-test-12345",
			setupFunc:     func() func() { return func() {} },
			wantErr:       true,
			errorContains: "does not exist",
		},
		{
			name:          "file instead of directory",
			projectPath:   "",
			setupFunc:     func() func() { return func() {} },
			wantErr:       true,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teardown := tt.setupFunc()
			defer teardown()

			linker := NewLinker()
			err := linker.validateProjectPath(tt.projectPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateProjectPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("validateProjectPath() error = %v, want error containing %v", err, tt.errorContains)
				}
			}
		})
	}
}

func TestLinker_checkPathExists(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (string, func())
		want    bool
		wantErr bool
	}{
		{
			name: "existing path",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				testPath := filepath.Join(tmpDir, "test")
				if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return testPath, func() {}
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "nonexistent path",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				testPath := filepath.Join(tmpDir, "nonexistent")
				return testPath, func() {}
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, teardown := tt.setup()
			defer teardown()

			linker := NewLinker()
			exists, err := linker.checkPathExists(path)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkPathExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if exists != tt.want {
				t.Errorf("checkPathExists() = %v, want %v", exists, tt.want)
			}
		})
	}
}

func TestLinker_LinkSkill_Integration(t *testing.T) {
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

	projectDir := t.TempDir()

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

	linker := NewLinker()
	ctx := context.Background()

	if err := linker.LinkSkill(ctx, "test-skill", projectDir); err != nil {
		t.Fatalf("LinkSkill() failed: %v", err)
	}

	targetPath := filepath.Join(projectDir, ".opencode/skills", "test-skill")
	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Errorf("symlink not created at %s: %v", targetPath, err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("target is not a symlink")
	}

	actualLink, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}

	if actualLink != skillsDir {
		t.Errorf("symlink points to %s, want %s", actualLink, skillsDir)
	}

	updatedSkill, err := registry.FindSkillByName("test-skill")
	if err != nil {
		t.Fatalf("failed to find updated skill: %v", err)
	}

	if updatedSkill.LinkedProjects == nil {
		t.Errorf("LinkedProjects not updated in registry")
	}

	if _, linked := updatedSkill.LinkedProjects[projectDir]; !linked {
		t.Errorf("Project not found in LinkedProjects")
	}

	os.Remove(targetPath)
}

func TestLinkError(t *testing.T) {
	tests := []struct {
		name       string
		errType    ErrorType
		message    string
		wrappedErr error
	}{
		{
			name:       "invalid path error",
			errType:    ErrorTypeInvalidPath,
			message:    "test message",
			wrappedErr: nil,
		},
		{
			name:       "symlink exists error",
			errType:    ErrorTypeSymlinkExists,
			message:    "test message",
			wrappedErr: nil,
		},
		{
			name:       "skill not found error",
			errType:    ErrorTypeSkillNotFound,
			message:    "test message",
			wrappedErr: nil,
		},
		{
			name:       "filesystem error with wrapped error",
			errType:    ErrorTypeFilesystem,
			message:    "test message",
			wrappedErr: &os.PathError{Op: "open", Path: "/test", Err: os.ErrNotExist},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linkErr := &LinkError{
				Type:    tt.errType,
				Message: tt.message,
				Err:     tt.wrappedErr,
			}

			if linkErr.Error() == "" {
				t.Error("Error() returned empty string")
			}

			if tt.wrappedErr != nil {
				if linkErr.Unwrap() != tt.wrappedErr {
					t.Errorf("Unwrap() = %v, want %v", linkErr.Unwrap(), tt.wrappedErr)
				}
			}

			targetErr := &LinkError{Type: tt.errType}
			if !linkErr.Is(targetErr) {
				t.Errorf("Is() should return true for same error type")
			}
		})
	}
}

func TestMain(m *testing.M) {
	exitCode := m.Run()
	os.Exit(exitCode)
}

func BenchmarkLinker_getSkillPath(b *testing.B) {
	homeDir := b.TempDir()

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	skillsDir := filepath.Join(homeDir, ".gskills", "skills", "test-skill")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		b.Fatalf("failed to create test skill directory: %v", err)
	}

	linker := NewLinker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = linker.getSkillPath("test-skill")
	}
}

func BenchmarkLinker_validateProjectPath(b *testing.B) {
	projectDir := b.TempDir()

	linker := NewLinker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = linker.validateProjectPath(projectDir)
	}
}

func BenchmarkLinker_checkPathExists(b *testing.B) {
	tmpDir := b.TempDir()
	testPath := filepath.Join(tmpDir, "test")
	if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
		b.Fatalf("failed to create test file: %v", err)
	}

	linker := NewLinker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = linker.checkPathExists(testPath)
	}
}
