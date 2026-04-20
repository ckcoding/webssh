<template>
    <div class="header-container">
        <div class="nav-bar">
            <div class="brand">
                <span class="brand-icon">🖥️</span>
                <span class="logo">Web</span><span class="logo-bold">SSH</span>
                <span class="brand-badge">终端管理</span>
            </div>

            <div class="nav-actions">
                <el-button type="primary" size="small" icon="el-icon-plus" @click="openConnectionDialog()"
                    class="connect-btn">
                    {{ $t('NewHost') }}
                </el-button>

                <el-dropdown trigger="click" @command="handleSavedCommand" placement="bottom-end">
                    <el-button size="small" icon="el-icon-menu" class="icon-btn">
                        {{ savedSSHButtonLabel }} <i class="el-icon-arrow-down el-icon--right"></i>
                    </el-button>
                    <el-dropdown-menu slot="dropdown" class="premium-dropdown">
                        <div v-if="savedSSHList.length === 0" class="empty-dropdown">
                            <i class="el-icon-monitor"></i>
                            <p>{{ $t('noSavedConnections') }}</p>
                        </div>
                        <el-dropdown-item v-for="item in savedSSHList" :key="item.id" :command="item">
                            <div class="dropdown-item-content">
                                <div class="item-avatar">{{ (item.host || '?')[0].toUpperCase() }}</div>
                                <div class="item-info-dropdown">
                                    <div class="item-host-row">
                                        <span class="item-status-dot" :class="savedSSHHealthClass(item)" :title="savedSSHHealthTitle(item)"></span>
                                        <div class="item-host">{{ item.host }}</div>
                                    </div>
                                    <div class="item-user">{{ item.username }}@{{ item.port }} · {{ authModeLabel(item)
                                        }}</div>
                                    <div v-if="buildServerInfoChips(item).length > 0" class="item-meta-row">
                                        <span v-for="chip in buildServerInfoChips(item)" :key="chip.key"
                                            class="info-chip">{{ chip.label }}</span>
                                    </div>
                                </div>
                                <i class="el-icon-edit edit-btn" @click.stop="editSavedSSHItem(item)"></i>
                                <i class="el-icon-delete delete-btn" @click.stop="deleteSavedSSHItem(item)"></i>
                            </div>
                        </el-dropdown-item>
                    </el-dropdown-menu>
                </el-dropdown>

                <file-list></file-list>

                <el-button size="small" icon="el-icon-question" circle class="lang-btn" @click="showShortcuts"
                    :title="$t('keyboardShortcuts') || '键盘快捷键'"></el-button>

                <el-button size="small" icon="el-icon-sort" circle class="lang-btn" @click="handleSetLanguage()"
                    :title="$i18n.locale === 'zh' ? 'Switch to English' : '切换中文'"></el-button>
            </div>
        </div>

        <el-dialog :title="editingSSHId ? $t('EditHost') : $t('NewHost')" :visible.sync="connectionDialogVisible" width="520px"
            custom-class="premium-dialog" :close-on-click-modal="false" :close-on-press-escape="false" append-to-body>
            <el-form ref="sshForm" :model="draftSSHInfo" :rules="checkRules" label-position="top" @submit.native.prevent>
                <el-row :gutter="20">
                    <el-col :span="16">
                        <el-form-item :label="$t('Host')" prop="host">
                            <el-input v-model.trim="draftSSHInfo.host" :placeholder="$t('hostTip')"
                                @keyup.enter.native="handleSaveAndConnect"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :span="8">
                        <el-form-item :label="$t('Port')" prop="port">
                            <el-input v-model="draftSSHInfo.port" :placeholder="$t('portTip')"
                                @keyup.enter.native="handleSaveAndConnect"></el-input>
                        </el-form-item>
                    </el-col>
                </el-row>

                <el-form-item :label="$t('Username')" prop="username">
                    <el-input v-model.trim="draftSSHInfo.username" :placeholder="$t('nameTip')"
                        @keyup.enter.native="handleSaveAndConnect"></el-input>
                </el-form-item>

                <el-form-item :label="$t('ExpiresAt')">
                    <el-date-picker
                        v-model="draftSSHInfo.expiresAt"
                        type="date"
                        value-format="yyyy-MM-dd 23:59:59"
                        format="yyyy-MM-dd"
                        :placeholder="$t('expiresAtTip')"
                        clearable
                        class="full-width-input"
                    ></el-date-picker>
                    <div class="field-hint">{{ $t('expiresAtTip') }}</div>
                </el-form-item>

                <div class="auth-section">
                    <div class="auth-header">
                        <span class="section-title">{{ privateKey ? $t('privateKeyPanel') : $t('passwordPanel')
                            }}</span>
                        <el-button type="text" size="mini" class="toggle-auth-btn" @click="toggleLoginType">
                            {{ privateKey ? $t('usePassword') : $t('usePrivateKey') }}
                        </el-button>
                    </div>

                    <el-form-item prop="password">
                        <el-input v-if="!privateKey" v-model="draftSSHInfo.password" show-password
                            :placeholder="credentialPlaceholder" @keyup.enter.native="handleSaveAndConnect"></el-input>
                        <el-input v-else :rows="6" v-model="draftSSHInfo.password" type="textarea"
                            :placeholder="$t('keyTip')"></el-input>
                    </el-form-item>

                    <div v-if="privateKey" class="key-actions">
                        <input ref="pkFile" @change="handleChangePKFile" type="file" class="hidden-file-input" />
                        <el-button type="primary" plain size="small" icon="el-icon-upload2" class="premium-upload-btn"
                            @click="$refs.pkFile.click()">
                            {{ $t('SelectFile') }}
                        </el-button>
                    </div>
                </div>

                <el-alert v-if="connectionCheckState !== ''"
                    :title="connectionCheckTitle"
                    :type="connectionCheckState"
                    :description="connectionCheckDescription"
                    :closable="false"
                    show-icon
                    class="connection-check-alert">
                </el-alert>
            </el-form>
            <div slot="footer" class="dialog-footer">
                <el-button size="small" @click="connectionDialogVisible = false">{{ $t('Cancel') }}</el-button>
                <el-button size="small" type="warning" plain :loading="connectionCheckLoading" @click="handleCheckConnection">
                    {{ connectionCheckLoading ? $t('checkingConnection') : $t('TestConnection') }}
                </el-button>
                <el-button size="small" type="primary" :loading="saveAndConnectLoading" @click="handleSaveAndConnect">{{ $t('SaveAndConnect') }}</el-button>
            </div>
        </el-dialog>

        <keyboard-shortcuts ref="keyboardShortcuts" />
    </div>
