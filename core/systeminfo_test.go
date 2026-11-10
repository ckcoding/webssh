package core

import (
	"encoding/base64"
	"testing"
)

func TestPopulateSSHSystemInfoAggregatesAllDiskEntries(t *testing.T) {
	disksJSON := `[{"device":"/dev/vda1","mount":"/","totalKB":10485760,"usedKB":5242880,"availableKB":5242880},{"device":"/dev/vdb1","mount":"/data","totalKB":20971520,"usedKB":5242880,"availableKB":15728640}]`
	parsed := map[string]string{
		"disk_mount":        "/",
		"disk_total_kb":     "10485760",
		"disk_used_kb":      "5242880",
		"disk_available_kb": "5242880",
		"disks_b64":         base64.StdEncoding.EncodeToString([]byte(disksJSON)),
	}

	var info SSHSystemInfo
	populateSSHSystemInfoWithKey(&info, parsed, "")

	if len(info.Disks) != 2 {
		t.Fatalf("len(info.Disks) = %d, want 2", len(info.Disks))
	}
	if info.Disks[0].Mount != "/" || info.Disks[1].Mount != "/data" {
		t.Fatalf("mounts = %q, %q; want / and /data", info.Disks[0].Mount, info.Disks[1].Mount)
	}
	if info.DiskTotalMB != 30720 {
		t.Fatalf("DiskTotalMB = %d, want 30720", info.DiskTotalMB)
	}
	if info.DiskUsedMB != 10240 {
		t.Fatalf("DiskUsedMB = %d, want 10240", info.DiskUsedMB)
	}
	if info.DiskMount != "/ +1" {
		t.Fatalf("DiskMount = %q, want / +1", info.DiskMount)
	}
}

func TestCalculateCPUUsageFromTicks(t *testing.T) {
	key := "127.0.0.1:22"
	cpuUsageSnapshotMu.Lock()
	delete(cpuUsageSnapshots, key)
	cpuUsageSnapshotMu.Unlock()

	if usage, ok := calculateCPUUsageFromTicks(key, 100, 80); ok || usage != 0 {
		t.Fatalf("first sample usage = %v ok = %v, want no calculated usage", usage, ok)
	}

	usage, ok := calculateCPUUsageFromTicks(key, 200, 150)
	if !ok {
		t.Fatalf("expected second sample to calculate usage")
	}
	if usage != 30 {
		t.Fatalf("usage = %.2f, want 30.00", usage)
	}
}

func TestPopulateSSHSystemInfoUsesCPUCounterDelta(t *testing.T) {
	key := "test-host:22"
	cpuUsageSnapshotMu.Lock()
	delete(cpuUsageSnapshots, key)
	cpuUsageSnapshotMu.Unlock()

	first := map[string]string{
		"cpu_usage":       "0.00",
		"cpu_total_ticks": "100",
		"cpu_idle_ticks":  "90",
	}
	var firstInfo SSHSystemInfo
	populateSSHSystemInfoWithKey(&firstInfo, first, key)
	if firstInfo.CPUUsage != 0 {
		t.Fatalf("first CPUUsage = %.2f, want 0", firstInfo.CPUUsage)
	}

	second := map[string]string{
		"cpu_usage":       "0.00",
		"cpu_total_ticks": "200",
		"cpu_idle_ticks":  "140",
	}
	var secondInfo SSHSystemInfo
	populateSSHSystemInfoWithKey(&secondInfo, second, key)
	if secondInfo.CPUUsage != 50 {
		t.Fatalf("second CPUUsage = %.2f, want 50", secondInfo.CPUUsage)
	}
}

func TestResolvedIPMissCache(t *testing.T) {
	address := "missing.example.test"
	setCachedResolvedIPMiss(address)

	ip, ok := getCachedResolvedIP(address)
	if !ok {
		t.Fatalf("expected cached miss")
	}
	if ip != "" {
		t.Fatalf("cached miss ip = %q, want empty", ip)
	}

	setCachedResolvedIP(address, "203.0.113.10")
	ip, ok = getCachedResolvedIP(address)
	if !ok {
		t.Fatalf("expected cached hit")
	}
	if ip != "203.0.113.10" {
		t.Fatalf("cached ip = %q, want 203.0.113.10", ip)
	}
}

func TestIPLocationMissCache(t *testing.T) {
	ip := "203.0.113.20"
	setCachedIPLocationMiss(ip)

	location, countryFlag, found, ok := getCachedIPLocation(ip)
	if !ok {
		t.Fatalf("expected cached miss")
	}
	if found {
		t.Fatalf("found = true, want false")
	}
	if location != "" || countryFlag != "" {
		t.Fatalf("cached miss location = %q flag = %q, want empty", location, countryFlag)
	}

	setCachedIPLocation(ip, "Test City", "TC")
	location, countryFlag, found, ok = getCachedIPLocation(ip)
	if !ok || !found {
		t.Fatalf("expected cached hit")
	}
	if location != "Test City" || countryFlag != "TC" {
		t.Fatalf("cached location = %q flag = %q, want Test City/TC", location, countryFlag)
	}
}
