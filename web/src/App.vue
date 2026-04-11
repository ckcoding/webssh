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
          <vheader @ssh-select="handleSSHSelect" />
        </el-header>
        <el-container class="workspace-container">
          <el-aside class="sidebar-wrapper" v-show="$store.state.termList.length > 0 && $store.state.currentTab && $store.state.currentTab.name">
            <server-sidebar />
          </el-aside>
          <el-main style="padding: 0" class="main-wrapper">
              <tabs ref="tabs"></tabs>
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

const statusEndpoint = process.env.NODE_ENV === 'production' ? '/api/status' : '/status'

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
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');

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
    padding: 12px;
    box-sizing: border-box;
}

.main-container {
    height: 100%;
    width: 100%;
    max-width: 1600px;
    background: #ffffff;
    border: 1px solid rgba(226, 232, 240, 0.8);
    border-radius: 16px;
    box-shadow:
        0 8px 24px -4px rgba(0, 0, 0, 0.08),
        0 2px 8px -2px rgba(0, 0, 0, 0.04),
        0 0 0 1px rgba(255, 255, 255, 0.5) inset;
    overflow: hidden;
    display: flex;
    flex-direction: column;
}

.header-wrapper {
    background: #ffffff;
    border-bottom: 1px solid #e2e8f0;
    z-index: 20;
}

.workspace-container {
    background: #ffffff;
    flex: 1;
    min-height: 0;
}

.sidebar-wrapper {
    width: 310px !important;
    background: linear-gradient(180deg, #f8fbff 0%, #f6f8fb 100%);
    border-right: 1px solid #e2e8f0;
    overflow: hidden;
}

.main-wrapper {
    height: 100%;
    position: relative;
    min-width: 0;
    min-height: 0;
    background: linear-gradient(180deg, #f8fafc 0%, #eef2ff 100%);
}

::-webkit-scrollbar {
    width: 8px;
    height: 8px;
}
::-webkit-scrollbar-track {
    background: transparent;
}
::-webkit-scrollbar-thumb {
    background: rgba(148, 163, 184, 0.3);
    border-radius: 4px;
}
::-webkit-scrollbar-thumb:hover {
    background: rgba(148, 163, 184, 0.6);
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
