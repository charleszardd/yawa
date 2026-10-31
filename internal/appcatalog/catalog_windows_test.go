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

func TestInstalledApplicationsPreferVisualStudioCodeCLI(t *testing.T) {
	localAppData := t.TempDir()
	programFiles := t.TempDir()
	windowsDirectory := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)
	t.Setenv("ProgramFiles", programFiles)
	t.Setenv("ProgramFiles(x86)", programFiles)
	t.Setenv("WINDIR", windowsDirectory)

	installationDirectory := filepath.Join(localAppData, `Programs\Microsoft VS Code`)
	cliPath := filepath.Join(installationDirectory, `bin\code.cmd`)
	guiPath := filepath.Join(installationDirectory, "Code.exe")
	for _, executablePath := range []string{cliPath, guiPath} {
		if err := os.MkdirAll(filepath.Dir(executablePath), 0o700); err != nil {
			t.Fatalf("create executable directory: %v", err)
		}
		if err := os.WriteFile(executablePath, []byte("test"), 0o600); err != nil {
			t.Fatalf("write executable fixture: %v", err)
		}
	}

	for _, application := range InstalledApplications() {
		if application.Name != "Visual Studio Code" {
			continue
		}
		if application.Executable != cliPath {
			t.Fatalf("Visual Studio Code executable = %q, want CLI %q", application.Executable, cliPath)
		}
		return
	}
	t.Fatal("Visual Studio Code was not detected")
}
