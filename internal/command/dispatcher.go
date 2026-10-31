package command

import (
	"context"
	"fmt"
	"strings"
)

const SelectedPathPlaceholder = "{selected_path}"

type Launcher interface {
	Execute(ctx context.Context, executable string, arguments []string, workingDirectory string) error
	OpenPath(path string) error
	OpenURL(url string) error
}

type SelectionKind int

const (
	SelectFile SelectionKind = iota
	SelectFolder
)

type Picker interface {
	Select(ctx context.Context, kind SelectionKind, startDirectory string) (path string, selected bool, err error)
}

type Dispatcher struct {
	launcher Launcher
	picker   Picker
}

func NewDispatcher(launcher Launcher, pickers ...Picker) Dispatcher {
	dispatcher := Dispatcher{launcher: launcher}
	if len(pickers) > 0 {
		dispatcher.picker = pickers[0]
	}
	return dispatcher
}

func (dispatcher Dispatcher) Dispatch(ctx context.Context, configuredCommand Definition) error {
	var err error
	switch configuredCommand.Type {
	case TypeExec:
		err = dispatcher.launcher.Execute(ctx, configuredCommand.Executable, configuredCommand.Arguments, configuredCommand.WorkingDir)
	case TypePickFolder:
		err = dispatcher.dispatchPicker(ctx, configuredCommand, SelectFolder)
	case TypePickFile:
		err = dispatcher.dispatchPicker(ctx, configuredCommand, SelectFile)
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

func (dispatcher Dispatcher) dispatchPicker(ctx context.Context, configuredCommand Definition, kind SelectionKind) error {
	if dispatcher.picker == nil {
		return fmt.Errorf("native picker is unavailable")
	}
	selectedPath, selected, err := dispatcher.picker.Select(ctx, kind, configuredCommand.StartDir)
	if err != nil || !selected {
		return err
	}
	arguments := make([]string, len(configuredCommand.Arguments))
	for index, argument := range configuredCommand.Arguments {
		arguments[index] = strings.ReplaceAll(argument, SelectedPathPlaceholder, selectedPath)
	}
	return dispatcher.launcher.Execute(ctx, configuredCommand.Executable, arguments, configuredCommand.WorkingDir)
}
