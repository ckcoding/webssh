<template>
    <aside class="sidebar-container">
        <section class="sidebar-panel current-panel">
            <div class="panel-header">
                <div class="panel-title-wrap">
                    <div class="panel-eyebrow">{{ $t('currentSSHInfo') }}</div>
                    <div class="panel-title">
                        <span v-if="hasActiveTerminal && currentConnectionInfo.countryFlag" style="margin-right: 4px;">{{ currentConnectionInfo.countryFlag }}</span>
                        {{ hasActiveTerminal ? (currentConnectionInfo.alias || currentConnectionInfo.host) : $t('waitingConnection') }}
                    </div>
                    <div class="panel-subtitle">
                        {{ hasActiveTerminal ? currentConnectionSummary : $t('connectUsingControls') }}
                    </div>
                </div>
                <div v-if="hasActiveTerminal" class="compact-metrics">
                    <div class="compact-metric">
                        <span class="compact-dot" :style="{ backgroundColor: cpuMetric.color }"></span>
                        <span class="compact-label">CPU</span>
                        <strong>{{ cpuMetric.percentText }}</strong>
                    </div>
                    <div class="compact-metric">
                        <span class="compact-dot" :style="{ backgroundColor: memoryMetric.color }"></span>
                        <span class="compact-label">RAM</span>
                        <strong>{{ memoryMetric.percentText }}</strong>
                    </div>
                    <div class="compact-metric">
                        <span class="compact-dot" :style="{ backgroundColor: diskMetric.color }"></span>
                        <span class="compact-label">Disk</span>
                        <strong>{{ diskMetric.percentText }}</strong>
                    </div>
                </div>
            </div>

            <div v-if="hasActiveTerminal" class="panel-body custom-scrollbar">
                <div class="chip-row">
                    <span class="meta-chip">{{ currentConnectionInfo.username }}</span>
                    <span class="meta-chip">{{ currentConnectionInfo.port }}</span>
                    <span class="meta-chip">{{ authModeLabel(currentConnectionInfo) }}</span>
                    <span v-if="trimOrEmpty(currentConnectionInfo.location) !== ''" class="meta-chip subtle">
                        {{ currentConnectionInfo.location }}
                    </span>
                </div>

                <div class="ring-grid single-ring">
                    <div class="ring-card">
                        <div class="ring-circle" :style="ringStyle(cpuMetric.percentage, cpuMetric.color)">
                            <div class="ring-inner">
                                <strong>{{ cpuMetric.percentText }}</strong>
                                <span>CPU</span>
                            </div>
                        </div>
                        <div v-if="cpuMetric.detail" class="ring-subtitle">{{ cpuMetric.detail }}</div>
                    </div>
                </div>

                <div class="disk-card">
                    <div class="disk-header">
                        <div class="disk-left">
                            <div class="disk-label">{{ memoryMetric.label }}</div>
                            <div v-if="memoryMetric.detail" class="disk-detail">{{ memoryMetric.detail }}</div>
                        </div>
                        <div class="disk-right">
                            <div class="disk-percent">{{ memoryMetric.percentText }}</div>
                        </div>
                    </div>
                    <div class="custom-progress">
                        <div class="custom-progress-bg">
                            <div class="custom-progress-fill" :style="progressFillStyle(memoryMetric.percentage, memoryMetric.color)"></div>
                        </div>
                    </div>
                </div>

                <div class="disk-card">
                    <div class="disk-header">
                        <div class="disk-left">
                            <div class="disk-label">{{ diskMetric.label }}</div>
                            <div v-if="diskMetric.detail" class="disk-detail">{{ diskMetric.detail }}</div>
                        </div>
                        <div class="disk-right">
                            <div class="disk-percent">{{ diskMetric.percentText }}</div>
                        </div>
                    </div>
                    <div class="custom-progress">
                        <div class="custom-progress-bg">
                            <div class="custom-progress-fill" :style="progressFillStyle(diskMetric.percentage, diskMetric.color)"></div>
                        </div>
                    </div>
                    <div v-if="diskEntries.length > 0" class="disk-volume-list">
                        <div class="disk-volume-title">{{ $t('MountedDisks') }}</div>
                        <div v-for="disk in diskEntries" :key="disk.key" class="disk-volume-row">
                            <div class="disk-volume-main">
                                <div class="disk-volume-mount">{{ disk.mount }}</div>
                                <div v-if="disk.device" class="disk-volume-device">{{ disk.device }}</div>
                            </div>
                            <div class="disk-volume-side">
                                <div class="disk-volume-size">{{ disk.detail }}</div>
                                <div class="disk-volume-percent">{{ disk.percentText }}</div>
                            </div>
                            <div class="disk-volume-progress">
                                <div class="disk-volume-progress-fill" :style="progressFillStyle(disk.percentage, disk.color)"></div>
                            </div>
                        </div>
                    </div>
                    <div v-if="gpuEntries.length > 0" class="gpu-section">
                        <div class="disk-volume-title">{{ $t('GraphicsCards') }}</div>
                        <div v-for="gpu in gpuEntries" :key="gpu.key" class="disk-card gpu-card">
                            <div class="disk-header">
                                <div class="disk-left">
                                    <div class="disk-label">{{ gpu.name }}</div>
                                    <div v-if="gpu.detail" class="disk-detail">{{ gpu.detail }}</div>
                                </div>
                                <div class="disk-right">
                                    <div class="disk-percent">{{ gpu.usagePercent }}</div>
                                </div>
                            </div>
                            <div class="custom-progress">
                                <div class="custom-progress-bg">
                                    <div class="custom-progress-fill" :style="progressFillStyle(gpu.usageValue, gpu.usageColor)"></div>
                                </div>
                            </div>
                            <div v-if="gpu.hasMemory" class="gpu-mem-row">
                                <div class="gpu-mem-label">{{ $t('MemoryUsage') }}</div>
                                <div class="gpu-mem-detail">{{ gpu.memoryDetail }}</div>
                                <div class="custom-progress">
                                    <div class="custom-progress-bg">
                                        <div class="custom-progress-fill" :style="progressFillStyle(gpu.memoryUsageValue, gpu.memoryUsageColor)"></div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div v-if="hasActiveTerminal" class="status-row">
                <div class="status-block-top">
                    <div class="status-badge" :class="{ online: hasSystemMetrics, loading: !hasSystemMetrics }">
                        <i v-if="!hasSystemMetrics" class="el-icon-loading" style="margin-right: 4px;"></i>
                        {{ hasSystemMetrics ? $t('systemMetrics') : $t('loadingSystemInfo') }}
                    </div>
                    <div v-if="systemInfo.uptime" class="status-uptime">
                        {{ $t('Uptime') }}: <span>{{ systemInfo.uptime }}</span>
                    </div>
                </div>
                <div class="status-block-bottom">
                    <div class="status-expire">
                        {{ $t('ExpiresAt') }}: <span>{{ formatExpiresAtLabel(currentConnectionInfo.expiresAt) }}</span>
                    </div>
                    <div class="status-time">
                        {{ browserTimeText }}
                    </div>
                </div>
            </div>
        </section>
    </aside>
