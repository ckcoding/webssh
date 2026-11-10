package core

import (
	"fmt"
	"sync"
	"time"
)

const (
	preparedTerminalTTL        = 30 * time.Second
	preparedTerminalMaxEntries = 32
)

type preparedTerminal struct {
	client  SSHClient
	expires time.Time
	timer   *time.Timer
}

var preparedTerminals = struct {
	sync.Mutex
	entries map[string]*preparedTerminal
}{entries: make(map[string]*preparedTerminal)}

// PrepareTerminalConnection hands an authenticated connection to one terminal.
// Other requests still resolve credentials and use their own SSH connections.
func PrepareTerminalConnection(client SSHClient) (*ConnectionSession, error) {
	if client.Client == nil {
		return nil, fmt.Errorf("SSH connection is not authenticated")
	}
	session, err := CreateConnectionSession(client)
	if err != nil {
		return nil, err
	}
	entry := &preparedTerminal{client: client, expires: time.Now().Add(preparedTerminalTTL)}
	preparedTerminals.Lock()
	var evicted *preparedTerminal
	if len(preparedTerminals.entries) >= preparedTerminalMaxEntries {
		var oldestID string
		for id, candidate := range preparedTerminals.entries {
			if evicted == nil || candidate.expires.Before(evicted.expires) {
				oldestID, evicted = id, candidate
			}
		}
		delete(preparedTerminals.entries, oldestID)
		evicted.timer.Stop()
	}
	preparedTerminals.entries[session.ID] = entry
	entry.timer = time.AfterFunc(preparedTerminalTTL, func() {
		expirePreparedTerminal(session.ID, entry)
	})
	preparedTerminals.Unlock()
	if evicted != nil {
		evicted.client.Close()
	}
	return session, nil
}

func expirePreparedTerminal(id string, entry *preparedTerminal) {
	preparedTerminals.Lock()
	if preparedTerminals.entries[id] != entry {
		preparedTerminals.Unlock()
		return
	}
	delete(preparedTerminals.entries, id)
	preparedTerminals.Unlock()
	entry.client.Close()
}

// TakePreparedTerminalConnection transfers ownership once; an expired or absent
// connection falls back to an ordinary terminal handshake using the session ID.
func TakePreparedTerminalConnection(id string) *SSHClient {
	preparedTerminals.Lock()
	entry := preparedTerminals.entries[id]
	if entry != nil {
		delete(preparedTerminals.entries, id)
		entry.timer.Stop()
	}
	preparedTerminals.Unlock()
	if entry == nil {
		return nil
	}
	if !time.Now().Before(entry.expires) {
		entry.client.Close()
		return nil
	}
	return &entry.client
}
