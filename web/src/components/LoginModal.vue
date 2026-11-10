<template>
    <el-dialog
        :title="$t('systemLogin')"
        :visible.sync="visible"
        width="1040px"
        top="0"
        class="login-screen"
        custom-class="login-dialog"
        :modal="false"
        :close-on-click-modal="false"
        :show-close="false"
        :close-on-press-escape="false"
        append-to-body>
        <div class="login-layout">
            <section class="login-hero" :aria-label="$t('workspaceCaption')">
                <div class="login-brand"><span class="brand-mark" aria-hidden="true">&gt;_</span><span>Web<strong>SSH</strong></span><span class="brand-tag">WORKSPACE</span></div>
                <div class="hero-eyebrow"><span></span> YOUR SERVERS. ONE WORKSPACE.</div>
                <h1>{{ $t('loginHeroTitle') }}<br><span>{{ $t('loginHeroAccent') }}</span></h1>
                <p class="hero-description">{{ $t('loginHeroDescription') }}</p>
                <div class="login-terminal" aria-hidden="true">
                    <div class="terminal-toolbar"><span class="terminal-dots"><i></i><i></i><i></i></span><span>~/workspace</span><i class="el-icon-full-screen"></i></div>
                    <div class="terminal-code">
                        <div class="code-comment"># {{ $t('loginTerminalHint') }}</div>
                        <div><span class="code-prompt">❯</span> ssh <span class="code-user">user</span><span class="code-host">@your-server</span></div>
                        <div class="code-response"><span>✓</span> {{ $t('loginTerminalReady') }}</div>
                        <div class="code-path">user@your-server <span>~</span></div>
                        <div class="code-last"><span class="code-prompt">❯</span> <span class="code-cursor"></span></div>
                    </div>
                    <div class="terminal-status"><span><i></i> REMOTE WORKSPACE</span><span>SSH / SFTP</span></div>
                </div>
                <div class="login-capabilities"><span><i class="el-icon-monitor"></i>{{ $t('terminal') }}</span><span><i class="el-icon-folder-opened"></i>{{ $t('FileBrowser') }}</span><span><i class="el-icon-data-line"></i>{{ $t('liveMetrics') }}</span></div>
            </section>
            <section class="login-card">
                <div class="card-icon" aria-hidden="true"><i class="el-icon-right"></i></div>
                <div class="card-eyebrow">LET’S GET CONNECTED</div>
                <h2>{{ $t('loginWelcome') }}</h2>
                <p class="login-subtitle">{{ $t('loginSubtitle') }}</p>
                <el-form ref="loginForm" :model="form" :rules="rules" @submit.native.prevent="handleLogin" label-position="top">
                    <el-form-item :label="$t('Username')" prop="username">
                        <el-input v-model="form.username" :placeholder="$t('loginUsernameTip')" prefix-icon="el-icon-user" autocomplete="username" name="username" autocapitalize="none" :spellcheck="false"></el-input>
                    </el-form-item>
                    <el-form-item :label="$t('Password')" prop="password">
                        <el-input v-model="form.password" type="password" :placeholder="$t('loginPasswordTip')" prefix-icon="el-icon-lock" autocomplete="current-password" name="password" show-password @keyup.enter.native="handleLogin"></el-input>
                    </el-form-item>
                    <el-button type="primary" class="login-btn" :loading="loading" @click="handleLogin"><span>{{ $t('loginButton') }}</span><i v-if="!loading" class="el-icon-right"></i></el-button>
                </el-form>
                <div class="login-footer-tip"><i class="el-icon-key"></i><span>{{ $t('loginAccountHint') }}</span></div>
            </section>
            <div class="login-page-footer"><span>WebSSH</span><span>{{ $t('loginFooter') }}</span></div>
        </div>
    </el-dialog>
</template>

<script>
import axios from '@/utils/request'
import { buildApiPath } from '@/utils/base'

const loginEndpoint = buildApiPath('/login')

export default {
    name: 'LoginModal',
    data() {
        return {
            visible: false,
            loading: false,
            form: {
                username: '',
                password: ''
            }
        }
    },
    computed: {
        rules() {
            return {
                username: [{ required: true, message: this.$t('usernameRequired'), trigger: 'blur' }],
                password: [{ required: true, message: this.$t('passwordRequired'), trigger: 'blur' }]
            }
        }
    },
    methods: {
        async handleLogin() {
            this.$refs.loginForm.validate(async (valid) => {
                if (valid) {
                    this.loading = true
                    try {
                        const res = await axios.post(loginEndpoint, this.form, { skipAuthRedirect: true })
                        if (res.Msg === 'success') {
                            this.$message.success(this.$t('loginSuccess'))
                            this.visible = false
                            if (res.token) {
                                localStorage.setItem('webssh_token', res.token)
                            }
                            this.form.password = ''
                            this.$emit('login-success')
                        } else {
                            this.$message.error(res.Msg || this.$t('invalidCredentials'))
                        }
                    } catch (err) {
                        if (err && err.status === 401) {
                            this.$message.error(this.$t('invalidCredentials'))
                        } else if (err && err.status === 400) {
                            this.$message.error(this.$t('badRequest'))
                        } else {
                            this.$message.error((err && err.Msg) || this.$t('loginFailed'))
                        }
                    } finally {
                        this.loading = false
                    }
                }
            })
        },
        show() {
            this.visible = true
            this.loading = false
            this.form.password = ''
            this.$nextTick(() => {
                if (this.$refs.loginForm) {
                    this.$refs.loginForm.clearValidate()
                }
            })
        }
    }
}
</script>

