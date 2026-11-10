<template>
    <div class="tabs-wrapper">
        <el-tabs v-if="termList.length > 0" v-model="currentTerm" type="card" @tab-remove="removeTab" @tab-click="clickTab" class="custom-tabs">
            <el-tab-pane
                v-for="(item, index) in termList"
                :key="item.name"
                :name="item.name"
                :closable="item.closable"
            >
                <template slot="label">
                    <span class="tab-label" :title="item.label">
                        <span class="tab-dot"></span>
                        <span class="tab-text">{{ item.label }}</span>
                    </span>
                </template>
                <terminal :id="'Terminal' + index" :ref="item.name"></terminal>
            </el-tab-pane>
        </el-tabs>
        
        <div v-else class="empty-state">
            <div class="welcome-workspace">
                <div class="welcome-main">
                    <div class="empty-content">
                        <div class="workspace-eyebrow"><span></span> REMOTE WORKSPACE</div>
                        <h1 class="empty-title"><span>{{ $t('workspaceTitle') }}</span><span>{{ $t('workspaceTitleAccent') }}</span></h1>
                        <p class="empty-subtitle">{{ $t('workspaceDescription') }}</p>
                        <div class="welcome-actions">
                            <el-button type="primary" icon="el-icon-plus" @click="$emit('new-connection')">{{ $t('NewHost') }}</el-button>
                            <el-button icon="el-icon-menu" @click="$emit('show-hosts')">{{ $t('SavedSSH') }}<span class="welcome-count">{{ $store.state.savedSSHList.length }}</span></el-button>
                        </div>
                    </div>
                    <div class="terminal-preview" aria-hidden="true">
                        <div class="preview-toolbar"><div class="preview-dots"><span></span><span></span><span></span></div><span>webssh — workspace</span><i class="el-icon-full-screen"></i></div>
                        <div class="preview-code">
                            <div class="preview-comment"># {{ $t('previewTerminalHint') }}</div>
                            <div><span class="preview-prompt">❯</span> ssh <span class="preview-user">root</span><span class="preview-host">@your-server</span></div>
                            <div class="preview-divider"></div>
                            <div class="preview-line"><i class="el-icon-key"></i> SSH</div>
                            <div class="preview-line"><i class="el-icon-folder-opened"></i> SFTP</div>
                            <div class="preview-line"><i class="el-icon-data-line"></i> {{ $t('liveMetrics') }}</div>
                            <div class="preview-cursor">❯ <span></span></div>
                        </div>
                        <div class="preview-footer"><span>SSH / SFTP</span><span>WebSSH</span></div>
                    </div>
                </div>
                <div class="workspace-features">
                    <div class="workspace-feature"><i class="el-icon-monitor"></i><div><strong>{{ $t('terminal') }}</strong><p>{{ $t('terminalFeature') }}</p></div></div>
                    <div class="workspace-feature"><i class="el-icon-folder-opened"></i><div><strong>{{ $t('FileBrowser') }}</strong><p>{{ $t('filesFeature') }}</p></div></div>
                    <div class="workspace-feature"><i class="el-icon-data-line"></i><div><strong>{{ $t('liveMetrics') }}</strong><p>{{ $t('metricsFeature') }}</p></div></div>
                </div>
            </div>
        </div>

        <div v-show="contextMenuVisible">
            <ul :style="{left:left+'px',top:top+'px'}" class="contextmenu premium-menu">
                <li @click="copyTab()"><i class="el-icon-document-copy"></i> {{$t('Copy')}}</li>
                <li @click="lockTab()"><i class="el-icon-lock"></i> {{ lockButtonShow(menuTab) }}</li>
                <li @click="setScreenfull()"><i class="el-icon-full-screen"></i> {{ $t('FullScreen') }}</li>
                <li @click="removeTab(menuTab)"><i class="el-icon-close"></i> {{$t('Close')}}</li>
                <el-divider></el-divider>
                <li @click="renameTab()"><i class="el-icon-edit"></i> {{$t('Rename')}}</li>
                <el-divider></el-divider>
                <li @click="closeTabs('left')"><i class="el-icon-back"></i> {{$t('CloseLeft')}}</li>
                <li @click="closeTabs('right')"><i class="el-icon-right"></i> {{$t('CloseRight')}}</li>
                <li @click="closeTabs('other')"><i class="el-icon-circle-close"></i> {{$t('CloseOther')}}</li>
                <li @click="closeTabs('all')"><i class="el-icon-error"></i> {{$t('CloseAll')}}</li>
            </ul>
        </div>
    </div>
