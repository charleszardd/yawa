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

func (spy *launcherSpy) Execute(_ context.Context, executable string, arguments []string, workingDirectory string) error {
	spy.executable = executable
	spy.arguments = arguments
	spy.workingDirectory = workingDirectory
	return nil
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
