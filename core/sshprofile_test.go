package core

import (
	"path/filepath"
	"testing"
)

func TestSSHProfileStoreSaveListDelete(t *testing.T) {
	store, err := NewSSHProfileStore(filepath.Join(t.TempDir(), "ssh_profiles.json"))
	if err != nil {
		t.Fatalf("NewSSHProfileStore() error = %v", err)
	}

	saved, err := store.Save(SavedSSHProfile{
		Host:        "192.168.1.2",
		IP:          "203.0.113.10",
		CountryFlag: "🇸🇬",
		Location:    "Singapore",
		OS:          "Ubuntu 24.04",
		Memory:      "8GB",
		ExpiresAt:   "2027-01-01 00:00:00",
		Username:    "root",
		Port:        22,
		Password:    "secret",
		LoginType:   0,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	list := store.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 saved profile, got %d", len(list))
	}
	if list[0].Password != "secret" {
		t.Fatalf("expected password to be persisted")
	}
	if list[0].IP != "203.0.113.10" || list[0].CountryFlag != "🇸🇬" || list[0].Location != "Singapore" || list[0].OS != "Ubuntu 24.04" || list[0].Memory != "8GB" || list[0].ExpiresAt != "2027-01-01 00:00:00" {
		t.Fatalf("expected server metadata to be persisted, got %+v", list[0])
	}

	updated, err := store.Save(SavedSSHProfile{
		Host:        "192.168.1.2",
		IP:          "203.0.113.20",
		CountryFlag: "🇯🇵",
		Location:    "Tokyo",
		OS:          "Debian 12",
		Memory:      "16GB",
		ExpiresAt:   "",
		Username:    "root",
		Port:        22,
		Password:    "changed",
		LoginType:   0,
	})
	if err != nil {
		t.Fatalf("Save() update error = %v", err)
	}
	if saved.ID != updated.ID {
		t.Fatalf("expected update to reuse profile id")
	}
	if len(store.List()) != 1 {
		t.Fatalf("expected update to keep a single profile")
	}
	if updated.IP != "203.0.113.20" || updated.CountryFlag != "🇯🇵" || updated.Location != "Tokyo" || updated.OS != "Debian 12" || updated.Memory != "16GB" || updated.ExpiresAt != "" {
		t.Fatalf("expected updated metadata to be returned, got %+v", updated)
	}

	if err := store.Delete(saved.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(store.List()) != 0 {
		t.Fatalf("expected empty store after delete")
	}
}
