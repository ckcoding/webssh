<template>
    <div class="ov-trigger">
        <el-button size="small" icon="el-icon-data-board" class="icon-btn ov-open-btn" @click="open">
            {{ $t('serverOverview') }}
        </el-button>

        <el-dialog :visible.sync="visible" fullscreen append-to-body custom-class="ov-dialog"
            :show-close="true" @open="handleOpen" @closed="handleClosed">
            <div slot="title" class="ov-title">
                <span class="ov-title-main"><i class="el-icon-data-board"></i>{{ $t('serverOverview') }}</span>
                <span class="ov-title-sub">{{ $t('ovAgentless') }}</span>
            </div>

            <div class="ov-body">
                <div class="ov-summary">
                    <button type="button" class="ov-stat" :class="{ active: statusFilter === 'all' }" @click="statusFilter = 'all'">
                        <span class="ov-stat-label">{{ $t('ovTotal') }}</span>
                        <span class="ov-stat-value">{{ summary.total }}</span>
                    </button>
                    <button type="button" class="ov-stat is-online" :class="{ active: statusFilter === 'online' }" @click="statusFilter = 'online'">
                        <span class="ov-stat-label"><i class="ov-dot is-online"></i>{{ $t('ovOnline') }}</span>
                        <span class="ov-stat-value">{{ summary.online }}</span>
                    </button>
                    <button type="button" class="ov-stat is-offline" :class="{ active: statusFilter === 'offline' }" @click="statusFilter = 'offline'">
                        <span class="ov-stat-label"><i class="ov-dot is-offline"></i>{{ $t('ovOffline') }}</span>
                        <span class="ov-stat-value">{{ summary.offline }}</span>
                    </button>
                    <div class="ov-stat is-static">
                        <span class="ov-stat-label">{{ $t('ovRealtimeSpeed') }}</span>
                        <span class="ov-stat-value ov-speed">
                            <span><i class="el-icon-bottom"></i>{{ formatRate(summary.netRxRate) }}</span>
                            <span><i class="el-icon-top"></i>{{ formatRate(summary.netTxRate) }}</span>
                        </span>
                    </div>
                    <div class="ov-stat is-static">
                        <span class="ov-stat-label">{{ $t('ovTrafficSinceBoot') }}</span>
                        <span class="ov-stat-value ov-speed">
                            <span><i class="el-icon-bottom"></i>{{ formatBytes(summary.netRxTotal) }}</span>
                            <span><i class="el-icon-top"></i>{{ formatBytes(summary.netTxTotal) }}</span>
                        </span>
                    </div>
                </div>

                <div class="ov-toolbar">
                    <el-input v-model="search" size="small" clearable prefix-icon="el-icon-search" class="ov-search"
                        :placeholder="$t('searchHosts')" :aria-label="$t('searchHosts')"></el-input>
                    <div class="ov-segment" role="group" :aria-label="$t('ovSortBy')">
                        <button v-for="option in sortOptions" :key="option.key" type="button"
                            :class="{ active: sortKey === option.key }" @click="setSort(option.key)">
                            {{ option.label }}<i v-if="sortKey === option.key && option.key !== 'default'"
                                :class="sortDesc ? 'el-icon-sort-down' : 'el-icon-sort-up'"></i>
                        </button>
                    </div>
                    <div class="ov-toolbar-right">
                        <span class="ov-updated" :class="{ 'is-loading': loading }">
                            <i :class="loading ? 'el-icon-loading' : 'el-icon-time'"></i>
                            {{ updatedLabel }}
                        </span>
                        <div class="ov-segment" role="group" :aria-label="$t('ovView')">
                            <button type="button" :class="{ active: viewMode === 'card' }" :title="$t('ovCardView')" @click="setView('card')"><i class="el-icon-menu"></i></button>
                            <button type="button" :class="{ active: viewMode === 'list' }" :title="$t('ovListView')" @click="setView('list')"><i class="el-icon-s-unfold"></i></button>
                        </div>
                    </div>
                </div>

                <div v-if="errorText" class="ov-error"><i class="el-icon-warning-outline"></i>{{ errorText }}</div>

                <div v-if="!firstLoaded" class="ov-empty"><i class="el-icon-loading"></i>{{ $t('ovCollecting') }}</div>
                <div v-else-if="servers.length === 0" class="ov-empty"><i class="el-icon-monitor"></i>{{ $t('noSavedConnections') }}</div>
                <div v-else-if="visibleServers.length === 0" class="ov-empty"><i class="el-icon-search"></i>{{ $t('noMatchingHosts') }}</div>

                <div v-else-if="viewMode === 'card'" class="ov-grid">
                    <div v-for="server in visibleServers" :key="server.id" class="ov-card" :class="'is-' + server.status">
                        <div class="ov-card-head">
                            <span class="ov-dot" :class="'is-' + server.status" :title="statusLabel(server)"></span>
                            <span class="ov-flag" v-if="server.countryFlag">{{ server.countryFlag }}</span>
                            <span class="ov-name" :title="server.alias || server.host">{{ server.alias || server.host }}</span>
                            <el-button type="text" size="mini" icon="el-icon-connection" class="ov-connect"
                                @click="connect(server)">{{ $t('ovConnect') }}</el-button>
                        </div>
                        <div class="ov-card-sub">
                            <span>{{ server.host }}</span>
                            <span v-if="server.info && server.info.os">&nbsp;· {{ server.info.os }}</span>
                        </div>

                        <template v-if="server.status === 'online' && server.info">
                            <div class="ov-meters">
                                <div class="ov-meter" v-for="meter in meters(server)" :key="meter.key">
                                    <div class="ov-meter-row">
                                        <span class="ov-meter-label">{{ meter.label }}</span>
                                        <span class="ov-meter-value">{{ meter.value.toFixed(1) }}%</span>
                                    </div>
                                    <div class="ov-bar"><span :class="levelClass(meter.value)" :style="{ width: Math.min(100, meter.value) + '%' }"></span></div>
                                    <div class="ov-meter-detail">{{ meter.detail }}</div>
                                </div>
                            </div>
                            <div class="ov-net">
                                <span class="is-down"><i class="el-icon-bottom"></i>{{ formatRate(server.netRxRate) }}</span>
                                <span class="is-up"><i class="el-icon-top"></i>{{ formatRate(server.netTxRate) }}</span>
                                <span class="ov-net-total">{{ $t('ovTraffic') }} {{ formatBytes(server.info.netRxBytes) }} / {{ formatBytes(server.info.netTxBytes) }}</span>
                            </div>
                            <div class="ov-facts">
                                <span :title="$t('ovUptime')"><i class="el-icon-time"></i>{{ server.info.uptime || '-' }}</span>
                                <span :title="$t('ovLoad')"><i class="el-icon-odometer"></i>{{ server.info.loadAverage || '-' }}</span>
                                <span :title="$t('ovProcesses')"><i class="el-icon-cpu"></i>{{ server.info.processCount || '-' }}</span>
                                <span :title="$t('ovTCP')">TCP {{ server.info.tcpConnCount || '-' }}</span>
                                <span v-if="server.expiresAt" :title="$t('ExpiresAt')"><i class="el-icon-date"></i>{{ server.expiresAt.slice(0, 10) }}</span>
                            </div>
                        </template>
                        <div v-else class="ov-card-offline">
                            <i :class="server.status === 'nocred' ? 'el-icon-lock' : 'el-icon-warning-outline'"></i>
                            <div>
                                <div class="ov-offline-title">{{ statusLabel(server) }}</div>
                                <div class="ov-offline-detail">{{ server.status === 'nocred' ? $t('ovNoCredHint') : server.error }}</div>
                                <div v-if="server.lastSeenAt" class="ov-offline-detail">{{ $t('ovLastSeen') }} {{ formatTime(server.lastSeenAt) }}</div>
                            </div>
                        </div>
                    </div>
                </div>

                <div v-else class="ov-table">
                    <div class="ov-row ov-row-head">
                        <span>{{ $t('ovServer') }}</span><span>CPU</span><span>{{ $t('ovMemory') }}</span><span>{{ $t('ovDisk') }}</span>
                        <span>{{ $t('ovNetSpeed') }}</span><span>{{ $t('ovTraffic') }}</span><span>{{ $t('ovUptime') }}</span><span>{{ $t('ovLoad') }}</span><span></span>
                    </div>
                    <div v-for="server in visibleServers" :key="server.id" class="ov-row" :class="'is-' + server.status">
                        <span class="ov-row-name">
                            <span class="ov-dot" :class="'is-' + server.status" :title="statusLabel(server)"></span>
                            <span v-if="server.countryFlag">{{ server.countryFlag }}</span>
                            <span class="ov-name">{{ server.alias || server.host }}</span>
                        </span>
                        <template v-if="server.status === 'online' && server.info">
                            <span v-for="meter in meters(server).slice(0, 3)" :key="meter.key" class="ov-cell-meter">
                                <span class="ov-bar"><span :class="levelClass(meter.value)" :style="{ width: Math.min(100, meter.value) + '%' }"></span></span>
                                {{ meter.value.toFixed(0) }}%
                            </span>
                            <span class="ov-mono">↓{{ formatRate(server.netRxRate) }} ↑{{ formatRate(server.netTxRate) }}</span>
                            <span class="ov-mono">{{ formatBytes(server.info.netRxBytes) }} / {{ formatBytes(server.info.netTxBytes) }}</span>
                            <span>{{ server.info.uptime || '-' }}</span>
                            <span class="ov-mono">{{ server.info.loadAverage || '-' }}</span>
                        </template>
                        <span v-else class="ov-row-offline">{{ statusLabel(server) }}<template v-if="server.status !== 'nocred' && server.error"> · {{ server.error }}</template></span>
                        <span class="ov-row-action">
                            <el-button type="text" size="mini" icon="el-icon-connection" @click="connect(server)">{{ $t('ovConnect') }}</el-button>
                        </span>
                    </div>
                </div>
            </div>
        </el-dialog>
    </div>
