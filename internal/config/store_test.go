package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/charleszardd/yawa/internal/command"
)

func TestStoreSavesAtomicallyAndCreatesBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")
	store := NewStore()
	configuration := File{General: General{ConfigVersion: CurrentVersion}, Commands: []command.Definition{{Name: "one", Type: command.TypeExec, Executable: "one.exe"}}}

	if err := store.Save(path, configuration, Revision{}); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}
	loaded, revision, err := store.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	loaded.Commands[0].Executable = "two.exe"
	if err := store.Save(path, loaded, revision); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}

	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	updated, _, err := store.Load(path)
	if err != nil || updated.Commands[0].Executable != "two.exe" {
		t.Fatalf("updated configuration = %+v, %v", updated, err)
	}
}

func TestStoreRejectsConcurrentModification(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	store := NewStore()
	configuration := File{General: General{ConfigVersion: CurrentVersion}}
	if err := store.Save(path, configuration, Revision{}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	_, revision, err := store.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("changed externally"), 0o600); err != nil {
		t.Fatalf("external write: %v", err)
	}

	err = store.Save(path, configuration, revision)
	if !errors.Is(err, ErrConcurrentModification) {
		t.Fatalf("Save() error = %v", err)
	}
}