</template>

<script>
import { getLanguage } from '@/lang/index'
import FileList from '@/components/FileList'
import KeyboardShortcuts from '@/components/KeyboardShortcuts'
import { buildSSHSystemStreamURL, checkSSHConnection, deleteSavedSSH, getSSHSystemInfo, listSavedSSH, saveSavedSSH } from '@/api/ssh'
import { mapState } from 'vuex'
import { decodeSSHList, encodeSSHInfo, encodeSSHList, isSameSSH, normalizeSSHInfo } from '@/utils/ssh'

const SYSTEM_INFO_REFRESH_INTERVAL = 5000
const SYSTEM_INFO_RETRY_MAX_INTERVAL = 30000
const SAVED_SSH_HEALTH_CONCURRENCY = 3

function createSSHInfo(source = {}) {
    const normalized = normalizeSSHInfo(source)
    return {
        ...normalized,
        password: source.password === undefined ? '' : source.password
    }
}

function buildSSHIdentity(source = {}) {
    const info = createSSHInfo(source)
    if (!info.host || !info.username || !info.port) {
        return ''
    }
    return [info.host, info.username, info.port, info.logintype].join('::')
}

async function runWithConcurrency(items, limit, handler) {
    const queue = items.slice()
    const workers = new Array(Math.max(1, Math.min(limit, queue.length))).fill(null).map(async () => {
        while (queue.length > 0) {
            const item = queue.shift()
            if (!item) {
                return
            }
            await handler(item)
        }
    })
    await Promise.all(workers)
}

