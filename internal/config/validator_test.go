package config

import (
	"strings"
	"testing"

	"github.com/charleszardd/yawa/internal/command"
)

func TestValidateAcceptsSupportedCommands(t *testing.T) {
	configuration := File{
		General: General{ConfigVersion: CurrentVersion},
		Commands: []command.Definition{
			{Name: "calculator", Type: command.TypeExec, Executable: "calc.exe"},
			{Name: "downloads", Type: command.TypeOpenPath, Path: `C:\Downloads`},
			{Name: "github", Type: command.TypeURL, URL: "https://github.com"},
		},
	}

	if err := Validate(configuration); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateReportsAllIndependentErrors(t *testing.T) {
	configuration := File{
		General: General{ConfigVersion: 99},
		Commands: []command.Definition{
			{Name: "Invalid Name", Type: command.TypeExec},
			{Name: "Invalid Name", Type: command.TypeURL, URL: "http://example.com"},
		},
	}

	err := Validate(configuration)
	if err == nil {
		t.Fatal("Validate() error = nil")
	}
	for _, expected := range []string{"config_version", "must match", "is duplicated", "absolute https URL"} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("Validate() error %q does not contain %q", err, expected)
		}
	}
}