</template>

<script>
import Sortable from 'sortablejs'
import screenfull from 'screenfull'
import { MessageBox } from 'element-ui'

export default {
    name: 'Tabs',
    components: {
        terminal: () => import(/* webpackChunkName: "terminal" */ '@/components/Terminal')
    },
    data () {
        return {
            currentTerm: '',
            currentTermIndex: 0,
            menuTab: '',
            contextMenuVisible: false,
            left: '',
            top: '',
            sortableInstance: null,
            shortcutHandler: null
        }
    },
    computed: {
        termList: {
            get() {
                return this.$store.state.termList
            },
            set(v) {
                this.$store.commit('SET_TERMLIST', v)
            }
        }
    },
    watch: {
        contextMenuVisible() {
            if (this.contextMenuVisible) {
                document.body.addEventListener('click', this.closeContextMenu)
            } else {
                document.body.removeEventListener('click', this.closeContextMenu)
            }
        },
        currentTerm(next, previous) {
            const previousTerm = this.getTermRef(previous)
            if (previousTerm && typeof previousTerm.deactivate === 'function') {
                previousTerm.deactivate()
            }
            this.$nextTick(() => {
                this.activateCurrentTerm()
            })
        },
        termList() {
            this.$nextTick(() => {
                this.bindTabInteractions()
            })
        }
    },
    mounted() {
        this.bindTabInteractions()
        this.shortcutHandler = (event) => this.handleGlobalShortcut(event)
        window.addEventListener('keydown', this.shortcutHandler, true)
    },
    methods: {
        getTermRef(name) {
            const target = this.$refs[`${name}`]
            if (Array.isArray(target)) {
                return target[0] || null
            }
            return target || null
        },
        activateCurrentTerm() {
            const term = this.getTermRef(this.currentTerm)
            if (term !== null && typeof term.activate === 'function') {
                term.activate()
            }
        },
        bindTabInteractions() {
            const tabTop = document.body.getElementsByClassName('el-tabs__nav-scroll')
            for (let i = 0; i < tabTop.length; ++i) {
                tabTop[i].oncontextmenu = this.openContextMenu
            }

            const el = document.querySelector('.el-tabs__nav')
            if (this.sortableInstance !== null && this.sortableInstance.el !== el) {
                this.sortableInstance.destroy()
                this.sortableInstance = null
            }
            if (el === null || this.sortableInstance !== null) {
                return
            }
            this.sortableInstance = Sortable.create(el, {
                animation: 200,
                onEnd: ({ newIndex, oldIndex }) => {
                    const currRow = this.termList.splice(oldIndex, 1)[0]
                    this.termList.splice(newIndex, 0, currRow)
                }
            })
        },
        lockButtonShow(targetName) {
            if (this.termList.length > 0 && targetName !== '') {
                const tab = this.termList.filter(tab => tab.name === targetName)[0]
                if (tab === undefined) {
                    return this.$t('Lock')
                } else {
                    return tab.closable? this.$t('Lock'):this.$t('Unlock')
                }
            } else {
                return this.$t('Lock')
            }
        },
        copyTab() {
            const term = this.getTermRef(this.menuTab)
            if (term !== null) {
                term.setSSH()
            }
            this.openTerm()
        },
        setScreenfull() {
            if (!screenfull.isEnabled) {
                this.$message({
                    message: this.$t('notSupportFullscreen'),
                    type: 'warning'
                })
                return false
            }
            screenfull.toggle()
        },
        getTabIndex(targetName) {
            let index = 0
            this.termList.forEach((tab, i) => {
                if (tab.name === targetName) {
                    index = i
                    return
                }
            })
            return index
        },
        closeTabs(par) {
            const setCurrentTerm = () => {
                this.currentTermIndex = currMenuIndex
                const tab = this.termList[currMenuIndex]
                this.currentTerm = tab.name
                document.title = tab.label ? `${tab.label} - WebSSH` : 'WebSSH'
                this.$store.commit('SET_TAB', this.termList[this.currentTermIndex])
                const term = this.getTermRef(tab.name)
                if (term !== null) {
                    term.setSSH()
                }
            }
            const filterTerm = (firstIndex, lastIndex) => {
                let tempList = []
                this.termList.forEach((tab, index) => {
                    if ((index >= firstIndex && index < lastIndex) || !tab.closable) {
                        tempList.push(tab)
                    }
                })
                this.termList = tempList
            }
            let currMenuIndex = this.getTabIndex(this.menuTab)
            switch (par) {
            case 'left':
                if (this.currentTermIndex < currMenuIndex) {
                    setCurrentTerm()
                }
                filterTerm(currMenuIndex, this.termList.length)
                break
            case 'right':
                if (this.currentTermIndex > currMenuIndex) {
                    setCurrentTerm()
                }
                filterTerm(0, currMenuIndex + 1)
                break
            case 'other':
                if (this.currentTermIndex !== currMenuIndex) {
                    setCurrentTerm()
                }
                filterTerm(currMenuIndex, currMenuIndex + 1)
                break
            case 'all':
                filterTerm(-1, -1)
                currMenuIndex = this.getTabIndex(this.menuTab)
                setCurrentTerm()
                break
            }
            this.closeContextMenu()
        },
        closeContextMenu() {
            this.contextMenuVisible = false
        },
        openContextMenu(e) {
            e.preventDefault() // 阻止浏览器默认右键菜单。
            const obj = e.srcElement ? e.srcElement : e.target
            if (obj.id) {
                this.menuTab = obj.id.substr(4)
                this.contextMenuVisible = true
                this.left = e.clientX
                this.top = 20
            }
        },
        genID(length) {
            return Number(Math.random().toString().substr(3, length) + Date.now()).toString(36)
        },
        openTerm() {
            const sshInfo = this.$store.state.sshInfo
            if (sshInfo.password === '') {
                return
            }
            this.termList.push({
                name: `${sshInfo.host}-${this.genID(5)}`,
                label: sshInfo.alias || sshInfo.host,
                path: '/',
                closable: true
            })
            const tab = this.termList[this.termList.length - 1]
            this.currentTerm = tab.name
            this.currentTermIndex = this.termList.length - 1
            this.$store.commit('SET_TAB', this.termList[this.currentTermIndex])
            this.$nextTick(() => {
                this.activateCurrentTerm()
            })
        },
        findTerm() {
            this.currentTermIndex = this.getTabIndex(this.currentTerm)
            this.$store.commit('SET_TAB', this.termList[this.currentTermIndex])
        },
        clickTab(tab) {
            this.findTerm()
            const term = this.getTermRef(tab.name)
            if (term !== null) {
                term.setSSH()
            }
            document.title = tab.label ? `${tab.label} - WebSSH` : 'WebSSH'
        },
        removeTab(targetName) {
            let activeName = this.currentTerm
            for (let i = 0; i < this.termList.length; ++i) {
                if (targetName == this.termList[i].name) {
                    if (!this.termList[i].closable) {
                        this.$message({
                            message: this.$t('unlockClose'),
                            type: 'warning'
                        })
                        return
                    }
                    const nextTab = this.termList[i + 1] || this.termList[i - 1]
                    if (nextTab) {
                        activeName = nextTab.name
                    }
                }
            }
            this.currentTerm = activeName
            this.termList = this.termList.filter(tab => tab.name !== targetName)
            this.$nextTick(() => {
                const term = this.getTermRef(this.currentTerm)
                if (term !== null) {
                    term.setSSH()
                    this.activateCurrentTerm()
                }
            })
            this.findTerm()
        },
        async renameTab() {
            for (const tab of this.termList) {
                if (tab.name === this.menuTab) {
                    let {value} = await MessageBox.prompt('', this.$t('Rename'), {
                        inputValue: tab.label,
                        inputErrorMessage: this.$t('pleaseInputValue'),
                        closeOnClickModal: false,
                        closeOnPressEscape: false,
                        inputValidator: function (label) {
                            return label !== null && label.length > 0
                        }
                    }).catch(null)
                    tab.label = value
                    if (this.currentTerm === this.menuTab) {
                        document.title = tab.label ? `${tab.label} - WebSSH` : 'WebSSH'
                    }
                    break
                }
            }
        },
        lockTab() {
            for (let tab of this.termList) {
                if (tab.name === this.menuTab) {
                    tab.closable = !tab.closable
                }
            }
        },
        isTypingTarget(target) {
            if (target === null || target === undefined) {
                return false
            }
            const tag = (target.tagName || '').toUpperCase()
            if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') {
                return true
            }
            if (target.isContentEditable) {
                return true
            }
            // xterm 的终端取用 .xterm-helper-textarea 接收键盘事件；它是预期的输入目标。
            if (target.classList && target.classList.contains('xterm-helper-textarea')) {
                return 'xterm'
            }
            return false
        },
        switchToTabByIndex(index) {
            if (index < 0 || index >= this.termList.length) {
                return
            }
            const tab = this.termList[index]
            if (!tab) {
                return
            }
            this.currentTerm = tab.name
            this.currentTermIndex = index
            this.$store.commit('SET_TAB', tab)
            document.title = tab.label ? `${tab.label} - WebSSH` : 'WebSSH'
            const term = this.getTermRef(tab.name)
            if (term !== null) {
                term.setSSH()
                this.$nextTick(() => this.activateCurrentTerm())
            }
        },
        handleGlobalShortcut(event) {
            const typing = this.isTypingTarget(event.target)
            const key = String(event.key || '').toLowerCase()
            const accelKey = event.ctrlKey || event.metaKey

            // Alt+1..9：切换到第 N 个 tab（无须 Shift/Ctrl）
            if (event.altKey && !event.ctrlKey && !event.metaKey && !event.shiftKey && /^[1-9]$/.test(event.key)) {
                if (this.termList.length === 0) {
                    return
                }
                event.preventDefault()
                event.stopPropagation()
                this.switchToTabByIndex(Number(event.key) - 1)
                return
            }

            // 以下快捷键仅在非普通输入焦点下生效；xterm 终端例外（它吃不到 Ctrl+Shift+... 我们主动接管）。
            if (typing === true) {
                return
            }

            if (accelKey && event.shiftKey && key === 'w') {
                if (this.termList.length === 0 || !this.currentTerm) {
                    return
                }
                event.preventDefault()
                event.stopPropagation()
                this.removeTab(this.currentTerm)
                return
            }

            if (accelKey && event.shiftKey && key === 'f') {
                event.preventDefault()
                event.stopPropagation()
                window.dispatchEvent(new Event('toggle-file-browser'))
                return
            }
        }
    },
    beforeDestroy() {
        if (this.sortableInstance !== null) {
            this.sortableInstance.destroy()
            this.sortableInstance = null
        }
        if (this.shortcutHandler !== null) {
            window.removeEventListener('keydown', this.shortcutHandler, true)
            this.shortcutHandler = null
        }
    }
}
</script>

