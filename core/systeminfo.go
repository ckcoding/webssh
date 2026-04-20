package core

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const linuxSystemInfoCommand = `sh -lc '
HOSTNAME=$(hostname 2>/dev/null || true)
OS=""
if [ -r /etc/os-release ]; then
  OS=$(awk -F= '\''$1=="PRETTY_NAME"{gsub(/^"|"$/, "", $2); print $2; exit}'\'' /etc/os-release 2>/dev/null)
fi
if [ -z "$OS" ]; then
  OS=$(uname -srm 2>/dev/null || true)
fi
KERNEL=$(uname -srmo 2>/dev/null || true)
CPU_CORES=$(nproc 2>/dev/null || getconf _NPROCESSORS_ONLN 2>/dev/null || echo 0)
MEM_TOTAL_KB=$(awk '\''/MemTotal:/ {print $2; exit}'\'' /proc/meminfo 2>/dev/null || echo 0)
MEM_AVAILABLE_KB=$(awk '\''/MemAvailable:/ {print $2; exit}'\'' /proc/meminfo 2>/dev/null || echo 0)
if [ -z "$MEM_AVAILABLE_KB" ] || [ "$MEM_AVAILABLE_KB" = "0" ]; then
  MEM_AVAILABLE_KB=$(awk '\''/MemFree:/ {free=$2} /Buffers:/ {buffers=$2} /^Cached:/ {cached=$2} END {print free+buffers+cached}'\'' /proc/meminfo 2>/dev/null || echo 0)
fi
DISK_TOTAL_KB=$(df -Pk / 2>/dev/null | awk '\''NR==2 {print $2; exit}'\'' || echo 0)
DISK_USED_KB=$(df -Pk / 2>/dev/null | awk '\''NR==2 {print $3; exit}'\'' || echo 0)
DISK_AVAILABLE_KB=$(df -Pk / 2>/dev/null | awk '\''NR==2 {print $4; exit}'\'' || echo 0)
DISK_MOUNT=$(df -Pk / 2>/dev/null | awk '\''NR==2 {print $6; exit}'\'' || echo /)
UPTIME_SECONDS=$(cut -d. -f1 /proc/uptime 2>/dev/null || echo 0)
LOAD_AVERAGE=$(cut -d" " -f1-3 /proc/loadavg 2>/dev/null || true)

CPU_USAGE="0.00"
if [ -r /proc/stat ]; then
  read _ U1 N1 S1 I1 IO1 IRQ1 SIRQ1 ST1 _ < /proc/stat
  T1=$((U1+N1+S1+I1+IO1+IRQ1+SIRQ1+ST1))
  IDLE1=$((I1+IO1))
  sleep 0.4
  read _ U2 N2 S2 I2 IO2 IRQ2 SIRQ2 ST2 _ < /proc/stat
  T2=$((U2+N2+S2+I2+IO2+IRQ2+SIRQ2+ST2))
  IDLE2=$((I2+IO2))
  TOTALD=$((T2-T1))
  IDLED=$((IDLE2-IDLE1))
  if [ "$TOTALD" -gt 0 ]; then
    CPU_USAGE=$(awk -v totald="$TOTALD" -v idled="$IDLED" '\''BEGIN { printf "%.2f", ((totald-idled) * 100) / totald }'\'')
  fi
fi

printf "hostname=%s\n" "$HOSTNAME"
printf "os=%s\n" "$OS"
printf "kernel=%s\n" "$KERNEL"
printf "load_average=%s\n" "$LOAD_AVERAGE"
printf "cpu_cores=%s\n" "$CPU_CORES"
printf "cpu_usage=%s\n" "$CPU_USAGE"
printf "mem_total_kb=%s\n" "$MEM_TOTAL_KB"
printf "mem_available_kb=%s\n" "$MEM_AVAILABLE_KB"
printf "disk_mount=%s\n" "$DISK_MOUNT"
printf "disk_total_kb=%s\n" "$DISK_TOTAL_KB"
printf "disk_available_kb=%s\n" "$DISK_AVAILABLE_KB"
printf "disk_used_kb=%s\n" "$DISK_USED_KB"
printf "uptime_seconds=%s\n" "$UPTIME_SECONDS"
'`

