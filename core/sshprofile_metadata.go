package core

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Keep background inventory work separate from interactive SSH connections.
var savedMetadataSlots = make(chan struct{}, 2)

func (s *SSHProfileStore) Get(id string) (SavedSSHProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, profile := range s.profiles {
		if profile.ID == strings.TrimSpace(id) {
			return profile, true
		}
	}
	return SavedSSHProfile{}, false
}

// UpdateMetadata merges into the latest profile, preserving user edits and credentials.
func (s *SSHProfileStore) UpdateMetadata(expected SavedSSHProfile, info SSHSystemInfo) (SavedSSHProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, current := range s.profiles {
		if current.ID != expected.ID {
			continue
		}
		if profileFingerprint(current) != profileFingerprint(expected) || current.Password != expected.Password {
			return SavedSSHProfile{}, fmt.Errorf("saved SSH credentials changed during collection")
		}
		updated := current
		for _, pair := range []struct {
			target *string
			value  string
		}{
			{&updated.IP, info.IP}, {&updated.CountryFlag, info.CountryFlag},
			{&updated.Location, info.Location}, {&updated.OS, info.OS},
			{&updated.Memory, formatProfileMemory(info)},
		} {
			if value := strings.TrimSpace(pair.value); value != "" {
				*pair.target = value
			}
		}
		updated.InfoUpdatedAt = time.Now().UTC().Format(time.RFC3339)
		updated.UpdatedAt = updated.InfoUpdatedAt
		s.profiles[i] = updated
		if err := s.persistLocked(); err != nil {
			s.profiles[i] = current
			return SavedSSHProfile{}, err
		}
		return updated, nil
	}
	return SavedSSHProfile{}, fmt.Errorf("saved ssh not found")
}

func formatProfileMemory(info SSHSystemInfo) string {
	parts := []string{}
	if info.CPUCores > 0 {
		parts = append(parts, fmt.Sprintf("%dC", info.CPUCores))
	}
	if info.MemoryTotalMB > 0 {
		if info.MemoryTotalMB >= 1024 {
			gb := float64(info.MemoryTotalMB) / 1024
			if gb == float64(int64(gb)) {
				parts = append(parts, fmt.Sprintf("%.0f GB", gb))
			} else {
				parts = append(parts, fmt.Sprintf("%.1f GB", gb))
			}
		} else {
			parts = append(parts, fmt.Sprintf("%d MB", info.MemoryTotalMB))
		}
	}
	return strings.Join(parts, " / ")
}

func RefreshSavedSSHMetadata(ctx context.Context, store *SSHProfileStore, id string) (SavedSSHProfile, SSHSystemInfo, error) {
	select {
	case savedMetadataSlots <- struct{}{}:
		defer func() { <-savedMetadataSlots }()
	case <-ctx.Done():
		return SavedSSHProfile{}, SSHSystemInfo{}, ctx.Err()
	}
	profile, found := store.Get(id)
	if !found {
		return SavedSSHProfile{}, SSHSystemInfo{}, fmt.Errorf("saved ssh not found")
	}
	client, err := BuildSSHClient(profile.Host, profile.Username, profile.Password, profile.Port, profile.LoginType)
	if err != nil {
		return SavedSSHProfile{}, SSHSystemInfo{}, err
	}
	if err := ctx.Err(); err != nil {
		return SavedSSHProfile{}, SSHSystemInfo{}, err
	}
	if err := client.GenerateClient(); err != nil {
		return SavedSSHProfile{}, SSHSystemInfo{}, err
	}
	defer client.Close()
	connection := client.Client
	cancelConnection := context.AfterFunc(ctx, func() { connection.Close() })
	defer cancelConnection()
	info, err := CollectSSHSystemInfo(&client)
	if ctx.Err() != nil {
		return SavedSSHProfile{}, SSHSystemInfo{}, ctx.Err()
	}
	if err != nil {
		return SavedSSHProfile{}, SSHSystemInfo{}, err
	}
	updated, err := store.UpdateMetadata(profile, info)
	return updated, info, err
}
