//go:build windows

package tray

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	windowClassName = "YawaBackgroundAgentWindow"
	trayIconID      = 1
	menuOpen        = 1001
	menuReload      = 1002
	menuExit        = 1003

	windowMessageClose   = 0x0010
	windowMessageDestroy = 0x0002
	windowMessageNull    = 0x0000
	windowMessageApp     = 0x8000
	windowMessageCommand = 0x0111
	windowMessageLeftDbl = 0x0203
	windowMessageRightUp = 0x0205
	windowMessageContext = 0x007B

	notifyIconAdd        = 0x00000000
	notifyIconDelete     = 0x00000002
	notifyIconSetVersion = 0x00000004
	notifyIconMessage    = 0x00000001
	notifyIconIcon       = 0x00000002
	notifyIconTip        = 0x00000004
	notifyIconVersion4   = 4

	menuString     = 0x00000000
	menuSeparator  = 0x00000800
	trackRight     = 0x0002
	trackReturnCmd = 0x0100

	defaultApplicationIcon = 32512
	defaultArrowCursor     = 32512
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	shell32                 = windows.NewLazySystemDLL("shell32.dll")
	kernel32                = windows.NewLazySystemDLL("kernel32.dll")
	registerClassEx         = user32.NewProc("RegisterClassExW")
	createWindowEx          = user32.NewProc("CreateWindowExW")
	defWindowProc           = user32.NewProc("DefWindowProcW")
	destroyWindow           = user32.NewProc("DestroyWindow")
	loadIcon                = user32.NewProc("LoadIconW")
	loadCursor              = user32.NewProc("LoadCursorW")
	createPopupMenu         = user32.NewProc("CreatePopupMenu")
	appendMenu              = user32.NewProc("AppendMenuW")
	trackPopupMenu          = user32.NewProc("TrackPopupMenu")
	destroyMenu             = user32.NewProc("DestroyMenu")
	getCursorPos            = user32.NewProc("GetCursorPos")
	setForegroundWindow     = user32.NewProc("SetForegroundWindow")
	postMessage             = user32.NewProc("PostMessageW")
	getMessage              = user32.NewProc("GetMessageW")
	translateMessage        = user32.NewProc("TranslateMessage")
	dispatchMessage         = user32.NewProc("DispatchMessageW")
	postQuitMessage         = user32.NewProc("PostQuitMessage")
	registerWindowMessage   = user32.NewProc("RegisterWindowMessageW")
	shellNotifyIcon         = shell32.NewProc("Shell_NotifyIconW")
	getModuleHandle         = kernel32.NewProc("GetModuleHandleW")
	windowProcedure         = syscall.NewCallback(handleWindowMessage)
	applications            sync.Map
	taskbarCreatedMessageID uint32
)

type Options struct {
	Tooltip string
	Open    func()
	Reload  func()
	Exit    func()
}

type Application struct {
	options Options
	window  uintptr
	icon    uintptr
}

type point struct {
	X int32
	Y int32
}

type message struct {
	Window  uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   point
	Private uint32
}

type windowClass struct {
	Size            uint32
	Style           uint32
	WindowProcedure uintptr
	ClassExtra      int32
	WindowExtra     int32
	Instance        uintptr
	Icon            uintptr
	Cursor          uintptr
	Background      uintptr
	MenuName        *uint16
	ClassName       *uint16
	SmallIcon       uintptr
}

type notifyIconData struct {
	Size        uint32
	Window      uintptr
	ID          uint32
	Flags       uint32
	Callback    uint32
	Icon        uintptr
	Tip         [128]uint16
	State       uint32
	StateMask   uint32
	Info        [256]uint16
	Version     uint32
	InfoTitle   [64]uint16
	InfoFlags   uint32
	ItemGUID    windows.GUID
	BalloonIcon uintptr
}

func New(options Options) *Application {
	return &Application{options: options}
}

func (application *Application) Run(ctx context.Context) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := application.createWindow(); err != nil {
		return err
	}
	defer application.destroy()

	go func() {
		<-ctx.Done()
		postMessage.Call(application.window, windowMessageClose, 0, 0)
	}()

	for {
		var event message
		result, _, callErr := getMessage.Call(uintptr(unsafe.Pointer(&event)), 0, 0, 0)
		if int32(result) == -1 {
			return fmt.Errorf("read tray message: %w", callErr)
		}
		if result == 0 {
			return nil
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&event)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&event)))
	}
}