<style scoped lang="scss">
.tabs-wrapper {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: transparent;
    min-height: 0;
    padding: 20px 24px;
    box-sizing: border-box;
}

.custom-tabs {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    background: #0f172a;
    border: 1px solid #d8e0ed;
    border-radius: 14px;
    box-shadow: 0 8px 24px rgba(23, 36, 59, 0.08);
    overflow: hidden;
}

.custom-tabs ::v-deep .el-tabs__header {
    flex-shrink: 0;
    margin: 0;
    padding: 10px 12px 0;
    background: #eef2f7;
    border-bottom: 1px solid rgba(226, 232, 240, 0.85);
}

.custom-tabs ::v-deep .el-tabs__nav-wrap {
    padding-bottom: 0;
}

.custom-tabs ::v-deep .el-tabs__nav-next,
.custom-tabs ::v-deep .el-tabs__nav-prev {
    line-height: 40px;
}

.custom-tabs ::v-deep .el-tabs__item {
    position: relative;
    height: 40px;
    line-height: 40px;
    margin-right: 8px;
    padding: 0 34px 0 12px !important;
    border: 1px solid rgba(226, 232, 240, 0.72) !important;
    border-bottom: 0 !important;
    border-radius: 8px 8px 0 0;
    background: #e8edf5;
    color: #475569;
    max-width: 220px;
}

