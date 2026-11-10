package core

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// windowsSystemInfoCommand 使用 PowerShell 采集 Windows 远端系统信息。
// 通过 base64 编码的 ScriptBlock 传入，避免引号/换行在 cmd 层被破坏。
// 输出与 POSIX 版相同的 key=value 行格式，便于 parseSSHSystemInfoOutput 复用。
const windowsSystemInfoCommandTemplate = `$ErrorActionPreference='SilentlyContinue'
$ProgressPreference='SilentlyContinue'
[Console]::OutputEncoding=[System.Text.Encoding]::UTF8

function Ensure-UInt($v){ if($v -match '^\d+$'){ [uint64]$v } else { 0 } }

# 并行采集：Win32_Processor / nvidia-smi / Win32_VideoController 三个慢查询同时跑，
# 主线程同时跑 Win32_OperatingSystem / Win32_LogicalDisk，将总耗时从串行 ~5s 压到 ~1.5s。
$pool = [RunspaceFactory]::CreateRunspacePool(1, 4)
$pool.Open()

$psProc = [PowerShell]::Create(); $psProc.RunspacePool = $pool
[void]$psProc.AddScript({ Get-CimInstance Win32_Processor -Property NumberOfLogicalProcessors,LoadPercentage })
$hProc = $psProc.BeginInvoke()

$psNv = [PowerShell]::Create(); $psNv.RunspacePool = $pool
[void]$psNv.AddScript({
  $nsm = Get-Command nvidia-smi -ErrorAction SilentlyContinue
  if($nsm){ & nvidia-smi --query-gpu=name,index,utilization.gpu,memory.used,memory.total,driver_version,temperature.gpu --format=csv,noheader,nounits 2>$null }
})
$hNv = $psNv.BeginInvoke()

$psVc = [PowerShell]::Create(); $psVc.RunspacePool = $pool
[void]$psVc.AddScript({ Get-CimInstance Win32_VideoController -Property Name,DriverVersion,AdapterRAM,CurrentHorizontalResolution,CurrentVerticalResolution })
$hVc = $psVc.BeginInvoke()

# 主线程：OS + 磁盘
$hostname = $env:COMPUTERNAME
$os = (Get-CimInstance Win32_OperatingSystem -Property Caption,Version,TotalVisibleMemorySize,FreePhysicalMemory,LastBootUpTime | Select-Object -First 1)
$osCaption = if($os.Caption){ $os.Caption.Trim() } else { '' }
$osVer = if($os.Version){ $os.Version } else { '' }
$kernel = if($osVer){ 'Windows ' + $osVer } else { '' }

$memTotalKB = [uint64]0
$memFreeKB = [uint64]0
if($os){ $memTotalKB = [uint64]$os.TotalVisibleMemorySize; $memFreeKB = [uint64]$os.FreePhysicalMemory }

$volList = Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' -Property DeviceID,Size,FreeSpace
$disksJson = '[]'
$diskArr = @()
foreach($v in $volList){
  $totalKB = [uint64]($v.Size / 1KB)
  $freeKB = [uint64]($v.FreeSpace / 1KB)
  if($totalKB -le 0){ continue }
  $usedKB = $totalKB - $freeKB
  $diskArr += [PSCustomObject]@{
    device = $v.DeviceID
    mount = $v.DeviceID
    totalKB = $totalKB
    usedKB = $usedKB
    availableKB = $freeKB
  }
}
if($diskArr.Count -gt 0){
  $disksJson = $diskArr | ConvertTo-Json -Compress -Depth 3
  if($diskArr.Count -eq 1){ $disksJson = '[' + $disksJson + ']' }
}
$disksB64 = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($disksJson))

# 根磁盘（系统盘）
$sysDrive = $env:SystemDrive
if(-not $sysDrive){ $sysDrive = 'C:' }
$rootDisk = $volList | Where-Object { $_.DeviceID -eq $sysDrive } | Select-Object -First 1
$diskTotalKB = 0; $diskFreeKB = 0; $diskUsedKB = 0; $diskMount = $sysDrive
if($rootDisk){
  $diskTotalKB = [uint64]($rootDisk.Size / 1KB)
  $diskFreeKB = [uint64]($rootDisk.FreeSpace / 1KB)
  $diskUsedKB = $diskTotalKB - $diskFreeKB
}

# 收集并行结果
$processors = $psProc.EndInvoke($hProc)
$cpuCores = 0
foreach($p in $processors){ $cpuCores += [int]$p.NumberOfLogicalProcessors }
if($cpuCores -le 0){ $cpuCores = $env:NUMBER_OF_PROCESSORS }
$cpuUsage = 0.0
$loadAvg = ($processors | Measure-Object -Property LoadPercentage -Average).Average
if($loadAvg -ne $null){ $cpuUsage = [double]$loadAvg }

# GPU 列表：优先用 nvidia-smi 采集 NVIDIA 卡（含利用率/显存/温度），
# 再用 Win32_VideoController 补充非 NVIDIA 卡（无利用率）。
$gpusJson = '[]'
$gpuArr = @()
$nvNames = @{}
$nvOut = $psNv.EndInvoke($hNv)
if($nvOut){
  foreach($line in $nvOut){
    $parts = $line -split ','
    if($parts.Count -lt 7){ continue }
    $gname = $parts[0].Trim()
    $gusage = [double]($parts[2].Trim())
    $gmemUsed = [uint64]($parts[3].Trim())
    $gmemTotal = [uint64]($parts[4].Trim())
    $gdriver = $parts[5].Trim()
    $gtemp = [int]($parts[6].Trim())
    if(-not $gname){ continue }
    $nvNames[$gname.ToLower()] = $true
    $gpuArr += [PSCustomObject]@{
      name = $gname
      driver = $gdriver
      vramKB = ($gmemTotal * 1KB)
      memTotalKB = ($gmemTotal * 1KB)
      memUsedKB = ($gmemUsed * 1KB)
      usage = $gusage
      temperature = $gtemp
      resolution = ''
    }
  }
}
$vcList = $psVc.EndInvoke($hVc)
foreach($g in $vcList){
  $name = if($g.Name){ $g.Name.Trim() } else { '' }
  if(-not $name){ continue }
  if($nvNames.ContainsKey($name.ToLower())){ continue }
  $nameLower = $name.ToLower()
  # 虚拟显示适配器关键词：远程桌面/串流/虚拟显卡软件创建的设备，非真实 GPU，过滤掉。
  # 注意：stdin -Command - 模式下字符串字面量里的 | 会被当成管道符，
  # 且过长的单行表达式可能解析异常，因此用多个 -like 分行拼接。
  $isVirtual = $false
  if($nameLower -like '*gameviewer*'){ $isVirtual = $true }
  if($nameLower -like '*remote display*'){ $isVirtual = $true }
  if($nameLower -like '*orayidd*'){ $isVirtual = $true }
  if($nameLower -like '*idddriver*'){ $isVirtual = $true }
  if($nameLower -like '*virtual display*'){ $isVirtual = $true }
  if($nameLower -like '*indirect display*'){ $isVirtual = $true }
  if($nameLower -like '*mirage driver*'){ $isVirtual = $true }
  if($nameLower -like '*parsec*'){ $isVirtual = $true }
  if($nameLower -like '*virtual adapter*'){ $isVirtual = $true }
  if($nameLower -like '*usb display*'){ $isVirtual = $true }
  if($isVirtual){ continue }
  $driver = if($g.DriverVersion){ $g.DriverVersion.Trim() } else { '' }
  $vramKB = [uint64]0
  if($g.AdapterRAM){ $vramKB = [uint64]($g.AdapterRAM / 1KB) }
  $res = ''
  if($g.CurrentHorizontalResolution -and $g.CurrentVerticalResolution){ $res = ('{0}x{1}' -f [uint64]$g.CurrentHorizontalResolution, [uint64]$g.CurrentVerticalResolution) }
  $gpuArr += [PSCustomObject]@{
    name = $name
    driver = $driver
    vramKB = $vramKB
    memTotalKB = $vramKB
    memUsedKB = 0
    usage = 0
    temperature = 0
    resolution = $res
  }
}
if($gpuArr.Count -gt 0){
  $gpusJson = $gpuArr | ConvertTo-Json -Compress -Depth 3
  if($gpuArr.Count -eq 1){ $gpusJson = '[' + $gpusJson + ']' }
}
$gpusB64 = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($gpusJson))

$pool.Close()
$pool.Dispose()

# 运行时间
$uptimeSec = 0
$bootTime = 0
$lastBoot = $os.LastBootUpTime
if($lastBoot){
  $bootTime = [uint64]((Get-Date '1970-01-01') - $lastBoot).TotalSeconds
  $uptimeSec = [uint64]((Get-Date) - $lastBoot).TotalSeconds
}
$currentEpoch = [uint64](Get-Date -UFormat %s)

Write-Output ("hostname=" + $hostname)
Write-Output ("os=" + $osCaption)
Write-Output ("kernel=" + $kernel)
Write-Output ("load_average=")
Write-Output ("cpu_cores=" + $cpuCores)
Write-Output ("cpu_usage=" + ('{0:F2}' -f $cpuUsage))
Write-Output ("cpu_total_ticks=0")
Write-Output ("cpu_idle_ticks=0")
Write-Output ("mem_total_kb=" + $memTotalKB)
Write-Output ("mem_available_kb=" + $memFreeKB)
Write-Output ("disk_mount=" + $diskMount)
Write-Output ("disk_total_kb=" + $diskTotalKB)
Write-Output ("disk_available_kb=" + $diskFreeKB)
Write-Output ("disk_used_kb=" + $diskUsedKB)
Write-Output ("disks_b64=" + $disksB64)
Write-Output ("gpus_b64=" + $gpusB64)
Write-Output ("uptime_seconds=" + $uptimeSec)
Write-Output ("boot_time=" + $bootTime)
Write-Output ("current_epoch=" + $currentEpoch)
`