</template>

<script>
import { getServerOverview } from '@/api/ssh'

const POLL_INTERVAL = 5000
const VIEW_KEY = 'webssh_overview_view'
const SORT_KEY = 'webssh_overview_sort'

function readStorage(key, fallback) {
    try {
        return localStorage.getItem(key) || fallback
    } catch (e) {
        return fallback
    }
}

function writeStorage(key, value) {
    try {
        localStorage.setItem(key, value)
    } catch (e) {
        // 隐私模式等场景下忽略
    }
}

export default {
    name: 'ServerOverview',
    data() {
        const savedSort = readStorage(SORT_KEY, 'default:desc').split(':')
        return {
            visible: false,
            loading: false,
            firstLoaded: false,
            servers: [],
            summary: { total: 0, online: 0, offline: 0, netRxRate: 0, netTxRate: 0, netRxTotal: 0, netTxTotal: 0 },
            generatedAt: '',
            errorText: '',
            search: '',
            statusFilter: 'all',
            sortKey: savedSort[0] || 'default',
            sortDesc: savedSort[1] !== 'asc',
            viewMode: readStorage(VIEW_KEY, 'card') === 'list' ? 'list' : 'card',
            timer: null,
            requestSeq: 0
        }
    },
    computed: {
        sortOptions() {
            return [
                { key: 'default', label: this.$t('ovSortDefault') },
                { key: 'name', label: this.$t('ovSortName') },
                { key: 'cpu', label: 'CPU' },
                { key: 'memory', label: this.$t('ovMemory') },
                { key: 'disk', label: this.$t('ovDisk') },
                { key: 'netIn', label: this.$t('ovSortNetIn') },
                { key: 'netOut', label: this.$t('ovSortNetOut') },
                { key: 'uptime', label: this.$t('ovUptime') }
            ]
        },
        visibleServers() {
            const keyword = this.search.trim().toLowerCase()
            const list = this.servers.filter(server => {
                if (this.statusFilter === 'online' && server.status !== 'online') return false
                if (this.statusFilter === 'offline' && server.status === 'online') return false
                if (!keyword) return true
                return [server.alias, server.host, server.remark, server.location, server.info && server.info.os]
                    .some(value => value && String(value).toLowerCase().includes(keyword))
            })
            if (this.sortKey === 'default') {
                return list
            }
            const metric = this.sortMetric
            const direction = this.sortDesc ? -1 : 1
            return list.slice().sort((a, b) => {
                if (this.sortKey !== 'name') {
                    // 与哪吒一致：除按名称外，在线主机始终排在前面
                    const onlineDiff = (b.status === 'online') - (a.status === 'online')
                    if (onlineDiff !== 0) return onlineDiff
                }
                const va = metric(a)
                const vb = metric(b)
                if (va < vb) return -1 * direction
                if (va > vb) return 1 * direction
                return 0
            })
        },
        sortMetric() {
            const info = server => server.info || {}
            switch (this.sortKey) {
            case 'name': return server => (server.alias || server.host || '').toLowerCase()
            case 'cpu': return server => info(server).cpuUsage || 0
            case 'memory': return server => info(server).memoryUsage || 0
            case 'disk': return server => info(server).diskUsage || 0
            case 'netIn': return server => server.netRxRate || 0
            case 'netOut': return server => server.netTxRate || 0
            case 'uptime': return server => info(server).uptimeSeconds || 0
            default: return () => 0
            }
        },
        updatedLabel() {
            if (!this.generatedAt) {
                return this.$t('ovCollecting')
            }
            return `${this.$t('ovUpdatedAt')} ${this.formatTime(this.generatedAt)}`
        }
    },
    beforeDestroy() {
        this.stopPolling()
        document.removeEventListener('visibilitychange', this.handleVisibility)
    },
    methods: {
        open() {
            this.visible = true
        },
        connect(server) {
            this.visible = false
            this.$emit('connect', server.id)
        },
        handleOpen() {
            document.addEventListener('visibilitychange', this.handleVisibility)
            this.refresh()
            this.startPolling()
        },
        handleClosed() {
            this.stopPolling()
            document.removeEventListener('visibilitychange', this.handleVisibility)
        },
        handleVisibility() {
            if (document.hidden) {
                this.stopPolling()
            } else if (this.visible) {
                this.refresh()
                this.startPolling()
            }
        },
        startPolling() {
            this.stopPolling()
            this.timer = setInterval(() => {
                if (!this.loading) this.refresh()
            }, POLL_INTERVAL)
        },
        stopPolling() {
            if (this.timer) {
                clearInterval(this.timer)
                this.timer = null
            }
        },
        async refresh() {
            const seq = ++this.requestSeq
            this.loading = true
            try {
                const res = await getServerOverview()
                if (seq !== this.requestSeq || !this.visible) return
                if (res.Msg !== 'success' || !res.Data) {
                    this.errorText = res.Msg || this.$t('serverError')
                    return
                }
                this.errorText = ''
                this.servers = res.Data.servers || []
                this.summary = res.Data.summary || this.summary
                this.generatedAt = res.Data.generatedAt || ''
            } catch (error) {
                if (seq === this.requestSeq) {
                    this.errorText = (error && error.message) || this.$t('serverConnectionFailed')
                }
            } finally {
                if (seq === this.requestSeq) {
                    this.loading = false
                    this.firstLoaded = true
                }
            }
        },
        setSort(key) {
            if (this.sortKey === key && key !== 'default') {
                this.sortDesc = !this.sortDesc
            } else {
                this.sortKey = key
                this.sortDesc = key !== 'name'
            }
            writeStorage(SORT_KEY, `${this.sortKey}:${this.sortDesc ? 'desc' : 'asc'}`)
        },
        setView(mode) {
            this.viewMode = mode
            writeStorage(VIEW_KEY, mode)
        },
        meters(server) {
            const info = server.info || {}
            const list = [
                { key: 'cpu', label: 'CPU', value: info.cpuUsage || 0, detail: `${info.cpuCores || '-'} ${this.$t('ovCores')}` },
                { key: 'memory', label: this.$t('ovMemory'), value: info.memoryUsage || 0, detail: `${this.formatMB(info.memoryUsedMB)} / ${this.formatMB(info.memoryTotalMB)}` },
                { key: 'disk', label: this.$t('ovDisk'), value: info.diskUsage || 0, detail: `${this.formatMB(info.diskUsedMB)} / ${this.formatMB(info.diskTotalMB)}` }
            ]
            if (info.swapTotalMB > 0) {
                list.push({ key: 'swap', label: 'Swap', value: (info.swapUsedMB || 0) * 100 / info.swapTotalMB, detail: `${this.formatMB(info.swapUsedMB)} / ${this.formatMB(info.swapTotalMB)}` })
            }
            return list
        },
        levelClass(value) {
            if (value >= 90) return 'is-danger'
            if (value >= 70) return 'is-warning'
            return 'is-ok'
        },
        statusLabel(server) {
            if (server.status === 'online') return this.$t('ovOnline')
            if (server.status === 'nocred') return this.$t('ovNoCred')
            return this.$t('ovOffline')
        },
        formatBytes(value) {
            let size = Number(value) || 0
            const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
            let index = 0
            while (size >= 1024 && index < units.length - 1) {
                size /= 1024
                index++
            }
            return `${size >= 100 || index === 0 ? size.toFixed(0) : size.toFixed(1)} ${units[index]}`
        },
        formatRate(value) {
            return `${this.formatBytes(value)}/s`
        },
        formatMB(value) {
            return this.formatBytes((Number(value) || 0) * 1024 * 1024)
        },
        formatTime(value) {
            const date = new Date(value)
            if (Number.isNaN(date.getTime())) return value
            return date.toLocaleTimeString()
        }
    }
}
</script>

