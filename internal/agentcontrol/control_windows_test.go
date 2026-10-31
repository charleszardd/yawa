//go:build windows

package agentcontrol

import (
	"context"
	"testing"
	"time"
)

func TestInstanceReceivesReloadAndStopSignals(t *testing.T) {
	instance, err := Acquire()
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer instance.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := instance.Signals(ctx)
	if IsActive() {
		t.Fatal("IsActive() = true before MarkActive()")
	}
	if err := instance.MarkActive(); err != nil {
		t.Fatalf("MarkActive() error = %v", err)
	}
	if !IsActive() {
		t.Fatal("IsActive() = false after MarkActive()")
	}
	if err := instance.MarkInactive(); err != nil {
		t.Fatalf("MarkInactive() error = %v", err)
	}
	if IsActive() {
		t.Fatal("IsActive() = true after MarkInactive()")
	}

	if err := RequestReload(); err != nil {
		t.Fatalf("RequestReload() error = %v", err)
	}
	assertSignal(t, signals, SignalReload)
	if err := RequestStop(); err != nil {
		t.Fatalf("RequestStop() error = %v", err)
	}
	assertSignal(t, signals, SignalStop)
}

func assertSignal(t *testing.T, signals <-chan Signal, expected Signal) {
	t.Helper()
	select {
	case actual := <-signals:
		if actual != expected {
			t.Fatalf("signal = %d, want %d", actual, expected)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for signal %d", expected)
	}
}
