//go:build windows

package config

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const replaceFileWriteThrough = 0x00000001

var replaceFileW = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReplaceFileW")

func replaceFile(temporaryPath string, destinationPath string, backupPath string) error {
	if _, err := os.Stat(destinationPath); errors.Is(err, os.ErrNotExist) {
		return os.Rename(temporaryPath, destinationPath)
	} else if err != nil {
		return err
	}

	destinationPointer, err := windows.UTF16PtrFromString(destinationPath)
	if err != nil {
		return err
	}
	temporaryPointer, err := windows.UTF16PtrFromString(temporaryPath)
	if err != nil {
		return err
	}
	backupPointer, err := windows.UTF16PtrFromString(backupPath)
	if err != nil {
		return err
	}
	result, _, callErr := replaceFileW.Call(
		uintptr(unsafe.Pointer(destinationPointer)),
		uintptr(unsafe.Pointer(temporaryPointer)),
		uintptr(unsafe.Pointer(backupPointer)),
		replaceFileWriteThrough,
		0,
		0,
	)
	if result == 0 {
		if callErr != syscall.Errno(0) {
			return callErr
		}
		return fmt.Errorf("ReplaceFileW failed")
	}
	return nil
}