<style lang="scss">
.ov-trigger {
    display: inline-block;
}

.ov-dialog {
    background: #f3f5f9;

    .el-dialog__header {
        background: #fff;
        border-bottom: 1px solid #e3e8f0;
        padding: 16px 24px;
    }

    .el-dialog__headerbtn {
        top: 20px;
    }

    .el-dialog__body {
        padding: 0;
    }
}

.ov-title {
    display: flex;
    align-items: baseline;
    gap: 12px;
    flex-wrap: wrap;
}

.ov-title-main {
    font-size: 18px;
    font-weight: 700;
    color: #17243b;

    i {
        margin-right: 8px;
        color: #2563eb;
    }
}

.ov-title-sub {
    font-size: 12px;
    color: #6b7a90;
}

.ov-body {
    padding: 20px 24px 32px;
    max-width: 1600px;
    margin: 0 auto;
}

.ov-summary {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 12px;
    margin-bottom: 16px;
}

.ov-stat {
    display: flex;
    flex-direction: column;
    gap: 6px;
    text-align: left;
    padding: 14px 16px;
    background: #fff;
    border: 1px solid #e3e8f0;
    border-radius: 12px;
    font: inherit;
    cursor: pointer;
    transition: border-color .15s, box-shadow .15s;

    &:hover:not(.is-static) {
        border-color: #93b4f5;
    }

    &.active {
        border-color: #2563eb;
        box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.12);
    }

    &.is-static {
        cursor: default;
    }
}