</template>

<script>
import { mapState } from 'vuex'

export default {
    name: 'ServerSidebar',
    data() {
        return {
            browserNow: new Date(),
            browserClockTimer: null
        }
    },
    computed: {
        ...mapState(['sshInfo', 'systemInfo', 'termList', 'currentTab']),
        hasActiveTerminal() {
            return this.termList.length > 0 && this.currentTab && this.currentTab.name
        },
        currentConnectionInfo() {
            const live = this.systemInfo || {}
            const current = this.sshInfo || {}
            return {
                host: current.host || '',
                alias: this.trimOrEmpty(current.alias),
                ip: this.trimOrEmpty(current.ip) || this.trimOrEmpty(live.ip) || this.resolveDisplayIP(current),
                countryFlag: this.trimOrEmpty(current.countryFlag) || this.trimOrEmpty(live.countryFlag),
                location: this.trimOrEmpty(current.location) || this.trimOrEmpty(live.location),
                os: this.trimOrEmpty(current.os) || this.trimOrEmpty(live.os),
                expiresAt: this.trimOrEmpty(current.expiresAt),
                username: current.username || 'root',
                port: Number(current.port) || 22,
                logintype: current.logintype === undefined ? 0 : Number(current.logintype)
            }
        },
        currentConnectionSummary() {
            return this.trimOrEmpty(this.currentConnectionInfo.os)
        },
        cpuMetric() {
            return {
                label: this.$t('CPUUsage'),
                percentText: this.formatPercent(this.systemInfo.cpuUsage) || '--',
                detail: this.cpuFootnote,
                percentage: this.progressValue(this.systemInfo.cpuUsage),
                color: '#22c55e'
            }
        },
        memoryMetric() {
            const usage = this.systemInfo.memoryUsage
            return {
                label: this.$t('MemoryUsage'),
                percentText: this.formatPercent(usage) || '--',
                detail: this.memoryMetricValue,
                percentage: this.progressValue(usage),
                color: this.usageColor(usage, '#3b82f6')
            }
        },
        diskMetric() {
            const usage = this.systemInfo.diskUsage
            return {
                label: this.$t('DiskUsage'),
                percentText: this.formatPercent(usage) || '--',
                detail: this.diskMetricValue,
                percentage: this.progressValue(usage),
                color: this.usageColor(usage, '#22c55e')
            }
        },
        diskEntries() {
            const disks = Array.isArray(this.systemInfo.disks) ? this.systemInfo.disks : []
            return disks
                .map((disk, index) => {
                    const mount = this.trimOrEmpty(disk.mount)
                    const device = this.trimOrEmpty(disk.device)
                    const totalMB = Number(disk.totalMB)
                    const usedMB = Number(disk.usedMB)
                    const usage = Number(disk.usage)
                    if ((!mount && !device) || !Number.isFinite(totalMB) || totalMB <= 0) {
                        return null
                    }
                    const total = this.formatSizeMB(totalMB)
                    const used = this.formatSizeMB(usedMB)
                    return {
                        key: `${mount || device}-${index}`,
                        mount: mount || device,
                        device: mount && device && device !== mount ? device : '',
                        detail: used !== '' && total !== '' ? `${used} / ${total}` : total,
                        percentText: this.formatPercent(usage) || '--',
                        percentage: this.progressValue(usage),
                        color: this.usageColor(usage, '#22c55e')
                    }
                })
                .filter(Boolean)
        },
        gpuEntries() {
            const gpus = Array.isArray(this.systemInfo.gpus) ? this.systemInfo.gpus : []
            return gpus
                .map((gpu, index) => {
                    const name = this.trimOrEmpty(gpu.name)
                    if (!name) {
                        return null
                    }
                    const usage = Number(gpu.usage)
                    const usageValue = this.progressValue(usage)
                    const memTotalMB = Number(gpu.memoryTotalMB)
                    const memUsedMB = Number(gpu.memoryUsedMB)
                    const hasMemory = Number.isFinite(memTotalMB) && memTotalMB > 0
                    const memUsage = Number(gpu.memoryUsage)
                    const memUsageValue = this.progressValue(memUsage)
                    const parts = []
                    const driver = this.trimOrEmpty(gpu.driver)
                    if (driver) {
                        parts.push(this.$i18n.locale === 'zh' ? '驱动 ' + driver : 'driver ' + driver)
                    }
                    const temp = Number(gpu.temperature)
                    if (Number.isFinite(temp) && temp > 0) {
                        parts.push(this.$i18n.locale === 'zh' ? temp + '°C' : temp + '°C')
                    }
                    const resolution = this.trimOrEmpty(gpu.resolution)
                    if (resolution) {
                        parts.push(resolution)
                    }
                    let memoryDetail = ''
                    if (hasMemory) {
                        const used = this.formatSizeMB(memUsedMB)
                        const total = this.formatSizeMB(memTotalMB)
                        if (used !== '' && total !== '') {
                            memoryDetail = `${used} / ${total}`
                        } else {
                            memoryDetail = total
                        }
                    }
                    return {
                        key: `${name}-${index}`,
                        name: name,
                        detail: parts.join(' · '),
                        usagePercent: this.formatPercent(usage) || '--',
                        usageValue: usageValue,
                        usageColor: this.usageColor(usage, '#a855f7'),
                        hasMemory: hasMemory,
                        memoryDetail: memoryDetail,
                        memoryUsageValue: memUsageValue,
                        memoryUsageColor: this.usageColor(memUsage, '#3b82f6')
                    }
                })
                .filter(Boolean)
        },
        memoryMetricValue() {
            const normalized = this.trimOrEmpty(this.memoryFootnote)
            if (normalized === '' || normalized === this.$t('loadingSystemInfo')) {
                return '--'
            }
            return normalized
        },
        diskMetricValue() {
            const normalized = this.trimOrEmpty(this.diskFootnote)
            if (normalized === '' || normalized === this.$t('loadingSystemInfo')) {
                return '--'
            }
            return normalized
        },
        cpuFootnote() {
            const cpuCores = Number(this.systemInfo.cpuCores)
            if (cpuCores > 0) {
                return this.$i18n.locale === 'zh' ? `${cpuCores} 核心` : `${cpuCores} cores`
            }
            return this.$t('loadingSystemInfo')
        },
        memoryFootnote() {
            const used = this.formatSizeMB(this.systemInfo.memoryUsedMB)
            const total = this.formatSizeMB(this.systemInfo.memoryTotalMB)
            if (used !== '' && total !== '') {
                return `${used} / ${total}`
            }
            return total || this.$t('loadingSystemInfo')
        },
        diskFootnote() {
            const used = this.formatSizeMB(this.systemInfo.diskUsedMB)
            const total = this.formatSizeMB(this.systemInfo.diskTotalMB)
            if (used !== '' && total !== '') {
                return `${used} / ${total}`
            }
            return total || this.$t('loadingSystemInfo')
        },
        hasSystemMetrics() {
            return Boolean(this.trimOrEmpty(this.systemInfo.updatedAt)) ||
                Number(this.systemInfo.cpuCores) > 0 ||
                Number(this.systemInfo.memoryTotalMB) > 0 ||
                Number(this.systemInfo.diskTotalMB) > 0
        },
        browserTimeText() {
            return this.formatBrowserTime(this.browserNow)
        }
    },
    mounted() {
        this.startBrowserClock()
    },
    beforeDestroy() {
        this.stopBrowserClock()
    },
    methods: {
        startBrowserClock() {
            this.stopBrowserClock()
            this.browserNow = new Date()
            this.browserClockTimer = window.setInterval(() => {
                this.browserNow = new Date()
            }, 1000)
        },
        stopBrowserClock() {
            if (this.browserClockTimer !== null) {
                window.clearInterval(this.browserClockTimer)
                this.browserClockTimer = null
            }
        },
        trimOrEmpty(value) {
            if (value === undefined || value === null) {
                return ''
            }
            return String(value).trim()
        },
        formatPercent(value) {
            const parsed = Number(value)
            if (!Number.isFinite(parsed)) {
                return ''
            }
            return `${parsed.toFixed(2)}%`
        },
        progressValue(value) {
            const parsed = Number(value)
            if (!Number.isFinite(parsed)) {
                return 0
            }
            return Math.max(0, Math.min(100, Math.round(parsed * 100) / 100))
        },
        usageColor(value, fallback) {
            const parsed = Number(value)
            if (!Number.isFinite(parsed)) {
                return fallback
            }
            if (parsed >= 90) {
                return '#ef4444'
            }
            if (parsed >= 75) {
                return '#f97316'
            }
            if (parsed >= 60) {
                return '#f59e0b'
            }
            return fallback
        },
        progressFillStyle(value, color) {
            const percentage = this.progressValue(value)
            const style = {
                width: `${percentage}%`,
                backgroundColor: color
            }
            if (percentage > 0 && percentage < 2) {
                style.minWidth = '4px'
            }
            return style
        },
        ringStyle(value, color) {
            const degree = Math.round(this.progressValue(value) * 3.6)
            return {
                background: `conic-gradient(${color} 0deg ${degree}deg, rgba(148, 163, 184, 0.18) ${degree}deg 360deg)`
            }
        },
        formatSizeMB(value) {
            const parsed = Number(value)
            if (!Number.isFinite(parsed) || parsed <= 0) {
                return ''
            }
            if (parsed >= 1024 * 1024) {
                const tb = parsed / (1024 * 1024)
                return `${tb % 1 === 0 ? tb.toFixed(0) : tb.toFixed(1)} TB`
            }
            if (parsed >= 1024) {
                const gb = parsed / 1024
                return `${gb % 1 === 0 ? gb.toFixed(0) : gb.toFixed(1)} GB`
            }
            return `${parsed.toFixed(0)} MB`
        },
        resolveDisplayIP(info = {}) {
            const ip = this.trimOrEmpty(info.ip)
            if (ip !== '') {
                return ip
            }
            const host = this.trimOrEmpty(info.host)
            return /^(?:\d{1,3}\.){3}\d{1,3}$/.test(host) || host.includes(':') ? host : ''
        },
        formatExpiresAtLabel(value) {
            const normalized = this.trimOrEmpty(value)
            if (normalized === '') {
                return this.$t('Permanent')
            }
            const parsed = new Date(normalized.replace(' ', 'T'))
            if (Number.isNaN(parsed.getTime())) {
                return normalized
            }
            return parsed.toLocaleString(this.$i18n.locale === 'zh' ? 'zh-CN' : 'en-US', {
                hour12: false
            })
        },
        authModeLabel(info = {}) {
            return this.$t(Number(info.logintype) === 1 ? 'privateKey' : 'password')
        },
        formatBrowserTime(value) {
            const date = value instanceof Date ? value : new Date(value)
            if (Number.isNaN(date.getTime())) {
                return ''
            }
            return date.toLocaleString(this.$i18n.locale === 'zh' ? 'zh-CN' : 'en-US', {
                hour12: false
            })
        }
    }
}
</script>

