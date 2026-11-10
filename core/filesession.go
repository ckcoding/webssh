package core

import (
	"hash/fnv"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	fileSessionIdleTimeout         = 5 * time.Minute
	fileSessionHealthCheckInterval = 20 * time.Second
	fileSessionCleanupInterval     = time.Minute
	fileSessionMaxEntries          = 200
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

func GetFileSessionByID(sessionID, sshInfo string) (*FileSession, error) {
	base, key, err := ResolveSSHClient(sessionID, sshInfo)
	if err != nil {
		return nil, err
	}
	return GetFileSessionByKey(key, base), nil
}

func GetFileSessionByKey(key string, base SSHClient) *FileSession {
	now := time.Now()
	key = strings.TrimSpace(key)

	fileSessionPool.mu.Lock()
	if session, ok := fileSessionPool.sessions[key]; ok {
		fileSessionPool.mu.Unlock()
		session.markUsed(now)
		return session
	}
	defer fileSessionPool.mu.Unlock()

	if now.Sub(fileSessionPool.lastCleanup) >= fileSessionCleanupInterval {
		for key, session := range fileSessionPool.sessions {
			if session.idleBeyond(now, fileSessionIdleTimeout) {
				delete(fileSessionPool.sessions, key)
			}
		}
		fileSessionPool.lastCleanup = now
	}

	// 超过最大条目数时淘汰最久未使用的会话
	if len(fileSessionPool.sessions) >= fileSessionMaxEntries {
		var oldestKey string
		var oldestTime time.Time
		for key, session := range fileSessionPool.sessions {
			if !session.mu.TryLock() {
				continue
			}
			used := session.lastUsed
			session.mu.Unlock()
			if oldestKey == "" || used.Before(oldestTime) {
				oldestKey = key
				oldestTime = used
			}
		}
		if oldestKey != "" {
			if session, ok := fileSessionPool.sessions[oldestKey]; ok && session.mu.TryLock() {
				session.closeLocked()
				delete(fileSessionPool.sessions, oldestKey)
				session.mu.Unlock()
			}
		}
	}

	session := &FileSession{
		base:     base,
		lastUsed: now,
	}
	fileSessionPool.sessions[key] = session
	return session
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
	// A slow operation on one host must not block every pool lookup.
	if !s.mu.TryLock() {
		return false
	}
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
	msg := err.Error()
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "connection reset by peer") ||
		strings.Contains(lower, "connection lost") ||
		strings.Contains(lower, "eof") ||
		strings.Contains(lower, "session is closed") ||
		strings.Contains(lower, "use of closed network connection")
}

func fingerprintFileSession(value string) string {
	h := fnv.New64a()
	h.Write([]byte(value))
	return strconv.FormatUint(h.Sum64(), 16)
}