.ov-stat-label {
    font-size: 12px;
    color: #6b7a90;
    display: inline-flex;
    align-items: center;
    gap: 6px;
}

.ov-stat-value {
    font-size: 24px;
    font-weight: 700;
    color: #17243b;
    font-variant-numeric: tabular-nums;

    &.ov-speed {
        font-size: 15px;
        display: flex;
        gap: 14px;
        flex-wrap: wrap;

        i {
            margin-right: 2px;
            color: #6b7a90;
        }
    }
}

.ov-toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
    margin-bottom: 16px;
}

.ov-search {
    width: 240px;
}

.ov-toolbar-right {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 12px;
}

.ov-updated {
    font-size: 12px;
    color: #6b7a90;

    i {
        margin-right: 4px;
    }
}

.ov-segment {
    display: inline-flex;
    flex-wrap: wrap;
    background: #fff;
    border: 1px solid #e3e8f0;
    border-radius: 8px;
    padding: 2px;

    button {
        border: 0;
        background: transparent;
        padding: 5px 10px;
        border-radius: 6px;
        font: inherit;
        font-size: 12px;
        color: #475569;
        cursor: pointer;

        i {
            margin-left: 2px;
        }

        &.active {
            background: #2563eb;
            color: #fff;
        }

        &:hover:not(.active) {
            background: #f1f5f9;
        }
    }
}

