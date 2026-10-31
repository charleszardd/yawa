//go:build windows

package hotkey

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/charleszardd/yawa/internal/command"
	"golang.org/x/sys/windows"
)

const (
	messageHotkey                    = 0x0312
	messageQuit                      = 0x0012
	peekNoRemove                     = 0x0000
	errorAlreadyExists syscall.Errno = 183
	jobQueueCapacity                 = 16
)

var (
	user32             = windows.NewLazySystemDLL("user32.dll")
	registerHotKey     = user32.NewProc("RegisterHotKey")
	unregisterHotKey   = user32.NewProc("UnregisterHotKey")
	getMessage         = user32.NewProc("GetMessageW")
	peekMessage        = user32.NewProc("PeekMessageW")
	postThreadMessage  = user32.NewProc("PostThreadMessageW")
	getCurrentThreadID = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentThreadId")
	createMutex        = windows.NewLazySystemDLL("kernel32.dll").NewProc("CreateMutexW")
	closeHandle        = windows.NewLazySystemDLL("kernel32.dll").NewProc("CloseHandle")
)

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

type Agent struct {
	dispatcher command.Dispatcher
	output     io.Writer
	logger     *slog.Logger
}

func NewAgent(dispatcher command.Dispatcher, output io.Writer, logger *slog.Logger) Agent {
	return Agent{dispatcher: dispatcher, output: output, logger: logger}
}

func (agent Agent) Run(ctx context.Context, registry command.Registry) error {
	mutex, err := acquireAgentMutex()
	if err != nil {
		return err
	}
	defer closeHandle.Call(mutex)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	registeredCommands, err := agent.registerCommands(registry.All())
	if err != nil {
		return err
	}
	defer unregisterAll(registeredCommands)

	jobs := make(chan command.Definition, jobQueueCapacity)
	var workers sync.WaitGroup
	workers.Add(1)
	go agent.runWorker(jobs, &workers)

	threadID, _, _ := getCurrentThreadID.Call()
	var pending message
	peekMessage.Call(uintptr(unsafe.Pointer(&pending)), 0, 0, 0, peekNoRemove)
	loopContext, cancelLoop := context.WithCancel(ctx)
	defer cancelLoop()
	go func() {
		<-loopContext.Done()
		postThreadMessage.Call(threadID, messageQuit, 0, 0)
	}()

	err = agent.messageLoop(registeredCommands, jobs)
	close(jobs)
	workers.Wait()
	return err
}

func (agent Agent) registerCommands(commands []command.Definition) (map[uintptr]command.Definition, error) {
	registered := make(map[uintptr]command.Definition)
	for _, configuredCommand := range commands {
		if configuredCommand.Hotkey == "" {
			continue
		}
		binding, err := Parse(configuredCommand.Hotkey)
		if err != nil {
			return nil, fmt.Errorf("parse hotkey for %q: %w", configuredCommand.Name, err)
		}
		identifier := registrationID(configuredCommand.Name)
		if existing, duplicate := registered[identifier]; duplicate {
			return nil, fmt.Errorf("registration ID collision between %q and %q", existing.Name, configuredCommand.Name)
		}
		result, _, callErr := registerHotKey.Call(0, identifier, uintptr(binding.Modifiers), uintptr(binding.VirtualKey))
		if result == 0 {
			fmt.Fprintf(agent.output, "hotkey unavailable: %s -> %s (%v)\n", configuredCommand.Hotkey, configuredCommand.Name, callErr)
			agent.logger.Warn("hotkey unavailable", "hotkey", configuredCommand.Hotkey, "command", configuredCommand.Name, "error", callErr)
			continue
		}
		registered[identifier] = configuredCommand
		fmt.Fprintf(agent.output, "hotkey registered: %s -> %s\n", configuredCommand.Hotkey, configuredCommand.Name)
		agent.logger.Info("hotkey registered", "hotkey", configuredCommand.Hotkey, "command", configuredCommand.Name)
	}
	if len(registered) == 0 {
		return nil, fmt.Errorf("no hotkeys were registered")
	}
	return registered, nil
}

func (agent Agent) messageLoop(commands map[uintptr]command.Definition, jobs chan<- command.Definition) error {
	for {
		var event message
		result, _, callErr := getMessage.Call(uintptr(unsafe.Pointer(&event)), 0, 0, 0)
		if int32(result) == -1 {
			return fmt.Errorf("read Windows message: %w", callErr)
		}
		if result == 0 || event.Message == messageQuit {
			return nil
		}
		if event.Message != messageHotkey {
			continue
		}
		configuredCommand, found := commands[event.WParam]
		if !found {
			continue
		}
		select {
		case jobs <- configuredCommand:
		default:
			fmt.Fprintf(agent.output, "hotkey ignored because the action queue is full: %s\n", configuredCommand.Name)
			agent.logger.Warn("hotkey action queue full", "command", configuredCommand.Name)
		}
	}
}

func (agent Agent) runWorker(jobs <-chan command.Definition, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for configuredCommand := range jobs {
		if err := agent.dispatcher.Dispatch(context.Background(), configuredCommand); err != nil {
			fmt.Fprintf(agent.output, "command failed: %s: %v\n", configuredCommand.Name, err)
			agent.logger.Error("hotkey command failed", "command", configuredCommand.Name, "error", err)
		}
	}
}

func acquireAgentMutex() (uintptr, error) {
	name, err := windows.UTF16PtrFromString(`Local\yawa-agent`)
	if err != nil {
		return 0, err
	}
	handle, _, callErr := createMutex.Call(0, 1, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		return 0, fmt.Errorf("create agent mutex: %w", callErr)
	}
	if callErr == errorAlreadyExists {
		closeHandle.Call(handle)
		return 0, fmt.Errorf("another yawa agent is already running")
	}
	return handle, nil
}

func unregisterAll(commands map[uintptr]command.Definition) {
	for identifier := range commands {
		unregisterHotKey.Call(0, identifier)
	}
}

func registrationID(name string) uintptr {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(name))
	return uintptr(hash.Sum32() & 0x7FFFFFFF)
}
