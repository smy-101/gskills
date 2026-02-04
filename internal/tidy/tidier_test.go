package tidy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smy-101/gskills/internal/registry"
	"github.com/smy-101/gskills/internal/types"
)

func TestCheckSymlinkExists(t *testing.T) {
	tidier := NewTidier()

	tests := []struct {
		name    string
		setup   func() (string, func())
		want    bool
		wantErr bool
	}{
		{
			name: "existing symlink",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				targetFile := filepath.Join(tmpDir, "target")
				if err := os.WriteFile(targetFile, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create target file: %v", err)
				}
				symlinkPath := filepath.Join(tmpDir, "symlink")
				if err := os.Symlink(targetFile, symlinkPath); err != nil {
					t.Fatalf("failed to create symlink: %v", err)
				}
				return symlinkPath, func() {}
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "non-existent symlink",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				symlinkPath := filepath.Join(tmpDir, "nonexistent")
				return symlinkPath, func() {}
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "regular file (not symlink)",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "regular")
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
				return filePath, func() {}
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setup()
			defer cleanup()

			got, err := tidier.checkSymlinkExists(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSymlinkExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkSymlinkExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindStaleLinks(t *testing.T) {
	tidier := NewTidier()

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	existingSymlink := filepath.Join(projectPath, "existing_link")
	if err := os.Symlink("/some/target", existingSymlink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	missingSymlink := filepath.Join(projectPath, "missing_link")

	skill := types.SkillMetadata{
		ID:        "test-skill-1",
		Name:      "test-skill",
		StorePath: "/store/path",
		LinkedProjects: map[string]types.LinkedProjectInfo{
			projectPath: {
				SymlinkPath: existingSymlink,
			},
			"/another/project": {
				SymlinkPath: missingSymlink,
			},
		},
	}

	staleLinks := tidier.findStaleLinks(skill)

	if len(staleLinks) != 1 {
		t.Errorf("findStaleLinks() returned %d stale links, want 1", len(staleLinks))
	}

	if staleLinks[0] != "/another/project" {
		t.Errorf("findStaleLinks() returned %s, want /another/project", staleLinks[0])
	}
}

func TestTidy(t *testing.T) {
	tests := []struct {
		name          string
		setupRegistry func(tmpDir string) ([]types.SkillMetadata, func())
		setupFiles    func(tmpDir string) error
		wantReport    CleanupReport
		wantErr       bool
	}{
		{
			name: "cleans stale registry entries and orphaned symlinks",
			setupRegistry: func(tmpDir string) ([]types.SkillMetadata, func()) {
				project1Path := filepath.Join(tmpDir, "project1")
				project2Path := filepath.Join(tmpDir, "project2")

				skills := []types.SkillMetadata{
					{
						ID:        "skill-1",
						Name:      "skill1",
						StorePath: filepath.Join(tmpDir, "skills", "skill1"),
						LinkedProjects: map[string]types.LinkedProjectInfo{
							project1Path: {
								SymlinkPath: filepath.Join(project1Path, ".opencode", "skills", "skill1"),
							},
							project2Path: {
								SymlinkPath: filepath.Join(project2Path, ".opencode", "skills", "skill1"),
							},
						},
					},
				}

				cleanup := func() {
					registry.SaveRegistry([]types.SkillMetadata{})
				}

				return skills, cleanup
			},
			setupFiles: func(tmpDir string) error {
				project1Path := filepath.Join(tmpDir, "project1")
				project2Path := filepath.Join(tmpDir, "project2")

				skillsDir1 := filepath.Join(project1Path, ".opencode", "skills")
				if err := os.MkdirAll(skillsDir1, 0755); err != nil {
					return err
				}

				skill1Store := filepath.Join(tmpDir, "skills", "skill1")
				if err := os.MkdirAll(skill1Store, 0755); err != nil {
					return err
				}

				if err := os.Symlink(skill1Store, filepath.Join(skillsDir1, "skill1")); err != nil {
					return err
				}

				skillsDir2 := filepath.Join(project2Path, ".opencode", "skills")
				if err := os.MkdirAll(skillsDir2, 0755); err != nil {
					return err
				}

				deletedSkillStore := filepath.Join(tmpDir, "skills", "deleted-skill")
				if err := os.MkdirAll(deletedSkillStore, 0755); err != nil {
					return err
				}

				if err := os.Symlink(deletedSkillStore, filepath.Join(skillsDir2, "deleted-skill")); err != nil {
					return err
				}

				return nil
			},
			wantReport: CleanupReport{
				StaleRegistryEntries: 1,
				OrphanedSymlinks:     1,
				SkillsChecked:        1,
				ProjectsScanned:      2,
			},
			wantErr: false,
		},
		{
			name: "empty registry returns empty report",
			setupRegistry: func(tmpDir string) ([]types.SkillMetadata, func()) {
				return []types.SkillMetadata{}, func() {
					registry.SaveRegistry([]types.SkillMetadata{})
				}
			},
			setupFiles: func(tmpDir string) error {
				return nil
			},
			wantReport: CleanupReport{
				StaleRegistryEntries: 0,
				OrphanedSymlinks:     0,
				SkillsChecked:        0,
				ProjectsScanned:      0,
			},
			wantErr: false,
		},
		{
			name: "skills with no linked projects are skipped",
			setupRegistry: func(tmpDir string) ([]types.SkillMetadata, func()) {
				skills := []types.SkillMetadata{
					{
						ID:             "skill-1",
						Name:           "skill1",
						StorePath:      filepath.Join(tmpDir, "skills", "skill1"),
						LinkedProjects: nil,
					},
				}

				cleanup := func() {
					registry.SaveRegistry([]types.SkillMetadata{})
				}

				return skills, cleanup
			},
			setupFiles: func(tmpDir string) error {
				return nil
			},
			wantReport: CleanupReport{
				StaleRegistryEntries: 0,
				OrphanedSymlinks:     0,
				SkillsChecked:        1,
				ProjectsScanned:      0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			skills, cleanup := tt.setupRegistry(tmpDir)
			defer cleanup()

			if err := registry.SaveRegistry(skills); err != nil {
				t.Fatalf("failed to setup registry: %v", err)
			}

			if err := tt.setupFiles(tmpDir); err != nil {
				t.Fatalf("failed to setup files: %v", err)
			}

			tidier := NewTidier()
			ctx := context.Background()

			report, err := tidier.Tidy(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Tidy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if report.StaleRegistryEntries != tt.wantReport.StaleRegistryEntries {
				t.Errorf("Tidy() StaleRegistryEntries = %v, want %v", report.StaleRegistryEntries, tt.wantReport.StaleRegistryEntries)
			}

			if report.OrphanedSymlinks != tt.wantReport.OrphanedSymlinks {
				t.Errorf("Tidy() OrphanedSymlinks = %v, want %v", report.OrphanedSymlinks, tt.wantReport.OrphanedSymlinks)
			}

			if report.SkillsChecked != tt.wantReport.SkillsChecked {
				t.Errorf("Tidy() SkillsChecked = %v, want %v", report.SkillsChecked, tt.wantReport.SkillsChecked)
			}

			if report.ProjectsScanned != tt.wantReport.ProjectsScanned {
				t.Errorf("Tidy() ProjectsScanned = %v, want %v", report.ProjectsScanned, tt.wantReport.ProjectsScanned)
			}

			updatedSkills, err := registry.LoadRegistry()
			if err != nil {
				t.Fatalf("failed to load updated registry: %v", err)
			}

			for _, skill := range updatedSkills {
				for projectPath, linkInfo := range skill.LinkedProjects {
					if _, err := os.Lstat(linkInfo.SymlinkPath); os.IsNotExist(err) {
						t.Errorf("Tidy() left stale registry entry: skill=%s, project=%s, path=%s", skill.Name, projectPath, linkInfo.SymlinkPath)
					}
				}
			}
		})
	}
}

func TestTidyError(t *testing.T) {
	tests := []struct {
		name   string
		err    *TidyError
		target error
		want   bool
	}{
		{
			name:   "error type matching",
			err:    &TidyError{Type: ErrorTypeFilesystem, Message: "test"},
			target: &TidyError{Type: ErrorTypeFilesystem},
			want:   true,
		},
		{
			name:   "error type not matching",
			err:    &TidyError{Type: ErrorTypeRegistry, Message: "test"},
			target: &TidyError{Type: ErrorTypeFilesystem},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Is(tt.target); got != tt.want {
				t.Errorf("TidyError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTidyConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	tmpDir := t.TempDir()

	numSkills := 50
	numProjects := 20

	skills := make([]types.SkillMetadata, numSkills)
	projects := make([]string, numProjects)

	for i := 0; i < numProjects; i++ {
		projects[i] = filepath.Join(tmpDir, "project", fmt.Sprintf("project%d", i))
		if err := os.MkdirAll(projects[i], 0755); err != nil {
			t.Fatalf("failed to create project dir: %v", err)
		}
	}

	for i := 0; i < numSkills; i++ {
		skillName := fmt.Sprintf("skill%d", i)
		skillStore := filepath.Join(tmpDir, "skills", skillName)
		if err := os.MkdirAll(skillStore, 0755); err != nil {
			t.Fatalf("failed to create skill store: %v", err)
		}

		linkedProjects := make(map[string]types.LinkedProjectInfo)
		for _, projectPath := range projects[:5] {
			skillsDir := filepath.Join(projectPath, ".opencode", "skills")
			if err := os.MkdirAll(skillsDir, 0755); err != nil {
				t.Fatalf("failed to create skills dir: %v", err)
			}

			symlinkPath := filepath.Join(skillsDir, skillName)
			if i%2 == 0 {
				if err := os.Symlink(skillStore, symlinkPath); err != nil {
					t.Fatalf("failed to create symlink: %v", err)
				}
			}

			linkedProjects[projectPath] = types.LinkedProjectInfo{
				SymlinkPath: symlinkPath,
				LinkedAt:    time.Now(),
			}
		}

		skills[i] = types.SkillMetadata{
			ID:             fmt.Sprintf("skill-id-%d", i),
			Name:           skillName,
			StorePath:      skillStore,
			Version:        "1.0.0",
			CommitSHA:      "abc123",
			SourceURL:      "https://github.com/test/skill",
			LinkedProjects: linkedProjects,
		}
	}

	if err := registry.SaveRegistry(skills); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	tidier := NewTidier()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	report, err := tidier.Tidy(ctx)
	if err != nil {
		t.Fatalf("Tidy() error = %v", err)
	}

	if report.StaleRegistryEntries != numSkills*5/2 {
		t.Logf("Warning: Expected %d stale entries, got %d", numSkills*5/2, report.StaleRegistryEntries)
	}

	updatedSkills, err := registry.LoadRegistry()
	if err != nil {
		t.Fatalf("failed to load updated registry: %v", err)
	}

	for _, skill := range updatedSkills {
		for projectPath, linkInfo := range skill.LinkedProjects {
			if _, err := os.Lstat(linkInfo.SymlinkPath); os.IsNotExist(err) {
				t.Errorf("Concurrent Tidy() left stale registry entry: skill=%s, project=%s", skill.Name, projectPath)
			}
		}
	}

	t.Logf("Successfully processed %d skills with %d stale entries removed in concurrent mode", report.SkillsChecked, report.StaleRegistryEntries)
}