type ipWhoisResponse struct {
	Success     bool   `json:"success"`
	City        string `json:"city"`
	Region      string `json:"region"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
}

type ipLocationCacheEntry struct {
	location    string
	countryFlag string
	expiresAt   time.Time
}

type resolvedIPCacheEntry struct {
	ip        string
	expiresAt time.Time
}

var (
	ipLocationCacheTTL = 24 * time.Hour
	ipLocationCacheMu  sync.RWMutex
	ipLocationCache    = map[string]ipLocationCacheEntry{}

	resolvedIPCacheTTL = 10 * time.Minute
	resolvedIPCacheMu  sync.RWMutex
	resolvedIPCache    = map[string]resolvedIPCacheEntry{}
)

// CollectSSHSystemInfo 采集 SSH 远端系统信息
func CollectSSHSystemInfo(sshClient *SSHClient) (SSHSystemInfo, error) {
	info := SSHSystemInfo{
		ConnectAddress: normalizeSSHAddress(sshClient.IPAddress),
		UpdatedAt:      time.Now().Format(time.RFC3339),
	}

	output, err := sshClient.RunCommand(linuxSystemInfoCommand)
	if err != nil {
		return info, err
	}

	parsed, err := parseSSHSystemInfoOutput(output)
	if err != nil {
		return info, err
	}

	info.Hostname = parsed["hostname"]
	info.OS = parsed["os"]
	info.Kernel = parsed["kernel"]
	info.LoadAverage = parsed["load_average"]
	info.CPUCores = parseInt(parsed["cpu_cores"])
	info.CPUUsage = parseFloat(parsed["cpu_usage"])

	memTotalKB := parseInt64(parsed["mem_total_kb"])
	memAvailableKB := parseInt64(parsed["mem_available_kb"])
	if memAvailableKB < 0 {
		memAvailableKB = 0
	}
	memUsedKB := memTotalKB - memAvailableKB
	if memUsedKB < 0 {
		memUsedKB = 0
	}
	info.MemoryTotalMB = memTotalKB / 1024
	info.MemoryAvailableMB = memAvailableKB / 1024
	info.MemoryUsedMB = memUsedKB / 1024
	if memTotalKB > 0 {
		info.MemoryUsage = float64(memUsedKB) * 100 / float64(memTotalKB)
	}

	diskTotalKB := parseInt64(parsed["disk_total_kb"])
	diskAvailableKB := parseInt64(parsed["disk_available_kb"])
	diskUsedKB := parseInt64(parsed["disk_used_kb"])
	if diskUsedKB <= 0 && diskTotalKB > 0 && diskAvailableKB >= 0 {
		diskUsedKB = diskTotalKB - diskAvailableKB
	}
	if diskUsedKB < 0 {
		diskUsedKB = 0
	}
	info.DiskMount = parsed["disk_mount"]
	info.DiskTotalMB = diskTotalKB / 1024
	info.DiskAvailableMB = diskAvailableKB / 1024
	info.DiskUsedMB = diskUsedKB / 1024
	if diskTotalKB > 0 {
		info.DiskUsage = float64(diskUsedKB) * 100 / float64(diskTotalKB)
	}
	info.Uptime = humanizeUptime(parseInt64(parsed["uptime_seconds"]))

	resolvedIP := resolveSSHIPAddress(info.ConnectAddress)
	info.IP = resolvedIP
	if resolvedIP != "" {
		info.Location, info.CountryFlag = lookupIPLocation(resolvedIP)
	}

	return info, nil
}

func parseSSHSystemInfoOutput(output string) (map[string]string, error) {
	result := make(map[string]string)
	lines := strings.Split(output, "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" {
			result[key] = value
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("empty system info output")
	}
	return result, nil
}

func normalizeSSHAddress(address string) string {
	address = strings.TrimSpace(address)
	if strings.HasPrefix(address, "[") && strings.HasSuffix(address, "]") {
		address = strings.TrimPrefix(strings.TrimSuffix(address, "]"), "[")
	}
	return address
}

func resolveSSHIPAddress(address string) string {
	address = normalizeSSHAddress(address)
	if address == "" {
		return ""
	}
	if ip := net.ParseIP(address); ip != nil {
		return ip.String()
	}
	if cachedIP, ok := getCachedResolvedIP(address); ok {
		return cachedIP
	}
	ips, err := net.LookupIP(address)
	if err != nil {
		return ""
	}
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		if ip.IsGlobalUnicast() {
			resolved := ip.String()
			setCachedResolvedIP(address, resolved)
			return resolved
		}
	}
	if len(ips) > 0 && ips[0] != nil {
		resolved := ips[0].String()
		setCachedResolvedIP(address, resolved)
		return resolved
	}
	return ""
}

func lookupIPLocation(ip string) (string, string) {
	if ip == "" {
		return "", ""
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || parsedIP.IsPrivate() || parsedIP.IsLoopback() || parsedIP.IsLinkLocalUnicast() {
		return "", ""
	}
	if location, countryFlag, ok := getCachedIPLocation(parsedIP.String()); ok {
		return location, countryFlag
	}

	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Get(fmt.Sprintf("https://ipwho.is/%s", ip))
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	var payload ipWhoisResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || !payload.Success {
		return "", ""
	}

	location := formatIPLocation(payload)
	countryFlag := countryFlagEmoji(payload.CountryCode)
	if location != "" || countryFlag != "" {
		setCachedIPLocation(parsedIP.String(), location, countryFlag)
	}
	return location, countryFlag
}

func formatIPLocation(payload ipWhoisResponse) string {
	parts := make([]string, 0, 3)
	for _, item := range []string{payload.City, payload.Region, payload.Country} {
		item = strings.TrimSpace(item)
		if item != "" {
			parts = append(parts, item)
		}
	}
	return strings.Join(parts, ", ")
}

func countryFlagEmoji(countryCode string) string {
	code := strings.ToUpper(strings.TrimSpace(countryCode))
	if len(code) != 2 {
		return ""
	}

	flag := make([]rune, 0, 2)
	for _, ch := range code {
		if ch < 'A' || ch > 'Z' {
			return ""
		}
		flag = append(flag, rune(127397)+ch)
	}
	return string(flag)
}

func getCachedIPLocation(ip string) (string, string, bool) {
	ipLocationCacheMu.RLock()
	entry, ok := ipLocationCache[ip]
	ipLocationCacheMu.RUnlock()
	if !ok {
		return "", "", false
	}
	if time.Now().After(entry.expiresAt) {
		ipLocationCacheMu.Lock()
		delete(ipLocationCache, ip)
		ipLocationCacheMu.Unlock()
		return "", "", false
	}
	return entry.location, entry.countryFlag, true
}

func setCachedIPLocation(ip, location, countryFlag string) {
	ipLocationCacheMu.Lock()
	ipLocationCache[ip] = ipLocationCacheEntry{
		location:    location,
		countryFlag: countryFlag,
		expiresAt:   time.Now().Add(ipLocationCacheTTL),
	}
	ipLocationCacheMu.Unlock()
}

func getCachedResolvedIP(address string) (string, bool) {
	resolvedIPCacheMu.RLock()
	entry, ok := resolvedIPCache[address]
	resolvedIPCacheMu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Now().After(entry.expiresAt) {
		resolvedIPCacheMu.Lock()
		delete(resolvedIPCache, address)
		resolvedIPCacheMu.Unlock()
		return "", false
	}
	return entry.ip, true
}

func setCachedResolvedIP(address, ip string) {
	if strings.TrimSpace(address) == "" || strings.TrimSpace(ip) == "" {
		return
	}
	resolvedIPCacheMu.Lock()
	resolvedIPCache[address] = resolvedIPCacheEntry{
		ip:        ip,
		expiresAt: time.Now().Add(resolvedIPCacheTTL),
	}
	resolvedIPCacheMu.Unlock()
}

func humanizeUptime(seconds int64) string {
	if seconds <= 0 {
		return ""
	}
	days := seconds / 86400
	seconds %= 86400
	hours := seconds / 3600
	seconds %= 3600
	minutes := seconds / 60

	parts := make([]string, 0, 3)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	return strings.Join(parts, " ")
}

func parseInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func parseInt64(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}
