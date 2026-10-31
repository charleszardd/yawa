package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = "config.toml"

func DefaultPath() (string, error) {
	appDataDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user configuration directory: %w", err)
	}
	return filepath.Join(appDataDirectory, "yawa", configFileName), nil
}
