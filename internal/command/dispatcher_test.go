package command

import (
	"context"
	"reflect"
	"testing"
)

type launcherSpy struct {
	executable       string
	arguments        []string
	workingDirectory string
	path             string
	url              string
}

type pickerStub struct {
	path           string
	selected       bool
	kind           SelectionKind
	startDirectory string
}

func (picker *pickerStub) Select(_ context.Context, kind SelectionKind, startDirectory string) (string, bool, error) {
	picker.kind = kind
	picker.startDirectory = startDirectory
	return picker.path, picker.selected, nil
}

func (spy *launcherSpy) Execute(_ context.Context, executable string, arguments []string, workingDirectory string) error {
	spy.executable = executable
	spy.arguments = arguments
	spy.workingDirectory = workingDirectory
	return nil
}

func TestDispatcherSubstitutesPickedPathAsOneArgument(t *testing.T) {
	spy := &launcherSpy{}
	picker := &pickerStub{path: `C:\Projects\Yawa nga Proyekto 😈`, selected: true}
	dispatcher := NewDispatcher(spy, picker)
	configuredCommand := Definition{
		Name: "code", Type: TypePickFolder, Executable: "code.exe",
		Arguments: []string{"--new-window", SelectedPathPlaceholder}, StartDir: `C:\Projects`,
	}

	if err := dispatcher.Dispatch(context.Background(), configuredCommand); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if !reflect.DeepEqual(spy.arguments, []string{"--new-window", picker.path}) {
		t.Fatalf("arguments = %v", spy.arguments)
	}
	if picker.kind != SelectFolder || picker.startDirectory != configuredCommand.StartDir {
		t.Fatalf("picker = %+v", picker)
	}
}

func TestDispatcherDoesNotLaunchWhenPickerIsCancelled(t *testing.T) {
	spy := &launcherSpy{}
	dispatcher := NewDispatcher(spy, &pickerStub{selected: false})
	if err := dispatcher.Dispatch(context.Background(), Definition{Name: "file", Type: TypePickFile}); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if spy.executable != "" {
		t.Fatalf("Execute() executable = %q", spy.executable)
	}
}

func (spy *launcherSpy) OpenPath(path string) error {
	spy.path = path
	return nil
}

func (spy *launcherSpy) OpenURL(url string) error {
	spy.url = url
	return nil
}

func TestDispatcherDispatchesExecCommand(t *testing.T) {
	spy := &launcherSpy{}
	dispatcher := NewDispatcher(spy)
	configuredCommand := Definition{Name: "editor", Type: TypeExec, Executable: "editor.exe", Arguments: []string{"project"}, WorkingDir: `C:\projects`}

	if err := dispatcher.Dispatch(context.Background(), configuredCommand); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if spy.executable != configuredCommand.Executable || !reflect.DeepEqual(spy.arguments, configuredCommand.Arguments) || spy.workingDirectory != configuredCommand.WorkingDir {
		t.Fatalf("Execute() received executable=%q arguments=%v workingDirectory=%q", spy.executable, spy.arguments, spy.workingDirectory)
	}
}

func TestDispatcherDispatchesShellCommands(t *testing.T) {
	tests := []struct {
		name    string
		command Definition
		assert  func(*testing.T, *launcherSpy)
	}{
		{name: "path", command: Definition{Name: "downloads", Type: TypeOpenPath, Path: `C:\Downloads`}, assert: func(t *testing.T, spy *launcherSpy) {
			if spy.path != `C:\Downloads` {
				t.Fatalf("OpenPath() path = %q", spy.path)
			}
		}},
		{name: "url", command: Definition{Name: "website", Type: TypeURL, URL: "https://example.com"}, assert: func(t *testing.T, spy *launcherSpy) {
			if spy.url != "https://example.com" {
				t.Fatalf("OpenURL() url = %q", spy.url)
			}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spy := &launcherSpy{}
			if err := NewDispatcher(spy).Dispatch(context.Background(), test.command); err != nil {
				t.Fatalf("Dispatch() error = %v", err)
			}
			test.assert(t, spy)
		})
	}
}