const systemInfoCommand = `PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin
export PATH
LC_ALL=C
LANG=C
export LC_ALL LANG
sh <<'EOF'
ensure_uint() {
  case "$1" in
    ''|*[!0-9]*)
      printf '0'
      ;;
    *)
      printf '%s' "$1"
      ;;
  esac
}

build_disk_json() {
  if [ "$1" = "Linux" ] && command -v findmnt >/dev/null 2>&1; then
    findmnt_json=$(findmnt -rn -b -o SOURCE,TARGET,FSTYPE,SIZE,USED,AVAIL 2>/dev/null | awk '
      function ignored_fstype(value) {
        return value ~ /^(tmpfs|devtmpfs|overlay|squashfs|proc|sysfs|cgroup|cgroup2|mqueue|autofs|nsfs|tracefs|debugfs|fusectl|configfs|securityfs|pstore|ramfs|hugetlbfs|bpf|binfmt_misc|rpc_pipefs|devpts|selinuxfs|efivarfs)$/
      }
      function ignored_mount(value) {
        return value ~ /^\/(proc|sys|dev|run)(\/|$)/ || value ~ /^\/var\/lib\/(docker|containers|kubelet)(\/|$)/
      }
      function json_escape(value) {
        gsub(/\\/,"\\\\", value)
        gsub(/"/,"\\\"", value)
        return value
      }
      BEGIN {
        first = 1
        printf "["
      }
      {
        filesystem = $1
        mount = $2
        fstype = $3
        total = $4
        used = $5
        available = $6
        if (filesystem == "" || mount == "" || fstype == "") next
        if (ignored_fstype(fstype) || ignored_mount(mount)) next
        if (total !~ /^[0-9]+$/ || used !~ /^[0-9]+$/ || available !~ /^[0-9]+$/) next
        if (total <= 0) next

        total = int(total / 1024)
        used = int(used / 1024)
        available = int(available / 1024)

        if (!first) {
          printf ","
        }
        first = 0
        printf "{\"device\":\"%s\",\"mount\":\"%s\",\"totalKB\":%s,\"usedKB\":%s,\"availableKB\":%s}", json_escape(filesystem), json_escape(mount), total, used, available
      }
      END {
        printf "]"
      }
    ')
    if [ -n "$findmnt_json" ] && [ "$findmnt_json" != "[]" ]; then
      printf '%s' "$findmnt_json"
      return
    fi
  fi

  df -PkP 2>/dev/null | awk -v uname="$1" '
    function ignored_linux_source(value) {
      return value ~ /^(tmpfs|devtmpfs|overlay|squashfs|proc|sysfs|cgroup|cgroup2|mqueue|autofs|nsfs|tracefs|debugfs|fusectl|configfs|securityfs|pstore|ramfs|hugetlbfs|bpf|binfmt_misc|rpc_pipefs|devpts|selinuxfs|efivarfs)$/
    }
    function ignored_linux_mount(value) {
      return value ~ /^\/(proc|sys|dev|run)(\/|$)/ || value ~ /^\/var\/lib\/(docker|containers|kubelet)(\/|$)/
    }
    function json_escape(value) {
      gsub(/\\/,"\\\\", value)
      gsub(/"/,"\\\"", value)
      return value
    }
    BEGIN {
      first = 1
      printf "["
    }
    NR == 1 { next }
    {
      filesystem = $1
      total = $2
      used = $3
      available = $4
      mount = $6
      if (filesystem == "" || mount == "") next
      if (total !~ /^[0-9]+$/ || used !~ /^[0-9]+$/ || available !~ /^[0-9]+$/) next

      if (uname == "Linux") {
        if (ignored_linux_source(filesystem) || ignored_linux_mount(mount)) next
      } else if (uname == "Darwin") {
        if (filesystem !~ /^\/dev\//) next
        if (mount != "/" && mount !~ /^\/Volumes\//) next
      } else {
        if (filesystem !~ /^\/dev\//) next
      }

      if (!first) {
        printf ","
      }
      first = 0
      printf "{\"device\":\"%s\",\"mount\":\"%s\",\"totalKB\":%s,\"usedKB\":%s,\"availableKB\":%s}", json_escape(filesystem), json_escape(mount), total, used, available
    }
    END {
      printf "]"
    }
  '
}

HOSTNAME=$(hostname 2>/dev/null || true)
OS=""
UNAME=$(uname -s 2>/dev/null || echo unknown)
KERNEL=$(uname -srm 2>/dev/null || uname -a 2>/dev/null || true)
LOAD_AVERAGE=""
CPU_CORES=0
CPU_USAGE="0.00"
CPU_TOTAL_TICKS=0
CPU_IDLE_TICKS=0
MEM_TOTAL_KB=0
MEM_AVAILABLE_KB=0
DISK_TOTAL_KB=$(df -Pk / 2>/dev/null | awk 'NR==2 {print $2; exit}' || echo 0)
DISK_USED_KB=$(df -Pk / 2>/dev/null | awk 'NR==2 {print $3; exit}' || echo 0)
DISK_AVAILABLE_KB=$(df -Pk / 2>/dev/null | awk 'NR==2 {print $4; exit}' || echo 0)
DISK_MOUNT=$(df -Pk / 2>/dev/null | awk 'NR==2 {print $6; exit}' || echo /)
DISKS_B64=""
UPTIME_SECONDS=0
BOOT_TIME=0
CURRENT_EPOCH=0

case "$UNAME" in
  Linux)
    if [ -r /etc/os-release ]; then
      OS=$(awk -F= '$1=="PRETTY_NAME"{gsub(/^"|"$/, "", $2); print $2; exit}' /etc/os-release 2>/dev/null)
    fi
    if [ -z "$OS" ]; then
      OS=$(uname -srm 2>/dev/null || true)
    fi
    CPU_CORES=$(nproc 2>/dev/null || getconf _NPROCESSORS_ONLN 2>/dev/null || grep -c '^processor[[:space:]]*:' /proc/cpuinfo 2>/dev/null || echo 0)
    MEM_TOTAL_KB=$(awk '/MemTotal:/ {print $2; exit}' /proc/meminfo 2>/dev/null || echo 0)
    MEM_AVAILABLE_KB=$(awk '/MemAvailable:/ {print $2; exit}' /proc/meminfo 2>/dev/null || echo 0)
    if [ -z "$MEM_AVAILABLE_KB" ] || [ "$MEM_AVAILABLE_KB" = "0" ]; then
      MEM_AVAILABLE_KB=$(awk '/MemFree:/ {free=$2} /Buffers:/ {buffers=$2} /^Cached:/ {cached=$2} END {print free+buffers+cached}' /proc/meminfo 2>/dev/null || echo 0)
    fi
    UPTIME_SECONDS=$(cut -d. -f1 /proc/uptime 2>/dev/null || echo 0)
    LOAD_AVERAGE=$(cut -d" " -f1-3 /proc/loadavg 2>/dev/null || true)

    if [ -r /proc/stat ]; then
      read _ U N S I IO IRQ SIRQ ST _ < /proc/stat
      CPU_TOTAL_TICKS=$((U+N+S+I+IO+IRQ+SIRQ+ST))
      CPU_IDLE_TICKS=$((I+IO))
    fi
    ;;
  Darwin)
    if [ -z "$HOSTNAME" ]; then
      HOSTNAME=$(scutil --get LocalHostName 2>/dev/null || scutil --get ComputerName 2>/dev/null || true)
    fi
    PRODUCT_NAME=$(sw_vers -productName 2>/dev/null || true)
    PRODUCT_VERSION=$(sw_vers -productVersion 2>/dev/null || true)
    BUILD_VERSION=$(sw_vers -buildVersion 2>/dev/null || true)
    if [ -n "$PRODUCT_NAME" ] || [ -n "$PRODUCT_VERSION" ]; then
      OS=$(printf '%s %s' "$PRODUCT_NAME" "$PRODUCT_VERSION" | awk '{$1=$1; print}')
      if [ -n "$BUILD_VERSION" ]; then
        OS=$(printf '%s (%s)' "$OS" "$BUILD_VERSION" | awk '{$1=$1; print}')
      fi
    fi
    if [ -z "$OS" ]; then
      OS=$(uname -srm 2>/dev/null || true)
    fi
    CPU_CORES=$(sysctl -n hw.logicalcpu 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 0)
    MEM_TOTAL_BYTES=$(sysctl -n hw.memsize 2>/dev/null || echo 0)
    MEM_TOTAL_KB=$(awk -v bytes="$MEM_TOTAL_BYTES" 'BEGIN { if (bytes > 0) printf "%.0f", bytes / 1024; else printf "0" }')
    PAGE_SIZE=$(pagesize 2>/dev/null || sysctl -n hw.pagesize 2>/dev/null || echo 4096)
    MEM_AVAILABLE_KB=$(vm_stat 2>/dev/null | awk -v page_size="$PAGE_SIZE" '
      /Pages free/ {gsub("\\.", "", $3); free=$3}
      /Pages inactive/ {gsub("\\.", "", $3); inactive=$3}
      /Pages speculative/ {gsub("\\.", "", $3); speculative=$3}
      END { printf "%.0f", ((free+inactive+speculative) * page_size) / 1024 }' || echo 0)
    LOAD_AVERAGE=$(uptime 2>/dev/null | sed 's/.*load averages*: *//' | sed 's/.*load average: *//' | tr ',' ' ' | awk '{$1=$1; print}' || true)
    CPU_USAGE=$(ps -A -o %cpu 2>/dev/null | awk -v cores="$CPU_CORES" 'NR>1 {sum+=$1} END { if (cores > 0) printf "%.2f", sum / cores; else printf "0.00" }' || echo 0.00)
    BOOT_TIME=$(sysctl -n kern.boottime 2>/dev/null | awk 'match($0, /[0-9]+/) { print substr($0, RSTART, RLENGTH); exit }' || echo 0)
    BOOT_TIME=$(ensure_uint "$BOOT_TIME")
    CURRENT_EPOCH=$(date +%s 2>/dev/null || echo 0)
    CURRENT_EPOCH=$(ensure_uint "$CURRENT_EPOCH")
    if [ "$BOOT_TIME" -gt 0 ] && [ "$CURRENT_EPOCH" -gt "$BOOT_TIME" ]; then
      UPTIME_SECONDS=$((CURRENT_EPOCH-BOOT_TIME))
    fi
    ;;
  *)
    OS=$(uname -srm 2>/dev/null || true)
    CPU_CORES=$(sysctl -n hw.ncpu 2>/dev/null || getconf _NPROCESSORS_ONLN 2>/dev/null || echo 0)
    LOAD_AVERAGE=$(uptime 2>/dev/null | sed 's/.*load averages*: *//' | sed 's/.*load average: *//' | tr ',' ' ' | awk '{$1=$1; print}' || true)
    MEM_TOTAL_BYTES=$(sysctl -n hw.memsize 2>/dev/null || sysctl -n hw.physmem 2>/dev/null || echo 0)
    MEM_TOTAL_KB=$(awk -v bytes="$MEM_TOTAL_BYTES" 'BEGIN { if (bytes > 0) printf "%.0f", bytes / 1024; else printf "0" }')
    BOOT_TIME=$(sysctl -n kern.boottime 2>/dev/null | awk 'match($0, /[0-9]+/) { print substr($0, RSTART, RLENGTH); exit }' || echo 0)
    BOOT_TIME=$(ensure_uint "$BOOT_TIME")
    CURRENT_EPOCH=$(date +%s 2>/dev/null || echo 0)
    CURRENT_EPOCH=$(ensure_uint "$CURRENT_EPOCH")
    if [ "$BOOT_TIME" -gt 0 ] && [ "$CURRENT_EPOCH" -gt "$BOOT_TIME" ]; then
      UPTIME_SECONDS=$((CURRENT_EPOCH-BOOT_TIME))
    fi
    ;;
esac

DISKS_JSON=$(build_disk_json "$UNAME")
DISKS_B64=$(printf '%s' "$DISKS_JSON" | base64 | tr -d '\n')

CPU_CORES=$(ensure_uint "$CPU_CORES")
CPU_TOTAL_TICKS=$(ensure_uint "$CPU_TOTAL_TICKS")
CPU_IDLE_TICKS=$(ensure_uint "$CPU_IDLE_TICKS")
MEM_TOTAL_KB=$(ensure_uint "$MEM_TOTAL_KB")
MEM_AVAILABLE_KB=$(ensure_uint "$MEM_AVAILABLE_KB")
DISK_TOTAL_KB=$(ensure_uint "$DISK_TOTAL_KB")
DISK_USED_KB=$(ensure_uint "$DISK_USED_KB")
DISK_AVAILABLE_KB=$(ensure_uint "$DISK_AVAILABLE_KB")
UPTIME_SECONDS=$(ensure_uint "$UPTIME_SECONDS")
BOOT_TIME=$(ensure_uint "$BOOT_TIME")
CURRENT_EPOCH=$(ensure_uint "$CURRENT_EPOCH")

printf "hostname=%s\n" "$HOSTNAME"
printf "os=%s\n" "$OS"
printf "kernel=%s\n" "$KERNEL"
printf "load_average=%s\n" "$LOAD_AVERAGE"
printf "cpu_cores=%s\n" "$CPU_CORES"
printf "cpu_usage=%s\n" "$CPU_USAGE"
printf "cpu_total_ticks=%s\n" "$CPU_TOTAL_TICKS"
printf "cpu_idle_ticks=%s\n" "$CPU_IDLE_TICKS"
printf "mem_total_kb=%s\n" "$MEM_TOTAL_KB"
printf "mem_available_kb=%s\n" "$MEM_AVAILABLE_KB"
printf "disk_mount=%s\n" "$DISK_MOUNT"
printf "disk_total_kb=%s\n" "$DISK_TOTAL_KB"
printf "disk_available_kb=%s\n" "$DISK_AVAILABLE_KB"
printf "disk_used_kb=%s\n" "$DISK_USED_KB"
printf "disks_b64=%s\n" "$DISKS_B64"
printf "uptime_seconds=%s\n" "$UPTIME_SECONDS"
printf "boot_time=%s\n" "$BOOT_TIME"
printf "current_epoch=%s\n" "$CURRENT_EPOCH"
EOF`

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
	found       bool
}

