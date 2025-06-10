// This new test file verifies the configuration loading logic using Viper.

package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("Defaults when no config file", func(t *testing.T) {
		// Ensure no config file exists for this test
		os.Remove("config.yml")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned an error: %v", err)
		}

		// Check if default values are set
		if cfg.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", cfg.Port)
		}
		if cfg.Database.Path != "./mango.db" {
			t.Errorf("Expected default db path './mango.db', got '%s'", cfg.Database.Path)
		}
		if cfg.Library.Path != "./manga" {
			t.Errorf("Expected default library path './manga', got '%s'", cfg.Library.Path)
		}
	})

	t.Run("Loads from config file", func(t *testing.T) {
		// Create a temporary config file for this test
		configContent := `
port: 9999
database:
  path: "/tmp/test.db"
library:
  path: "/tmp/test-manga"
unknown_setting: "should be ignored"
`
		// Create the config file in the current directory so Viper can find it.
		// Note: `t.TempDir()` is not used here because Viper looks in the CWD.
		configPath := "config.yml"
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write test config file: %v", err)
		}
		// Clean up the file after the test
		defer os.Remove(configPath)

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned an error: %v", err)
		}

		// Check if values from the file were loaded
		if cfg.Port != 9999 {
			t.Errorf("Expected port 9999, got %d", cfg.Port)
		}
		if cfg.Database.Path != "/tmp/test.db" {
			t.Errorf("Expected db path '/tmp/test.db', got '%s'", cfg.Database.Path)
		}
		if cfg.Library.Path != "/tmp/test-manga" {
			t.Errorf("Expected library path '/tmp/test-manga', got '%s'", cfg.Library.Path)
		}
		if cfg.ScanInterval != 60 {
			t.Errorf("Expected default scan interval of 60, got %d", cfg.ScanInterval)
		}
	})
}