.custom-tabs ::v-deep .el-tabs__item.is-active {
    background: #ffffff;
    color: #0f172a;
    border-color: rgba(203, 213, 225, 0.95) !important;
    box-shadow: inset 0 2px 0 var(--accent);
}

.custom-tabs ::v-deep .el-tabs__item:hover {
    color: #0f172a;
    background: rgba(255, 255, 255, 0.95);
}

.custom-tabs ::v-deep .el-tabs__item .el-icon-close {
    position: absolute;
    right: 10px;
    top: 50%;
    margin-top: -7px;
    width: 14px;
    height: 14px;
    line-height: 14px;
    text-align: center;
    border-radius: 50%;
    background: rgba(148, 163, 184, 0.14);
    color: #64748b;
    opacity: 0.65;
}

.custom-tabs ::v-deep .el-tabs__item .el-icon-close:hover {
    background: rgba(239, 68, 68, 0.12);
    color: #dc2626;
}

.custom-tabs ::v-deep .el-tabs__content {
    flex: 1;
    min-height: 0;
    padding: 0;
    background: linear-gradient(180deg, #0f172a 0%, #111827 100%);
}

.custom-tabs ::v-deep .el-tab-pane {
    height: 100%;
}

.tab-label {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    max-width: 100%;
    min-width: 0;
}

.tab-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #94a3b8;
    flex-shrink: 0;
}

