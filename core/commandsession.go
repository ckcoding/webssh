package core

import (
	"strings"
	"sync"
	"time"
)

const (
	commandSessionIdleTimeout         = 5 * time.Minute
	commandSessionHealthCheckInterval = 20 * time.Second
	commandSessionCleanupInterval     = time.Minute
)

// CommandSession 复用短命令所需的 SSH 连接，避免状态轮询反复握手。
type CommandSession struct {
	base        SSHClient
	client      *SSHClient
	lastUsed    time.Time
	lastChecked time.Time
	mu          sync.Mutex
}

var commandSessionPool = struct {
	sessions    map[string]*CommandSession
	lastCleanup time.Time
	mu          sync.Mutex
}{
	sessions: make(map[string]*CommandSession),
}

// GetCommandSession 获取短命令复用会话。
func GetCommandSession(sshInfo string) (*CommandSession, error) {
	base, err := DecodedMsgToSSHClient(sshInfo)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	key := fingerprintFileSession(strings.TrimSpace(sshInfo))

	commandSessionPool.mu.Lock()
	defer commandSessionPool.mu.Unlock()

	if now.Sub(commandSessionPool.lastCleanup) >= commandSessionCleanupInterval {
		for key, session := range commandSessionPool.sessions {
			if session.idleBeyond(now, commandSessionIdleTimeout) {
				delete(commandSessionPool.sessions, key)
			}
		}
		commandSessionPool.lastCleanup = now
	}

	if session, ok := commandSessionPool.sessions[key]; ok {
		session.markUsed(now)
		return session, nil
	}

	session := &CommandSession{
		base:     base,
		lastUsed: now,
	}
	commandSessionPool.sessions[key] = session
	return session, nil
}

// Use 在复用连接上执行短命令。
func (s *CommandSession) Use(fn func(*SSHClient) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureConnectedLocked(); err != nil {
		return err
	}

	err := fn(s.client)
	s.lastUsed = time.Now()
	if err != nil && isBrokenFileSessionError(err) {
		s.closeLocked()
	}
	return err
}

func (s *CommandSession) ensureConnectedLocked() error {
	now := time.Now()
	if s.client != nil && s.client.Client != nil {
		if now.Sub(s.lastChecked) < commandSessionHealthCheckInterval {
			s.lastUsed = now
			return nil
		}
		if _, _, err := s.client.Client.SendRequest("keepalive@webssh", false, nil); err == nil {
			s.lastChecked = now
			s.lastUsed = now
			return nil
		}
		s.closeLocked()
	}

	client := s.base
	if err := client.GenerateClient(); err != nil {
		return err
	}
	s.client = &client
	s.lastChecked = now
	s.lastUsed = now
	return nil
}

func (s *CommandSession) idleBeyond(now time.Time, timeout time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if now.Sub(s.lastUsed) < timeout {
		return false
	}
	s.closeLocked()
	return true
}

func (s *CommandSession) markUsed(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUsed = now
}

func (s *CommandSession) closeLocked() {
	if s.client != nil {
		s.client.Close()
		s.client = nil
	}
}
