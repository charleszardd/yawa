package config

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

var ErrConcurrentModification = errors.New("configuration changed on disk; reload it and retry")

type Revision [sha256.Size]byte

type Store struct {
	loader Loader
}

func NewStore() Store {
	return Store{loader: Loader{}}
}

func (store Store) Load(path string) (File, Revision, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return File{}, Revision{}, fmt.Errorf("read configuration %s: %w", path, err)
	}
	configuration, err := store.loader.Decode(path, contents)
	if err != nil {
		return File{}, Revision{}, err
	}
	return configuration, sha256.Sum256(contents), nil
}

func (store Store) LoadOrDefault(path string) (File, Revision, error) {
	configuration, revision, err := store.Load(path)
	if errors.Is(err, os.ErrNotExist) {
		return File{General: General{ConfigVersion: CurrentVersion, LogLevel: "info"}}, Revision{}, nil
	}
	return configuration, revision, err
}

func (Store) Save(path string, configuration File, expectedRevision Revision) error {
	if err := Validate(configuration); err != nil {
		return fmt.Errorf("refuse to save invalid configuration: %w", err)
	}
	if err := ensureRevision(path, expectedRevision); err != nil {
		return err
	}

	var encoded bytes.Buffer
	if err := toml.NewEncoder(&encoded).Encode(configuration); err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	temporaryFile, err := os.CreateTemp(directory, configFileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary configuration: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)

	if err := writeAndSync(temporaryFile, encoded.Bytes()); err != nil {
		return err
	}
	if err := replaceFile(temporaryPath, path, path+".bak"); err != nil {
		return fmt.Errorf("replace configuration: %w", err)
	}
	return nil
}

func ensureRevision(path string, expected Revision) error {
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && expected == (Revision{}) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check configuration revision: %w", err)
	}
	if sha256.Sum256(contents) != expected {
		return ErrConcurrentModification
	}
	return nil
}

func writeAndSync(file *os.File, contents []byte) error {
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return fmt.Errorf("write temporary configuration: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("flush temporary configuration: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary configuration: %w", err)
	}
	return nil
}
