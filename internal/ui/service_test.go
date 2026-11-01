package ui

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charleszardd/yawa/internal/command"
)

func TestServiceLoadSerializesEmptyCommandsAsArray(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "config.toml"), command.NewDispatcher(launcherStub{}))

	snapshot, err := service.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if snapshot.Commands == nil {
		t.Fatal("Load().Commands = nil, want an empty slice")
	}

	encodedSnapshot, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(encodedSnapshot), `"commands":[]`) {
		t.Fatalf("json.Marshal() = %s, want commands to be an array", encodedSnapshot)
	}
}

type launcherStub struct{}

func (launcherStub) Execute(context.Context, string, []string, string) error { return nil }
func (launcherStub) OpenPath(string) error                                   { return nil }
func (launcherStub) OpenURL(string) error                                    { return nil }

type launcherSpy struct {
	launcherStub
	openedPath string
}

func (launcher *launcherSpy) OpenPath(path string) error {
	launcher.openedPath = path
	return nil
}

func TestServiceOpensWindowsAppsFolder(t *testing.T) {
	launcher := &launcherSpy{}
	service := NewService(filepath.Join(t.TempDir(), "config.toml"), command.NewDispatcher(launcher))

	if err := service.OpenWindowsApps(); err != nil {
		t.Fatalf("OpenWindowsApps() error = %v", err)
	}
	if launcher.openedPath != windowsAppsFolderPath {
		t.Fatalf("OpenWindowsApps() path = %q, want %q", launcher.openedPath, windowsAppsFolderPath)
	}
}

func TestServiceCommandLifecycle(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "config.toml"), command.NewDispatcher(launcherStub{}))
	configuredCommand := command.Definition{Name: "calculator", Type: command.TypeExec, Executable: "calc.exe", Hotkey: "Ctrl+Alt+C"}
	if err := service.Save(configuredCommand, ""); err != nil {
		t.Fatalf("Save(add) error = %v", err)
	}
	snapshot, err := service.Load()
	if err != nil || len(snapshot.Commands) != 1 {
		t.Fatalf("Load() = %+v, %v", snapshot, err)
	}
	configuredCommand.Description = "Updated"
	if err := service.Save(configuredCommand, "calculator"); err != nil {
		t.Fatalf("Save(update) error = %v", err)
	}
	if err := service.Delete("calculator"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	snapshot, err = service.Load()
	if err != nil || len(snapshot.Commands) != 0 {
		t.Fatalf("Load() after delete = %+v, %v", snapshot, err)
	}
}
