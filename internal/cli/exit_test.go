package cli

import (
	"errors"
	"testing"
)

func TestExitCodePreservesCategoryThroughWrapping(t *testing.T) {
	err := withExitCode(ExitExecution, errors.New("launch failed"))
	if code := ExitCode(err); code != ExitExecution {
		t.Fatalf("ExitCode() = %d", code)
	}
}
