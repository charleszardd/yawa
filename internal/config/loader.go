package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Loader struct{}

func (Loader) Load(path string) (File, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{}, fmt.Errorf("configuration file does not exist at %s", path)
		}
		return File{}, fmt.Errorf("read configuration %s: %w", path, err)
	}

	return (Loader{}).Decode(path, contents)
}

func (Loader) Decode(source string, contents []byte) (File, error) {
	var configuration File
	metadata, err := toml.Decode(string(contents), &configuration)
	if err != nil {
		return File{}, fmt.Errorf("parse configuration %s: %w", source, err)
	}
	if undecodedKeys := metadata.Undecoded(); len(undecodedKeys) > 0 {
		return File{}, fmt.Errorf("configuration contains unknown field %q", undecodedKeys[0])
	}
	return configuration, nil
}