.ov-error {
    padding: 10px 14px;
    margin-bottom: 16px;
    border-radius: 8px;
    background: #fef2f2;
    color: #b91c1c;
    font-size: 13px;

    i {
        margin-right: 6px;
    }
}

.ov-empty {
    padding: 80px 0;
    text-align: center;
    color: #6b7a90;

    i {
        display: block;
        font-size: 28px;
        margin-bottom: 10px;
    }
}

.ov-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #94a3b8;
    flex: none;

    &.is-online {
        background: #16a34a;
        box-shadow: 0 0 0 3px rgba(22, 163, 74, 0.15);
    }

    &.is-offline {
        background: #dc2626;
    }

    &.is-nocred {
        background: #94a3b8;
    }
}

.ov-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 14px;
}

.ov-card {
    background: #fff;
    border: 1px solid #e3e8f0;
    border-radius: 14px;
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-width: 0;

    &.is-offline,
    &.is-nocred {
        background: #fbfcfd;
    }
}

.ov-card-head {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
}

.ov-flag {
    font-size: 16px;
}

.ov-name {
    font-weight: 700;
    color: #17243b;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
}

.ov-connect {
    margin-left: auto;
    padding: 0 !important;
}

.ov-card-sub {
    font-size: 12px;
    color: #6b7a90;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    margin-top: -4px;
}

