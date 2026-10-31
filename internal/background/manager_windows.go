//go:build windows

package background

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/charleszardd/yawa/internal/agentcontrol"
	"github.com/charleszardd/yawa/internal/hotkey"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	startupRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	startupValueName    = "Yawa Background Agent"
	operationTimeout    = 5 * time.Second
)

type Status struct {
	Available        bool   `json:"available"`
	Running          bool   `json:"running"`
	Active           bool   `json:"active"`
	StartWithWindows bool   `json:"startWithWindows"`
	Executable       string `json:"executable"`
	Message          string `json:"message"`
}

type Manager struct{}

func (Manager) Status() Status {
	executablePath, executableErr := resolveAgentExecutable()
	startupEnabled, startupErr := isStartupEnabled()
	status := Status{
		Available:        executableErr == nil,
		Running:          agentcontrol.IsRunning(),
		Active:           agentcontrol.IsActive(),
		StartWithWindows: startupEnabled,
		Executable:       executablePath,
	}
	switch {
	case executableErr != nil:
		status.Message = executableErr.Error()
	case startupErr != nil:
		status.Message = startupErr.Error()
	case status.Active:
		status.Message = "Shortcuts are active in the background."
	case status.Running:
		status.Message = "Agent is running, but shortcuts are unavailable. Stop any terminal agent and reload."
	default:
		status.Message = "Background agent is stopped."
	}
	return status
}

func (Manager) Start() error {
	if agentcontrol.IsRunning() {
		return nil
	}
	if hotkey.IsRunning() {
		return fmt.Errorf("a terminal-based Yawa agent is already running; stop it with Ctrl+C before starting the background agent")
	}
	executablePath, err := resolveAgentExecutable()
	if err != nil {
		return err
	}
	process := exec.Command(executablePath)
	process.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW | windows.CREATE_NEW_PROCESS_GROUP,
	}
	if err := process.Start(); err != nil {
		return fmt.Errorf("start background agent: %w", err)
	}
	if err := process.Process.Release(); err != nil {
		return fmt.Errorf("release background agent process: %w", err)
	}
	return waitForState(true)
}

func (manager Manager) Reload() error {
	if !agentcontrol.IsRunning() {
		return manager.Start()
	}
	if err := agentcontrol.RequestReload(); err != nil {
		return fmt.Errorf("reload background agent: %w", err)
	}
	return nil
}

func (Manager) Stop() error {
	if !agentcontrol.IsRunning() {
		return nil
	}
	if err := agentcontrol.RequestStop(); err != nil {
		return fmt.Errorf("stop background agent: %w", err)
	}
	return waitForState(false)
}

func (Manager) SetStartWithWindows(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRegistryPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Windows startup settings: %w", err)
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(startupValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("disable start with Windows: %w", err)
		}
		return nil
	}
	executablePath, err := resolveAgentExecutable()
	if err != nil {
		return err
	}
	if err := key.SetStringValue(startupValueName, startupCommand(executablePath)); err != nil {
		return fmt.Errorf("enable start with Windows: %w", err)
	}
	return nil
}

func resolveAgentExecutable() (string, error) {
	currentExecutable, err := os.Executable()
	if err == nil {
		siblingPath := filepath.Join(filepath.Dir(currentExecutable), "yawa-agent.exe")
		if fileInfo, statErr := os.Stat(siblingPath); statErr == nil && !fileInfo.IsDir() {
			return siblingPath, nil
		}
	}
	resolvedPath, err := exec.LookPath("yawa-agent.exe")
	if err != nil {
		return "", fmt.Errorf("yawa-agent.exe was not found; reinstall Yawa v1.0.0 or later")
	}
	absolutePath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return filepath.Clean(resolvedPath), nil
	}
	return filepath.Clean(absolutePath), nil
}

func isStartupEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRegistryPath, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("open Windows startup settings: %w", err)
	}
	defer key.Close()
	_, _, err = key.GetStringValue(startupValueName)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read start with Windows setting: %w", err)
	}
	return true, nil
}

func startupCommand(executablePath string) string {
	return `"` + executablePath + `"`
}

func waitForState(expectedRunning bool) error {
	deadline := time.Now().Add(operationTimeout)
	for time.Now().Before(deadline) {
		if agentcontrol.IsRunning() == expectedRunning {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	if expectedRunning {
		return fmt.Errorf("background agent did not start within %s", operationTimeout)
	}
	return fmt.Errorf("background agent did not stop within %s", operationTimeout)
}