<style scoped lang="scss">
.login-screen {
    padding: 40px 36px;
    isolation: isolate;
    background:
        radial-gradient(ellipse at 13% 18%, rgba(48, 88, 199, .42), transparent 48%),
        radial-gradient(ellipse at 92% 76%, rgba(10, 138, 163, .24), transparent 43%),
        radial-gradient(ellipse at 55% 0%, rgba(98, 68, 177, .2), transparent 48%),
        #0b1224;

    &::before {
        content: '';
        position: fixed;
        inset: 0;
        z-index: -1;
        pointer-events: none;
        background-image: linear-gradient(rgba(157, 186, 246, .035) 1px, transparent 1px), linear-gradient(90deg, rgba(157, 186, 246, .035) 1px, transparent 1px);
        background-size: 64px 64px;
    }

    &::after {
        content: '';
        position: fixed;
        width: 640px;
        height: 640px;
        left: -290px;
        bottom: -330px;
        z-index: -1;
        pointer-events: none;
        border: 1px solid rgba(140, 176, 255, .13);
        border-radius: 50%;
        box-shadow: 0 0 0 70px rgba(140, 176, 255, .025), 0 0 0 140px rgba(140, 176, 255, .02);
    }

    ::v-deep .login-dialog {
        max-width: 100%;
        max-height: calc(100vh - 80px);
        max-height: calc(100dvh - 80px);
        display: flex;
        flex-direction: column;
        background: transparent !important;
        border: 0;
        border-radius: 0;
        box-shadow: none;

        .el-dialog__header { display: none; }
        .el-dialog__body { padding: 0; overflow-y: auto; }
    }
}

