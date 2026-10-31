//go:build windows

package agentcontrol

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

const (
	instanceMutexName = `Local\yawa-background-agent`
	stopEventName     = `Local\yawa-background-agent-stop`
	reloadEventName   = `Local\yawa-background-agent-reload`
	activeEventName   = `Local\yawa-background-agent-active`
)

type Signal int

const (
	SignalStop Signal = iota + 1
	SignalReload
)

type Instance struct {
	mutex       windows.Handle
	stopEvent   windows.Handle
	reloadEvent windows.Handle
	activeEvent windows.Handle
}

func Acquire() (*Instance, error) {
	mutexName, err := windows.UTF16PtrFromString(instanceMutexName)
	if err != nil {
		return nil, err
	}
	mutex, err := windows.CreateMutex(nil, false, mutexName)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		windows.CloseHandle(mutex)
		return nil, fmt.Errorf("another Yawa background agent is already running")
	}
	if err != nil {
		return nil, fmt.Errorf("create background agent mutex: %w", err)
	}

	instance := &Instance{mutex: mutex}
	if instance.stopEvent, err = createEvent(stopEventName, false); err != nil {
		instance.Close()
		return nil, err
	}
	if instance.reloadEvent, err = createEvent(reloadEventName, false); err != nil {
		instance.Close()
		return nil, err
	}
	if instance.activeEvent, err = createEvent(activeEventName, true); err != nil {
		instance.Close()
		return nil, err
	}
	return instance, nil
}

func (instance *Instance) Signals(ctx context.Context) <-chan Signal {
	signals := make(chan Signal, 1)
	go func() {
		defer close(signals)
		for {
			result, err := windows.WaitForMultipleObjects([]windows.Handle{instance.stopEvent, instance.reloadEvent}, false, 250)
			if err != nil {
				return
			}
			switch result {
			case windows.WAIT_OBJECT_0:
				signals <- SignalStop
				return
			case windows.WAIT_OBJECT_0 + 1:
				select {
				case signals <- SignalReload:
				case <-ctx.Done():
					return
				}
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
	return signals
}

func (instance *Instance) Close() {
	for _, handle := range []windows.Handle{instance.activeEvent, instance.reloadEvent, instance.stopEvent, instance.mutex} {
		if handle != 0 {
			windows.CloseHandle(handle)
		}
	}
}

func (instance *Instance) MarkActive() error {
	return windows.SetEvent(instance.activeEvent)
}

func (instance *Instance) MarkInactive() error {
	return windows.ResetEvent(instance.activeEvent)
}

func IsRunning() bool {
	name, err := windows.UTF16PtrFromString(instanceMutexName)
	if err != nil {
		return false
	}
	handle, err := windows.OpenMutex(windows.SYNCHRONIZE, false, name)
	if err != nil {
		return false
	}
	windows.CloseHandle(handle)
	return true
}

func IsActive() bool {
	name, err := windows.UTF16PtrFromString(activeEventName)
	if err != nil {
		return false
	}
	handle, err := windows.OpenEvent(windows.SYNCHRONIZE, false, name)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)
	result, err := windows.WaitForSingleObject(handle, 0)
	return err == nil && result == windows.WAIT_OBJECT_0
}

func RequestStop() error {
	return signalEvent(stopEventName)
}

func RequestReload() error {
	return signalEvent(reloadEventName)
}

func createEvent(name string, manualReset bool) (windows.Handle, error) {
	namePointer, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	manualResetValue := uint32(0)
	if manualReset {
		manualResetValue = 1
	}
	handle, err := windows.CreateEvent(nil, manualResetValue, 0, namePointer)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return 0, fmt.Errorf("create control event %q: %w", name, err)
	}
	return handle, nil
}

func signalEvent(name string) error {
	namePointer, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	handle, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, namePointer)
	if err != nil {
		return fmt.Errorf("background agent is not running: %w", err)
	}
	defer windows.CloseHandle(handle)
	if err := windows.SetEvent(handle); err != nil {
		return fmt.Errorf("signal background agent: %w", err)
	}
	return nil
}