func (application *Application) createWindow() error {
	instance, _, callErr := getModuleHandle.Call(0)
	if instance == 0 {
		return fmt.Errorf("get module handle: %w", callErr)
	}
	className, err := windows.UTF16PtrFromString(windowClassName)
	if err != nil {
		return err
	}
	application.icon, _, _ = loadIcon.Call(0, defaultApplicationIcon)
	cursor, _, _ := loadCursor.Call(0, defaultArrowCursor)
	class := windowClass{
		Size:            uint32(unsafe.Sizeof(windowClass{})),
		WindowProcedure: windowProcedure,
		Instance:        instance,
		Icon:            application.icon,
		Cursor:          cursor,
		ClassName:       className,
		SmallIcon:       application.icon,
	}
	if atom, _, registerErr := registerClassEx.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		return fmt.Errorf("register tray window class: %w", registerErr)
	}
	title, err := windows.UTF16PtrFromString("Yawa Background Agent")
	if err != nil {
		return err
	}
	application.window, _, callErr = createWindowEx.Call(
		0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), 0,
		0, 0, 0, 0, 0, 0, instance, 0,
	)
	if application.window == 0 {
		return fmt.Errorf("create tray window: %w", callErr)
	}
	applications.Store(application.window, application)
	messageName, _ := windows.UTF16PtrFromString("TaskbarCreated")
	messageID, _, _ := registerWindowMessage.Call(uintptr(unsafe.Pointer(messageName)))
	taskbarCreatedMessageID = uint32(messageID)
	if err := application.addIcon(); err != nil {
		return err
	}
	return nil
}

func (application *Application) addIcon() error {
	data := application.iconData()
	result, _, callErr := shellNotifyIcon.Call(notifyIconAdd, uintptr(unsafe.Pointer(&data)))
	if result == 0 {
		return fmt.Errorf("add tray icon: %w", callErr)
	}
	data.Version = notifyIconVersion4
	shellNotifyIcon.Call(notifyIconSetVersion, uintptr(unsafe.Pointer(&data)))
	return nil
}

func (application *Application) iconData() notifyIconData {
	data := notifyIconData{
		Size:     uint32(unsafe.Sizeof(notifyIconData{})),
		Window:   application.window,
		ID:       trayIconID,
		Flags:    notifyIconMessage | notifyIconIcon | notifyIconTip,
		Callback: windowMessageApp + 1,
		Icon:     application.icon,
	}
	copy(data.Tip[:], windows.StringToUTF16(application.options.Tooltip))
	return data
}

func (application *Application) showMenu() {
	menu, _, _ := createPopupMenu.Call()
	if menu == 0 {
		return
	}
	defer destroyMenu.Call(menu)
	appendMenuItem(menu, menuString, menuOpen, "Open Yawa")
	appendMenuItem(menu, menuString, menuReload, "Reload shortcuts")
	appendMenu.Call(menu, menuSeparator, 0, 0)
	appendMenuItem(menu, menuString, menuExit, "Exit Yawa")

	var cursor point
	getCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	setForegroundWindow.Call(application.window)
	commandID, _, _ := trackPopupMenu.Call(menu, trackRight|trackReturnCmd, uintptr(cursor.X), uintptr(cursor.Y), 0, application.window, 0)
	postMessage.Call(application.window, windowMessageNull, 0, 0)
	application.handleCommand(commandID)
}

func (application *Application) handleCommand(commandID uintptr) {
	switch commandID {
	case menuOpen:
		if application.options.Open != nil {
			go application.options.Open()
		}
	case menuReload:
		if application.options.Reload != nil {
			go application.options.Reload()
		}
	case menuExit:
		if application.options.Exit != nil {
			application.options.Exit()
		}
		postMessage.Call(application.window, windowMessageClose, 0, 0)
	}
}

func (application *Application) destroy() {
	if application.window == 0 {
		return
	}
	application.removeIcon()
	applications.Delete(application.window)
	destroyWindow.Call(application.window)
	application.window = 0
}

func (application *Application) removeIcon() {
	data := application.iconData()
	shellNotifyIcon.Call(notifyIconDelete, uintptr(unsafe.Pointer(&data)))
}

func appendMenuItem(menu uintptr, flags uintptr, identifier uintptr, label string) {
	labelPointer, err := windows.UTF16PtrFromString(label)
	if err != nil {
		return
	}
	appendMenu.Call(menu, flags, identifier, uintptr(unsafe.Pointer(labelPointer)))
}

func handleWindowMessage(window uintptr, message uint32, wordParameter uintptr, longParameter uintptr) uintptr {
	value, found := applications.Load(window)
	if !found {
		result, _, _ := defWindowProc.Call(window, uintptr(message), wordParameter, longParameter)
		return result
	}
	application := value.(*Application)
	switch message {
	case windowMessageApp + 1:
		notification := uint32(longParameter & 0xFFFF)
		switch notification {
		case windowMessageLeftDbl:
			application.handleCommand(menuOpen)
		case windowMessageRightUp, windowMessageContext:
			application.showMenu()
		}
		return 0
	case windowMessageCommand:
		application.handleCommand(wordParameter & 0xFFFF)
		return 0
	case windowMessageClose:
		application.removeIcon()
		destroyWindow.Call(window)
		return 0
	case windowMessageDestroy:
		applications.Delete(window)
		application.window = 0
		postQuitMessage.Call(0)
		return 0
	default:
		if message == taskbarCreatedMessageID {
			_ = application.addIcon()
			return 0
		}
	}
	result, _, _ := defWindowProc.Call(window, uintptr(message), wordParameter, longParameter)
	return result
}
