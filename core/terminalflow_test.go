package core

import (
	"errors"
	"io"
	"testing"
	"time"
)

func TestTerminalFlowWaitsForParsedOutput(t *testing.T) {
	f := newTerminalFlow()
	if err := f.reserve(terminalOutputWindow); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- f.reserve(1) }()
	select {
	case <-done:
		t.Fatal("output exceeded the browser parsing window")
	case <-time.After(20 * time.Millisecond):
	}
	f.acknowledge(terminalOutputWindow + 1)
	f.acknowledge(-1)
	select {
	case <-done:
		t.Fatal("invalid acknowledgement released the output window")
	default:
	}
	f.acknowledge(64 * 1024)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("parsed output did not release the writer")
	}
	if f.pending != terminalOutputWindow-64*1024+1 {
		t.Fatalf("unexpected pending bytes: %d", f.pending)
	}
}

func TestTerminalFlowDisconnectUnblocksWriter(t *testing.T) {
	f := newTerminalFlow()
	_ = f.reserve(terminalOutputWindow)
	done := make(chan error, 1)
	go func() { done <- f.reserve(1) }()
	f.close()
	select {
	case err := <-done:
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("disconnect left an output goroutine blocked")
	}
}
