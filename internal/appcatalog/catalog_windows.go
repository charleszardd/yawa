//go:build windows

package appcatalog

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Application struct {
	Name       string `json:"name"`
	Executable string `json:"executable"`
}

type ExecutableStatus struct {
	Found        bool   `json:"found"`
	ResolvedPath string `json:"resolvedPath"`
	Message      string `json:"message"`
}

type candidate struct {
	name  string
	paths []string
}

func InstalledApplications() []Application {
	applicationCandidates := commonApplicationCandidates()
	applications := make([]Application, 0, len(applicationCandidates))
	for _, applicationCandidate := range applicationCandidates {
		for _, pathCandidate := range applicationCandidate.paths {
			status := ResolveExecutable(pathCandidate)
			if !status.Found {
				continue
			}
			applications = append(applications, Application{Name: applicationCandidate.name, Executable: status.ResolvedPath})
			break
		}
	}
	sort.Slice(applications, func(left, right int) bool { return applications[left].Name < applications[right].Name })
	return applications
}

func commonApplicationCandidates() []candidate {
	localAppData := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	windowsDirectory := os.Getenv("WINDIR")
	return []candidate{
		{name: "Visual Studio Code", paths: []string{
			filepath.Join(localAppData, `Programs\Microsoft VS Code\bin\code.cmd`),
			filepath.Join(programFiles, `Microsoft VS Code\bin\code.cmd`),
			"code",
			filepath.Join(localAppData, `Programs\Microsoft VS Code\Code.exe`),
			filepath.Join(programFiles, `Microsoft VS Code\Code.exe`),
		}},
		{name: "Google Chrome", paths: []string{
			filepath.Join(programFiles, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(programFilesX86, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(localAppData, `Google\Chrome\Application\chrome.exe`),
		}},
		{name: "Microsoft Edge", paths: []string{
			filepath.Join(programFilesX86, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(programFiles, `Microsoft\Edge\Application\msedge.exe`),
			"msedge",
		}},
		{name: "Windows Terminal", paths: []string{"wt.exe", "wt"}},
		{name: "File Explorer", paths: []string{filepath.Join(windowsDirectory, "explorer.exe")}},
		{name: "Notepad", paths: []string{filepath.Join(windowsDirectory, `System32\notepad.exe`), "notepad"}},
	}
}

func ResolveExecutable(value string) ExecutableStatus {
	trimmedValue := strings.Trim(strings.TrimSpace(value), `"`)
	if trimmedValue == "" {
		return ExecutableStatus{Message: "Choose an installed application or browse for an executable."}
	}

	expandedValue := os.ExpandEnv(trimmedValue)
	if filepath.IsAbs(expandedValue) {
		return inspectAbsolutePath(expandedValue)
	}

	resolvedPath, err := exec.LookPath(expandedValue)
	if err != nil {
		return ExecutableStatus{Message: "Application was not found on PATH."}
	}
	absolutePath, err := filepath.Abs(resolvedPath)
	if err != nil {
		absolutePath = resolvedPath
	}
	return ExecutableStatus{Found: true, ResolvedPath: filepath.Clean(absolutePath), Message: "Application found."}
}

func inspectAbsolutePath(path string) ExecutableStatus {
	cleanedPath := filepath.Clean(path)
	fileInfo, err := os.Stat(cleanedPath)
	if err != nil {
		return ExecutableStatus{ResolvedPath: cleanedPath, Message: "Executable does not exist or is not accessible."}
	}
	if fileInfo.IsDir() {
		return ExecutableStatus{ResolvedPath: cleanedPath, Message: "Select an executable file, not a directory."}
	}
	return ExecutableStatus{Found: true, ResolvedPath: cleanedPath, Message: "Application found."}
}
