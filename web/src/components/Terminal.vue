<template>
    <div class="terminal-shell">
        <div :id="id" class="terminal-host"></div>
    </div>
</template>

<script>
import 'xterm/css/xterm.css'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { TerminalTransport } from '@/utils/terminalTransport'
import { WebglAddon } from 'xterm-addon-webgl'
import { buildWsURL } from '@/utils/base'
import { buildHistoryItem, decodeSSHList, encodeSSHList, isSameSSH } from '@/utils/ssh'

export default {
    name: 'Terminal',
    props: ['id'],
    data() {
        return {
            resetClose: false,
            ssh: null,
            reconnectSSH: null,
            reconnectReady: false,
            disconnectMessage: null,
            savePass: false,
            fontSize: 15,
            resizeHandler: null,
            resizeSyncTimer: null,
            wheelEventName: '',
            wheelHandler: null,

        }
    },
    created() {
        // xterm and its renderer mutate large object graphs on every frame.
        // Keep runtime objects outside Vue 2 observation and DOM rendering.
        this.term = null
        this.fitAddon = null
        this.ws = null
        this.resizeObserver = null
        this.resizeFrame = null
        this.lastResizeDimensions = ''
        this.lastContainerSize = ''
        this.sessionId = ''
    },
    mounted() {
        this.createTerm()
    },
    methods: {
        setSSH() {
            const ssh = this.ssh || this.reconnectSSH
            if (ssh !== null) {
                this.$store.commit('SET_SSH', ssh)
                this.$store.commit('SET_CONNECTION_SESSION_ID', this.sessionId)
            }
        },
        fitTerminal() {
            if (this.fitAddon === null || !this.isVisible()) {
                return
            }
            try {
                this.fitAddon.fit()
            } catch (error) {
                // 容器尺寸尚未稳定时跳过一次 fit，后续 resize 会自动补齐。
            }
        },
        isVisible() {
            const host = document.getElementById(this.id)
            return Boolean(host && host.offsetWidth > 0 && host.offsetHeight > 0)
        },
        focusTerminal() {
            if (this.term !== null && this.isVisible()) {
                this.term.focus()
            }
        },
        closeDisconnectMessage() {
            if (this.disconnectMessage !== null && typeof this.disconnectMessage.close === 'function') {
                this.disconnectMessage.close()
            }
            this.disconnectMessage = null
        },
        copySelection() {
            if (this.term === null || !this.term.hasSelection()) {
                return
            }
            const selection = this.term.getSelection()
            if (navigator.clipboard && window.isSecureContext) {
                navigator.clipboard.writeText(selection).then(() => {
                    this.$message({
                        message: this.$t('copySuccess') || '已复制',
                        type: 'success',
                        duration: 1000,
                        showClose: false
                    })
                }).catch(() => {
                    const textarea = document.createElement('textarea')
                    textarea.value = selection
                    textarea.style.position = 'fixed'
                    textarea.style.opacity = '0'
                    document.body.appendChild(textarea)
                    textarea.select()
                    document.execCommand('copy')
                    document.body.removeChild(textarea)
                })
                return
            }
            const textarea = document.createElement('textarea')
            textarea.value = selection
            textarea.style.position = 'fixed'
            textarea.style.opacity = '0'
            document.body.appendChild(textarea)
            textarea.select()
            document.execCommand('copy')
            document.body.removeChild(textarea)
        },
        pasteClipboard() {
            if (!navigator.clipboard || !window.isSecureContext || this.term === null) {
                return false
            }
            navigator.clipboard.readText().then((text) => {
                if (text && this.term !== null) {
                    this.term.paste(text)
                }
            }).catch(() => {
                // 浏览器或权限拒绝时交给默认行为兜底。
            })
            return true
        },
        isMacCommandPaste(event) {
            const platform = navigator.userAgentData && Array.isArray(navigator.userAgentData.platform)
                ? navigator.userAgentData.platform.join(' ')
                : (navigator.platform || navigator.userAgent || '')
            return /mac|iphone|ipad|ipod/i.test(platform) && event.metaKey && !event.ctrlKey && !event.shiftKey
        },
        adjustFontSize(delta) {
            const nextSize = Math.max(11, Math.min(28, this.fontSize + delta))
            if (nextSize === this.fontSize || this.term === null) {
                return
            }
            this.fontSize = nextSize
            this.term.setOption('fontSize', this.fontSize)
            this.lastContainerSize = ''
            this.scheduleResizeSync(0)
        },
        sendResize() {
            if (this.ws !== null && this.ws.readyState === 1 && this.term !== null && this.isVisible()) {
                const dimensions = `${this.term.rows}:${this.term.cols}`
                if (dimensions !== this.lastResizeDimensions) {
                    this.ws.send(`resize:${dimensions}`)
                    this.lastResizeDimensions = dimensions
                }
            }
        },
        scheduleResizeSync(delay = 40) {
            if (this.resizeSyncTimer !== null || this.resizeFrame !== null) {
                return
            }
            this.resizeSyncTimer = window.setTimeout(() => {
                this.resizeSyncTimer = null
                this.resizeFrame = window.requestAnimationFrame(() => {
                    this.resizeFrame = null
                    const termWeb = document.getElementById(this.id)
                    if (!termWeb || this.term === null || !this.isVisible()) {
                        return
                    }
                    const rect = termWeb.parentElement.getBoundingClientRect()
                    const size = `${Math.floor(rect.width)}:${Math.floor(rect.height)}`
                    if (size !== this.lastContainerSize) {
                        this.lastContainerSize = size
                        this.resizeTerm(termWeb)
                        this.fitTerminal()
                    }
                    this.sendResize()
                })
            }, delay)
        },
        deactivate() {
            if (this.term !== null) {
                this.term.setOption('cursorBlink', false)
            }
        },
        activate() {
            const termWeb = document.getElementById(this.id)
            if (!termWeb || this.term === null || this.fitAddon === null) {
                return
            }
            this.term.setOption('cursorBlink', true)
            this.$nextTick(() => {
                this.scheduleResizeSync(0)
                this.focusTerminal()
            })
        },
        resizeTerm(termWeb) {
            const container = termWeb.parentElement
            if (container === null) {
                return
            }
            const height = Math.max(0, Math.floor(container.getBoundingClientRect().height))
            if (termWeb.style.height !== `${height}px`) {
                termWeb.style.height = `${height}px`
            }
        },
        createTerm() {
            if (this.$store.state.sshInfo.password === '') {
                return
            }
            const sshSnapshot = Object.assign({}, this.$store.state.sshInfo)
            const termWeb = document.getElementById(this.id)
            if (!termWeb) {
                return
            }
            this.resizeTerm(termWeb)
            const sshReq = this.$store.getters.sshReq
            const sessionId = this.$store.getters.sshSessionId
            this.close()
            this.sessionId = sessionId
            this.ssh = sshSnapshot
            this.reconnectSSH = sshSnapshot
            this.reconnectReady = false
            this.fitAddon = new FitAddon()
            this.term = new Terminal({
                fontFamily: "'JetBrains Mono', 'Fira Code', 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace",
                fontSize: this.fontSize,
                lineHeight: 1.18,
                scrollback: 10000,
                cursorBlink: this.isVisible(),
                convertEol: true,
                fastScrollModifier: 'shift',
                fastScrollSensitivity: 5,
                theme: {
                    background: '#0f172a',
                    foreground: '#e2e8f0',
                    cursor: '#f8fafc',
                    cursorAccent: '#0f172a',
                    selectionBackground: 'rgba(148, 163, 184, 0.28)',
                    black: '#0f172a',
                    red: '#f87171',
                    green: '#4ade80',
                    yellow: '#fbbf24',
                    blue: '#93c5fd',
                    magenta: '#d8b4fe',
                    cyan: '#67e8f9',
                    white: '#e2e8f0',
                    brightBlack: '#475569',
                    brightRed: '#fca5a5',
                    brightGreen: '#86efac',
                    brightYellow: '#fcd34d',
                    brightBlue: '#bfdbfe',
                    brightMagenta: '#e9d5ff',
                    brightCyan: '#a5f3fc',
                    brightWhite: '#f8fafc'
                }
            })
            this.term.loadAddon(this.fitAddon)
            this.term.open(document.getElementById(this.id))
            try {
                const webgl = new WebglAddon()
                webgl.onContextLoss(() => { webgl.dispose() })
                this.term.loadAddon(webgl)
            } catch (e) {
                // WebGL unavailable, fall back to canvas renderer
            }
            this.fitTerminal()
            this.focusTerminal()
            const self = this
            let closeTip = '已超时关闭!'
            if (this.$store.state.language === 'en') {
                closeTip = 'Connection timed out!'
            }
            const wsParams = {
                rows: this.term.rows,
                cols: this.term.cols,
                closeTip,
                flowControl: 1
            }
            if (sessionId) {
                wsParams.sessionId = sessionId
            } else {
                wsParams.sshInfo = sshReq
            }
            const token = localStorage.getItem('webssh_token')
            if (token) {
                wsParams.token = token
            }
            const wsUrl = buildWsURL('/term', wsParams)

            this.ws = new WebSocket(wsUrl)
            this.ws.binaryType = 'arraybuffer'
            const activeWs = this.ws
            this.ws.onopen = () => {
                if (self.ws !== activeWs) {
                    return
                }
                self.connected(sshSnapshot)
                self.sendResize()
                self.focusTerminal()
            }
            this.ws.onclose = () => {
                if (self.ws !== activeWs) {
                    return
                }
                if (!self.resetClose) {
                    this.ws = null
                    self.onFinalDisconnect()
                }
                self.resetClose = false
            }
            this.term.loadAddon(new TerminalTransport(this.ws))
            this.term.attachCustomKeyEventHandler((e) => {
                if (self.reconnectReady) {
                    if (e.type === 'keydown' && e.key === 'Enter') {
                        self.reconnect()
                    }
                    return false
                }
                const keyArray = ['F5', 'F11', 'F12']
                if (keyArray.indexOf(e.key) > -1) {
                    return false
                }
                const lowerKey = String(e.key || '').toLowerCase()
                const accelKey = e.ctrlKey || e.metaKey
                const isKeydown = e.type === 'keydown'
                if (lowerKey === 'v' && self.isMacCommandPaste(e)) {
                    return true
                }
                if (accelKey && lowerKey === 'v') {
                    if (!isKeydown) {
                        return false
                    }
                    if (self.pasteClipboard()) {
                        return false
                    }
                    return true
                }
                if (accelKey && lowerKey === 'c' && self.term.hasSelection()) {
                    if (!isKeydown) {
                        return false
                    }
                    self.copySelection()
                    return false
                }
                if ((e.ctrlKey || e.metaKey) && e.shiftKey && lowerKey === 'v') {
                    if (!isKeydown) {
                        return false
                    }
                    if (self.pasteClipboard()) {
                        return false
                    }
                    return true
                }
                if ((e.ctrlKey || e.metaKey) && e.shiftKey && lowerKey === 'c' && self.term.hasSelection()) {
                    if (!isKeydown) {
                        return false
                    }
                    self.copySelection()
                    return false
                }
                if ((e.ctrlKey || e.metaKey) && lowerKey === '=' && isKeydown) {
                    e.preventDefault()
                    self.adjustFontSize(1)
                    return false
                }
                if ((e.ctrlKey || e.metaKey) && lowerKey === '-' && isKeydown) {
                    e.preventDefault()
                    self.adjustFontSize(-1)
                    return false
                }
                if ((e.ctrlKey || e.metaKey) && lowerKey === '0' && isKeydown) {
                    e.preventDefault()
                    self.fontSize = 15
                    self.term.setOption('fontSize', self.fontSize)
                    self.lastContainerSize = ''
                    self.scheduleResizeSync(0)
                    return false
                }
                if ((e.ctrlKey || e.metaKey) && e.shiftKey && lowerKey === 'l' && isKeydown) {
                    e.preventDefault()
                    if (self.term !== null) {
                        self.term.clear()
                    }
                    return false
                }
            })
            this.wheelEventName = 'onwheel' in document.createElement('div') ? 'wheel' : document.onmousewheel !== undefined ? 'mousewheel' : 'DOMMouseScroll'
            this.wheelHandler = (e) => {
                if (e.ctrlKey) {
                    e.preventDefault()
                    self.adjustFontSize(e.deltaY < 0 ? 1 : -1)
                }
            }
            termWeb.addEventListener(this.wheelEventName, this.wheelHandler)
            this.resizeHandler = () => {
                self.scheduleResizeSync()
            }
            window.addEventListener('resize', this.resizeHandler)
            if (typeof window.ResizeObserver === 'function') {
                this.resizeObserver = new window.ResizeObserver(this.resizeHandler)
                this.resizeObserver.observe(termWeb.parentElement)
            }
        },
        hasReconnectSSH() {
            return this.reconnectSSH !== null && this.reconnectSSH.password !== undefined && this.reconnectSSH.password !== ''
        },
        onFinalDisconnect() {
            if (!this.savePass) {
                const activeSSH = this.$store.state.sshInfo || {}
                const closedSSH = this.ssh || this.reconnectSSH
                if (closedSSH !== null && isSameSSH(activeSSH, closedSSH)) {
                    this.$store.commit('SET_PASS', '')
                }
            }
            this.closeDisconnectMessage()
            this.disconnectMessage = this.$message({
                message: this.$t('wsClose'),
                type: 'warning',
                duration: 0,
                showClose: true
            })
            this.reconnectReady = true
            this.writeReconnectPrompt()
        },
        writeReconnectPrompt() {
            if (this.term === null) {
                return
            }
            const prompt = this.hasReconnectSSH() ? this.$t('terminalPressEnterReconnect') : this.$t('terminalReconnectNeedCredential')
            this.term.write(`\r\n\u001B[33m${this.$t('wsClose')}\u001B[0m\r\n${prompt}\r\n`)
        },
        reconnect() {
            if (!this.hasReconnectSSH()) {
                this.writeReconnectPrompt()
                return
            }
            this.reconnectReady = false
            this.$store.commit('SET_SSH', this.reconnectSSH)
            if (this.term !== null) {
                this.term.write(`\r\n\u001B[36m${this.$t('terminalReconnecting')}\u001B[0m\r\n`)
            }
            this.$nextTick(() => {
                this.createTerm()
            })
        },
        connected(sshInfo = this.ssh || this.reconnectSSH) {
            this.ssh = Object.assign({}, sshInfo)
            this.reconnectSSH = Object.assign({}, sshInfo)
            this.reconnectReady = false
            this.closeDisconnectMessage()
            this.savePass = this.$store.state.savePass
            if (this.isVisible()) {
                document.title = sshInfo.host ? `${sshInfo.host} - WebSSH` : 'WebSSH'
            }
            const sshList = decodeSSHList(this.$store.state.sshList)
                .filter(item => !isSameSSH(item, sshInfo))
            sshList.push(buildHistoryItem(sshInfo, this.savePass))
            this.$store.commit('SET_LIST', encodeSSHList(sshList))
        },
        close() {
            if (this.resizeObserver !== null) {
                this.resizeObserver.disconnect()
                this.resizeObserver = null
            }
            if (this.resizeFrame !== null) {
                window.cancelAnimationFrame(this.resizeFrame)
                this.resizeFrame = null
            }
            this.lastResizeDimensions = ''
            this.lastContainerSize = ''
            const termWeb = document.getElementById(this.id)
            if (termWeb !== null && this.wheelEventName !== '' && this.wheelHandler !== null) {
                termWeb.removeEventListener(this.wheelEventName, this.wheelHandler)
            }
            if (this.resizeHandler !== null) {
                window.removeEventListener('resize', this.resizeHandler)
                this.resizeHandler = null
            }
            this.wheelEventName = ''
            this.wheelHandler = null
            if (this.resizeSyncTimer !== null) {
                window.clearTimeout(this.resizeSyncTimer)
                this.resizeSyncTimer = null
            }
            this.reconnectReady = false
            this.closeDisconnectMessage()
            if (this.ws !== null) {
                const ws = this.ws
                this.resetClose = true
                this.ws = null
                ws.close()
                this.resetClose = false
            }
            if (this.term !== null) {
                this.term.dispose()
                this.term = null
            }
            this.fitAddon = null
        }
    },
    beforeDestroy() {
        this.close()
    }
}
</script>

<style scoped lang="scss">
.terminal-shell,
.terminal-host {
    width: 100%;
    height: 100%;
    background-color: #0f172a !important;
}
</style>
