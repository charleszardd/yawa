//go:build windows

package appcatalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveExecutableAcceptsAccessibleAbsoluteFile(t *testing.T) {
	executablePath := filepath.Join(t.TempDir(), "example.exe")
	if err := os.WriteFile(executablePath, []byte("test"), 0o600); err != nil {
		t.Fatalf("write executable fixture: %v", err)
	}

	status := ResolveExecutable(executablePath)
	if !status.Found || status.ResolvedPath != executablePath {
		t.Fatalf("ResolveExecutable() = %+v", status)
	}
}

func TestResolveExecutableRejectsMissingFile(t *testing.T) {
	status := ResolveExecutable(filepath.Join(t.TempDir(), "missing.exe"))
	if status.Found || status.Message == "" {
		t.Fatalf("ResolveExecutable() = %+v", status)
	}
}

func TestInstalledApplicationsHaveUniqueNamesAndValidPaths(t *testing.T) {
	seenNames := make(map[string]struct{})
	for _, application := range InstalledApplications() {
		if _, duplicate := seenNames[application.Name]; duplicate {
			t.Fatalf("duplicate application %q", application.Name)
		}
		seenNames[application.Name] = struct{}{}
		if status := ResolveExecutable(application.Executable); !status.Found {
			t.Fatalf("application %q has invalid executable %q", application.Name, application.Executable)
		}
	}
}