type resolvedIPCacheEntry struct {
	ip        string
	expiresAt time.Time
	found     bool
}

type cpuUsageSnapshot struct {
	totalTicks int64
	idleTicks  int64
	updatedAt  time.Time
}

var (
	ipLocationCacheTTL = 24 * time.Hour
	ipLocationMissTTL  = 10 * time.Minute
	ipLocationCacheMu  sync.RWMutex
	ipLocationCache    = map[string]ipLocationCacheEntry{}

	resolvedIPCacheTTL = 10 * time.Minute
	resolvedIPMissTTL  = time.Minute
	resolvedIPCacheMu  sync.RWMutex
	resolvedIPCache    = map[string]resolvedIPCacheEntry{}

	cpuUsageSnapshotTTL = time.Hour
	cpuUsageSnapshotMu  sync.Mutex
	cpuUsageSnapshots   = map[string]cpuUsageSnapshot{}

	ipWhoisClient = &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			MaxIdleConns:        20,
			MaxIdleConnsPerHost: 4,
			IdleConnTimeout:     90 * time.Second,
		},
	}
)

// remoteOSCache 缓存 detectRemoteOS 的结果，避免每次采集都发起探测会话。
// 同一台主机的操作系统不会在运行期间变化，SSE 流每 5 秒采集一次时收益明显。
var (
	remoteOSCacheMu  sync.RWMutex
	remoteOSCache    = map[string]remoteOSCacheEntry{}
	remoteOSCacheTTL = 30 * time.Minute
)

