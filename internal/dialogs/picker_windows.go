//go:build windows

package dialogs

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/charleszardd/yawa/internal/command"
	"golang.org/x/sys/windows"
)

const (
	coinitApartmentThreaded = 0x2
	clsctxInprocServer      = 0x1
	fosPickFolders          = 0x20
	fosForceFileSystem      = 0x40
	fosPathMustExist        = 0x800
	fosFileMustExist        = 0x1000
	sigdnFileSystemPath     = 0x80058000
	hresultCancelled        = 0x800704C7
)

var (
	ole32                       = windows.NewLazySystemDLL("ole32.dll")
	coInitializeEx              = ole32.NewProc("CoInitializeEx")
	coUninitialize              = ole32.NewProc("CoUninitialize")
	coCreateInstance            = ole32.NewProc("CoCreateInstance")
	coTaskMemFree               = ole32.NewProc("CoTaskMemFree")
	shCreateItemFromParsingName = windows.NewLazySystemDLL("shell32.dll").NewProc("SHCreateItemFromParsingName")

	clsidFileOpenDialog = windows.GUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileOpenDialog  = windows.GUID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768, Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	iidIShellItem       = windows.GUID{Data1: 0x43826D1E, Data2: 0xE718, Data3: 0x42EE, Data4: [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
)

type comObject struct{ vtable *[29]uintptr }

type filterSpec struct {
	name    *uint16
	pattern *uint16
}

type Picker struct{}

func (Picker) Select(_ context.Context, kind command.SelectionKind, startDirectory string) (string, bool, error) {
	return selectPath(kind, startDirectory, false)
}

func (Picker) SelectExecutable(_ context.Context) (string, bool, error) {
	return selectPath(command.SelectFile, os.Getenv("ProgramFiles"), true)
}

func selectPath(kind command.SelectionKind, startDirectory string, executableOnly bool) (string, bool, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	initialized, err := initializeCOM()
	if err != nil {
		return "", false, err
	}
	if initialized {
		defer coUninitialize.Call()
	}

	dialog, err := createDialog()
	if err != nil {
		return "", false, err
	}
	defer dialog.release()

	options := uintptr(fosForceFileSystem | fosPathMustExist)
	if kind == command.SelectFolder {
		options |= fosPickFolders
	} else {
		options |= fosFileMustExist
	}
	if err := dialog.callHRESULT(9, options); err != nil {
		return "", false, fmt.Errorf("set picker options: %w", err)
	}
	if executableOnly {
		if err := dialog.setExecutableFilter(); err != nil {
			return "", false, err
		}
	}
	if startDirectory != "" {
		if err := dialog.setFolder(os.ExpandEnv(startDirectory)); err != nil {
			return "", false, err
		}
	}
	if err := dialog.callHRESULT(3, 0); err != nil {
		if hresultCode(err) == hresultCancelled {
			return "", false, nil
		}
		return "", false, fmt.Errorf("show picker: %w", err)
	}
	return dialog.resultPath()
}

func (object *comObject) setExecutableFilter() error {
	filterName, err := windows.UTF16PtrFromString("Applications (*.exe)")
	if err != nil {
		return fmt.Errorf("encode executable filter name: %w", err)
	}
	filterPattern, err := windows.UTF16PtrFromString("*.exe")
	if err != nil {
		return fmt.Errorf("encode executable filter pattern: %w", err)
	}
	filter := filterSpec{name: filterName, pattern: filterPattern}
	if err := object.callHRESULT(4, 1, uintptr(unsafe.Pointer(&filter))); err != nil {
		return fmt.Errorf("set executable picker filter: %w", err)
	}
	return nil
}

func initializeCOM() (bool, error) {
	result, _, _ := coInitializeEx.Call(0, coinitApartmentThreaded)
	code := uint32(result)
	if code == 0 || code == 1 {
		return true, nil
	}
	return false, fmt.Errorf("CoInitializeEx: HRESULT 0x%08X", code)
}

func createDialog() (*comObject, error) {
	var dialog *comObject
	result, _, _ := coCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)), 0, clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIFileOpenDialog)), uintptr(unsafe.Pointer(&dialog)),
	)
	if code := uint32(result); code != 0 {
		return nil, fmt.Errorf("CoCreateInstance: HRESULT 0x%08X", code)
	}
	return dialog, nil
}

func (object *comObject) callHRESULT(method int, arguments ...uintptr) error {
	callArguments := append([]uintptr{uintptr(unsafe.Pointer(object))}, arguments...)
	result, _, _ := syscall.SyscallN(object.vtable[method], callArguments...)
	if code := uint32(result); code != 0 {
		return hresultError(code)
	}
	return nil
}

func (object *comObject) release() {
	syscall.SyscallN(object.vtable[2], uintptr(unsafe.Pointer(object)))
}

func (object *comObject) setFolder(path string) error {
	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("encode picker start directory: %w", err)
	}
	var item *comObject
	result, _, _ := shCreateItemFromParsingName.Call(
		uintptr(unsafe.Pointer(pathPointer)), 0, uintptr(unsafe.Pointer(&iidIShellItem)), uintptr(unsafe.Pointer(&item)),
	)
	if code := uint32(result); code != 0 {
		return fmt.Errorf("resolve picker start directory %q: HRESULT 0x%08X", path, code)
	}
	defer item.release()
	if err := object.callHRESULT(12, uintptr(unsafe.Pointer(item))); err != nil {
		return fmt.Errorf("set picker start directory: %w", err)
	}
	return nil
}

func (object *comObject) resultPath() (string, bool, error) {
	var item *comObject
	if err := object.callHRESULT(20, uintptr(unsafe.Pointer(&item))); err != nil {
		return "", false, fmt.Errorf("get picker result: %w", err)
	}
	defer item.release()

	var pathPointer *uint16
	if err := item.callHRESULT(5, sigdnFileSystemPath, uintptr(unsafe.Pointer(&pathPointer))); err != nil {
		return "", false, fmt.Errorf("get selected path: %w", err)
	}
	defer coTaskMemFree.Call(uintptr(unsafe.Pointer(pathPointer)))
	return windows.UTF16PtrToString(pathPointer), true, nil
}

type hresultError uint32

func (err hresultError) Error() string { return fmt.Sprintf("HRESULT 0x%08X", uint32(err)) }
func hresultCode(err error) uint32 {
	if result, ok := err.(hresultError); ok {
		return uint32(result)
	}
	return 0
}