.custom-tabs ::v-deep .el-tabs__item.is-active .tab-dot {
    background: #22c55e;
    box-shadow: 0 0 0 4px rgba(34, 197, 94, 0.14), 0 0 8px rgba(34, 197, 94, 0.3);
}

.tab-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

/* The welcome screen shares the workspace palette with connected sessions. */
.empty-state {
    flex: 1;
    min-height: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: auto;
    padding: 28px;
}

.welcome-workspace {
    width: 100%;
    max-width: 1080px;
    margin: auto;
}

.welcome-main {
    display: grid;
    grid-template-columns: 1.05fr 1fr;
    align-items: center;
    gap: 60px;
    padding: 36px 0 56px;
}

.workspace-eyebrow {
    display: flex;
    align-items: center;
    gap: 9px;
    color: #6e7f99;
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.18em;
    margin-bottom: 22px;

    span {
        width: 7px;
        height: 7px;
        border-radius: 2px;
        background: var(--accent);
    }
}

.empty-title {
    margin: 0 0 22px;
    max-width: 460px;
    font-size: clamp(28px, 3.2vw, 44px);
    font-weight: 700;
    color: var(--text);
    line-height: 1.35;
    letter-spacing: -0.045em;
    text-wrap: balance;
    span { display: block; }
    span:last-child { color: #315c9e; }
}

.empty-subtitle {
    font-size: 14px;
    color: var(--text-muted);
    max-width: 390px;
    margin: 0;
    line-height: 1.9;
}

.welcome-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    margin-top: 28px;

    .el-button {
        margin: 0;
        padding: 12px 18px;
    }
}

.welcome-count {
    margin-left: 7px;
    padding: 1px 5px;
    border-radius: 4px;
    background: #edf1f7;
    color: #7a8ba3;
    font: 11px var(--mono-font);
}

.terminal-preview {
    border: 1px solid #273750;
    border-radius: 16px;
    overflow: hidden;
    background: #101b2e;
    box-shadow: 0 24px 48px -16px rgba(23, 36, 59, 0.3);
    font-family: var(--mono-font);
    transform: rotate(-2deg);
}