type remoteOSCacheEntry struct {
	os      string
	expires time.Time
}

// detectRemoteOS 通过单次会话执行组合命令判断远端是 POSIX 还是 Windows。
// 命令 "uname -s 2>/dev/null || ver" 在 POSIX 上 uname 成功直接输出内核名；
// 在 Windows 上 uname 不存在会失败，|| 触发 ver 输出包含 "windows" 的版本信息。
// 返回 "windows" / "posix" / "unknown"。结果按主机缓存。
func detectRemoteOS(sshClient *SSHClient) string {
	if sshClient == nil || sshClient.Client == nil {
		return "unknown"
	}
	cacheKey := net.JoinHostPort(normalizeSSHAddress(sshClient.IPAddress), strconv.Itoa(sshClient.Port))
	if cacheKey != "" {
		remoteOSCacheMu.RLock()
		if cached, ok := remoteOSCache[cacheKey]; ok && time.Now().Before(cached.expires) {
			remoteOSCacheMu.RUnlock()
			return cached.os
		}
		remoteOSCacheMu.RUnlock()
	}

	session, err := sshClient.Client.NewSession()
	if err != nil {
		return "unknown"
	}
	defer session.Close()
	var out bytes.Buffer
	session.Stdout = &out
	// 单次会话同时探测 POSIX 与 Windows：uname 成功则 POSIX，否则回退 ver。
	if err := session.Run("uname -s 2>/dev/null || ver"); err == nil {
		name := strings.TrimSpace(out.String())
		lower := strings.ToLower(name)
		if name != "" && !strings.Contains(lower, "windows") {
			if cacheKey != "" {
				remoteOSCacheMu.Lock()
				remoteOSCache[cacheKey] = remoteOSCacheEntry{os: "posix", expires: time.Now().Add(remoteOSCacheTTL)}
				remoteOSCacheMu.Unlock()
			}
			return "posix"
		}
		if strings.Contains(lower, "windows") {
			if cacheKey != "" {
				remoteOSCacheMu.Lock()
				remoteOSCache[cacheKey] = remoteOSCacheEntry{os: "windows", expires: time.Now().Add(remoteOSCacheTTL)}
				remoteOSCacheMu.Unlock()
			}
			return "windows"
		}
	}
	return "unknown"
}

