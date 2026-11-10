package controller

import (
	"context"
	"path/filepath"
	"testing"
	"time"
	"webssh/core"
)

func TestInventoryCollectionAndCancellation(t *testing.T) {
	for _, slow := range []bool{false, true} {
		t.Run(map[bool]string{false: "collection", true: "cancelled"}[slow], func(t *testing.T) {
			delay := time.Duration(0)
			if slow {
				delay = 200 * time.Millisecond
			}
			port, auths, _ := echoSSHServer(t, delay)
			store, err := core.NewSSHProfileStore(filepath.Join(t.TempDir(), "profiles.json"))
			if err != nil {
				t.Fatal(err)
			}
			original, err := store.Save(core.SavedSSHProfile{Host: "127.0.0.1", Port: port, Username: "fixture", Password: "fixture", Alias: "Keep alias", Remark: "Keep remark"})
			if err != nil {
				t.Fatal(err)
			}
			duration := 3 * time.Second
			if slow {
				duration = 80 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), duration)
			defer cancel()
			updated, info, err := core.RefreshSavedSSHMetadata(ctx, store, original.ID)
			if slow {
				if err != context.DeadlineExceeded {
					t.Fatalf("Expected timeout, got %v", err)
				}
				got, _ := store.Get(original.ID)
				if got != original {
					t.Fatal("Failed collection changed saved data")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if info.OS != "Fixture Linux" || updated.OS != info.OS || updated.Memory != "4C / 8 GB" || updated.Alias != original.Alias || auths.Load() != 1 {
				t.Fatal("Inventory output or connection reuse is incorrect")
			}
		})
	}
}
