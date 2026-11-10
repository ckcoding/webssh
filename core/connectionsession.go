package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	connectionSessionIdleTimeout     = 8 * time.Hour
	connectionSessionCleanupInterval = time.Minute
	connectionSessionMaxEntries      = 500
)

type ConnectionSession struct {
	ID       string
	Client   SSHClient
	LastUsed time.Time
}

var connectionSessionPool = struct {
	sessions    map[string]*ConnectionSession
	lastCleanup time.Time
	mu          sync.Mutex
}{
	sessions: make(map[string]*ConnectionSession),
}

func CreateConnectionSession(client SSHClient) (*ConnectionSession, error) {
	id, err := randomSessionID()
	if err != nil {
		return nil, err
	}
	client.Client = nil
	client.Sftp = nil
	client.StdinPipe = nil
	client.Session = nil
	client.wsWriteMu = nil
	client.wsOutput = nil
	now := time.Now()
	session := &ConnectionSession{
		ID:       id,
		Client:   client,
		LastUsed: now,
	}

	connectionSessionPool.mu.Lock()
	defer connectionSessionPool.mu.Unlock()
	cleanupConnectionSessionsLocked(now)
	evictOldestConnectionSessionLocked()
	connectionSessionPool.sessions[id] = session
	return session, nil
}

func evictOldestConnectionSessionLocked() {
	if len(connectionSessionPool.sessions) < connectionSessionMaxEntries {
		return
	}
	var oldestID string
	var oldestUsed time.Time
	for id, session := range connectionSessionPool.sessions {
		if oldestID == "" || session.LastUsed.Before(oldestUsed) {
			oldestID = id
			oldestUsed = session.LastUsed
		}
	}
	if oldestID != "" {
		delete(connectionSessionPool.sessions, oldestID)
	}
}

func ResolveSSHClient(sessionID, sshInfo string) (SSHClient, string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID != "" {
		session, ok := getConnectionSession(sessionID)
		if !ok {
			return SSHClient{}, "", fmt.Errorf("connection session expired")
		}
		return session.Client, "session:" + session.ID, nil
	}

	client, err := DecodedMsgToSSHClient(sshInfo)
	if err != nil {
		return SSHClient{}, "", err
	}
	return client, fingerprintFileSession(strings.TrimSpace(sshInfo)), nil
}

func getConnectionSession(id string) (*ConnectionSession, bool) {
	now := time.Now()
	connectionSessionPool.mu.Lock()
	defer connectionSessionPool.mu.Unlock()
	cleanupConnectionSessionsLocked(now)

	session, ok := connectionSessionPool.sessions[id]
	if !ok {
		return nil, false
	}
	if now.Sub(session.LastUsed) >= connectionSessionIdleTimeout {
		delete(connectionSessionPool.sessions, id)
		return nil, false
	}
	session.LastUsed = now
	return session, true
}

func cleanupConnectionSessionsLocked(now time.Time) {
	if now.Sub(connectionSessionPool.lastCleanup) < connectionSessionCleanupInterval {
		return
	}
	for id, session := range connectionSessionPool.sessions {
		if now.Sub(session.LastUsed) >= connectionSessionIdleTimeout {
			delete(connectionSessionPool.sessions, id)
		}
	}
	connectionSessionPool.lastCleanup = now
}

func randomSessionID() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