// windowsSystemInfoShell 是启动 Windows 信息采集的命令行。
// 使用 powershell.exe 的 -Command - 从 stdin 读取脚本执行，
// 避免命令行长度限制（cmd.exe 约 8191 字符）导致 -EncodedCommand 被截断。
const windowsSystemInfoShell = "powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command -"

// windowsSystemInfoScript 返回通过 stdin 传入的 PowerShell 脚本原文。
func windowsSystemInfoScript() string {
	return windowsSystemInfoCommandTemplate
}

// CollectSSHSystemInfo 采集 SSH 远端系统信息
func CollectSSHSystemInfo(sshClient *SSHClient) (SSHSystemInfo, error) {
	info := SSHSystemInfo{
		ConnectAddress: normalizeSSHAddress(sshClient.IPAddress),
		UpdatedAt:      time.Now().Format(time.RFC3339),
	}

	command := systemInfoCommand
	remoteOS := detectRemoteOS(sshClient)
	var output string
	var err error
	if remoteOS == "windows" {
		output, err = sshClient.RunScript(windowsSystemInfoShell, windowsSystemInfoScript())
	} else {
		output, err = sshClient.RunCommand(command)
	}
	if err != nil {
		return info, err
	}

	parsed, err := parseSSHSystemInfoOutput(output)
	if err != nil {
		return info, err
	}

	populateSSHSystemInfoWithKey(&info, parsed, cpuUsageCacheKey(sshClient))

	resolvedIP := resolveSSHIPAddress(info.ConnectAddress)
	info.IP = resolvedIP
	if resolvedIP != "" {
		info.Location, info.CountryFlag = lookupIPLocation(resolvedIP)
	}

	return info, nil
}