.preview-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 17px 20px;
    background: #18243a;
    border-bottom: 1px solid #25324a;
    color: #7d8da8;
    font-size: 10px;
}

.preview-dots {
    display: flex;
    gap: 5px;

    span {
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background: #ff7c72;
        &:nth-child(2) { background: #edc16d; }
        &:nth-child(3) { background: #75c8a2; }
    }
}

.preview-code {
    padding: 28px;
    color: #d7e1f3;
    font-size: 13px;
    line-height: 1.8;
}

.preview-comment {
    margin-bottom: 14px;
    color: #7183a1;
    font-size: 11px;
}

.preview-prompt, .preview-cursor { color: #7acfb5; }
.preview-prompt { margin-right: 10px; }
.preview-user { color: #a3c4ff; }
.preview-host { color: #dce6f8; }
.preview-divider { height: 1px; margin: 24px 0 18px; background: #22314b; }
.preview-line {
    color: #94a7c5;
    font-size: 12px;
    margin-bottom: 7px;
    i { margin-right: 10px; color: #7899ca; }
}

.preview-cursor {
    margin-top: 24px;
    display: flex;
    align-items: center;
    gap: 12px;
    span { width: 7px; height: 15px; background: #8cacdd; border-radius: 1px; }
}

.preview-footer {
    display: flex;
    justify-content: space-between;
    padding: 10px 20px;
    border-top: 1px solid #25324a;
    color: #7386a6;
    font-size: 9px;
    letter-spacing: 0.08em;
}

.workspace-features {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
    border-top: 1px solid #dfe5ef;
    padding-top: 28px;
}

.workspace-feature {
    display: flex;
    align-items: flex-start;
    gap: 14px;
    padding: 8px;
    > i { padding: 10px; border: 1px solid #e0e6ef; border-radius: 10px; color: #5879aa; background: #fff; font-size: 18px; }
    strong { color: #33435c; font-size: 13px; font-weight: 600; }
    p { margin: 8px 0 0; color: #7c8aa0; font-size: 11px; line-height: 1.8; }
}

@media (max-width: 900px) {
    .tabs-wrapper { padding: 16px; }
    .empty-state { padding: 16px; }
    .welcome-main { gap: 24px; }
    .preview-code { padding: 22px; }
    .workspace-features { gap: 8px; }
    .workspace-feature { gap: 8px; > i { padding: 8px; font-size: 16px; } }
}

@media (max-width: 650px) {
    .tabs-wrapper { padding: 12px; }
    .empty-state { padding: 12px; }
    .welcome-main { grid-template-columns: 1fr; padding: 12px 0 28px; gap: 32px; }
    .empty-title { font-size: 30px; }
    .workspace-eyebrow { margin-bottom: 16px; }
    .terminal-preview { transform: none; }
    .workspace-features { grid-template-columns: 1fr; padding-top: 20px; gap: 12px; }
    .workspace-feature p { margin-top: 4px; }
}

.contextmenu {
    position: absolute;
    width: 160px;
    background: rgba(255, 255, 255, 0.9);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border: 1px solid rgba(226, 232, 240, 0.8);
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.12), 0 4px 12px rgba(0, 0, 0, 0.08);
    z-index: 4000;
    list-style: none;
    padding: 6px;
    border-radius: 12px;
    
    li {
        padding: 8px 12px;
        font-size: 13px;
        color: #475569;
        font-weight: 500;
        cursor: pointer;
        border-radius: 8px;
        display: flex;
        align-items: center;
        gap: 10px;
        transition: all 0.2s;
        
        i {
            font-size: 15px;
            color: #94a3b8;
        }
        
        &:hover {
            background-color: #f1f5f9;
            color: #3b82f6;
            i { color: #3b82f6; }
        }
        
        &.active {
            background-color: #3b82f6;
            color: #ffffff;
            i { color: #ffffff; }
        }
    }
    
    .el-divider--horizontal {
        margin: 4px 6px;
        width: auto;
    }
}

@media (max-width: 900px) {
    .custom-tabs ::v-deep .el-tabs__item {
        max-width: 168px;
    }
}
</style>
