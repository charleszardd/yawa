package ui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/charleszardd/yawa/internal/command"
)

type launcherStub struct{}

func (launcherStub) Execute(context.Context, string, []string, string) error { return nil }
func (launcherStub) OpenPath(string) error                                   { return nil }
func (launcherStub) OpenURL(string) error                                    { return nil }

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