func populateSSHSystemInfoWithKey(info *SSHSystemInfo, parsed map[string]string, cacheKey string) {
	if info == nil {
		return
	}

	info.Hostname = parsed["hostname"]
	info.OS = parsed["os"]
	info.Kernel = parsed["kernel"]
	info.LoadAverage = parsed["load_average"]
	info.CPUCores = parseInt(parsed["cpu_cores"])
	info.CPUUsage = normalizeUsagePercent(parseFloat(parsed["cpu_usage"]))
	if usage, ok := calculateCPUUsageFromTicks(cacheKey, parseInt64(parsed["cpu_total_ticks"]), parseInt64(parsed["cpu_idle_ticks"])); ok {
		info.CPUUsage = usage
	}

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
		info.MemoryUsage = normalizeUsagePercent(float64(memUsedKB) * 100 / float64(memTotalKB))
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
	if info.DiskMount == "" {
		info.DiskMount = "/"
	}
	info.DiskTotalMB = diskTotalKB / 1024
	info.DiskAvailableMB = diskAvailableKB / 1024
	info.DiskUsedMB = diskUsedKB / 1024
	if diskTotalKB > 0 {
		info.DiskUsage = normalizeUsagePercent(float64(diskUsedKB) * 100 / float64(diskTotalKB))
	}
	info.Disks = parseDiskEntries(parsed["disks_b64"])
	info.GPUs = parseGPUEntries(parsed["gpus_b64"])
	if len(info.Disks) == 0 && diskTotalKB > 0 {
		info.Disks = []SSHDiskInfo{
			{
				Mount:       info.DiskMount,
				TotalMB:     info.DiskTotalMB,
				AvailableMB: info.DiskAvailableMB,
				UsedMB:      info.DiskUsedMB,
				Usage:       info.DiskUsage,
			},
		}
	}
	if len(info.Disks) > 0 {
		totalMB, availableMB, usedMB := summarizeDisks(info.Disks)
		info.DiskTotalMB = totalMB
		info.DiskAvailableMB = availableMB
		info.DiskUsedMB = usedMB
		if totalMB > 0 {
			info.DiskUsage = normalizeUsagePercent(float64(usedMB) * 100 / float64(totalMB))
		}
		if len(info.Disks) == 1 {
			info.DiskMount = info.Disks[0].Mount
		} else {
			info.DiskMount = fmt.Sprintf("%s +%d", info.Disks[0].Mount, len(info.Disks)-1)
		}
	}
	info.Uptime = humanizeUptime(resolveUptimeSeconds(
		parseInt64(parsed["uptime_seconds"]),
		parseInt64(parsed["boot_time"]),
		parseInt64(parsed["current_epoch"]),
	))
}

