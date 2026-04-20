<template>
    <div class="terminal-shell">
        <div :id="id" class="terminal-host"></div>
    </div>
</template>

<script>
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { AttachAddon } from 'xterm-addon-attach'
import { buildHistoryItem, decodeSSHList, encodeSSHList, isSameSSH } from '@/utils/ssh'

export default {
    name: 'Terminal',
    props: ['id'],
    data() {
        return {
            term: null,
            fitAddon: null,
            ws: null,
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
            wheelHandler: null
        }
    },
    mounted() {
        this.createTerm()
    },
    methods: {
        setSSH() {
            const ssh = this.ssh || this.reconnectSSH
            if (ssh !== null) {
                this.$store.commit('SET_SSH', ssh)
            }
        },
        fitTerminal() {
            if (this.fitAddon === null) {
                return
            }
            try {
                this.fitAddon.fit()
            } catch (error) {
                // 容器尺寸尚未稳定时跳过一次 fit，后续 resize 会自动补齐。
            }
        },
        focusTerminal() {
            if (this.term !== null) {
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
        adjustFontSize(delta) {
            const nextSize = Math.max(11, Math.min(28, this.fontSize + delta))
            if (nextSize === this.fontSize || this.term === null) {
                return
            }
            this.fontSize = nextSize
            this.term.setOption('fontSize', this.fontSize)
            this.scheduleResizeSync(0)
        },
        sendResize() {
            if (this.ws !== null && this.ws.readyState === 1 && this.term !== null) {
                this.ws.send(`resize:${this.term.rows}:${this.term.cols}`)
            }
        },
        scheduleResizeSync(delay = 60) {
            if (this.resizeSyncTimer !== null) {
                window.clearTimeout(this.resizeSyncTimer)
            }
            this.resizeSyncTimer = window.setTimeout(() => {
                this.resizeSyncTimer = null
                const termWeb = document.getElementById(this.id)
                if (!termWeb || this.term === null) {
                    return
                }
                this.resizeTerm(termWeb)
                this.fitTerminal()
                this.sendResize()
            }, delay)
        },
        activate() {
            const termWeb = document.getElementById(this.id)
            if (!termWeb || this.term === null || this.fitAddon === null) {
                return
            }
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
            const height = Math.max(260, Math.floor(container.getBoundingClientRect().height))
            termWeb.style.height = `${height}px`
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
            this.close()
            this.reconnectSSH = sshSnapshot
            this.reconnectReady = false
            const prefix = process.env.NODE_ENV === 'production' ? '' : '/ws'
            this.fitAddon = new FitAddon()
            this.term = new Terminal({
                fontFamily: "'JetBrains Mono', 'Fira Code', 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace",
                fontSize: this.fontSize,
                lineHeight: 1.18,
                scrollback: 10000,
                cursorBlink: true,
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
            this.fitTerminal()
            this.focusTerminal()
            const self = this
            let closeTip = '已超时关闭!'
            if (this.$store.state.language === 'en') {
                closeTip = 'Connection timed out!'
            }
            const wsParams = {
                sshInfo: sshReq,
                rows: this.term.rows,
                cols: this.term.cols,
                closeTip
            }
            const token = localStorage.getItem('webssh_token')
            if (token) {
                wsParams.token = token
            }
            const wsQuery = new URLSearchParams(wsParams).toString()
            const wsProtocol = location.protocol === 'http:' ? 'ws' : 'wss'
            const wsUrl = `${wsProtocol}://${location.host}${prefix}/term?${wsQuery}`

            this.ws = new WebSocket(wsUrl)
            this.ws.binaryType = 'arraybuffer'
            const activeWs = this.ws
            this.ws.onopen = () => {
                if (self.ws !== activeWs) {
                    return
                }
                self.connected()
                self.sendResize()
                self.focusTerminal()
                if (self.term !== null) {
                    self.term.write('\r\n\u001B[32m✓ 连接成功\u001B[0m\r\n')
                }
            }
            this.ws.onclose = () => {
                if (self.ws !== activeWs) {
                    return
                }
                if (!self.resetClose) {
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
                    this.ws = null
                }
                self.resetClose = false
            }
            const attachAddon = new AttachAddon(this.ws)
            this.term.loadAddon(attachAddon)
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
                if (accelKey && lowerKey === 'v') {
                    if (self.pasteClipboard()) {
                        return false
                    }
                    return true
                }
                if (accelKey && lowerKey === 'c' && self.term.hasSelection()) {
                    self.copySelection()
                    return false
                }
                if ((e.ctrlKey || e.metaKey) && e.shiftKey && lowerKey === 'v') {
                    if (self.pasteClipboard()) {
                        return false
                    }
                    return true
                }
                if ((e.ctrlKey || e.metaKey) && e.shiftKey && lowerKey === 'c' && self.term.hasSelection()) {
                    self.copySelection()
                    return false
                }
                if ((e.ctrlKey || e.metaKey) && lowerKey === '=' && e.type === 'keydown') {
                    e.preventDefault()
                    self.adjustFontSize(1)
                    return false
                }
                if ((e.ctrlKey || e.metaKey) && lowerKey === '-' && e.type === 'keydown') {
                    e.preventDefault()
                    self.adjustFontSize(-1)
                    return false
                }
                if ((e.ctrlKey || e.metaKey) && lowerKey === '0' && e.type === 'keydown') {
                    e.preventDefault()
                    self.fontSize = 15
                    self.term.setOption('fontSize', self.fontSize)
                    self.fitTerminal()
                    self.sendResize()
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
        },
        hasReconnectSSH() {
            return this.reconnectSSH !== null && this.reconnectSSH.password !== undefined && this.reconnectSSH.password !== ''
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
        connected() {
            const sshInfo = this.$store.state.sshInfo
            this.ssh = Object.assign({}, sshInfo)
            this.reconnectSSH = Object.assign({}, sshInfo)
            this.reconnectReady = false
            this.closeDisconnectMessage()
            this.savePass = this.$store.state.savePass
            document.title = sshInfo.host ? `${sshInfo.host} - WebSSH` : 'WebSSH'
            const sshList = decodeSSHList(this.$store.state.sshList)
                .filter(item => !isSameSSH(item, sshInfo))
            sshList.push(buildHistoryItem(sshInfo, this.savePass))
            this.$store.commit('SET_LIST', encodeSSHList(sshList))
        },
        close() {
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
