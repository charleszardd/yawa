package config

import "github.com/charleszardd/yawa/internal/command"

const CurrentVersion = 1

type General struct {
	ConfigVersion int    `toml:"config_version"`
	LogLevel      string `toml:"log_level"`
}

type File struct {
	General  General              `toml:"general"`
	Commands []command.Definition `toml:"commands"`
}
