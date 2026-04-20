<template>
    <aside class="sidebar-container">
        <section class="sidebar-panel current-panel">
            <div class="panel-header">
                <div class="panel-title-wrap">
                    <div class="panel-eyebrow">{{ $t('currentSSHInfo') }}</div>
                    <div class="panel-title">
                        <span v-if="hasActiveTerminal && currentConnectionInfo.countryFlag" style="margin-right: 4px;">{{ currentConnectionInfo.countryFlag }}</span>
                        {{ hasActiveTerminal ? currentConnectionInfo.host : $t('waitingConnection') }}
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
                        <div class="ring-title">{{ cpuMetric.label }}</div>
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
                            <div class="custom-progress-fill" :style="{ width: memoryMetric.percentage + '%', backgroundColor: memoryMetric.color }"></div>
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
                            <div class="custom-progress-fill" :style="{ width: diskMetric.percentage + '%', backgroundColor: diskMetric.color }"></div>
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
        formattedServerIP() {
            return this.formatServerIP(this.currentConnectionInfo)
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
            return {
                label: this.$t('MemoryUsage'),
                percentText: this.formatPercent(this.systemInfo.memoryUsage) || '--',
                detail: this.memoryMetricValue,
                percentage: this.progressValue(this.systemInfo.memoryUsage),
                color: '#3b82f6'
            }
        },
        diskMetric() {
            return {
                label: this.$t('DiskUsage'),
                percentText: this.formatPercent(this.systemInfo.diskUsage) || '--',
                detail: this.diskMetricValue,
                percentage: this.progressValue(this.systemInfo.diskUsage),
                color: '#f59e0b'
            }
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
                return `${this.$t('CPUCores')}: ${cpuCores}`
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
            return Math.max(0, Math.min(100, Math.round(parsed)))
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
        formatServerIP(info = {}) {
            const ip = this.resolveDisplayIP(info)
            const countryFlag = this.trimOrEmpty(info.countryFlag)
            if (ip === '') {
                return ''
            }
            return countryFlag === '' ? ip : `${ip} ${countryFlag}`
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
        },
        formatUpdatedAt(value) {
            const normalized = this.trimOrEmpty(value)
            if (normalized === '' || normalized === '--') {
                return ''
            }
            const parsed = new Date(normalized)
            if (Number.isNaN(parsed.getTime())) {
                return normalized
            }
            return parsed.toLocaleString(this.$i18n.locale === 'zh' ? 'zh-CN' : 'en-US', {
                hour12: false
            })
        },
        isValidTimestamp(value) {
            const normalized = this.trimOrEmpty(value)
            if (normalized === '' || normalized === '--') {
                return false
            }
            return !Number.isNaN(new Date(normalized).getTime())
        }
    }
}
</script>

<style scoped lang="scss">
.sidebar-container {
    height: 100%;
    display: flex;
    flex-direction: column;
    padding: 20px;
    box-sizing: border-box;
    min-height: 0;
    background: 
        radial-gradient(circle at top right, rgba(99, 102, 241, 0.15), transparent 40%),
        radial-gradient(circle at bottom left, rgba(56, 189, 248, 0.15), transparent 40%),
        radial-gradient(circle at center, rgba(168, 85, 247, 0.08), transparent 50%),
        linear-gradient(180deg, #f4f7f9 0%, #f9fafb 100%);
    position: relative;
    overflow: hidden;
}

.sidebar-container::before {
    content: '';
    position: absolute;
    inset: 0;
    background: url('data:image/svg+xml;utf8,<svg viewBox="0 0 200 200" xmlns="http://www.w3.org/2000/svg"><filter id="noiseFilter"><feTurbulence type="fractalNoise" baseFrequency="0.65" numOctaves="3" stitchTiles="stitch"/></filter><rect width="100%" height="100%" filter="url(%23noiseFilter)"/></svg>');
    opacity: 0.03;
    pointer-events: none;
}

.sidebar-panel {
    background: rgba(255, 255, 255, 0.55);
    border: 1px solid rgba(255, 255, 255, 0.9);
    border-radius: 24px;
    padding: 20px;
    box-shadow: 
        0 20px 40px -15px rgba(0, 0, 0, 0.05),
        0 0 0 1px rgba(255, 255, 255, 0.8) inset;
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    z-index: 1;
}

.current-panel {
    height: 100%;
    display: flex;
    flex-direction: column;
}

.panel-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    padding-bottom: 8px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.panel-body {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    margin: 0 -12px;
    padding: 0 12px;
}

.custom-scrollbar {
    &::-webkit-scrollbar {
        width: 4px;
    }
    &::-webkit-scrollbar-track {
        background: transparent;
    }
    &::-webkit-scrollbar-thumb {
        background: rgba(148, 163, 184, 0.2);
        border-radius: 10px;
    }
    &::-webkit-scrollbar-thumb:hover {
        background: rgba(148, 163, 184, 0.4);
    }
}

.panel-title-wrap {
    flex: 1;
    min-width: 0;
    padding-right: 8px;
}

.panel-eyebrow {
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    background: linear-gradient(90deg, #3b82f6, #8b5cf6);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
}

.panel-title {
    margin-top: 8px;
    font-size: 18px;
    font-weight: 800;
    color: #0f172a;
    line-height: 1.25;
    word-break: break-all;
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
    padding: 6px 14px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.8);
    color: #334155;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.02em;
    box-shadow: 0 2px 6px rgba(0,0,0,0.02), inset 0 1px 0 rgba(255,255,255,1);
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
    padding: 24px 16px;
    border-radius: 16px;
    background: rgba(255, 255, 255, 0.7);
    border: 1px solid rgba(255, 255, 255, 0.9);
    box-shadow: 0 4px 15px -5px rgba(0, 0, 0, 0.03);
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
    border-radius: 16px;
    background: rgba(255, 255, 255, 0.7);
    border: 1px solid rgba(255, 255, 255, 0.9);
    box-shadow: 0 4px 15px -5px rgba(0, 0, 0, 0.03);
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

.disk-mount {
    margin-top: 4px;
    font-size: 11px;
    color: #94a3b8;
    font-weight: 700;
}

.custom-progress {
    width: 100%;
    margin-top: 8px;
}

.custom-progress-bg {
    width: 100%;
    height: 16px;
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
    background-image: linear-gradient(
        45deg,
        rgba(255, 255, 255, 0.15) 25%,
        transparent 25%,
        transparent 50%,
        rgba(255, 255, 255, 0.15) 50%,
        rgba(255, 255, 255, 0.15) 75%,
        transparent 75%,
        transparent
    );
    background-size: 1rem 1rem;
}

.status-row {
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