<style scoped lang="scss">
.sidebar-container {
    height: 100%;
    display: flex;
    flex-direction: column;
    padding: 20px 0 20px 24px;
    box-sizing: border-box;
    min-height: 0;
    background: transparent;
    position: relative;
    overflow: hidden;
}

.sidebar-panel {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 18px;
    box-shadow: var(--panel-shadow);
    z-index: 1;
}

.current-panel {
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
}

.panel-header {
    flex-shrink: 0;
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border);
}

.panel-body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    margin: 0 -12px;
    padding: 0 12px;
}


.panel-title-wrap {
    flex: 1;
    min-width: 0;
    padding-right: 8px;
}

.panel-eyebrow {
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--text-muted);
}

.panel-title {
    margin-top: 8px;
    font-size: 18px;
    font-weight: 700;
    color: #0f172a;
    line-height: 1.25;
    word-break: break-all;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    overflow: hidden;
    letter-spacing: -0.01em;
}

.panel-subtitle {
    margin-top: 6px;
    font-size: 13px;
    color: #64748b;
    line-height: 1.5;
    font-weight: 500;
    word-break: break-word;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 3;
    overflow: hidden;
}


.chip-row {
    margin-top: 16px;
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

.meta-chip {
    display: inline-flex;
    align-items: center;
    padding: 5px 9px;
    border-radius: 6px;
    background: #f5f7fb;
    color: #334155;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.02em;
    font-family: var(--mono-font);
    border: 1px solid rgba(226, 232, 240, 0.8);
}

.meta-chip.subtle {
    background: rgba(239, 246, 255, 0.8);
    color: #2563eb;
    border-color: rgba(191, 219, 254, 0.5);
}

.ring-grid {
    margin-top: 16px;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
}

.ring-grid.single-ring {
    grid-template-columns: 1fr;
}

.ring-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    padding: 20px 16px;
    border-radius: 10px;
    background: #f8fafd;
    border: 1px solid #edf0f6;
}

