//go:build windows

package launcher

import (
	"context"
	"strings"
	"testing"
)

func TestExecuteReportsMissingExecutable(t *testing.T) {
	err := (Windows{}).Execute(context.Background(), "yawa-definitely-missing-executable.exe", nil, "")
	if err == nil || !strings.Contains(err.Error(), "resolve executable") {
		t.Fatalf("Execute() error = %v", err)
	}
}
