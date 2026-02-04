package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func setupConfigTest(t *testing.T) (func(), string) {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	originalConfigFile := viper.ConfigFileUsed()

	viper.Reset()
	viper.SetConfigFile(configPath)
	viper.Set("github_token", "")
	viper.Set("proxy", "")

	cleanup := func() {
		viper.Reset()
		if originalConfigFile != "" {
			viper.SetConfigFile(originalConfigFile)
		}
	}

	return cleanup, tempDir
}

func TestExecuteConfigGet(t *testing.T) {
	t.Run("valid key with value", func(t *testing.T) {
		cleanup, _ := setupConfigTest(t)
		defer cleanup()

		viper.Set("github_token", "test-token-123")

		err := executeConfigGet("github_token")
		if err != nil {
			t.Errorf("executeConfigGet() error = %v", err)
		}
	})

	t.Run("valid key without value", func(t *testing.T) {
		cleanup, _ := setupConfigTest(t)
		defer cleanup()

		viper.Set("proxy", "")

		err := executeConfigGet("proxy")
		if err != nil {
			t.Errorf("executeConfigGet() error = %v", err)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		cleanup, _ := setupConfigTest(t)
		defer cleanup()

		err := executeConfigGet("invalid_key")
		if err == nil {
			t.Error("executeConfigGet() expected error for invalid key, got nil")
		}
	})
}

func TestConcurrentConfigAccess(t *testing.T) {
	cleanup, tempDir := setupConfigTest(t)
	defer cleanup()

	configPath := filepath.Join(tempDir, "config.json")
	viper.SetConfigFile(configPath)

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 5

	// 并发写入配置
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := configKeys[index%len(configKeys)]
				value := fmt.Sprintf("concurrent-value-%d-%d", index, j)
				if err := executeConfigSet(key, value); err != nil {
					t.Errorf("concurrent set failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证配置文件存在且可读
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file does not exist after concurrent writes")
	}
}

func TestExecuteConfigList(t *testing.T) {
	t.Run("list all configs", func(t *testing.T) {
		cleanup, tempDir := setupConfigTest(t)
		defer cleanup()

		configPath := filepath.Join(tempDir, "config.json")
		viper.SetConfigFile(configPath)
		viper.Set("github_token", "test-token")
		viper.Set("proxy", "http://proxy.example.com")

		err := executeConfigList()
		if err != nil {
			t.Errorf("executeConfigList() error = %v", err)
		}
	})

	t.Run("list empty configs", func(t *testing.T) {
		cleanup, tempDir := setupConfigTest(t)
		defer cleanup()

		configPath := filepath.Join(tempDir, "config.json")
		viper.SetConfigFile(configPath)

		err := executeConfigList()
		if err != nil {
			t.Errorf("executeConfigList() error = %v", err)
		}
	})
}

func TestConfigGetCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		setup       func()
	}{
		{
			name:        "no args",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "one arg",
			args:        []string{"github_token"},
			expectError: false,
		},
		{
			name:        "too many args",
			args:        []string{"github_token", "extra"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup, _ := setupConfigTest(t)
			defer cleanup()

			cmd := &cobra.Command{}
			err := configGetCmd.Args(cmd, tt.args)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigSetCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "no args",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "one arg",
			args:        []string{"github_token"},
			expectError: true,
		},
		{
			name:        "two args",
			args:        []string{"github_token", "value"},
			expectError: false,
		},
		{
			name:        "too many args",
			args:        []string{"github_token", "value", "extra"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup, _ := setupConfigTest(t)
			defer cleanup()

			cmd := &cobra.Command{}
			err := configSetCmd.Args(cmd, tt.args)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
