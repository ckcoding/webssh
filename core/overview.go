package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// 服务器总览：无需在被控端安装 Agent，直接复用已保存主机的 SSH 凭据，
// 通过连接池中的长连接执行只读采集脚本（/proc、df 等）获取指标。
// 只有在有人打开总览页时才会轮询，空闲时对被控端零负载。

const (
	overviewHostTimeout   = 12 * time.Second
	overviewMaxConcurrent = 8
	overviewCacheTTL      = 3 * time.Second
)

const (
	OverviewStatusOnline  = "online"
	OverviewStatusOffline = "offline"
	OverviewStatusNoCred  = "nocred"
)

// OverviewServer 单台主机在总览中的状态。
type OverviewServer struct {
	ID          string         `json:"id"`
	Alias       string         `json:"alias"`
	Host        string         `json:"host"`
	Port        int            `json:"port"`
	Username    string         `json:"username"`
	Remark      string         `json:"remark"`
	ExpiresAt   string         `json:"expiresAt"`
	CountryFlag string         `json:"countryFlag"`
	Location    string         `json:"location"`
	Status      string         `json:"status"`
	Error       string         `json:"error,omitempty"`
	Info        *SSHSystemInfo `json:"info,omitempty"`
	NetRxRate   int64          `json:"netRxRate"`
	NetTxRate   int64          `json:"netTxRate"`
	LatencyMs   int64          `json:"latencyMs"`
	CheckedAt   string         `json:"checkedAt"`
	LastSeenAt  string         `json:"lastSeenAt,omitempty"`
}

// OverviewSummary 总览顶部汇总。
type OverviewSummary struct {
	Total     int   `json:"total"`
	Online    int   `json:"online"`
	Offline   int   `json:"offline"`
	NetRxRate int64 `json:"netRxRate"`
	NetTxRate int64 `json:"netTxRate"`
	NetRxSum  int64 `json:"netRxTotal"`
	NetTxSum  int64 `json:"netTxTotal"`
}

// Overview 总览接口返回值。
type Overview struct {
	Summary     OverviewSummary  `json:"summary"`
	Servers     []OverviewServer `json:"servers"`
	GeneratedAt string           `json:"generatedAt"`
}

type overviewHostState struct {
	mu         sync.Mutex
	inflight   bool
	done       chan struct{}
	last       OverviewServer
	hasLast    bool
	prevRx     int64
	prevTx     int64
	prevAt     time.Time
	lastSeenAt time.Time
}

var overviewState = struct {
	mu     sync.Mutex
	hosts  map[string]*overviewHostState
	cached *Overview
	at     time.Time
}{hosts: make(map[string]*overviewHostState)}

// overviewCollector 可在测试中替换。
var overviewCollector = func(ctx context.Context, profile SavedSSHProfile) (SSHSystemInfo, error) {
	client, err := BuildSSHClient(profile.Host, profile.Username, profile.Password, profile.Port, profile.LoginType)
	if err != nil {
		return SSHSystemInfo{}, err
	}
	session := GetCommandSessionByKey(overviewSessionKey(profile), client)
	var info SSHSystemInfo
	err = session.Use(func(sshClient *SSHClient) error {
		var collectErr error
		info, collectErr = CollectSSHSystemInfo(sshClient)
		return collectErr
	})
	return info, err
}

func overviewSessionKey(profile SavedSSHProfile) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{profile.ID, profile.Host, profile.Username, profile.Password}, "\x00")))
	return "overview:" + hex.EncodeToString(sum[:12])
}

// CollectOverview 并发采集所有已保存主机的状态。
func CollectOverview(ctx context.Context, store *SSHProfileStore) Overview {
	overviewState.mu.Lock()
	defer overviewState.mu.Unlock()

	now := time.Now()
	if overviewState.cached != nil && now.Sub(overviewState.at) < overviewCacheTTL {
		return *overviewState.cached
	}

	profiles := store.List()
	servers := make([]OverviewServer, len(profiles))
	slots := make(chan struct{}, overviewMaxConcurrent)
	var wg sync.WaitGroup
	alive := make(map[string]struct{}, len(profiles))
	for i, profile := range profiles {
		state := overviewState.hosts[profile.ID]
		if state == nil {
			state = &overviewHostState{}
			overviewState.hosts[profile.ID] = state
		}
		alive[profile.ID] = struct{}{}
		wg.Add(1)
		go func(i int, profile SavedSSHProfile, state *overviewHostState) {
			defer wg.Done()
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-ctx.Done():
				servers[i] = baseOverviewServer(profile, OverviewStatusOffline, ctx.Err().Error())
				return
			}
			servers[i] = pollOverviewHost(ctx, profile, state)
		}(i, profile, state)
	}
	wg.Wait()

	for id := range overviewState.hosts {
		if _, ok := alive[id]; !ok {
			delete(overviewState.hosts, id)
		}
	}

	result := Overview{Servers: servers, GeneratedAt: time.Now().Format(time.RFC3339)}
	result.Summary = summarizeOverview(servers)
	overviewState.cached = &result
	overviewState.at = time.Now()
	return result
}

