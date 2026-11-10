package core

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMetadataPreservesLatestEditsAndPersists(t *testing.T) {
	store, err := NewSSHProfileStore(filepath.Join(t.TempDir(), "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	original, err := store.Save(SavedSSHProfile{Host: "127.0.0.1", Username: "fixture", Port: 22, Password: "fixture", Location: "Known location", ExpiresAt: "2027-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	edited := original
	edited.Alias = "New alias"
	edited.Remark = "New remark"
	if _, err := store.Save(edited); err != nil {
		t.Fatal(err)
	}
	got, err := store.UpdateMetadata(original, SSHSystemInfo{IP: "127.0.0.1", OS: "Linux", CPUCores: 4, MemoryTotalMB: 8192})
	if err != nil {
		t.Fatal(err)
	}
	if got.Alias != edited.Alias || got.Remark != edited.Remark || got.ExpiresAt != original.ExpiresAt || got.Password != original.Password || got.Location != original.Location || got.CreatedAt != original.CreatedAt {
		t.Fatal("Metadata overwrote credentials, user edits or existing location")
	}
	if got.Memory != "4C / 8 GB" || got.InfoUpdatedAt == "" {
		t.Fatal("Missing collected metadata")
	}
	loaded, err := NewSSHProfileStore(store.path)
	if err != nil {
		t.Fatal(err)
	}
	saved, found := loaded.Get(got.ID)
	if !found || saved != got {
		t.Fatal("Metadata did not survive reload")
	}
	saved.Password = "changed"
	if _, err := store.Save(saved); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateMetadata(original, SSHSystemInfo{OS: "Stale OS"}); err == nil {
		t.Fatal("Collection with stale credentials was accepted")
	}
	if err := store.Delete(saved.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateMetadata(saved, SSHSystemInfo{OS: "Stale OS"}); err == nil {
		t.Fatal("Deleted profile was restored")
	}
}

func TestMetadataCancelledWhileWaitingForSlot(t *testing.T) {
	savedMetadataSlots <- struct{}{}
	savedMetadataSlots <- struct{}{}
	defer func() { <-savedMetadataSlots; <-savedMetadataSlots }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := RefreshSavedSSHMetadata(ctx, nil, "fixture"); err != context.Canceled {
		t.Fatalf("Expected cancellation, got %v", err)
	}
}

func TestFormatProfileMemory(t *testing.T) {
	for _, tc := range []struct {
		cores int
		mb    int64
		want  string
	}{
		{4, 8192, "4C / 8 GB"}, {0, 1536, "1.5 GB"}, {2, 512, "2C / 512 MB"}, {2, 0, "2C"}, {0, 0, ""},
	} {
		if got := formatProfileMemory(SSHSystemInfo{CPUCores: tc.cores, MemoryTotalMB: tc.mb}); got != tc.want {
			t.Fatalf("Got %q, want %q", got, tc.want)
		}
	}
}