func cpuUsageCacheKey(sshClient *SSHClient) string {
	if sshClient == nil {
		return ""
	}
	return strings.Join([]string{
		normalizeSSHAddress(sshClient.IPAddress),
		strconv.Itoa(sshClient.Port),
	}, ":")
}

func calculateCPUUsageFromTicks(cacheKey string, totalTicks, idleTicks int64) (float64, bool) {
	if cacheKey == "" || totalTicks <= 0 || idleTicks < 0 || idleTicks > totalTicks {
		return 0, false
	}

	now := time.Now()
	cpuUsageSnapshotMu.Lock()
	defer cpuUsageSnapshotMu.Unlock()

	for key, snapshot := range cpuUsageSnapshots {
		if now.Sub(snapshot.updatedAt) > cpuUsageSnapshotTTL {
			delete(cpuUsageSnapshots, key)
		}
	}

	previous, ok := cpuUsageSnapshots[cacheKey]
	cpuUsageSnapshots[cacheKey] = cpuUsageSnapshot{
		totalTicks: totalTicks,
		idleTicks:  idleTicks,
		updatedAt:  now,
	}
	if !ok {
		return 0, false
	}

	totalDelta := totalTicks - previous.totalTicks
	idleDelta := idleTicks - previous.idleTicks
	if totalDelta <= 0 || idleDelta < 0 || idleDelta > totalDelta {
		return 0, false
	}
	return normalizeUsagePercent(float64(totalDelta-idleDelta) * 100 / float64(totalDelta)), true
}

func parseDiskEntries(encoded string) []SSHDiskInfo {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil
	}

	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}

	var rawEntries []struct {
		Device      string `json:"device"`
		Mount       string `json:"mount"`
		TotalKB     int64  `json:"totalKB"`
		UsedKB      int64  `json:"usedKB"`
		AvailableKB int64  `json:"availableKB"`
	}
	if err := json.Unmarshal(payload, &rawEntries); err != nil {
		return nil
	}

	disks := make([]SSHDiskInfo, 0, len(rawEntries))
	seenMounts := make(map[string]struct{}, len(rawEntries))
	for _, entry := range rawEntries {
		mount := strings.TrimSpace(entry.Mount)
		if mount == "" {
			continue
		}
		if _, exists := seenMounts[mount]; exists {
			continue
		}
		seenMounts[mount] = struct{}{}

		totalKB := entry.TotalKB
		availableKB := entry.AvailableKB
		usedKB := entry.UsedKB
		if totalKB <= 0 {
			continue
		}
		if usedKB <= 0 && availableKB >= 0 {
			usedKB = totalKB - availableKB
		}
		if usedKB < 0 {
			usedKB = 0
		}

		disk := SSHDiskInfo{
			Device:      strings.TrimSpace(entry.Device),
			Mount:       mount,
			TotalMB:     totalKB / 1024,
			AvailableMB: availableKB / 1024,
			UsedMB:      usedKB / 1024,
		}
		if totalKB > 0 {
			disk.Usage = normalizeUsagePercent(float64(usedKB) * 100 / float64(totalKB))
		}
		disks = append(disks, disk)
	}

	return disks
}