export default {
    components: {
        'file-list': FileList,
        'keyboard-shortcuts': KeyboardShortcuts
    },
    data() {
        return {
            connectionDialogVisible: false,
            draftSSHInfo: createSSHInfo(),
            connectionCheckLoading: false,
            saveAndConnectLoading: false,
            connectionCheckResult: null,
            systemInfoLoading: false,
            systemInfoSource: null,
            systemInfoTimer: null,
            systemInfoReconnectTimer: null,
            systemInfoRequestId: 0,
            systemInfoFailureCount: 0,
            savedSSHHealthMap: {},
            savedSSHHealthSnapshotKey: '',
            savedSSHHealthRequestId: 0,
            editingSSHId: '',
            pageVisibilityHandler: null
        }
    },
    computed: {
        ...mapState(['sshInfo', 'systemInfo', 'currentTab', 'termList']),
        savedSSHList() {
            return this.$store.state.savedSSHList || []
        },
        savedSSHButtonLabel() {
            if (this.savedSSHList.length === 0) {
                return this.$t('SavedSSH')
            }
            return `${this.$t('SavedSSH')} (${this.healthySavedSSHCount}/${this.savedSSHList.length})`
        },
        checkRules() {
            return {
                host: [
                    { required: true, trigger: 'blur', message: this.$t('hostRequired') }
                ],
                port: [
                    { required: true, trigger: 'blur', type: 'number', transform(value) { return Number(value) }, message: this.$t('portRequired') }
                ],
                username: [
                    { required: true, trigger: 'blur', message: this.$t('usernameRequired') }
                ],
                password: [
                    { required: true, trigger: 'blur', message: this.$t('passwordRequired') }
                ]
            }
        },
        privateKey() {
            return this.draftSSHInfo.logintype === 1
        },
        credentialPlaceholder() {
            return this.$t('inputTip') + this.$t('Password')
        },
        sshList() {
            return decodeSSHList(this.$store.state.sshList)
        },
        healthySavedSSHCount() {
            return this.savedSSHList.filter(item => {
                const health = this.savedSSHHealthMap[item.id]
                return health && health.status === 'success'
            }).length
        },
        draftConnectionFingerprint() {
            return [
                this.trimOrEmpty(this.draftSSHInfo.host),
                this.trimOrEmpty(this.draftSSHInfo.username),
                Number(this.draftSSHInfo.port) || 22,
                this.draftSSHInfo.logintype === undefined ? 0 : Number(this.draftSSHInfo.logintype),
                this.draftSSHInfo.password || ''
            ].join('\u0001')
        },
        connectionCheckState() {
            return this.connectionCheckResult && this.connectionCheckResult.state ? this.connectionCheckResult.state : ''
        },
        connectionCheckTitle() {
            return this.connectionCheckResult && this.connectionCheckResult.title ? this.connectionCheckResult.title : ''
        },
        connectionCheckDescription() {
            return this.connectionCheckResult && this.connectionCheckResult.description ? this.connectionCheckResult.description : ''
        },
        hasActiveTerminal() {
            return this.termList.length > 0 && this.currentTab && this.currentTab.name
        },
        activeSSHIdentity() {
            if (!this.hasActiveTerminal) {
                return ''
            }
            if (this.trimOrEmpty(this.sshInfo.password) === '') {
                return ''
            }
            return buildSSHIdentity(this.sshInfo)
        },
        activeSystemInfoKey() {
            if (this.activeSSHIdentity === '' || !this.currentTab || !this.currentTab.name) {
                return ''
            }
            return `${this.currentTab.name}::${this.activeSSHIdentity}`
        },
        hasSystemMetrics() {
            const info = this.systemInfo || {}
            return Boolean(this.trimOrEmpty(info.updatedAt)) ||
                Number(info.cpuCores) > 0 ||
                Number(info.memoryTotalMB) > 0 ||
                Number(info.diskTotalMB) > 0
        }
    },
    watch: {
        draftConnectionFingerprint(next, prev) {
            if (prev !== undefined && next !== prev) {
                this.resetConnectionCheckState()
            }
        },
        activeSystemInfoKey: {
            immediate: true,
            handler(next, prev) {
                this.stopSystemInfoUpdates()
                this.systemInfoRequestId += 1
                this.systemInfoLoading = false
                this.systemInfoFailureCount = 0
                if (next === '') {
                    this.$store.commit('CLEAR_SYSTEM_INFO')
                    return
                }
                this.hydrateSSHMetadataFromCollections(this.sshInfo)
                if (next !== prev) {
                    this.$store.commit('CLEAR_SYSTEM_INFO')
                }
                if (!this.connectionDialogVisible) {
                    this.startSystemInfoUpdates()
                }
            }
        },
        connectionDialogVisible(visible) {
            if (visible) {
                this.stopSystemInfoUpdates()
                this.systemInfoRequestId += 1
                this.systemInfoLoading = false
                this.systemInfoFailureCount = 0
                this.resetConnectionCheckState()
                return
            }
            this.resetConnectionCheckState()
            if (this.activeSystemInfoKey !== '') {
                this.startSystemInfoUpdates()
            }
        }
    },
    methods: {
        openConnectionDialog(source = this.sshInfo) {
            this.draftSSHInfo = createSSHInfo(source)
            this.resetConnectionCheckState()
            this.editingSSHId = ''
            this.connectionDialogVisible = true
            this.$nextTick(() => {
                if (this.$refs.sshForm) {
                    this.$refs.sshForm.clearValidate()
                }
            })
        },
        resetConnectionCheckState() {
            this.connectionCheckLoading = false
            this.saveAndConnectLoading = false
            this.connectionCheckResult = null
        },
        buildDraftSSHPayload(source = this.draftSSHInfo) {
            return {
                host: this.trimOrEmpty(source.host),
                ip: this.trimOrEmpty(source.ip),
                countryFlag: this.trimOrEmpty(source.countryFlag),
                location: this.trimOrEmpty(source.location),
                os: this.trimOrEmpty(source.os),
                memory: this.trimOrEmpty(source.memory),
                expiresAt: this.trimOrEmpty(source.expiresAt),
                username: this.trimOrEmpty(source.username),
                port: Number(source.port) || 0,
                password: source.password || '',
                logintype: source.logintype === undefined ? 0 : Number(source.logintype)
            }
        },
        buildCheckPayload(source = this.draftSSHInfo, options = {}) {
            const payload = this.buildDraftSSHPayload(source)
            return {
                host: payload.host,
                username: payload.username,
                port: payload.port,
                password: payload.password,
                logintype: payload.logintype,
                probeOnly: Boolean(options.probeOnly)
            }
        },
        async validateSSHForm() {
            if (!this.$refs.sshForm) {
                return false
            }
            return new Promise((resolve) => {
                this.$refs.sshForm.validate((valid) => {
                    resolve(valid)
                })
            })
        },
        connectDraftSSH() {
            this.$store.commit('SET_SSH', createSSHInfo(this.draftSSHInfo))
            this.$store.commit('CLEAR_SYSTEM_INFO')
            this.connectionDialogVisible = false
            this.$emit('ssh-select')
        },
        trimOrEmpty(value) {
            if (value === undefined || value === null) {
                return ''
            }
            return String(value).trim()
        },
        formatMemorySizeMB(value) {
            const parsed = Number(value)
            if (!Number.isFinite(parsed) || parsed <= 0) {
                return ''
            }
            if (parsed >= 1024) {
                const gb = parsed / 1024
                return `${gb % 1 === 0 ? gb.toFixed(0) : gb.toFixed(1)} GB`
            }
            return `${parsed.toFixed(0)} MB`
        },
        isIPAddress(value) {
            const normalized = this.trimOrEmpty(value)
            return /^(?:\d{1,3}\.){3}\d{1,3}$/.test(normalized) || normalized.includes(':')
        },
        buildConnectionMemory(info = {}) {
            const total = this.formatMemorySizeMB(info.memoryTotalMB)
            const cpuCores = Number(info.cpuCores)
            if (cpuCores > 0 && total !== '') {
                return `${cpuCores}C / ${total}`
            }
            if (cpuCores > 0) {
                return `${cpuCores}C`
            }
            return total
        },
        resolveDisplayIP(info = {}) {
            const ip = this.trimOrEmpty(info.ip)
            if (ip !== '') {
                return ip
            }
            const host = this.trimOrEmpty(info.host)
            return this.isIPAddress(host) ? host : ''
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
        buildServerInfoChips(info = {}) {
            const chips = []
            const ip = this.formatServerIP(info)
            const location = this.trimOrEmpty(info.location)
            const osName = this.trimOrEmpty(info.os)
            const memory = this.trimOrEmpty(info.memory)
            const expiresAt = this.formatExpiresAtLabel(info.expiresAt)

            if (ip !== '') {
                chips.push({ key: `ip-${ip}`, label: `${this.$t('ServerIP')}: ${ip}` })
            }
            if (location !== '') {
                chips.push({ key: `location-${location}`, label: `${this.$t('ServerLocation')}: ${location}` })
            }
            if (osName !== '') {
                chips.push({ key: `os-${osName}`, label: `${this.$t('OperatingSystem')}: ${osName}` })
            }
            if (memory !== '') {
                chips.push({ key: `memory-${memory}`, label: `${this.$t('Memory')}: ${memory}` })
            }
            chips.push({ key: `expires-${expiresAt}`, label: `${this.$t('ExpiresAt')}: ${expiresAt}` })
            return chips
        },
        authModeLabel(info = {}) {
            return this.$t(Number(info.logintype) === 1 ? 'privateKey' : 'password')
        },
        formatProbeError(error, fallback) {
            return this.trimOrEmpty(error && error.Msg) || this.trimOrEmpty(error && error.message) || fallback
        },
        showShortcuts() {
            if (this.$refs.keyboardShortcuts) {
                this.$refs.keyboardShortcuts.show()
            }
        },
        isPageHidden() {
            return typeof document !== 'undefined' && document.visibilityState === 'hidden'
        },
        handlePageVisibilityChange() {
            if (this.isPageHidden()) {
                this.stopSystemInfoUpdates()
                return
            }
            if (!this.connectionDialogVisible && this.activeSystemInfoKey !== '') {
                this.startSystemInfoUpdates()
            }
        },
        async probeSSH(payload) {
            const result = await checkSSHConnection(payload)
            if (result.Msg !== 'success' || !result.Data) {
                throw new Error(this.trimOrEmpty(result.Msg) || this.$t('connectionCheckFailed'))
            }
            return result.Data
        },
        applyDraftSSHMetadata(info = {}) {
            const resolved = this.buildResolvedSSHMetadata(this.draftSSHInfo, info)
            this.draftSSHInfo = createSSHInfo({
                ...this.draftSSHInfo,
                ...resolved
            })
        },
        formatConnectionCheckDescription(data = {}) {
            const info = data.info || {}
            const parts = []
            const latencyMs = Number(data.latencyMs)
            if (Number.isFinite(latencyMs) && latencyMs > 0) {
                parts.push(`${latencyMs} ms`)
            }
            const ip = this.trimOrEmpty(info.ip)
            if (ip !== '') {
                parts.push(ip)
            }
            const location = this.trimOrEmpty(info.location)
            if (location !== '') {
                parts.push(location)
            }
            const osName = this.trimOrEmpty(info.os)
            if (osName !== '') {
                parts.push(osName)
            }
            const memory = this.buildConnectionMemory(info)
            if (memory !== '') {
                parts.push(memory)
            }
            return parts.join(' · ') || this.$t('connectionCheckReady')
        },
        buildSavedSSHHealthSnapshotKey(list = this.savedSSHList) {
            return list.map(item => item.id || '').filter(Boolean).sort().join('|')
        },
        syncSavedSSHHealthMap(list = this.savedSSHList) {
            if (list.length === 0) {
                this.savedSSHHealthMap = {}
                this.savedSSHHealthSnapshotKey = ''
                return
            }
            const nextMap = {}
            list.forEach((item) => {
                if (this.savedSSHHealthMap[item.id]) {
                    nextMap[item.id] = this.savedSSHHealthMap[item.id]
                }
            })
            this.savedSSHHealthMap = nextMap
        },
        setSavedSSHHealth(item = {}, health = {}) {
            const id = item && item.id ? item.id : ''
            if (id === '') {
                return
            }
            this.$set(this.savedSSHHealthMap, id, {
                status: health.status || 'checking',
                message: health.message || this.$t('hostChecking')
            })
        },
        findDuplicateSavedSSH(source = this.draftSSHInfo, editingId = this.editingSSHId) {
            return this.savedSSHList.find(item => item.id !== editingId && isSameSSH(item, source)) || null
        },
        getSavedSSHHealth(item = {}) {
            const id = item.id || ''
            return this.savedSSHHealthMap[id] || {
                status: 'checking',
                message: this.$t('hostChecking')
            }
        },
        savedSSHHealthClass(item = {}) {
            const status = this.getSavedSSHHealth(item).status
            if (status === 'success') {
                return 'is-up'
            }
            if (status === 'error') {
                return 'is-down'
            }
            return 'is-checking'
        },
        savedSSHHealthTitle(item = {}) {
            return this.getSavedSSHHealth(item).message
        },
        async refreshSavedSSHHealth() {
            const list = this.savedSSHList.slice()
            if (list.length === 0) {
                this.syncSavedSSHHealthMap([])
                return
            }

            const snapshotKey = this.buildSavedSSHHealthSnapshotKey(list)
            const targets = list.filter((item) => {
                const health = this.savedSSHHealthMap[item.id]
                return !health || health.status === 'checking'
            })

            if (targets.length === 0 && this.savedSSHHealthSnapshotKey === snapshotKey) {
                return
            }

            const requestId = ++this.savedSSHHealthRequestId
            const nextMap = { ...this.savedSSHHealthMap }
            targets.forEach((item) => {
                nextMap[item.id] = {
                    status: 'checking',
                    message: this.$t('hostChecking')
                }
            })
            this.savedSSHHealthMap = nextMap
            if (targets.length === 0) {
                this.savedSSHHealthSnapshotKey = snapshotKey
                return
            }
            await runWithConcurrency(targets, SAVED_SSH_HEALTH_CONCURRENCY, async (item) => {
                let health
                if (this.trimOrEmpty(item.password) === '') {
                    health = {
                        status: 'error',
                        message: this.$t('hostMissingCredential')
                    }
                } else {
                    try {
                        await this.probeSSH(this.buildCheckPayload(item, { probeOnly: true }))
                        health = {
                            status: 'success',
                            message: this.$t('hostReachable')
                        }
                    } catch (error) {
                        health = {
                            status: 'error',
                            message: this.formatProbeError(error, this.$t('hostUnreachable'))
                        }
                    }
                }
                if (requestId === this.savedSSHHealthRequestId) {
                    this.setSavedSSHHealth(item, health)
                }
            })
            if (requestId === this.savedSSHHealthRequestId) {
                this.savedSSHHealthSnapshotKey = snapshotKey
            }
        },
        toggleLoginType() {
            this.draftSSHInfo.logintype = this.privateKey ? 0 : 1
            this.draftSSHInfo.password = ''
            this.resetConnectionCheckState()
            this.$nextTick(() => {
                if (this.$refs.sshForm) {
                    this.$refs.sshForm.clearValidate('password')
                }
            })
        },
        handleSetLanguage() {
            const oldLang = getLanguage()
            const lang = oldLang === 'zh' ? 'en' : 'zh'
            this.$i18n.locale = lang
            this.$store.dispatch('setLanguage', lang)
        },
        async handleCheckConnection() {
            const valid = await this.validateSSHForm()
            if (!valid) {
                return false
            }
            this.connectionCheckLoading = true
            this.connectionCheckResult = null
            try {
                const data = await this.probeSSH(this.buildCheckPayload())
                this.applyDraftSSHMetadata(data.info || {})
                this.connectionCheckResult = {
                    state: 'success',
                    title: this.$t('connectionCheckSuccess'),
                    description: this.formatConnectionCheckDescription(data)
                }
                this.$message.success(this.$t('connectionCheckSuccess'))
                return true
            } catch (error) {
                const message = this.formatProbeError(error, this.$t('connectionCheckFailed'))
                this.connectionCheckResult = {
                    state: 'error',
                    title: this.$t('connectionCheckFailed'),
                    description: message
                }
                this.$message.error(message)
                return false
            } finally {
                this.connectionCheckLoading = false
            }
        },
        applySSHCommand(command) {
            const normalized = createSSHInfo(command)
            this.draftSSHInfo = createSSHInfo(normalized)

            if (!normalized.password) {
                this.openConnectionDialog(normalized)
                return
            }
            this.$store.commit('SET_SSH', normalized)
            this.$store.commit('CLEAR_SYSTEM_INFO')
            this.connectionDialogVisible = false
            this.$emit('ssh-select')
        },
        handleSavedCommand(command) {
            this.applySSHCommand(command)
        },
        async fetchSavedSSHList() {
            try {
                const result = await listSavedSSH()
                if (result.Msg !== 'success') {
                    this.$message.error(result.Msg)
                    return
                }
                const nextList = result.Data && result.Data.list ? result.Data.list : []
                this.$store.commit('SET_SAVED_LIST', nextList)
                this.syncSavedSSHHealthMap(nextList)
                this.refreshSavedSSHHealth()
            } catch {
                // 错误提示由请求拦截器统一处理。
            }
        },
        editSavedSSHItem(item) {
            this.draftSSHInfo = createSSHInfo(item)
            this.resetConnectionCheckState()
            this.editingSSHId = item.id || ''
            this.connectionDialogVisible = true
            this.$nextTick(() => {
                if (this.$refs.sshForm) {
                    this.$refs.sshForm.clearValidate()
                }
            })
        },
        async persistDraftSSH(source = this.draftSSHInfo, options = {}) {
            const { showSuccess = true, knownHealth = null } = options
            const payload = this.buildDraftSSHPayload(source)
            const editingId = this.editingSSHId
            if (editingId) {
                payload.id = editingId
            }
            if (payload.host === '') {
                this.$message.warning(this.$t('saveSSHNeedHost'))
                return null
            }
            if (payload.username === '') {
                this.$message.warning(this.$t('saveSSHNeedUser'))
                return null
            }
            if (payload.port === 0) {
                this.$message.warning(this.$t('saveSSHNeedPort'))
                return null
            }

            const duplicate = this.findDuplicateSavedSSH(payload, editingId)
            if (duplicate) {
                if (knownHealth) {
                    this.setSavedSSHHealth(duplicate, knownHealth)
                    this.savedSSHHealthSnapshotKey = this.buildSavedSSHHealthSnapshotKey(this.savedSSHList)
                }
                this.$message.warning(this.$t('saveSSHExists'))
                return duplicate
            }

            try {
                const result = await saveSavedSSH(payload)
                if (result.Msg !== 'success') {
                    if (result.Msg === 'saved ssh already exists') {
                        const existing = result.Data && result.Data.profile ? result.Data.profile : null
                        if (existing && knownHealth) {
                            this.setSavedSSHHealth(existing, knownHealth)
                            this.savedSSHHealthSnapshotKey = this.buildSavedSSHHealthSnapshotKey(this.savedSSHList)
                        }
                        this.$message.warning(this.$t('saveSSHExists'))
                        return existing
                    }
                    this.$message.error(result.Msg)
                    return null
                }

                const nextList = result.Data && result.Data.list ? result.Data.list : []
                const savedProfile = result.Data && result.Data.profile ? result.Data.profile : payload
                this.$store.commit('SET_SAVED_LIST', nextList)
                this.syncSavedSSHHealthMap(nextList)
                if (showSuccess) {
                    this.$message.success(this.$t('saveSSHSuccess'))
                }
                if (result.Data && result.Data.profile && result.Data.profile.id) {
                    this.editingSSHId = result.Data.profile.id
                } else {
                    this.editingSSHId = ''
                }
                if (knownHealth) {
                    this.setSavedSSHHealth(savedProfile, knownHealth)
                    this.savedSSHHealthSnapshotKey = this.buildSavedSSHHealthSnapshotKey(nextList)
                } else if (savedProfile && savedProfile.id) {
                    this.setSavedSSHHealth(savedProfile)
                    this.savedSSHHealthSnapshotKey = ''
                    this.refreshSavedSSHHealth()
                }
                return savedProfile
            } catch {
                // 错误提示由请求拦截器统一处理。
                return null
            }
        },
        async handleSaveAndConnect() {
            const valid = await this.validateSSHForm()
            if (!valid) {
                return false
            }
            this.saveAndConnectLoading = true
            this.connectionCheckResult = null
            try {
                const data = await this.probeSSH(this.buildCheckPayload())
                this.applyDraftSSHMetadata(data.info || {})
                this.connectionCheckResult = {
                    state: 'success',
                    title: this.$t('connectionCheckSuccess'),
                    description: this.formatConnectionCheckDescription(data)
                }
                const draftToPersist = createSSHInfo(this.draftSSHInfo)
                this.connectDraftSSH()
                void this.persistDraftSSH(draftToPersist, {
                    knownHealth: {
                        status: 'success',
                        message: this.$t('hostReachable')
                    }
                })
                return true
            } catch (error) {
                const message = this.formatProbeError(error, this.$t('connectionCheckFailed'))
                this.connectionCheckResult = {
                    state: 'error',
                    title: this.$t('connectionCheckFailed'),
                    description: message
                }
                this.$message.error(message)
                return false
            } finally {
                this.saveAndConnectLoading = false
            }
        },
        async deleteSavedSSHItem(item) {
            try {
                const result = await deleteSavedSSH(item.id)
                if (result.Msg !== 'success') {
                    this.$message.error(result.Msg)
                    return
                }
                const nextList = result.Data && result.Data.list ? result.Data.list : []
                this.$store.commit('SET_SAVED_LIST', nextList)
                this.syncSavedSSHHealthMap(nextList)
                this.savedSSHHealthSnapshotKey = this.buildSavedSSHHealthSnapshotKey(nextList)
                this.$message.success(this.$t('deleteSSHSuccess'))
            } catch {
                // 错误提示由请求拦截器统一处理。
            }
        },
        handleChangePKFile(event) {
            const file = event.target.files[0]
            if (file) {
                const reader = new FileReader()
                reader.onload = e => {
                    this.draftSSHInfo.password = e.target.result
                    this.resetConnectionCheckState()
                }
                reader.readAsText(file)
            }
        },
        stopSystemInfoStream() {
            if (this.systemInfoSource !== null) {
                this.systemInfoSource.close()
                this.systemInfoSource = null
            }
            if (this.systemInfoReconnectTimer !== null) {
                window.clearTimeout(this.systemInfoReconnectTimer)
                this.systemInfoReconnectTimer = null
            }
        },
        stopSystemInfoUpdates() {
            this.stopSystemInfoStream()
            this.stopSystemInfoRefresh()
        },
        startSystemInfoUpdates() {
            this.stopSystemInfoUpdates()
            if (this.activeSystemInfoKey === '' || this.connectionDialogVisible || this.isPageHidden()) {
                return
            }
            if (typeof window !== 'undefined' && typeof window.EventSource === 'function') {
                this.openSystemInfoStream()
                return
            }
            this.refreshSystemInfo()
        },
        openSystemInfoStream(delay = 0) {
            this.stopSystemInfoStream()
            if (this.activeSystemInfoKey === '' || this.connectionDialogVisible || this.isPageHidden()) {
                return
            }
            const requestKey = this.activeSystemInfoKey
            const sshInfo = createSSHInfo(this.sshInfo)
            const requestIdentity = buildSSHIdentity(sshInfo)
            if (requestIdentity === '' || requestIdentity !== this.activeSSHIdentity) {
                return
            }

            const start = () => {
                if (requestKey !== this.activeSystemInfoKey || requestIdentity !== this.activeSSHIdentity || this.connectionDialogVisible || this.isPageHidden()) {
                    return
                }
                this.systemInfoLoading = true
                const source = new window.EventSource(buildSSHSystemStreamURL(encodeSSHInfo(sshInfo)))
                this.systemInfoSource = source

                source.onmessage = (event) => {
                    if (this.systemInfoSource !== source || requestKey !== this.activeSystemInfoKey || requestIdentity !== this.activeSSHIdentity) {
                        return
                    }
                    this.systemInfoLoading = false
                    try {
                        const payload = JSON.parse(event.data)
                        if (payload.Msg !== 'success' || !payload.Data || !payload.Data.info) {
                            throw new Error(this.trimOrEmpty(payload.Msg) || this.$t('systemInfoUnavailable'))
                        }
                        this.systemInfoFailureCount = 0
                        this.$store.commit('SET_SYSTEM_INFO', payload.Data.info)
                        this.syncResolvedSSHMetadata(sshInfo, payload.Data.info)
                    } catch (error) {
                        this.systemInfoFailureCount += 1
                        if (!this.hasSystemMetrics) {
                            this.$store.commit('CLEAR_SYSTEM_INFO')
                        }
                    }
                }

                source.onerror = () => {
                    if (this.systemInfoSource !== source) {
                        return
                    }
                    source.close()
                    this.systemInfoSource = null
                    this.systemInfoFailureCount += 1
                    this.systemInfoLoading = false
                    if (requestKey !== this.activeSystemInfoKey || requestIdentity !== this.activeSSHIdentity || this.connectionDialogVisible || this.isPageHidden()) {
                        return
                    }
                    const retryDelay = Math.min(SYSTEM_INFO_RETRY_MAX_INTERVAL, SYSTEM_INFO_REFRESH_INTERVAL * Math.pow(2, Math.max(0, this.systemInfoFailureCount - 1)))
                    this.systemInfoReconnectTimer = window.setTimeout(() => {
                        this.systemInfoReconnectTimer = null
                        this.openSystemInfoStream()
                    }, retryDelay)
                }
            }

            if (delay > 0) {
                this.systemInfoReconnectTimer = window.setTimeout(() => {
                    this.systemInfoReconnectTimer = null
                    start()
                }, delay)
                return
            }
            start()
        },
        scheduleSystemInfoRefresh(delay = SYSTEM_INFO_REFRESH_INTERVAL) {
            this.stopSystemInfoRefresh()
            if (this.activeSystemInfoKey === '' || this.connectionDialogVisible || this.isPageHidden()) {
                return
            }
            this.systemInfoTimer = window.setTimeout(() => {
                this.refreshSystemInfo()
            }, delay)
        },
        stopSystemInfoRefresh() {
            if (this.systemInfoTimer !== null) {
                window.clearTimeout(this.systemInfoTimer)
                this.systemInfoTimer = null
            }
        },
        // 用最近一次采集到的系统信息回填主机元数据，避免列表信息长期陈旧。
        buildResolvedSSHMetadata(baseInfo, systemInfo) {
            const base = createSSHInfo(baseInfo)
            return {
                ip: this.trimOrEmpty(systemInfo.ip) || this.resolveDisplayIP(base),
                countryFlag: this.trimOrEmpty(systemInfo.countryFlag) || this.trimOrEmpty(base.countryFlag),
                location: this.trimOrEmpty(systemInfo.location) || this.trimOrEmpty(base.location),
                os: this.trimOrEmpty(systemInfo.os) || this.trimOrEmpty(base.os),
                memory: this.buildConnectionMemory(systemInfo) || this.trimOrEmpty(base.memory),
                expiresAt: this.trimOrEmpty(base.expiresAt)
            }
        },
        hydrateSSHMetadataFromCollections(baseInfo) {
            const current = createSSHInfo(baseInfo)
            const matches = [...this.sshList, ...this.savedSSHList]
            const existing = matches.find(item => isSameSSH(item, current))
            if (!existing) {
                return
            }
            const metadata = {
                ip: this.trimOrEmpty(existing.ip),
                countryFlag: this.trimOrEmpty(existing.countryFlag),
                location: this.trimOrEmpty(existing.location),
                os: this.trimOrEmpty(existing.os),
                memory: this.trimOrEmpty(existing.memory),
                expiresAt: this.trimOrEmpty(existing.expiresAt)
            }
            if (!this.hasServerMetadataChange(current, metadata)) {
                return
            }
            this.$store.commit('SET_SSH', {
                ...current,
                ...metadata
            })
        },
        hasServerMetadataChange(source = {}, metadata = {}) {
            return ['ip', 'countryFlag', 'location', 'os', 'memory', 'expiresAt'].some(key => this.trimOrEmpty(source[key]) !== this.trimOrEmpty(metadata[key]))
        },
        syncHistoryMetadata(baseInfo, metadata) {
            let changed = false
            const nextList = this.sshList.map(item => {
                if (!isSameSSH(item, baseInfo)) {
                    return item
                }
                const nextItem = {
                    ...item,
                    ...metadata
                }
                changed = changed || this.hasServerMetadataChange(item, metadata)
                return nextItem
            })
            if (changed) {
                this.$store.commit('SET_LIST', encodeSSHList(nextList))
            }
        },
        async syncSavedSSHMetadata(baseInfo, metadata) {
            let changed = false
            const nextList = this.savedSSHList.map(item => {
                if (!isSameSSH(item, baseInfo)) {
                    return item
                }
                const nextItem = {
                    ...item,
                    ...metadata
                }
                changed = changed || this.hasServerMetadataChange(item, metadata)
                return nextItem
            })
            if (!changed) {
                return
            }
            this.$store.commit('SET_SAVED_LIST', nextList)
            const target = nextList.find(item => isSameSSH(item, baseInfo))
            if (!target) {
                return
            }
            try {
                const result = await saveSavedSSH(target)
                if (result.Msg === 'success' && result.Data && result.Data.list) {
                    this.$store.commit('SET_SAVED_LIST', result.Data.list)
                }
            } catch {
                // 后台回写失败不影响当前终端会话。
            }
        },
        syncResolvedSSHMetadata(baseInfo, info) {
            const metadata = this.buildResolvedSSHMetadata(baseInfo, info)
            if (!this.hasServerMetadataChange(baseInfo, metadata)) {
                return
            }
            const current = createSSHInfo(this.sshInfo)
            if (buildSSHIdentity(current) === buildSSHIdentity(baseInfo)) {
                this.$store.commit('SET_SSH', {
                    ...current,
                    ...metadata
                })
            }
            this.syncHistoryMetadata(baseInfo, metadata)
            this.syncSavedSSHMetadata(baseInfo, metadata)
        },
        async refreshSystemInfo() {
            if (this.activeSystemInfoKey === '' || this.connectionDialogVisible || this.systemInfoLoading || this.isPageHidden()) {
                return
            }
            const sshInfo = createSSHInfo(this.sshInfo)
            const requestIdentity = buildSSHIdentity(sshInfo)
            const requestKey = this.activeSystemInfoKey
            if (requestIdentity === '' || requestKey === '' || requestIdentity !== this.activeSSHIdentity) {
                return
            }

            this.systemInfoLoading = true
            const requestId = ++this.systemInfoRequestId

            try {
                const result = await getSSHSystemInfo(encodeSSHInfo(sshInfo))
                if (requestId !== this.systemInfoRequestId || requestKey !== this.activeSystemInfoKey || requestIdentity !== this.activeSSHIdentity) {
                    return
                }
                if (result.Msg !== 'success' || !result.Data || !result.Data.info) {
                    throw new Error(this.trimOrEmpty(result.Msg) || this.$t('systemInfoUnavailable'))
                }
                this.systemInfoFailureCount = 0
                this.$store.commit('SET_SYSTEM_INFO', result.Data.info)
                this.syncResolvedSSHMetadata(sshInfo, result.Data.info)
            } catch (error) {
                if (requestId !== this.systemInfoRequestId || requestKey !== this.activeSystemInfoKey || requestIdentity !== this.activeSSHIdentity) {
                    return
                }
                if (!this.hasSystemMetrics) {
                    this.$store.commit('CLEAR_SYSTEM_INFO')
                }
                this.systemInfoFailureCount += 1
            } finally {
                if (requestId === this.systemInfoRequestId && requestKey === this.activeSystemInfoKey && requestIdentity === this.activeSSHIdentity) {
                    this.systemInfoLoading = false
                    const retryDelay = this.systemInfoFailureCount === 0
                        ? SYSTEM_INFO_REFRESH_INTERVAL
                        : Math.min(SYSTEM_INFO_RETRY_MAX_INTERVAL, SYSTEM_INFO_REFRESH_INTERVAL * Math.pow(2, this.systemInfoFailureCount))
                    this.scheduleSystemInfoRefresh(retryDelay)
                }
            }
        }
    },
    mounted() {
        this.fetchSavedSSHList()
        if (this.sshList.length > 0) {
            this.draftSSHInfo = createSSHInfo(this.sshList[this.sshList.length - 1])
        }
        this.pageVisibilityHandler = () => {
            this.handlePageVisibilityChange()
        }
        document.addEventListener('visibilitychange', this.pageVisibilityHandler)
    },
    beforeDestroy() {
        this.stopSystemInfoUpdates()
        this.systemInfoRequestId += 1
        if (this.pageVisibilityHandler !== null) {
            document.removeEventListener('visibilitychange', this.pageVisibilityHandler)
            this.pageVisibilityHandler = null
        }
        if (!this.hasActiveTerminal) {
            this.$store.commit('CLEAR_SYSTEM_INFO')
        }
    }
}
</script>

<style scoped lang="scss">
.header-container {
    padding: 10px 20px;
    background: transparent;
}

.nav-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
}

