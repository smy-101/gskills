package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestInitViper(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T) string
		checkFunc func(*testing.T, string)
		cleanup   bool
	}{
		{
			name: "creates new config file when it doesn't exist",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				homeDir := t.TempDir()
				return homeDir
			},
			checkFunc: func(t *testing.T, homeDir string) {
				t.Helper()
				configPath := filepath.Join(homeDir, ".gskills", "config.json")

				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					t.Errorf("config file was not created at %s", configPath)
					return
				}

				data, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatalf("failed to read config file: %v", err)
				}

				var config map[string]interface{}
				if err := json.Unmarshal(data, &config); err != nil {
					t.Fatalf("failed to unmarshal config: %v", err)
				}

				if githubToken, ok := config["github_token"]; !ok || githubToken != "" {
					t.Errorf("expected github_token to be empty string, got %v", githubToken)
				}

				if proxy, ok := config["proxy"]; !ok || proxy != "" {
					t.Errorf("expected proxy to be empty string, got %v", proxy)
				}
			},
			cleanup: true,
		},
		{
			name: "reads existing config file",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				homeDir := t.TempDir()
				configDir := filepath.Join(homeDir, ".gskills")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatalf("failed to create config dir: %v", err)
				}

				configPath := filepath.Join(configDir, "config.json")
				testConfig := map[string]interface{}{
					"github_token": "test_token_123",
					"proxy":        "http://proxy.example.com:8080",
				}

				data, err := json.MarshalIndent(testConfig, "", "  ")
				if err != nil {
					t.Fatalf("failed to marshal test config: %v", err)
				}

				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Fatalf("failed to write test config: %v", err)
				}

				return homeDir
			},
			checkFunc: func(t *testing.T, homeDir string) {
				t.Helper()

				if token := viper.GetString("github_token"); token != "test_token_123" {
					t.Errorf("expected github_token to be 'test_token_123', got '%s'", token)
				}

				if proxy := viper.GetString("proxy"); proxy != "http://proxy.example.com:8080" {
					t.Errorf("expected proxy to be 'http://proxy.example.com:8080', got '%s'", proxy)
				}
			},
			cleanup: true,
		},
		{
			name: "reads existing config file and uses viper defaults",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				homeDir := t.TempDir()
				configDir := filepath.Join(homeDir, ".gskills")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatalf("failed to create config dir: %v", err)
				}

				return homeDir
			},
			checkFunc: func(t *testing.T, homeDir string) {
				t.Helper()

				if token := viper.GetString("github_token"); token != "" {
					t.Errorf("expected github_token to be empty string by default, got '%s'", token)
				}

				if proxy := viper.GetString("proxy"); proxy != "" {
					t.Errorf("expected proxy to be empty string by default, got '%s'", proxy)
				}

				configPath := filepath.Join(homeDir, ".gskills", "config.json")
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					t.Errorf("config file should be created with defaults")
				}
			},
			cleanup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc == nil {
				t.Fatal("setupFunc is required")
			}

			homeDir := tt.setupFunc(t)

			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", homeDir)
			defer func() {
				if oldHome != "" {
					os.Setenv("HOME", oldHome)
				} else {
					os.Unsetenv("HOME")
				}
			}()

			viper.Reset()

			initViper()

			if tt.checkFunc != nil {
				tt.checkFunc(t, homeDir)
			}
		})
	}
}

func TestMain(t *testing.T) {
	oldHome := os.Getenv("HOME")
	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	defer func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	viper.Reset()

	t.Run("main executes without panicking", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("main() panicked: %v", r)
			}
		}()

		os.Args = []string{"gskills", "--help"}
		main()
	})
}
