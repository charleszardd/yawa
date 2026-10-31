//go:build windows

package background

import "testing"

func TestStartupCommandQuotesExecutablePath(t *testing.T) {
	executablePath := `C:\Program Files\Yawa\yawa-agent.exe`
	if actual, expected := startupCommand(executablePath), `"C:\Program Files\Yawa\yawa-agent.exe"`; actual != expected {
		t.Fatalf("startupCommand() = %q, want %q", actual, expected)
	}
}
