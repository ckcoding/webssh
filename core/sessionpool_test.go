package core

import (
	"fmt"
	"testing"
	"time"
)

func TestCommandPoolSlowHostDoesNotBlockOtherHosts(t *testing.T) {
	commandSessionPool.mu.Lock()
	previous, previousCleanup := commandSessionPool.sessions, commandSessionPool.lastCleanup
	busy := &CommandSession{lastUsed: time.Now().Add(-time.Hour)}
	commandSessionPool.sessions = map[string]*CommandSession{"busy": busy}
	commandSessionPool.lastCleanup = time.Time{}
	commandSessionPool.mu.Unlock()
	busy.mu.Lock()
	busyLookup := make(chan struct{})
	go func() { GetCommandSessionByKey("busy", NewSSHClient()); close(busyLookup) }()
	t.Cleanup(func() {
		busy.mu.Unlock()
		<-busyLookup
		commandSessionPool.mu.Lock()
		commandSessionPool.sessions, commandSessionPool.lastCleanup = previous, previousCleanup
		commandSessionPool.mu.Unlock()
	})
	done := make(chan struct{})
	go func() { GetCommandSessionByKey("other", NewSSHClient()); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("slow host blocked the entire command pool")
	}
}

func TestFilePoolSlowHostDoesNotBlockOtherHosts(t *testing.T) {
	fileSessionPool.mu.Lock()
	previous, previousCleanup := fileSessionPool.sessions, fileSessionPool.lastCleanup
	busy := &FileSession{lastUsed: time.Now().Add(-time.Hour)}
	fileSessionPool.sessions = map[string]*FileSession{"busy": busy}
	fileSessionPool.lastCleanup = time.Time{}
	fileSessionPool.mu.Unlock()
	busy.mu.Lock()
	busyLookup := make(chan struct{})
	go func() { GetFileSessionByKey("busy", NewSSHClient()); close(busyLookup) }()
	t.Cleanup(func() {
		busy.mu.Unlock()
		<-busyLookup
		fileSessionPool.mu.Lock()
		fileSessionPool.sessions, fileSessionPool.lastCleanup = previous, previousCleanup
		fileSessionPool.mu.Unlock()
	})
	done := make(chan struct{})
	go func() { GetFileSessionByKey("other", NewSSHClient()); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("slow host blocked the entire file pool")
	}
}

func TestSessionPoolsReuseCachedEntryAtCapacity(t *testing.T) {
	commandSessionPool.mu.Lock()
	previousCommands, previousCommandCleanup := commandSessionPool.sessions, commandSessionPool.lastCleanup
	commandSessionPool.sessions = make(map[string]*CommandSession)
	commandSessionPool.lastCleanup = time.Now()
	fileSessionPool.mu.Lock()
	previousFiles, previousFileCleanup := fileSessionPool.sessions, fileSessionPool.lastCleanup
	fileSessionPool.sessions = make(map[string]*FileSession)
	fileSessionPool.lastCleanup = time.Now()
	for i := range commandSessionMaxEntries {
		key := fmt.Sprintf("host-%d", i)
		used := time.Now().Add(time.Duration(i) * time.Second)
		commandSessionPool.sessions[key] = &CommandSession{lastUsed: used}
		fileSessionPool.sessions[key] = &FileSession{lastUsed: used}
	}
	command := commandSessionPool.sessions["host-0"]
	file := fileSessionPool.sessions["host-0"]
	fileSessionPool.mu.Unlock()
	commandSessionPool.mu.Unlock()
	t.Cleanup(func() {
		commandSessionPool.mu.Lock()
		commandSessionPool.sessions, commandSessionPool.lastCleanup = previousCommands, previousCommandCleanup
		commandSessionPool.mu.Unlock()
		fileSessionPool.mu.Lock()
		fileSessionPool.sessions, fileSessionPool.lastCleanup = previousFiles, previousFileCleanup
		fileSessionPool.mu.Unlock()
	})
	if GetCommandSessionByKey("host-0", NewSSHClient()) != command {
		t.Fatal("cached command connection was evicted before lookup")
	}
	if GetFileSessionByKey("host-0", NewSSHClient()) != file {
		t.Fatal("cached file connection was evicted before lookup")
	}
}
