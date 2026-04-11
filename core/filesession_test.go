package core

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestGetFileSessionReuseAndCleanup(t *testing.T) {
	resetFileSessionPoolForTest()

	sshInfoA := encodeSSHInfoForTest(t, SSHClient{
		Username:  "root",
		Password:  "secret-a",
		IPAddress: "192.168.1.10",
		Port:      22,
		LoginType: 0,
	})
	sshInfoB := encodeSSHInfoForTest(t, SSHClient{
		Username:  "root",
		Password:  "secret-b",
		IPAddress: "192.168.1.11",
		Port:      22,
		LoginType: 0,
	})

	sessionA1, err := GetFileSession(sshInfoA)
	if err != nil {
		t.Fatalf("GetFileSession() error = %v", err)
	}
	sessionA2, err := GetFileSession(sshInfoA)
	if err != nil {
		t.Fatalf("GetFileSession() second call error = %v", err)
	}
	if sessionA1 != sessionA2 {
		t.Fatalf("expected the same session instance to be reused")
	}

	sessionA1.lastUsed = time.Now().Add(-fileSessionIdleTimeout - time.Minute)
	fileSessionPool.mu.Lock()
	fileSessionPool.lastCleanup = time.Now().Add(-fileSessionCleanupInterval - time.Second)
	fileSessionPool.mu.Unlock()
	if _, err := GetFileSession(sshInfoB); err != nil {
		t.Fatalf("GetFileSession() with another ssh info error = %v", err)
	}

	fileSessionPool.mu.Lock()
	_, stillExists := fileSessionPool.sessions[fingerprintFileSession(sshInfoA)]
	fileSessionPool.mu.Unlock()
	if stillExists {
		t.Fatalf("expected idle session to be cleaned up")
	}
}

func TestIsBrokenFileSessionError(t *testing.T) {
	if !isBrokenFileSessionError(errors.New("EOF")) {
		t.Fatalf("expected EOF to be treated as a broken file session")
	}
	if isBrokenFileSessionError(errors.New("permission denied")) {
		t.Fatalf("did not expect permission errors to invalidate the session")
	}
}

func encodeSSHInfoForTest(t *testing.T, client SSHClient) string {
	t.Helper()
	payload, err := json.Marshal(client)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return base64.StdEncoding.EncodeToString(payload)
}

func resetFileSessionPoolForTest() {
	fileSessionPool.mu.Lock()
	defer fileSessionPool.mu.Unlock()
	fileSessionPool.sessions = make(map[string]*FileSession)
	fileSessionPool.lastCleanup = time.Time{}
}
