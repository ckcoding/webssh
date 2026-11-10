package core

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

type trackedSSHConn struct {
	ssh.Conn
	closed atomic.Int32
}

func (c *trackedSSHConn) Close() error { c.closed.Add(1); return nil }

func preparedTestClient() (SSHClient, *trackedSSHConn) {
	conn := &trackedSSHConn{}
	return SSHClient{IPAddress: "127.0.0.1", Port: 22, Username: "test", Password: "test", Client: &ssh.Client{Conn: conn}}, conn
}

func resetPreparedTerminals(t *testing.T) {
	t.Helper()
	clear := func() {
		preparedTerminals.Lock()
		entries := preparedTerminals.entries
		preparedTerminals.entries = make(map[string]*preparedTerminal)
		for _, entry := range entries {
			entry.timer.Stop()
		}
		preparedTerminals.Unlock()
		for _, entry := range entries {
			entry.client.Close()
		}
	}
	clear()
	t.Cleanup(clear)
}

func TestPreparedTerminalHasOneConsumer(t *testing.T) {
	resetPreparedTerminals(t)
	client, tracked := preparedTestClient()
	session, err := PrepareTerminalConnection(client)
	if err != nil {
		t.Fatal(err)
	}
	resolved, _, err := ResolveSSHClient(session.ID, "")
	if err != nil || resolved.Client != nil {
		t.Fatalf("credentials must not share the live terminal connection: %v", err)
	}
	var winners atomic.Int32
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if taken := TakePreparedTerminalConnection(session.ID); taken != nil {
				winners.Add(1)
				taken.Close()
			}
		}()
	}
	wg.Wait()
	if winners.Load() != 1 || tracked.closed.Load() != 1 {
		t.Fatalf("consumers=%d closes=%d", winners.Load(), tracked.closed.Load())
	}
}

func TestPreparedTerminalExpiryPreservesCredentialFallback(t *testing.T) {
	resetPreparedTerminals(t)
	client, tracked := preparedTestClient()
	session, err := PrepareTerminalConnection(client)
	if err != nil {
		t.Fatal(err)
	}
	preparedTerminals.Lock()
	entry := preparedTerminals.entries[session.ID]
	entry.expires = time.Now().Add(-time.Second)
	preparedTerminals.Unlock()
	if TakePreparedTerminalConnection(session.ID) != nil {
		t.Fatal("expired connection was reused")
	}
	if tracked.closed.Load() != 1 {
		t.Fatal("expired connection was not closed")
	}
	if _, _, err := ResolveSSHClient(session.ID, ""); err != nil {
		t.Fatalf("credential fallback lost: %v", err)
	}
	expirePreparedTerminal(session.ID, entry)
	if tracked.closed.Load() != 1 {
		t.Fatal("late timer closed a consumed connection twice")
	}
}

func TestPreparedTerminalCapacityClosesAbandonedConnections(t *testing.T) {
	resetPreparedTerminals(t)
	client, first := preparedTestClient()
	if _, err := PrepareTerminalConnection(client); err != nil {
		t.Fatal(err)
	}
	for range preparedTerminalMaxEntries {
		client, _ := preparedTestClient()
		if _, err := PrepareTerminalConnection(client); err != nil {
			t.Fatal(err)
		}
	}
	preparedTerminals.Lock()
	count := len(preparedTerminals.entries)
	preparedTerminals.Unlock()
	if count != preparedTerminalMaxEntries || first.closed.Load() != 1 {
		t.Fatalf("entries=%d first closes=%d", count, first.closed.Load())
	}
}
