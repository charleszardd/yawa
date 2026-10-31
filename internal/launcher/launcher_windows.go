//go:build windows

package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const showNormal = 1

var (
	shell32           = windows.NewLazySystemDLL("shell32.dll")
	shellExecuteWProc = shell32.NewProc("ShellExecuteW")
)

type Windows struct{}

func (Windows) Execute(ctx context.Context, executable string, arguments []string, workingDirectory string) error {
	executablePath, err := exec.LookPath(os.ExpandEnv(executable))
	if err != nil {
		return fmt.Errorf("resolve executable %q: %w", executable, err)
	}

	process := exec.CommandContext(ctx, executablePath, arguments...)
	if workingDirectory != "" {
		process.Dir = filepath.Clean(os.ExpandEnv(workingDirectory))
	}
	if err := process.Start(); err != nil {
		return fmt.Errorf("start executable %q: %w", executablePath, err)
	}
	if err := process.Process.Release(); err != nil {
		return fmt.Errorf("release process handle: %w", err)
	}
	return nil
}

func (Windows) OpenPath(path string) error {
	return shellExecute(os.ExpandEnv(path))
}

func (Windows) OpenURL(url string) error {
	return shellExecute(url)
}

func shellExecute(target string) error {
	operationPointer, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return fmt.Errorf("encode shell operation: %w", err)
	}
	targetPointer, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("encode shell target: %w", err)
	}

	result, _, callErr := shellExecuteWProc.Call(
		0,
		uintptr(unsafe.Pointer(operationPointer)),
		uintptr(unsafe.Pointer(targetPointer)),
		0,
		0,
		showNormal,
	)
	if result <= 32 {
		if callErr != syscall.Errno(0) {
			return fmt.Errorf("open %q: %w", target, callErr)
		}
		return fmt.Errorf("open %q: ShellExecuteW returned code %d", target, result)
	}
	return nil
}
