package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smy-101/gskills/internal/types"
)

func TestLoadRegistry(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() func()
		want    int
		wantErr bool
	}{
		{
			name: "non-existent registry returns empty list",
			setup: func() func() {
				home := t.TempDir()
				oldGetHomeDir := os.Getenv("HOME")
				os.Setenv("HOME", home)
				return func() {
					if oldGetHomeDir != "" {
						os.Setenv("HOME", oldGetHomeDir)
					} else {
						os.Unsetenv("HOME")
					}
				}
			},
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			got, err := LoadRegistry()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRegistry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("LoadRegistry() got %d skills, want %d", len(got), tt.want)
			}
		})
	}
}

func TestSaveRegistry(t *testing.T) {
	home := t.TempDir()
	gskillsDir := filepath.Join(home, ".gskills")
	err := os.MkdirAll(gskillsDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .gskills directory: %v", err)
	}

	oldGetHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer func() {
		if oldGetHomeDir != "" {
			os.Setenv("HOME", oldGetHomeDir)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	skills := []types.SkillMetadata{
		{
			ID:        "test@main",
			Name:      "test",
			Version:   "main",
			SourceURL: "https://github.com/test/test",
			StorePath: filepath.Join(home, ".gskills", "skills", "test"),
			UpdatedAt: time.Now(),
		},
	}

	err = SaveRegistry(skills)
	if err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	registryPath := filepath.Join(home, ".gskills", "skills.json")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Errorf("SaveRegistry() registry file not created at %s", registryPath)
	}

	loaded, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}

	if len(loaded) != len(skills) {
		t.Errorf("LoadRegistry() got %d skills, want %d", len(loaded), len(skills))
	}

	if loaded[0].ID != skills[0].ID {
		t.Errorf("LoadRegistry() got ID %s, want %s", loaded[0].ID, skills[0].ID)
	}
}

