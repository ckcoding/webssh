<template>
  <div id="app">
    <div v-if="!appReady && !showLoginModal" class="initial-loading">
      <div class="loading-content">
        <div class="loading-ring">
          <div class="loading-ring-outer"></div>
          <div class="loading-ring-inner"></div>
        </div>
        <div class="loading-brand">
          <span class="loading-logo">Web</span><span class="loading-logo-bold">SSH</span>
        </div>
        <div class="loading-text">正在初始化...</div>
      </div>
    </div>

    <div class="app-layout" v-if="appReady">
      <el-container class="main-container">
        <el-header style="height: auto; padding: 0" class="header-wrapper">
          <vheader ref="header" @ssh-select="handleSSHSelect" />
        </el-header>
        <el-container class="workspace-container">
          <el-aside class="sidebar-wrapper" v-show="$store.state.termList.length > 0 && $store.state.currentTab && $store.state.currentTab.name">
            <server-sidebar />
          </el-aside>
          <el-main style="padding: 0" class="main-wrapper">
              <tabs ref="tabs" @new-connection="$refs.header.openConnectionDialog()" @show-hosts="$refs.header.openSavedSSHMenu()"></tabs>
          </el-main>
        </el-container>
      </el-container>
    </div>
    
    <login-modal ref="loginModal" @login-success="handleLoginSuccess" />
  </div>
</template>

<script>
import Header from '@/components/Header'
import ServerSidebar from '@/components/ServerSidebar'
import Tabs from '@/components/Tabs'
import LoginModal from '@/components/LoginModal'
import axios from '@/utils/request'
import { buildApiPath } from '@/utils/base'

const statusEndpoint = buildApiPath('/status')

export default {
    name: 'App',
    components: {
        vheader: Header,
        'server-sidebar': ServerSidebar,
        tabs: Tabs,
        'login-modal': LoginModal
    },
    data() {
        return {
            appReady: false,
            showLoginModal: false,
            loginEventHandler: null
        }
    },
    async mounted() {
        this.loginEventHandler = () => {
            this.handleShowLogin()
        }
        window.addEventListener('show-login', this.loginEventHandler)

        try {
            const res = await axios.get(statusEndpoint, { skipAuthRedirect: true })
            if (res && typeof res.savePass === 'boolean') {
                this.$store.commit('SET_SAVE_PASS', res.savePass)
            }
            if (res && res.auth && !res.authenticated) {
                this.handleShowLogin()
                return
            }
            this.appReady = true
        } catch (err) {
            console.error('获取系统状态失败', err)
            this.appReady = true
        }
    },
    beforeDestroy() {
        if (this.loginEventHandler !== null) {
            window.removeEventListener('show-login', this.loginEventHandler)
            this.loginEventHandler = null
        }
    },
    methods: {
        handleShowLogin() {
            this.appReady = false
            this.showLoginModal = true
            this.$nextTick(() => {
                if (this.$refs.loginModal) {
                    this.$refs.loginModal.show()
                }
            })
        },
        handleLoginSuccess() {
            this.showLoginModal = false
            this.appReady = true
        },
        handleSSHSelect() {
            if (this.$refs.tabs) {
                this.$refs.tabs.openTerm()
            }
        }
    }
}
</script>

<style lang="scss">
html, body {
    margin: 0;
    padding: 0;
    height: 100%;
    width: 100%;
    font-family: 'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    background-color: #f1f5f9;
    overflow: hidden;
    color: #1e293b;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
}

#app {
    height: 100%;
    height: 100dvh;
    width: 100%;
    position: relative;
    background-color: #f1f5f9;
}

.initial-loading {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    display: flex;
    justify-content: center;
    align-items: center;
    background-color: #f1f5f9;
    z-index: 9999;
}

.loading-content {
    text-align: center;
}

.loading-ring {
    width: 56px;
    height: 56px;
    margin: 0 auto 20px;
    position: relative;
}

.loading-ring-outer {
    position: absolute;
    width: 100%;
    height: 100%;
    border: 3px solid #e2e8f0;
    border-top-color: #3b82f6;
    border-radius: 50%;
}

.loading-ring-inner {
    position: absolute;
    top: 8px;
    left: 8px;
    width: calc(100% - 16px);
    height: calc(100% - 16px);
    border: 3px solid #e2e8f0;
    border-bottom-color: #7c3aed;
    border-radius: 50%;
}

.loading-brand {
    margin-bottom: 8px;
}

.loading-logo {
    font-size: 22px;
    color: #64748b;
    font-weight: 300;
    letter-spacing: -0.3px;
}

.loading-logo-bold {
    font-size: 22px;
    color: #3b82f6;
    font-weight: 800;
    letter-spacing: -0.5px;
}

.loading-text {
    font-size: 13px;
    color: #94a3b8;
    font-weight: 500;
    letter-spacing: 0.01em;
}

.app-layout {
    height: 100%;
    width: 100%;
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 0;
    box-sizing: border-box;
}

.main-container {
    height: 100%;
    width: 100%;
    max-width: none;
    background: #ffffff;
    border: none;
    border-radius: 0;
    box-shadow: none;
    overflow: hidden;
    display: flex;
    flex-direction: column;
}

.header-wrapper {
    background: #ffffff;
    border-bottom: 1px solid var(--border);
    box-shadow: 0 2px 12px rgba(15, 23, 42, 0.03);
    z-index: 20;
}

.workspace-container {
    background: var(--workspace-bg);
    position: relative;
    flex: 1;
    min-height: 0;
}

.workspace-container::before {
    content: '';
    position: absolute;
    inset: 0;
    background-image: radial-gradient(#cbd5e1 0.7px, transparent 0.7px);
    background-size: 24px 24px;
    opacity: 0.3;
    pointer-events: none;
    z-index: 0;
}

.sidebar-wrapper {
    width: 320px !important;
    background: transparent;
    border-right: 0;
    overflow: hidden;
    position: relative;
    z-index: 1;
}

.main-wrapper {
    flex: 1;
    overflow: hidden;
    position: relative;
    min-width: 0;
    min-height: 0;
    background: transparent;
    z-index: 1;
}


@media (max-width: 1100px) {
    .workspace-container {
        flex-direction: column;
    }

    .sidebar-wrapper {
        width: 100% !important;
        height: auto !important;
        max-height: none !important;
        border-right: 0;
        border-bottom: 1px solid #e2e8f0;
        overflow: visible !important;
        flex: none;
    }
}

@media (max-width: 768px) {
    .app-layout {
        padding: 0;
    }

    .main-container {
        border-radius: 0;
        border: 0;
        box-shadow: none;
        max-width: none;
    }

    .sidebar-wrapper {
        max-height: none !important;
    }
}
</style>
