package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderLoadsConfiguration(t *testing.T) {
	configPath := writeTestConfig(t, `[general]
config_version = 1
log_level = "info"

[[commands]]
name = "calculator"
type = "exec"
executable = "calc.exe"
`)

	configuration, err := (Loader{}).Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if configuration.General.ConfigVersion != 1 || len(configuration.Commands) != 1 {
		t.Fatalf("Load() = %+v", configuration)
	}
}

func TestLoaderRejectsUnknownFields(t *testing.T) {
	configPath := writeTestConfig(t, "[general]\nconfig_version = 1\nunknown = true\n")

	_, err := (Loader{}).Load(configPath)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Load() error = %v", err)
	}
}

func writeTestConfig(t *testing.T, contents string) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}
