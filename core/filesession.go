package core

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

const (
	fileSessionIdleTimeout         = 5 * time.Minute
	fileSessionHealthCheckInterval = 20 * time.Second
	fileSessionCleanupInterval     = time.Minute
)

// FileSession 复用文件操作所需的 SSH/SFTP 连接，避免每次请求重新建连。
type FileSession struct {
	base        SSHClient
	client      *SSHClient
	lastUsed    time.Time
	lastChecked time.Time
	mu          sync.Mutex
}

var fileSessionPool = struct {
	sessions    map[string]*FileSession
	lastCleanup time.Time
	mu          sync.Mutex
}{
	sessions: make(map[string]*FileSession),
}

// GetFileSession 获取文件操作复用会话。
func GetFileSession(sshInfo string) (*FileSession, error) {
	base, err := DecodedMsgToSSHClient(sshInfo)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	key := fingerprintFileSession(strings.TrimSpace(sshInfo))

	fileSessionPool.mu.Lock()
	defer fileSessionPool.mu.Unlock()

	if now.Sub(fileSessionPool.lastCleanup) >= fileSessionCleanupInterval {
		for key, session := range fileSessionPool.sessions {
			if session.idleBeyond(now, fileSessionIdleTimeout) {
				delete(fileSessionPool.sessions, key)
			}
		}
		fileSessionPool.lastCleanup = now
	}

	if session, ok := fileSessionPool.sessions[key]; ok {
		session.markUsed(now)
		return session, nil
	}

	session := &FileSession{
		base:     base,
		lastUsed: now,
	}
	fileSessionPool.sessions[key] = session
	return session, nil
}

// Use 在复用连接上执行文件操作。
func (s *FileSession) Use(fn func(*SSHClient) error) error {
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

func (s *FileSession) ensureConnectedLocked() error {
	now := time.Now()
	if s.client != nil && s.client.Client != nil && s.client.Sftp != nil {
		if now.Sub(s.lastChecked) < fileSessionHealthCheckInterval {
			s.lastUsed = now
			return nil
		}
		if _, err := s.client.Sftp.Stat("."); err == nil {
			s.lastChecked = now
			s.lastUsed = now
			return nil
		}
		s.closeLocked()
	}

	client := s.base
	if err := client.CreateSftp(); err != nil {
		return err
	}
	s.client = &client
	s.lastChecked = now
	s.lastUsed = now
	return nil
}

func (s *FileSession) idleBeyond(now time.Time, timeout time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if now.Sub(s.lastUsed) < timeout {
		return false
	}
	s.closeLocked()
	return true
}

func (s *FileSession) markUsed(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUsed = now
}

func (s *FileSession) closeLocked() {
	if s.client != nil {
		s.client.Close()
		s.client = nil
	}
}

func isBrokenFileSessionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "connection lost") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "session is closed") ||
		strings.Contains(msg, "use of closed network connection")
}

func fingerprintFileSession(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}
