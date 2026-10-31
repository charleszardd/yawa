//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestExecutableContainsTrayIcon(t *testing.T) {
	var module windows.Handle
	if err := windows.GetModuleHandleEx(windows.GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT, nil, &module); err != nil {
		t.Fatalf("GetModuleHandleEx() error = %v", err)
	}
	if _, err := windows.FindResource(module, windows.ResourceID(32512), windows.RT_GROUP_ICON); err != nil {
		t.Fatalf("tray icon resource 32512 is unavailable: %v", err)
	}
}
