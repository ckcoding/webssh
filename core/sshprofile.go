package core

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// SavedSSHProfile 已保存的 SSH 配置
type SavedSSHProfile struct {
	ID          string `json:"id"`
	Host        string `json:"host"`
	IP          string `json:"ip"`
	CountryFlag string `json:"countryFlag"`
	Location    string `json:"location"`
	OS          string `json:"os"`
	Memory      string `json:"memory"`
	ExpiresAt   string `json:"expiresAt"`
	Username    string `json:"username"`
	Port        int    `json:"port"`
	Password    string `json:"password"`
	LoginType   int    `json:"logintype"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// SSHProfileStore JSON 文件存储
type SSHProfileStore struct {
	path     string
	mu       sync.RWMutex
	profiles []SavedSSHProfile
}

var (
	ErrSavedSSHAlreadyExists = errors.New("saved ssh already exists")
	sshProfileStore          *SSHProfileStore
	sshProfileStoreErr       error
	sshProfileStoreOnce      sync.Once
)

// NewSSHProfileStore 创建存储实例
func NewSSHProfileStore(path string) (*SSHProfileStore, error) {
	store := &SSHProfileStore{
		path: path,
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

// GetSSHProfileStore 获取全局存储实例
func GetSSHProfileStore() (*SSHProfileStore, error) {
	sshProfileStoreOnce.Do(func() {
		profilePath, err := resolveSSHProfilePath()
		if err != nil {
			sshProfileStoreErr = err
			return
		}
		sshProfileStore, sshProfileStoreErr = NewSSHProfileStore(profilePath)
	})
	return sshProfileStore, sshProfileStoreErr
}

func resolveSSHProfilePath() (string, error) {
	if customPath := strings.TrimSpace(os.Getenv("WEBSSH_PROFILE_FILE")); customPath != "" {
		return customPath, nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "webssh", "ssh_profiles.json"), nil
}

func (s *SSHProfileStore) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	content, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.profiles = []SavedSSHProfile{}
			return nil
		}
		return err
	}
	if len(content) == 0 {
		s.profiles = []SavedSSHProfile{}
		return nil
	}
	var profiles []SavedSSHProfile
	if err := json.Unmarshal(content, &profiles); err != nil {
		return err
	}
	s.profiles = profiles
	return nil
}

func (s *SSHProfileStore) persistLocked() error {
	content, err := json.MarshalIndent(s.profiles, "", "  ")
	if err != nil {
		return err
	}
	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, content, 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, s.path)
}

// List 列出所有已保存配置
func (s *SSHProfileStore) List() []SavedSSHProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]SavedSSHProfile, len(s.profiles))
	copy(result, s.profiles)
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].CreatedAt < result[j].CreatedAt
	})
	return result
}

// FindDuplicate 查找同一主机身份是否已存在，编辑自身时会自动跳过。
func (s *SSHProfileStore) FindDuplicate(profile SavedSSHProfile) (SavedSSHProfile, bool, error) {
	normalized, err := normalizeSavedSSHProfile(profile)
	if err != nil {
		return SavedSSHProfile{}, false, err
	}
	requestedID := strings.TrimSpace(profile.ID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.profiles {
		if s.profiles[i].ID != normalized.ID {
			continue
		}
		if requestedID != "" && s.profiles[i].ID == requestedID {
			return SavedSSHProfile{}, false, nil
		}
		return s.profiles[i], true, nil
	}
	return SavedSSHProfile{}, false, nil
}

// Save 保存或更新 SSH 配置
func (s *SSHProfileStore) Save(profile SavedSSHProfile) (SavedSSHProfile, error) {
	normalized, err := normalizeSavedSSHProfile(profile)
	if err != nil {
		return SavedSSHProfile{}, err
	}
	requestedID := strings.TrimSpace(profile.ID)
	now := time.Now().Format(time.RFC3339)
	s.mu.Lock()
	defer s.mu.Unlock()

	updateIndex := -1
	duplicateIndex := -1
	for i := range s.profiles {
		if requestedID != "" && s.profiles[i].ID == requestedID {
			updateIndex = i
		}
		if s.profiles[i].ID == normalized.ID {
			duplicateIndex = i
		}
	}

	if updateIndex >= 0 {
		normalized.CreatedAt = s.profiles[updateIndex].CreatedAt
		normalized.UpdatedAt = now
		s.profiles[updateIndex] = normalized
		if duplicateIndex >= 0 && duplicateIndex != updateIndex {
			s.profiles = append(s.profiles[:duplicateIndex], s.profiles[duplicateIndex+1:]...)
		}
		if err := s.persistLocked(); err != nil {
			return SavedSSHProfile{}, err
		}
		return normalized, nil
	}

	if duplicateIndex >= 0 {
		normalized.CreatedAt = s.profiles[duplicateIndex].CreatedAt
		normalized.UpdatedAt = now
		s.profiles[duplicateIndex] = normalized
		if err := s.persistLocked(); err != nil {
			return SavedSSHProfile{}, err
		}
		return normalized, nil
	}

	normalized.CreatedAt = now
	normalized.UpdatedAt = now
	s.profiles = append(s.profiles, normalized)
	if err := s.persistLocked(); err != nil {
		return SavedSSHProfile{}, err
	}
	return normalized, nil
}

// Delete 删除 SSH 配置
func (s *SSHProfileStore) Delete(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.profiles {
		if s.profiles[i].ID == id {
			s.profiles = append(s.profiles[:i], s.profiles[i+1:]...)
			return s.persistLocked()
		}
	}
	return fmt.Errorf("saved ssh not found")
}

func normalizeSavedSSHProfile(profile SavedSSHProfile) (SavedSSHProfile, error) {
	host := strings.TrimSpace(profile.Host)
	ip := strings.TrimSpace(profile.IP)
	countryFlag := strings.TrimSpace(profile.CountryFlag)
	location := strings.TrimSpace(profile.Location)
	osName := strings.TrimSpace(profile.OS)
	memory := strings.TrimSpace(profile.Memory)
	expiresAt := strings.TrimSpace(profile.ExpiresAt)
	username := strings.TrimSpace(profile.Username)
	if host == "" {
		return SavedSSHProfile{}, fmt.Errorf("host is required")
	}
	if username == "" {
		return SavedSSHProfile{}, fmt.Errorf("username is required")
	}
	port := profile.Port
	if port == 0 {
		port = 22
	}
	loginType := profile.LoginType
	if loginType != 1 {
		loginType = 0
	}
	normalized := SavedSSHProfile{
		Host:        host,
		IP:          ip,
		CountryFlag: countryFlag,
		Location:    location,
		OS:          osName,
		Memory:      memory,
		ExpiresAt:   expiresAt,
		Username:    username,
		Port:        port,
		Password:    profile.Password,
		LoginType:   loginType,
	}
	normalized.ID = profileFingerprint(normalized)
	return normalized, nil
}

func profileFingerprint(profile SavedSSHProfile) string {
	sum := sha1.Sum([]byte(strings.Join([]string{
		strings.ToLower(profile.Host),
		profile.Username,
		fmt.Sprintf("%d", profile.Port),
		fmt.Sprintf("%d", profile.LoginType),
	}, "|")))
	return hex.EncodeToString(sum[:])
}