func TestAddOrUpdateSkill(t *testing.T) {
	home := t.TempDir()
	gskillsDir := filepath.Join(home, ".gskills")
	err := os.MkdirAll(gskillsDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .gskills directory: %v", err)
	}

	oldGetHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer func() {
		if oldGetHomeDir != "" {
			os.Setenv("HOME", oldGetHomeDir)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	tests := []struct {
		name    string
		skill   *types.SkillMetadata
		wantErr bool
	}{
		{
			name: "add new skill",
			skill: &types.SkillMetadata{
				ID:        "test@main",
				Name:      "test",
				Version:   "main",
				CommitSHA: "abc123",
				SourceURL: "https://github.com/test/test",
				StorePath: filepath.Join(home, ".gskills", "skills", "test"),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "update existing skill",
			skill: &types.SkillMetadata{
				ID:        "test@main",
				Name:      "test",
				Version:   "main",
				CommitSHA: "def456",
				SourceURL: "https://github.com/test/test",
				StorePath: filepath.Join(home, ".gskills", "skills", "test-updated"),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddOrUpdateSkill(tt.skill)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddOrUpdateSkill() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				skills, err := LoadRegistry()
				if err != nil {
					t.Fatalf("LoadRegistry() error = %v", err)
				}

				found := false
				for _, s := range skills {
					if s.ID == tt.skill.ID {
						found = true
						if tt.skill.Name != "" && s.Name != tt.skill.Name {
							t.Errorf("Skill Name = %s, want %s", s.Name, tt.skill.Name)
						}
						break
					}
				}

				if !found && tt.skill.ID != "" {
					t.Errorf("AddOrUpdateSkill() skill not found in registry")
				}
			}
		})
	}
}

func TestFindSkillByName(t *testing.T) {
	home := t.TempDir()
	gskillsDir := filepath.Join(home, ".gskills")
	err := os.MkdirAll(gskillsDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .gskills directory: %v", err)
	}

	oldGetHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer func() {
		if oldGetHomeDir != "" {
			os.Setenv("HOME", oldGetHomeDir)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	skills := []types.SkillMetadata{
		{
			ID:        "test@main",
			Name:      "test",
			Version:   "main",
			SourceURL: "https://github.com/test/test",
			StorePath: filepath.Join(home, ".gskills", "skills", "test"),
			UpdatedAt: time.Now(),
		},
	}

	err = SaveRegistry(skills)
	if err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	tests := []struct {
		name      string
		skillName string
		wantErr   bool
	}{
		{
			name:      "find existing skill",
			skillName: "test",
			wantErr:   false,
		},
		{
			name:      "skill not found",
			skillName: "nonexistent",
			wantErr:   true,
		},
		{
			name:      "empty skill name",
			skillName: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindSkillByName(tt.skillName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindSkillByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Name != tt.skillName {
					t.Errorf("FindSkillByName() got Name %s, want %s", got.Name, tt.skillName)
				}
			}
		})
	}
}

func TestRemoveSkill(t *testing.T) {
	home := t.TempDir()
	gskillsDir := filepath.Join(home, ".gskills")
	err := os.MkdirAll(gskillsDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .gskills directory: %v", err)
	}

	oldGetHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer func() {
		if oldGetHomeDir != "" {
			os.Setenv("HOME", oldGetHomeDir)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	skills := []types.SkillMetadata{
		{
			ID:        "test@main",
			Name:      "test",
			Version:   "main",
			SourceURL: "https://github.com/test/test",
			StorePath: filepath.Join(home, ".gskills", "skills", "test"),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "another@main",
			Name:      "another",
			Version:   "main",
			SourceURL: "https://github.com/test/another",
			StorePath: filepath.Join(home, ".gskills", "skills", "another"),
			UpdatedAt: time.Now(),
		},
	}

	err = SaveRegistry(skills)
	if err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	tests := []struct {
		name      string
		skillID   string
		wantErr   bool
		wantCount int
	}{
		{
			name:      "remove existing skill",
			skillID:   "test@main",
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:      "remove non-existent skill",
			skillID:   "nonexistent@main",
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:      "empty skill ID",
			skillID:   "",
			wantErr:   true,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RemoveSkill(tt.skillID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveSkill() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				skills, err := LoadRegistry()
				if err != nil {
					t.Fatalf("LoadRegistry() error = %v", err)
				}

				if len(skills) != tt.wantCount {
					t.Errorf("RemoveSkill() got %d skills, want %d", len(skills), tt.wantCount)
				}
			}
		})
	}
}

func TestUpdateSkill(t *testing.T) {
	home := t.TempDir()
	gskillsDir := filepath.Join(home, ".gskills")
	err := os.MkdirAll(gskillsDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .gskills directory: %v", err)
	}

	oldGetHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer func() {
		if oldGetHomeDir != "" {
			os.Setenv("HOME", oldGetHomeDir)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	skills := []types.SkillMetadata{
		{
			ID:        "test@main",
			Name:      "test",
			Version:   "main",
			SourceURL: "https://github.com/test/test",
			StorePath: filepath.Join(home, ".gskills", "skills", "test"),
			UpdatedAt: time.Now(),
		},
	}

	err = SaveRegistry(skills)
	if err != nil {
		t.Fatalf("SaveRegistry() error = %v", err)
	}

	tests := []struct {
		name    string
		skill   *types.SkillMetadata
		wantErr bool
	}{
		{
			name: "update existing skill",
			skill: &types.SkillMetadata{
				ID:        "test@main",
				Name:      "test-updated",
				Version:   "main",
				SourceURL: "https://github.com/test/test",
				StorePath: filepath.Join(home, ".gskills", "skills", "test"),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "update non-existent skill",
			skill: &types.SkillMetadata{
				ID:        "nonexistent@main",
				Name:      "nonexistent",
				Version:   "main",
				SourceURL: "https://github.com/test/nonexistent",
				StorePath: filepath.Join(home, ".gskills", "skills", "nonexistent"),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateSkill(tt.skill)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateSkill() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.skill.ID == "test@main" {
				skills, err := LoadRegistry()
				if err != nil {
					t.Fatalf("LoadRegistry() error = %v", err)
				}

				found := false
				for _, s := range skills {
					if s.ID == tt.skill.ID {
						found = true
						if s.Name != tt.skill.Name {
							t.Errorf("UpdateSkill() Name = %s, want %s", s.Name, tt.skill.Name)
						}
						break
					}
				}

				if !found {
					t.Errorf("UpdateSkill() skill not found in registry")
				}
			}
		})
	}
}