.ov-meters {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px 14px;
}

.ov-meter-row {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    margin-bottom: 4px;
}

.ov-meter-label {
    color: #6b7a90;
}

.ov-meter-value {
    font-weight: 600;
    color: #17243b;
    font-variant-numeric: tabular-nums;
}

.ov-meter-detail {
    font-size: 11px;
    color: #94a3b8;
    margin-top: 3px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.ov-bar {
    display: block;
    height: 6px;
    border-radius: 3px;
    background: #eef2f7;
    overflow: hidden;

    > span {
        display: block;
        height: 100%;
        border-radius: 3px;
        transition: width .4s ease;

        &.is-ok {
            background: #2563eb;
        }

        &.is-warning {
            background: #f59e0b;
        }

        &.is-danger {
            background: #dc2626;
        }
    }
}

.ov-net {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
    font-size: 12px;
    font-variant-numeric: tabular-nums;
    padding-top: 8px;
    border-top: 1px dashed #e3e8f0;

    .is-down {
        color: #16a34a;
        font-weight: 600;
    }

    .is-up {
        color: #2563eb;
        font-weight: 600;
    }
}

.ov-net-total {
    margin-left: auto;
    color: #94a3b8;
}

.ov-facts {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;

    span {
        font-size: 11px;
        color: #475569;
        background: #f1f5f9;
        border-radius: 6px;
        padding: 2px 8px;

        i {
            margin-right: 3px;
        }
    }
}

.ov-card-offline {
    display: flex;
    gap: 10px;
    padding: 12px;
    border-radius: 10px;
    background: #fef2f2;
    color: #b91c1c;
    font-size: 12px;

    > i {
        font-size: 18px;
    }
}

.ov-card.is-nocred .ov-card-offline {
    background: #f1f5f9;
    color: #475569;
}

.ov-offline-title {
    font-weight: 700;
    margin-bottom: 2px;
}

.ov-offline-detail {
    word-break: break-all;
    opacity: .85;
}

.ov-table {
    background: #fff;
    border: 1px solid #e3e8f0;
    border-radius: 12px;
    overflow-x: auto;
}

.ov-row {
    display: grid;
    grid-template-columns: minmax(180px, 1.6fr) repeat(3, minmax(110px, 1fr)) minmax(170px, 1.2fr) minmax(150px, 1.1fr) minmax(90px, .8fr) minmax(110px, .8fr) 70px;
    align-items: center;
    gap: 12px;
    padding: 10px 16px;
    border-bottom: 1px solid #f1f5f9;
    font-size: 13px;
    color: #17243b;
    min-width: 1100px;

    &:last-child {
        border-bottom: 0;
    }
}

.ov-row-head {
    font-size: 12px;
    color: #6b7a90;
    background: #f8fafc;
    font-weight: 600;
}

.ov-row-name {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
}

.ov-cell-meter {
    display: flex;
    align-items: center;
    gap: 8px;
    font-variant-numeric: tabular-nums;

    .ov-bar {
        flex: 1;
    }
}

.ov-mono {
    font-variant-numeric: tabular-nums;
    font-size: 12px;
}

.ov-row-offline {
    grid-column: 2 / -2;
    color: #b91c1c;
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.ov-row.is-nocred .ov-row-offline {
    color: #6b7a90;
}

.ov-row-action {
    text-align: right;
}

@media (max-width: 640px) {
    .ov-body {
        padding: 16px;
    }

    .ov-search {
        width: 100%;
    }

    .ov-toolbar-right {
        margin-left: 0;
    }

    .ov-grid {
        grid-template-columns: 1fr;
    }
}
</style>