.login-layout { display: grid; grid-template-columns: minmax(0, 1fr) 400px; gap: 30px 72px; align-items: center; }
.login-hero { min-width: 0; color: #f1f5ff; }
.login-brand { display: flex; align-items: center; gap: 12px; font-size: 25px; letter-spacing: -.7px; font-weight: 400; }
.login-brand strong { font-weight: 750; }
.brand-mark { display: grid; place-items: center; width: 43px; height: 43px; background: rgba(119, 161, 255, .12); border: 1px solid rgba(149, 184, 255, .22); border-radius: 12px; color: #99bfff; font: 700 20px var(--mono-font); }
.brand-tag { margin-left: 3px; padding: 5px 8px; border: 1px solid rgba(155, 180, 230, .2); border-radius: 5px; color: #96a9ce; font: 10px var(--mono-font); letter-spacing: 1px; }
.hero-eyebrow { display: flex; align-items: center; gap: 9px; margin-top: 42px; color: #99b8ed; font-size: 10px; font-weight: 600; letter-spacing: 1.7px; }
.hero-eyebrow span { width: 6px; height: 6px; border-radius: 50%; background: #7daaff; box-shadow: 0 0 0 4px rgba(125, 170, 255, .1); }
h1 { margin: 18px 0 18px; font-size: clamp(32px, 3.5vw, 44px); font-weight: 650; line-height: 1.4; letter-spacing: -1.8px; }
h1 span { color: #91baff; }
.hero-description { max-width: 410px; margin: 0; color: #a5b3ce; font-size: 14px; line-height: 1.9; }
.login-terminal { margin-top: 30px; background: rgba(8, 17, 34, .62); border: 1px solid rgba(148, 178, 226, .2); border-radius: 14px; box-shadow: 0 20px 55px rgba(0, 0, 0, .2); overflow: hidden; font-family: var(--mono-font); }
.terminal-toolbar { display: flex; align-items: center; justify-content: space-between; height: 38px; padding: 0 16px; color: #8a9ab6; background: rgba(152, 180, 230, .045); border-bottom: 1px solid rgba(148, 178, 226, .09); font-size: 10px; }
.terminal-dots { display: flex; gap: 6px; }
.terminal-dots i { width: 7px; height: 7px; border-radius: 50%; background: #e08a98; }
.terminal-dots i:nth-child(2) { background: #e4bc75; }
.terminal-dots i:nth-child(3) { background: #71c7ad; }
.terminal-code { padding: 22px; color: #d7e1f6; font-size: 12px; line-height: 1.9; }
.code-comment { margin-bottom: 10px; color: #7487a9; }
.code-prompt { margin-right: 10px; color: #8dafff; }
.code-user { color: #98d4c0; }
.code-host { color: #bdd0ed; }
.code-response { margin-top: 10px; color: #8fa2bf; }
.code-response span { margin-right: 8px; color: #79c9ab; }
.code-path { margin-top: 16px; color: #93c7dd; }
.code-path span { color: #86a8e1; }
.code-last { display: flex; align-items: center; height: 24px; }
.code-cursor { width: 7px; height: 14px; background: #96b6ee; }
.terminal-status { display: flex; justify-content: space-between; padding: 9px 16px; border-top: 1px solid rgba(148, 178, 226, .09); color: #7e91b0; font-size: 9px; letter-spacing: .5px; }
.terminal-status span:first-child { display: flex; align-items: center; gap: 7px; }
.terminal-status i { width: 5px; height: 5px; border-radius: 50%; background: #7daaff; }
.login-capabilities { display: flex; flex-wrap: wrap; gap: 24px; margin-top: 21px; color: #a5b5d1; font-size: 11px; }
.login-capabilities span { display: flex; align-items: center; gap: 7px; }
.login-capabilities i { font-size: 15px; color: #81a8e9; }
.login-card { padding: 36px; border: 1px solid rgba(255, 255, 255, .8); border-radius: 22px; background: linear-gradient(160deg, #fff, #f6f8fd); box-shadow: 0 28px 80px rgba(0, 0, 0, .24), 0 0 0 6px rgba(157, 182, 233, .055); }
.card-icon { display: grid; place-items: center; width: 42px; height: 42px; margin-bottom: 23px; border-radius: 12px; background: #e9f0ff; color: #3266d8; font-size: 23px; }
.card-eyebrow { margin-bottom: 10px; color: #8490a6; font-size: 9px; font-weight: 650; letter-spacing: 1.8px; }
h2 { margin: 0; color: #17243b; font-size: 28px; font-weight: 650; letter-spacing: -.8px; }
.login-subtitle { margin: 12px 0 28px; color: #7a879c; font-size: 12px; line-height: 1.8; }
.login-card ::v-deep .el-form-item { margin-bottom: 23px; }
.login-card ::v-deep .el-form-item__label { line-height: 24px; padding-bottom: 7px; font-size: 12px; }
.login-card ::v-deep .el-input__inner { height: 46px; background: #fff !important; border-color: #dfe5f0 !important; border-radius: 9px; padding-left: 38px; font-size: 13px; transition: border-color .15s, box-shadow .15s; }
.login-card ::v-deep .el-input__inner:focus { border-color: #6690ee !important; box-shadow: 0 0 0 3px rgba(63, 111, 219, .09); }
.login-card ::v-deep .el-input__prefix { left: 12px; color: #8a9bb7; }
.login-btn { display: block; width: 100%; margin-top: 6px; height: 46px; padding: 0 20px !important; border: 0 !important; border-radius: 9px !important; background: linear-gradient(110deg, #3069e0, #2454c4) !important; box-shadow: 0 5px 14px rgba(47, 97, 210, .22) !important; font-size: 13px !important; font-weight: 600 !important; }
.login-btn ::v-deep > span { display: inline-flex; align-items: center; justify-content: center; gap: 12px; }
.login-btn:hover { background: linear-gradient(110deg, #255dce, #1949b5) !important; }
.login-btn:focus-visible { outline: 3px solid #9cbafa; outline-offset: 3px; }
.login-footer-tip { display: flex; justify-content: center; align-items: flex-start; gap: 7px; margin-top: 24px; color: #929daf; font-size: 10px; line-height: 1.7; }
.login-footer-tip i { margin-top: 3px; color: #8091af; font-size: 12px; }
.login-page-footer { grid-column: 1 / -1; display: flex; justify-content: space-between; padding-top: 10px; color: #6f83a8; font-size: 10px; letter-spacing: .2px; }
.login-page-footer span:first-child { font-family: var(--mono-font); }

@media (max-width: 900px) {
    .login-layout { gap: 28px 36px; grid-template-columns: minmax(0, 1fr) 360px; }
    .login-card { padding: 30px; }
    .brand-tag { display: none; }
    .terminal-code { padding: 18px; font-size: 11px; }
}
@media (max-width: 760px) {
    .login-screen { padding: 28px 20px; }
    .login-screen ::v-deep .login-dialog { width: 400px !important; max-height: calc(100vh - 56px); max-height: calc(100dvh - 56px); }
    .login-layout { grid-template-columns: minmax(0, 1fr); gap: 22px; }
    .login-brand { justify-content: center; font-size: 24px; }
    .brand-mark { width: 38px; height: 38px; font-size: 18px; }
    .hero-eyebrow, .hero-description, .login-terminal, .login-capabilities, .login-hero h1 { display: none; }
    .login-card { padding: 28px; border-radius: 18px; }
    .card-icon { display: none; }
    .login-page-footer { justify-content: center; padding-top: 0; }
    .login-page-footer span:first-child { display: none; }
}
@media (max-width: 360px), (max-height: 600px) {
    .login-screen { padding: 16px; }
    .login-screen ::v-deep .login-dialog { max-height: calc(100vh - 32px); max-height: calc(100dvh - 32px); }
    .login-layout { gap: 18px; }
    .login-card { padding: 23px; }
    .card-icon { display: none; }
    .login-subtitle { margin: 8px 0 20px; }
    .login-footer-tip { margin-top: 18px; }
    .login-page-footer { display: none; }
}
</style>
