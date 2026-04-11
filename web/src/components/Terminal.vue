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
            savePass: false,
            fontSize: 15,
            resizeHandler: null,
            wheelEventName: '',
            wheelHandler: null
        }
    },
    mounted() {
        this.createTerm()
    },
    methods: {
        setSSH() {
            this.$store.commit('SET_SSH', this.ssh)
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
        sendResize() {
            if (this.ws !== null && this.ws.readyState === 1 && this.term !== null) {
                this.ws.send(`resize:${this.term.rows}:${this.term.cols}`)
            }
        },
        activate() {
            const termWeb = document.getElementById(this.id)
            if (!termWeb || this.term === null || this.fitAddon === null) {
                return
            }
            this.$nextTick(() => {
                this.resizeTerm(termWeb)
                this.fitTerminal()
                this.sendResize()
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
            const termWeb = document.getElementById(this.id)
            if (!termWeb) {
                return
            }
            this.resizeTerm(termWeb)
            const sshReq = this.$store.getters.sshReq
            this.close()
            const prefix = process.env.NODE_ENV === 'production' ? '' : '/ws'
            this.fitAddon = new FitAddon()
            this.term = new Terminal({
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
            const self = this
            const heartCheck = {
                timeout: 5000,
                intervalObj: null,
                stop: function() {
                    clearInterval(this.intervalObj)
                },
                start: function() {
                    this.intervalObj = setInterval(function() {
                        if (self.ws !== null && self.ws.readyState === 1) {
                            self.ws.send('ping')
                        }
                    }, this.timeout)
                }
            }
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
            this.ws = new WebSocket(`${(location.protocol === 'http:' ? 'ws' : 'wss')}://${location.host}${prefix}/term?${wsQuery}`)
            this.ws.onopen = () => {
                self.connected()
                heartCheck.start()
            }
            this.ws.onclose = () => {
                if (!self.resetClose) {
                    if (!this.savePass) {
                        this.$store.commit('SET_PASS', '')
                        if (this.ssh !== null) {
                            this.ssh.password = ''
                        }
                    }
                    this.$message({
                        message: this.$t('wsClose'),
                        type: 'warning',
                        duration: 0,
                        showClose: true
                    })
                    this.ws = null
                }
                heartCheck.stop()
                self.resetClose = false
            }
            const attachAddon = new AttachAddon(this.ws)
            this.term.loadAddon(attachAddon)
            this.term.attachCustomKeyEventHandler((e) => {
                const keyArray = ['F5', 'F11', 'F12']
                if (keyArray.indexOf(e.key) > -1) {
                    return false
                }
                if (e.ctrlKey && e.key === 'v') {
                    document.execCommand('copy')
                    return false
                }
                if (e.ctrlKey && e.key === 'c' && self.term.hasSelection()) {
                    document.execCommand('copy')
                    return false
                }
            })
            this.wheelEventName = 'onwheel' in document.createElement('div') ? 'wheel' : document.onmousewheel !== undefined ? 'mousewheel' : 'DOMMouseScroll'
            this.wheelHandler = (e) => {
                if (e.ctrlKey) {
                    e.preventDefault()
                    if (e.deltaY < 0) {
                        self.term.setOption('fontSize', ++this.fontSize)
                    } else {
                        self.term.setOption('fontSize', --this.fontSize)
                    }
                    self.fitTerminal()
                    self.sendResize()
                }
            }
            termWeb.addEventListener(this.wheelEventName, this.wheelHandler)
            this.resizeHandler = () => {
                self.resizeTerm(termWeb)
                self.fitTerminal()
                self.sendResize()
            }
            window.addEventListener('resize', this.resizeHandler)
        },
        connected() {
            const sshInfo = this.$store.state.sshInfo
            this.ssh = Object.assign({}, sshInfo)
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
            if (this.ws !== null) {
                this.ws.close()
                this.resetClose = true
                this.ws = null
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