.ring-circle {
    width: 96px;
    height: 96px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 4px 14px rgba(0, 0, 0, 0.05);
    flex-shrink: 0;
}

.ring-inner {
    width: 76px;
    height: 76px;
    border-radius: 50%;
    background: #ffffff;
    box-shadow: inset 0 0 0 1px rgba(226, 232, 240, 0.6), inset 0 2px 4px rgba(0,0,0,0.02);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
}

.ring-inner strong {
    font-size: 16px;
    font-weight: 800;
    color: #0f172a;
    line-height: 1;
}

.ring-inner span {
    margin-top: 2px;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.1em;
    color: #64748b;
}

.ring-title {
    font-size: 14px;
    font-weight: 800;
    color: #1e293b;
    text-align: center;
    white-space: nowrap;
}

.ring-subtitle {
    font-size: 12px;
    color: #64748b;
    line-height: 1.3;
    text-align: center;
    word-break: break-word;
}

.disk-card {
    margin-top: 8px;
    padding: 14px 16px;
    border-radius: 10px;
    background: #f8fafd;
    border: 1px solid #edf0f6;
}

.disk-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 12px;
}

.disk-left {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
}

.disk-label {
    font-size: 13px;
    font-weight: 800;
    color: #1e293b;
}

.disk-detail {
    font-size: 12px;
    color: #64748b;
    font-weight: 500;
}

