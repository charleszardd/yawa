//go:build windows

package agentapp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/charleszardd/yawa/internal/agentcontrol"
	"github.com/charleszardd/yawa/internal/command"
	"github.com/charleszardd/yawa/internal/config"
	"github.com/charleszardd/yawa/internal/dialogs"
	"github.com/charleszardd/yawa/internal/hotkey"
	"github.com/charleszardd/yawa/internal/launcher"
	"github.com/charleszardd/yawa/internal/tray"
)

const shutdownTimeout = 5 * time.Second

type Application struct {
	logger     *slog.Logger
	supervisor *supervisor
}

func New(logger *slog.Logger) *Application {
	windowsLauncher := launcher.Windows{}
	dispatcher := command.NewDispatcher(windowsLauncher, dialogs.Picker{})
	return &Application{
		logger:     logger,
		supervisor: &supervisor{dispatcher: dispatcher, logger: logger},
	}
}

func (application *Application) Run(parentContext context.Context) error {
	instance, err := agentcontrol.Acquire()
	if err != nil {
		return err
	}
	defer instance.Close()

	ctx, cancel := context.WithCancel(parentContext)
	defer cancel()
	application.supervisor.SetReadinessCallbacks(instance.MarkInactive, instance.MarkActive)
	application.supervisor.Reload(ctx)
	defer application.supervisor.Stop()

	go func() {
		for signal := range instance.Signals(ctx) {
			switch signal {
			case agentcontrol.SignalReload:
				application.supervisor.Reload(ctx)
			case agentcontrol.SignalStop:
				cancel()
				return
			}
		}
	}()

	trayApplication := tray.New(tray.Options{
		Tooltip: "Yawa — global shortcuts",
		Open: func() {
			if err := openUI(); err != nil {
				application.logger.Error("open Yawa UI", "error", err)
			}
		},
		Reload: func() { application.supervisor.Reload(ctx) },
		Exit:   cancel,
	})
	application.logger.Info("background agent started")
	if err := trayApplication.Run(ctx); err != nil {
		return err
	}
	application.logger.Info("background agent stopped")
	return nil
}

type supervisor struct {
	lifecycle    sync.Mutex
	cancel       context.CancelFunc
	done         chan struct{}
	dispatcher   command.Dispatcher
	logger       *slog.Logger
	markInactive func() error
	markActive   func() error
}

func (worker *supervisor) SetReadinessCallbacks(markInactive func() error, markActive func() error) {
	worker.lifecycle.Lock()
	defer worker.lifecycle.Unlock()
	worker.markInactive = markInactive
	worker.markActive = markActive
}

func (worker *supervisor) Reload(parentContext context.Context) {
	worker.lifecycle.Lock()
	defer worker.lifecycle.Unlock()
	worker.stopLocked()
	if worker.markInactive != nil {
		_ = worker.markInactive()
	}
	ctx, cancel := context.WithCancel(parentContext)
	done := make(chan struct{})
	worker.cancel = cancel
	worker.done = done
	go func() {
		defer close(done)
		if err := worker.run(ctx); err != nil && ctx.Err() == nil {
			worker.logger.Error("hotkey agent stopped", "error", err)
		}
	}()
}

func (worker *supervisor) Stop() {
	worker.lifecycle.Lock()
	defer worker.lifecycle.Unlock()
	worker.stopLocked()
}

func (worker *supervisor) stopLocked() {
	cancel := worker.cancel
	done := worker.done
	worker.cancel = nil
	worker.done = nil
	if cancel == nil {
		return
	}
	cancel()
	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		worker.logger.Warn("hotkey agent shutdown timed out")
	}
}

func (worker *supervisor) run(ctx context.Context) error {
	configPath, err := config.DefaultPath()
	if err != nil {
		return err
	}
	configuration, _, err := config.NewStore().Load(configPath)
	if err != nil {
		return err
	}
	if err := config.Validate(configuration); err != nil {
		return fmt.Errorf("validate configuration: %w", err)
	}
	registry := command.NewRegistry(configuration.Commands)
	worker.logger.Info("loading hotkeys", "commands", len(registry.All()))
	return hotkey.NewAgent(worker.dispatcher, io.Discard, worker.logger).RunWithReady(ctx, registry, func() {
		if worker.markActive != nil {
			_ = worker.markActive()
		}
	})
}

func openUI() error {
	agentExecutable, err := os.Executable()
	if err == nil {
		uiPath := filepath.Join(filepath.Dir(agentExecutable), "yawa-ui.exe")
		if fileInfo, statErr := os.Stat(uiPath); statErr == nil && !fileInfo.IsDir() {
			return startDetached(uiPath)
		}
	}
	uiPath, err := exec.LookPath("yawa-ui.exe")
	if err != nil {
		return fmt.Errorf("yawa-ui.exe was not found")
	}
	return startDetached(uiPath)
}

func startDetached(executablePath string) error {
	process := exec.Command(executablePath)
	if err := process.Start(); err != nil {
		return err
	}
	return process.Process.Release()
}