func baseOverviewServer(profile SavedSSHProfile, status, errText string) OverviewServer {
	return OverviewServer{
		ID:          profile.ID,
		Alias:       profile.Alias,
		Host:        profile.Host,
		Port:        profile.Port,
		Username:    profile.Username,
		Remark:      profile.Remark,
		ExpiresAt:   profile.ExpiresAt,
		CountryFlag: profile.CountryFlag,
		Location:    profile.Location,
		Status:      status,
		Error:       errText,
		CheckedAt:   time.Now().Format(time.RFC3339),
	}
}

func pollOverviewHost(ctx context.Context, profile SavedSSHProfile, state *overviewHostState) OverviewServer {
	if strings.TrimSpace(profile.Password) == "" {
		return baseOverviewServer(profile, OverviewStatusNoCred, "no saved credentials")
	}

	state.mu.Lock()
	if state.inflight {
		// 上一次采集仍未返回（主机卡顿），直接复用上次结果，避免堆积连接。
		done := state.done
		state.mu.Unlock()
		select {
		case <-done:
		case <-time.After(overviewHostTimeout):
		case <-ctx.Done():
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if state.hasLast {
			return state.last
		}
		return baseOverviewServer(profile, OverviewStatusOffline, "timeout")
	}
	state.inflight = true
	state.done = make(chan struct{})
	done := state.done
	state.mu.Unlock()

	go func() {
		started := time.Now()
		info, err := overviewCollector(context.Background(), profile)
		elapsed := time.Since(started)
		state.mu.Lock()
		defer state.mu.Unlock()
		state.inflight = false
		state.last = buildOverviewServer(profile, state, info, err, elapsed)
		state.hasLast = true
		close(done)
	}()

	select {
	case <-done:
		state.mu.Lock()
		defer state.mu.Unlock()
		return state.last
	case <-time.After(overviewHostTimeout):
		return baseOverviewServer(profile, OverviewStatusOffline, "timeout")
	case <-ctx.Done():
		return baseOverviewServer(profile, OverviewStatusOffline, ctx.Err().Error())
	}
}

// buildOverviewServer 必须在持有 state.mu 时调用。
func buildOverviewServer(profile SavedSSHProfile, state *overviewHostState, info SSHSystemInfo, err error, elapsed time.Duration) OverviewServer {
	now := time.Now()
	if err != nil {
		server := baseOverviewServer(profile, OverviewStatusOffline, err.Error())
		if !state.lastSeenAt.IsZero() {
			server.LastSeenAt = state.lastSeenAt.Format(time.RFC3339)
		}
		state.prevAt = time.Time{}
		return server
	}

	server := baseOverviewServer(profile, OverviewStatusOnline, "")
	server.Info = &info
	server.LatencyMs = elapsed.Milliseconds()
	if info.CountryFlag != "" {
		server.CountryFlag = info.CountryFlag
	}
	if info.Location != "" {
		server.Location = info.Location
	}
	server.NetRxRate, server.NetTxRate = overviewRates(state, info.NetRxBytes, info.NetTxBytes, now)
	state.lastSeenAt = now
	server.LastSeenAt = now.Format(time.RFC3339)
	return server
}

// overviewRates 根据两次采集之间的累计字节差计算速率（字节/秒）。
func overviewRates(state *overviewHostState, rx, tx int64, now time.Time) (int64, int64) {
	var rxRate, txRate int64
	if !state.prevAt.IsZero() {
		seconds := now.Sub(state.prevAt).Seconds()
		if seconds > 0.5 {
			if delta := rx - state.prevRx; delta >= 0 {
				rxRate = int64(float64(delta) / seconds)
			}
			if delta := tx - state.prevTx; delta >= 0 {
				txRate = int64(float64(delta) / seconds)
			}
		}
	}
	state.prevRx, state.prevTx, state.prevAt = rx, tx, now
	return rxRate, txRate
}

func summarizeOverview(servers []OverviewServer) OverviewSummary {
	summary := OverviewSummary{Total: len(servers)}
	for _, server := range servers {
		if server.Status != OverviewStatusOnline {
			summary.Offline++
			continue
		}
		summary.Online++
		summary.NetRxRate += server.NetRxRate
		summary.NetTxRate += server.NetTxRate
		if server.Info != nil {
			summary.NetRxSum += server.Info.NetRxBytes
			summary.NetTxSum += server.Info.NetTxBytes
		}
	}
	return summary
}

// resetOverviewStateForTest 清空总览缓存，仅供测试使用。
func resetOverviewStateForTest() {
	overviewState.mu.Lock()
	defer overviewState.mu.Unlock()
	overviewState.hosts = make(map[string]*overviewHostState)
	overviewState.cached = nil
	overviewState.at = time.Time{}
}
