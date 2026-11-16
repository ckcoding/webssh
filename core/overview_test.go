package core

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func newOverviewTestStore(t *testing.T, profiles ...SavedSSHProfile) *SSHProfileStore {
	t.Helper()
	t.Setenv("WEBSSH_SECRET_KEY", "")
	store, err := NewSSHProfileStore(filepath.Join(t.TempDir(), "profiles.json"))
	if err != nil {
		t.Fatalf("NewSSHProfileStore() error = %v", err)
	}
	for _, profile := range profiles {
		if _, err := store.Save(profile); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}
	return store
}

func withOverviewCollector(t *testing.T, fn func(context.Context, SavedSSHProfile) (SSHSystemInfo, error)) {
	t.Helper()
	original := overviewCollector
	overviewCollector = fn
	resetOverviewStateForTest()
	t.Cleanup(func() {
		overviewCollector = original
		resetOverviewStateForTest()
	})
}

func TestCollectOverviewStatusesAndSummary(t *testing.T) {
	store := newOverviewTestStore(t,
		SavedSSHProfile{Host: "web.example.test", Username: "demo", Port: 22, Password: "pw", Alias: "web"},
		SavedSSHProfile{Host: "down.example.test", Username: "demo", Port: 22, Password: "pw", Alias: "down"},
		SavedSSHProfile{Host: "nopass.example.test", Username: "demo", Port: 22, Alias: "nopass"},
	)
	withOverviewCollector(t, func(_ context.Context, profile SavedSSHProfile) (SSHSystemInfo, error) {
		if profile.Alias == "down" {
			return SSHSystemInfo{}, errors.New("dial tcp: connection refused")
		}
		return SSHSystemInfo{CPUUsage: 12.5, NetRxBytes: 1000, NetTxBytes: 500}, nil
	})

	overview := CollectOverview(context.Background(), store)
	if overview.Summary.Total != 3 || overview.Summary.Online != 1 || overview.Summary.Offline != 2 {
		t.Fatalf("unexpected summary %+v", overview.Summary)
	}
	statuses := map[string]string{}
	for _, server := range overview.Servers {
		statuses[server.Alias] = server.Status
	}
	if statuses["web"] != OverviewStatusOnline || statuses["down"] != OverviewStatusOffline || statuses["nopass"] != OverviewStatusNoCred {
		t.Fatalf("unexpected statuses %v", statuses)
	}
	if overview.Summary.NetRxSum != 1000 || overview.Summary.NetTxSum != 500 {
		t.Fatalf("unexpected traffic totals %+v", overview.Summary)
	}
}

func TestCollectOverviewCachesWithinTTL(t *testing.T) {
	store := newOverviewTestStore(t, SavedSSHProfile{Host: "web.example.test", Username: "demo", Port: 22, Password: "pw"})
	var calls int32
	withOverviewCollector(t, func(context.Context, SavedSSHProfile) (SSHSystemInfo, error) {
		atomic.AddInt32(&calls, 1)
		return SSHSystemInfo{}, nil
	})
	CollectOverview(context.Background(), store)
	CollectOverview(context.Background(), store)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected cached second call, collector ran %d times", got)
	}
}

func TestOverviewRates(t *testing.T) {
	state := &overviewHostState{}
	start := time.Now()
	if rx, tx := overviewRates(state, 1000, 2000, start); rx != 0 || tx != 0 {
		t.Fatalf("first sample should have no rate, got %d/%d", rx, tx)
	}
	rx, tx := overviewRates(state, 11000, 4000, start.Add(5*time.Second))
	if rx != 2000 || tx != 400 {
		t.Fatalf("rates = %d/%d, want 2000/400", rx, tx)
	}
	// 计数器回绕（如重启）时不应出现负速率
	rx, tx = overviewRates(state, 10, 10, start.Add(10*time.Second))
	if rx != 0 || tx != 0 {
		t.Fatalf("counter reset should yield zero rate, got %d/%d", rx, tx)
	}
}

func TestCollectOverviewTimesOutSlowHost(t *testing.T) {
	store := newOverviewTestStore(t, SavedSSHProfile{Host: "slow.example.test", Username: "demo", Port: 22, Password: "pw"})
	release := make(chan struct{})
	withOverviewCollector(t, func(context.Context, SavedSSHProfile) (SSHSystemInfo, error) {
		<-release
		return SSHSystemInfo{}, nil
	})
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	started := time.Now()
	overview := CollectOverview(ctx, store)
	if time.Since(started) > 2*time.Second {
		t.Fatalf("slow host blocked overview for %s", time.Since(started))
	}
	if overview.Servers[0].Status != OverviewStatusOffline {
		t.Fatalf("slow host status = %q, want offline", overview.Servers[0].Status)
	}
}
