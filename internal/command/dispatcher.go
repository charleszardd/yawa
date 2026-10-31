package command

import (
	"context"
	"fmt"
)

type Launcher interface {
	Execute(ctx context.Context, executable string, arguments []string, workingDirectory string) error
	OpenPath(path string) error
	OpenURL(url string) error
}

type Dispatcher struct {
	launcher Launcher
}

func NewDispatcher(launcher Launcher) Dispatcher {
	return Dispatcher{launcher: launcher}
}

func (dispatcher Dispatcher) Dispatch(ctx context.Context, configuredCommand Definition) error {
	var err error
	switch configuredCommand.Type {
	case TypeExec:
		err = dispatcher.launcher.Execute(ctx, configuredCommand.Executable, configuredCommand.Arguments, configuredCommand.WorkingDir)
	case TypeOpenPath:
		err = dispatcher.launcher.OpenPath(configuredCommand.Path)
	case TypeURL:
		err = dispatcher.launcher.OpenURL(configuredCommand.URL)
	default:
		return fmt.Errorf("unsupported command type %q", configuredCommand.Type)
	}

	if err != nil {
		return fmt.Errorf("run command %q: %w", configuredCommand.Name, err)
	}
	return nil
}