.brand {
    user-select: none;
    display: flex;
    align-items: center;
    gap: 8px;

    .brand-icon {
        font-size: 18px;
        line-height: 1;
    }

    .logo {
        font-size: 20px;
        color: #64748b;
        font-weight: 300;
        letter-spacing: -0.3px;
    }

    .logo-bold {
        font-size: 20px;
        color: #3b82f6;
        font-weight: 800;
        letter-spacing: -0.5px;
    }

    .brand-badge {
        font-size: 10px;
        font-weight: 600;
        color: #ffffff;
        background: linear-gradient(135deg, #3b82f6, #7c3aed);
        padding: 3px 10px;
        border-radius: 20px;
        letter-spacing: 0.03em;
        white-space: nowrap;
        box-shadow: 0 2px 6px rgba(124, 58, 237, 0.3);
    }
}

.nav-actions {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
    justify-content: flex-end;
}

.connect-btn {
    font-weight: 600 !important;
    letter-spacing: 0.01em;
    border-radius: 10px !important;
    padding: 8px 16px !important;
}

.icon-btn {
    font-weight: 500 !important;
    border-radius: 10px !important;
}

.lang-btn {
    width: 32px !important;
    height: 32px !important;
    padding: 0 !important;
    font-size: 16px !important;
    color: #64748b !important;
    border-radius: 50% !important;
    flex-shrink: 0;

    &:hover {
        color: #3b82f6 !important;
        background: rgba(59, 130, 246, 0.08) !important;
        border-color: rgba(59, 130, 246, 0.2) !important;
    }
}

.auth-section {
    margin-top: 12px;
    padding: 16px;
    background: rgba(248, 250, 252, 0.7);
    border-radius: 12px;
    border: 1.5px solid rgba(226, 232, 240, 0.8);

    ::v-deep .el-textarea__inner {
        font-family: 'JetBrains Mono', 'Fira Code', 'SF Mono', Consolas, monospace;
        font-size: 13px;
        line-height: 1.6;
        background: #ffffff;
        border: 1.5px solid #cbd5e1;
        border-radius: 8px;
        padding: 12px;
        color: #1e293b;

        &:focus {
            border-color: #3b82f6;
            box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
        }
    }
}

.full-width-input {
    width: 100% !important;
}

::v-deep .el-date-editor.el-input,
::v-deep .el-date-editor.el-input__inner {
    width: 100% !important;
}

::v-deep .el-date-editor .el-input__inner {
    border: 1px solid #dcdfe6;
    border-radius: 4px;
    background-color: #fff;
    color: #606266;
}

.field-hint {
    margin-top: 6px;
    font-size: 12px;
    color: #94a3b8;
    line-height: 1.5;
}

.auth-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 14px;
}