func parseGPUEntries(encoded string) []SSHGPUInfo {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil
	}

	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}

	var rawEntries []struct {
		Name        string  `json:"name"`
		Driver      string  `json:"driver"`
		VRAMKB      int64   `json:"vramKB"`
		MemTotalKB  int64   `json:"memTotalKB"`
		MemUsedKB   int64   `json:"memUsedKB"`
		Usage       float64 `json:"usage"`
		Temperature int     `json:"temperature"`
		Resolution  string  `json:"resolution"`
	}
	if err := json.Unmarshal(payload, &rawEntries); err != nil {
		return nil
	}

	gpus := make([]SSHGPUInfo, 0, len(rawEntries))
	for _, entry := range rawEntries {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			continue
		}
		memTotalKB := entry.MemTotalKB
		if memTotalKB <= 0 {
			memTotalKB = entry.VRAMKB
		}
		memUsedKB := entry.MemUsedKB
		if memUsedKB < 0 {
			memUsedKB = 0
		}
		var memUsage float64
		if memTotalKB > 0 {
			memUsage = normalizeUsagePercent(float64(memUsedKB) * 100 / float64(memTotalKB))
		}
		gpus = append(gpus, SSHGPUInfo{
			Name:          name,
			Driver:        strings.TrimSpace(entry.Driver),
			VRAMMB:        memTotalKB / 1024,
			MemoryTotalMB: memTotalKB / 1024,
			MemoryUsedMB:  memUsedKB / 1024,
			MemoryUsage:   memUsage,
			Usage:         normalizeUsagePercent(entry.Usage),
			Temperature:   entry.Temperature,
			Resolution:    strings.TrimSpace(entry.Resolution),
		})
	}

	return gpus
}

func summarizeDisks(disks []SSHDiskInfo) (int64, int64, int64) {
	var totalMB int64
	var availableMB int64
	var usedMB int64

	for _, disk := range disks {
		totalMB += disk.TotalMB
		availableMB += disk.AvailableMB
		usedMB += disk.UsedMB
	}

	return totalMB, availableMB, usedMB
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
		setCachedResolvedIPMiss(address)
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
	setCachedResolvedIPMiss(address)
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
	if location, countryFlag, found, ok := getCachedIPLocation(parsedIP.String()); ok {
		if !found {
			return "", ""
		}
		return location, countryFlag
	}

	resp, err := ipWhoisClient.Get(fmt.Sprintf("https://ipwho.is/%s", ip))
	if err != nil {
		setCachedIPLocationMiss(parsedIP.String())
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		setCachedIPLocationMiss(parsedIP.String())
		return "", ""
	}

	var payload ipWhoisResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || !payload.Success {
		setCachedIPLocationMiss(parsedIP.String())
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

func getCachedIPLocation(ip string) (string, string, bool, bool) {
	ipLocationCacheMu.RLock()
	entry, ok := ipLocationCache[ip]
	ipLocationCacheMu.RUnlock()
	if !ok {
		return "", "", false, false
	}
	if time.Now().After(entry.expiresAt) {
		ipLocationCacheMu.Lock()
		delete(ipLocationCache, ip)
		ipLocationCacheMu.Unlock()
		return "", "", false, false
	}
	return entry.location, entry.countryFlag, entry.found, true
}

func setCachedIPLocation(ip, location, countryFlag string) {
	ipLocationCacheMu.Lock()
	ipLocationCache[ip] = ipLocationCacheEntry{
		location:    location,
		countryFlag: countryFlag,
		expiresAt:   time.Now().Add(ipLocationCacheTTL),
		found:       true,
	}
	ipLocationCacheMu.Unlock()
}

func setCachedIPLocationMiss(ip string) {
	if strings.TrimSpace(ip) == "" {
		return
	}
	ipLocationCacheMu.Lock()
	ipLocationCache[ip] = ipLocationCacheEntry{
		expiresAt: time.Now().Add(ipLocationMissTTL),
		found:     false,
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
	if !entry.found {
		return "", true
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
		found:     true,
	}
	resolvedIPCacheMu.Unlock()
}

func setCachedResolvedIPMiss(address string) {
	if strings.TrimSpace(address) == "" {
		return
	}
	resolvedIPCacheMu.Lock()
	resolvedIPCache[address] = resolvedIPCacheEntry{
		expiresAt: time.Now().Add(resolvedIPMissTTL),
		found:     false,
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

func resolveUptimeSeconds(uptimeSeconds, bootTime, currentEpoch int64) int64 {
	const maxReasonableUptime = int64(20 * 365 * 24 * 3600)
	const allowedDrift = int64(5 * 60)
	const minPlausibleUnixTime = int64(946684800)

	if currentEpoch <= 0 {
		currentEpoch = time.Now().Unix()
	}

	if bootTime >= minPlausibleUnixTime && currentEpoch > bootTime {
		derived := currentEpoch - bootTime
		if uptimeSeconds <= 0 {
			return derived
		}
		diff := uptimeSeconds - derived
		if diff < 0 {
			diff = -diff
		}
		if diff <= allowedDrift {
			return uptimeSeconds
		}
		if uptimeSeconds > maxReasonableUptime || uptimeSeconds > derived*2 || uptimeSeconds < derived/2 {
			return derived
		}
	}

	if uptimeSeconds > maxReasonableUptime {
		return 0
	}
	return uptimeSeconds
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

func normalizeUsagePercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