.disk-right {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    flex-shrink: 0;
}

.disk-percent {
    font-size: 16px;
    font-weight: 800;
    color: #0f172a;
    line-height: 1;
}

.custom-progress {
    width: 100%;
    margin-top: 8px;
}

.custom-progress-bg {
    width: 100%;
    height: 6px;
    background-color: rgba(226, 232, 240, 0.6);
    border-radius: 4px;
    overflow: hidden;
    position: relative;
    box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.05);
}

.custom-progress-fill {
    height: 100%;
    border-radius: 4px;
    transition: width 0.5s ease-in-out;
}

.disk-volume-list {
    margin-top: 14px;
    padding-top: 12px;
    border-top: 1px solid rgba(148, 163, 184, 0.16);
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.disk-volume-title {
    font-size: 11px;
    font-weight: 800;
    color: #64748b;
    letter-spacing: 0.08em;
    text-transform: uppercase;
}

.gpu-section {
    margin-top: 14px;
    padding-top: 12px;
    border-top: 1px solid rgba(148, 163, 184, 0.16);
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.gpu-card {
    padding: 10px 12px;
    border: 1px solid rgba(148, 163, 184, 0.16);
    border-radius: 10px;
    background: rgba(248, 250, 252, 0.6);
}

.gpu-card .disk-label {
    font-size: 12px;
    font-weight: 700;
    color: #1e293b;
    word-break: break-word;
}

.gpu-mem-row {
    margin-top: 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.gpu-mem-label {
    font-size: 10px;
    font-weight: 700;
    color: #64748b;
    letter-spacing: 0.04em;
}

.gpu-mem-detail {
    font-size: 11px;
    color: #475569;
}

.disk-volume-row {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: 12px;
    align-items: flex-start;
}

.disk-volume-main {
    flex: 1 1 0;
    min-width: 0;
}

.disk-volume-mount {
    font-size: 12px;
    font-weight: 700;
    color: #0f172a;
    word-break: break-all;
}

.disk-volume-device {
    margin-top: 2px;
    font-size: 11px;
    color: #94a3b8;
    word-break: break-all;
}

.disk-volume-side {
    flex-shrink: 0;
    text-align: right;
}

.disk-volume-progress {
    flex: 1 0 100%;
    height: 6px;
    border-radius: 3px;
    background-color: rgba(226, 232, 240, 0.72);
    overflow: hidden;
}

.disk-volume-progress-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.5s ease-in-out;
}

.disk-volume-size {
    font-size: 12px;
    font-weight: 600;
    color: #334155;
}

.disk-volume-percent {
    margin-top: 2px;
    font-size: 11px;
    color: #64748b;
    font-weight: 700;
}

.status-row {
    flex-shrink: 0;
    margin-top: 12px;
    padding-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px; /* Breathe room between badge & text */
    border-top: 1px solid rgba(0, 0, 0, 0.04);
}

.status-block-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    gap: 8px;
    flex-wrap: wrap;
}

.status-uptime {
    font-size: 11px;
    color: #94a3b8;
    white-space: nowrap;
}

.status-uptime span {
    font-weight: 600;
    color: #64748b;
}

.status-block-bottom {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    gap: 6px;
    flex-wrap: wrap;
}

.status-time {
    font-size: 11px;
    font-weight: 500;
    color: #94a3b8;
    text-align: right;
    white-space: nowrap;
}

.status-badge {
    display: inline-flex;
    align-items: center;
    padding: 6px 12px;
    border-radius: 999px;
    background: rgba(241, 245, 249, 0.8);
    color: #475569;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.02em;
    white-space: nowrap;
    flex-shrink: 0;
}

.status-badge.online {
    background: rgba(220, 252, 231, 0.6);
    color: #166534;
    box-shadow: inset 0 0 0 1px rgba(134, 239, 172, 0.5);
}

.status-expire {
    font-size: 11px;
    color: #94a3b8;
    white-space: nowrap;
}

.status-expire span {
    font-weight: 600;
    color: #64748b;
}

.compact-metrics {
    display: none;
}

@media (max-width: 1100px) {
    .sidebar-container {
        padding: 12px;
        height: auto !important;
    }

    .sidebar-panel {
        padding: 16px;
        height: auto !important;
        min-height: 0;
    }
    
    .panel-header {
        border-bottom: none;
        padding-bottom: 0;
        align-items: center;
        flex-wrap: wrap;
    }

    .panel-body, .status-row {
        display: none !important;
    }

    .compact-metrics {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        align-items: center;
        margin-left: auto;
    }

    .compact-metric {
        display: inline-flex;
        align-items: center;
        background: rgba(255, 255, 255, 0.8);
        padding: 4px 10px;
        border-radius: 999px;
        border: 1px solid rgba(226, 232, 240, 0.8);
        box-shadow: 0 1px 3px rgba(0,0,0,0.02);
    }

    .compact-dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        margin-right: 6px;
    }

    .compact-label {
        font-size: 11px;
        color: #64748b;
        font-weight: 700;
        text-transform: uppercase;
        margin-right: 4px;
    }

    .compact-metric strong {
        font-size: 12px;
        color: #0f172a;
        font-weight: 800;
        line-height: 1;
    }
}

@media (max-width: 768px) {
    .sidebar-container {
        padding: 12px;
    }
}
</style>