.section-title {
    font-size: 13px;
    font-weight: 700;
    color: #334155;
    display: flex;
    align-items: center;
    gap: 8px;

    &::before {
        content: '';
        width: 3px;
        height: 14px;
        background: linear-gradient(135deg, #3b82f6, #7c3aed);
        border-radius: 2px;
        flex-shrink: 0;
    }
}

.toggle-auth-btn {
    font-size: 12px;
    font-weight: 600;
    color: #3b82f6;
    background: rgba(59, 130, 246, 0.08) !important;
    padding: 5px 12px !important;
    border-radius: 20px !important;
    border: 1px solid rgba(59, 130, 246, 0.2) !important;

    &:hover {
        background: linear-gradient(135deg, #3b82f6, #2563eb) !important;
        border-color: #3b82f6 !important;
        color: #ffffff !important;
        transform: none !important;
        box-shadow: 0 4px 10px rgba(59, 130, 246, 0.3) !important;
    }
}

.key-actions {
    margin-top: 12px;
    display: flex;
    justify-content: flex-end;
}

.premium-upload-btn {
    border-radius: 8px;
    font-weight: 600;
}

.hidden-file-input {
    display: none;
}

.premium-dropdown {
    min-width: 340px;
    padding: 8px !important;
    border-radius: 14px !important;
    border: 1px solid rgba(226, 232, 240, 0.9) !important;
    box-shadow: 0 16px 40px -8px rgba(0, 0, 0, 0.12), 0 6px 16px -4px rgba(0, 0, 0, 0.06) !important;
    backdrop-filter: blur(12px);
}

.empty-dropdown {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 28px 20px;
    color: #94a3b8;

    i {
        font-size: 28px;
        margin-bottom: 8px;
        opacity: 0.6;
    }

    p {
        font-size: 13px;
        margin: 0;
        font-weight: 500;
    }
}

.dropdown-item-content {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    width: 100%;
    padding: 2px 4px;
}

.item-avatar {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    background: linear-gradient(135deg, #3b82f6, #7c3aed);
    color: #ffffff;
    font-size: 13px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    margin-top: 2px;
}

.item-info-dropdown {
    display: flex;
    flex-direction: column;
    gap: 2px;
    line-height: 1.4;
    flex: 1;
    min-width: 0;

    .item-host {
        font-size: 13px;
        font-weight: 600;
        color: #0f172a;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .item-user {
        font-size: 11px;
        color: #64748b;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
}

.item-host-row {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
}

.item-status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
    background: #cbd5e1;
    box-shadow: 0 0 0 4px rgba(203, 213, 225, 0.18);

    &.is-up {
        background: #22c55e;
        box-shadow: 0 0 0 4px rgba(34, 197, 94, 0.16);
    }

    &.is-down {
        background: #ef4444;
        box-shadow: 0 0 0 4px rgba(239, 68, 68, 0.14);
    }

    &.is-checking {
        background: #f59e0b;
        box-shadow: 0 0 0 4px rgba(245, 158, 11, 0.14);
    }
}

.connection-check-alert {
    margin-top: 14px;
    border-radius: 12px;
}

.item-meta-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 3px;
}

.info-chip {
    display: inline-flex;
    align-items: center;
    max-width: 100%;
    padding: 3px 8px;
    border-radius: 999px;
    background: #eff6ff;
    border: 1px solid rgba(59, 130, 246, 0.12);
    color: #2563eb;
    font-size: 10px;
    font-weight: 600;
    line-height: 1.3;
}

.delete-btn {
    color: #cbd5e1;
    font-size: 14px;
    padding: 5px;
    transition: all 0.2s;
    border-radius: 6px;
    flex-shrink: 0;

    &:hover {
        color: #ef4444;
        background: #fee2e2;
    }
}

.edit-btn {
    color: #cbd5e1;
    font-size: 14px;
    padding: 5px;
    transition: all 0.2s;
    border-radius: 6px;
    flex-shrink: 0;
    cursor: pointer;

    &:hover {
        color: #3b82f6;
        background: #eff6ff;
    }
}

::v-deep .premium-dialog {
    border-radius: 18px;
    overflow: hidden;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.2);
    margin-top: 5vh !important;
    max-height: 90vh;
    display: flex;
    flex-direction: column;

    .el-dialog__header {
        background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
        border-bottom: 1px solid #e2e8f0;
        padding: 18px 24px;
        flex-shrink: 0;

        .el-dialog__title {
            font-weight: 700;
            font-size: 16px;
            color: #0f172a;
        }
    }

    .el-dialog__body {
        padding: 16px 24px 20px;
        flex: 1;
        overflow-y: auto;
        min-height: 0;
    }

    .el-form-item {
        margin-bottom: 16px;
    }
    
    .el-form-item__label {
        padding-bottom: 4px !important;
        line-height: 1.2 !important;
    }

    .el-dialog__footer {
        padding: 14px 24px;
        background: #f8fafc;
        border-top: 1px solid #e2e8f0;
        flex-shrink: 0;
    }
}

@media (max-width: 900px) {
    .header-container {
        padding: 10px 14px;
    }

    .nav-bar {
        flex-direction: column;
        align-items: stretch;
    }

    .nav-actions {
        justify-content: flex-start;
    }
}
</style>
