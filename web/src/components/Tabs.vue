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
            <div class="empty-content">
                <img src="@/assets/empty_terminal.png" :alt="$t('emptyTerminalAlt')" class="empty-image" />
                <div class="empty-title">{{ $t('waitingConnection') }}</div>
                <div class="empty-subtitle">{{ $t('connectUsingControls') }}</div>
            </div>
            <div class="glow-effect"></div>
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
import Terminal from '@/components/Terminal'
import { MessageBox } from 'element-ui'

export default {
    name: 'Tabs',
    components: {
        terminal: Terminal
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
        currentTerm() {
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
                label: sshInfo.host,
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
}

.custom-tabs {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    background: rgba(255, 255, 255, 0.55);
    border: 1px solid rgba(255, 255, 255, 0.9);
    border-radius: 24px;
    box-shadow: 0 20px 40px -15px rgba(0, 0, 0, 0.05), 0 0 0 1px rgba(255, 255, 255, 0.8) inset;
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    overflow: hidden;
}

.custom-tabs ::v-deep .el-tabs__header {
    margin: 0;
    padding: 8px 12px 0;
    background: rgba(255, 255, 255, 0.78);
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
    border-radius: 12px 12px 0 0;
    background: rgba(241, 245, 249, 0.88);
    color: #475569;
    max-width: 220px;
}

.custom-tabs ::v-deep .el-tabs__item.is-active {
    background: #ffffff;
    color: #0f172a;
    border-color: rgba(203, 213, 225, 0.95) !important;
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

.empty-state {
    flex-grow: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
    overflow: hidden;
}

.empty-content {
    text-align: center;
    z-index: 10;
}

.empty-image {
    max-width: 420px;
    height: auto;
    opacity: 0.9;
    margin-bottom: 12px;
    filter: drop-shadow(0 20px 40px rgba(0, 0, 0, 0.04));
    mix-blend-mode: multiply;
    mask-image: radial-gradient(circle, black 50%, transparent 95%);
    -webkit-mask-image: radial-gradient(circle, black 50%, transparent 95%);
}



.empty-title {
    font-size: 22px;
    font-weight: 800;
    color: #1e293b;
    margin-bottom: 10px;
    letter-spacing: -0.03em;
}

.empty-subtitle {
    font-size: 14px;
    color: #64748b;
    max-width: 360px;
    margin: 0 auto;
    line-height: 1.7;
}

.glow-effect {
    position: absolute;
    width: 400px;
    height: 400px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(59, 130, 246, 0.06) 0%, transparent 70%);
    pointer-events: none;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
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
