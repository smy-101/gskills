package initializer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectShell(t *testing.T) {
	tests := []struct {
		name          string
		shellEnv      string
		wantShell     Shell
		wantConfigHas bool
		wantErr       bool
	}{
		{
			name:      "zsh shell",
			shellEnv:  "/bin/zsh",
			wantShell: ShellZsh,
			wantErr:   false,
		},
		{
			name:      "bash shell",
			shellEnv:  "/bin/bash",
			wantShell: ShellBash,
			wantErr:   false,
		},
		{
			name:      "fish shell",
			shellEnv:  "/usr/local/bin/fish",
			wantShell: ShellFish,
			wantErr:   false,
		},
		{
			name:          "unknown shell",
			shellEnv:      "/bin/unknown",
			wantShell:     ShellUnknown,
			wantConfigHas: false,
			wantErr:       true,
		},
		{
			name:          "empty shell",
			shellEnv:      "",
			wantShell:     ShellUnknown,
			wantConfigHas: false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalShell := os.Getenv("SHELL")
			defer func() {
				if originalShell != "" {
					os.Setenv("SHELL", originalShell)
				} else {
					os.Unsetenv("SHELL")
				}
			}()

			if tt.shellEnv != "" {
				os.Setenv("SHELL", tt.shellEnv)
			} else {
				os.Unsetenv("SHELL")
			}

			gotShell, gotConfig, err := DetectShell()
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectShell() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotShell != tt.wantShell {
				t.Errorf("DetectShell() shell = %v, want %v", gotShell, tt.wantShell)
			}

			if !tt.wantErr && gotConfig == "" {
				t.Errorf("DetectShell() config path is empty, expected non-empty")
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME environment variable not set")
	}

	tests := []struct {
		name  string
		shell Shell
	}{
		{
			name:  "zsh config",
			shell: ShellZsh,
		},
		{
			name:  "bash config",
			shell: ShellBash,
		},
		{
			name:  "fish config",
			shell: ShellFish,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getConfigPath(tt.shell)
			if got == "" {
				t.Errorf("getConfigPath() returned empty string for shell %v", tt.shell)
			}

			if !filepath.IsAbs(got) {
				t.Errorf("getConfigPath() returned relative path: %s", got)
			}
		})
	}
}

func TestGeneratePATHExport(t *testing.T) {
	binPath := "/home/user/.gskills/bin"

	tests := []struct {
		name        string
		shell       Shell
		wantContain string
	}{
		{
			name:        "zsh export",
			shell:       ShellZsh,
			wantContain: "export PATH=",
		},
		{
			name:        "bash export",
			shell:       ShellBash,
			wantContain: "export PATH=",
		},
		{
			name:        "fish export",
			shell:       ShellFish,
			wantContain: "fish_add_path",
		},
		{
			name:        "unknown shell",
			shell:       ShellUnknown,
			wantContain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePATHExport(binPath, tt.shell)

			if tt.wantContain == "" {
				if got != "" {
					t.Errorf("GeneratePATHExport() = %q, want empty string", got)
				}
				return
			}

			if got == "" {
				t.Errorf("GeneratePATHExport() returned empty string")
				return
			}

			if !contains(got, tt.wantContain) {
				t.Errorf("GeneratePATHExport() = %q, does not contain %q", got, tt.wantContain)
			}

			if !contains(got, binPath) {
				t.Errorf("GeneratePATHExport() = %q, does not contain binPath %q", got, binPath)
			}
		})
	}
}

func TestIsInPATH(t *testing.T) {
	originalPath := os.Getenv("PATH")
	defer func() {
		if originalPath != "" {
			os.Setenv("PATH", originalPath)
		} else {
			os.Unsetenv("PATH")
		}
	}()

	tests := []struct {
		name    string
		pathEnv string
		binPath string
		want    bool
	}{
		{
			name:    "path exists",
			pathEnv: "/usr/bin:/home/user/.gskills/bin:/usr/local/bin",
			binPath: "/home/user/.gskills/bin",
			want:    true,
		},
		{
			name:    "path does not exist",
			pathEnv: "/usr/bin:/usr/local/bin",
			binPath: "/home/user/.gskills/bin",
			want:    false,
		},
		{
			name:    "empty path env",
			pathEnv: "",
			binPath: "/home/user/.gskills/bin",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pathEnv != "" {
				os.Setenv("PATH", tt.pathEnv)
			} else {
				os.Unsetenv("PATH")
			}

			if got := IsInPATH(tt.binPath); got != tt.want {
				t.Errorf("IsInPATH() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasExportInConfig(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		configContent string
		binPath       string
		want          bool
		wantErr       bool
	}{
		{
			name:          "export exists with full path",
			configContent: `export PATH="/home/user/.gskills/bin:$PATH"`,
			binPath:       "/home/user/.gskills/bin",
			want:          true,
		},
		{
			name: "export exists with gskills/bin",
			configContent: `# gskills
export PATH="$HOME/.gskills/bin:$PATH"`,
			binPath: "/home/user/.gskills/bin",
			want:    true,
		},
		{
			name:          "export does not exist",
			configContent: `export PATH="/usr/bin:$PATH"`,
			binPath:       "/home/user/.gskills/bin",
			want:          false,
		},
		{
			name:          "file does not exist",
			configContent: "",
			binPath:       "/home/user/.gskills/bin",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			if tt.configContent != "" {
				configPath = filepath.Join(tmpDir, "config")
				if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
					t.Fatalf("Failed to write test config: %v", err)
				}
			} else {
				configPath = filepath.Join(tmpDir, "nonexistent")
			}

			got, err := HasExportInConfig(configPath, tt.binPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasExportInConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("HasExportInConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME environment variable not set")
	}

	init := New()
	if init == nil {
		t.Fatal("New() returned nil")
	}

	expectedBinDir := filepath.Join(home, ".gskills", "bin")
	if init.GetBinDir() != expectedBinDir {
		t.Errorf("New().GetBinDir() = %v, want %v", init.GetBinDir(), expectedBinDir)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
