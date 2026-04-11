<template>
    <el-dialog
        :title="$t('systemLogin')"
        :visible.sync="visible"
        width="400px"
        custom-class="premium-dialog login-dialog"
        :close-on-click-modal="false"
        :show-close="false"
        :close-on-press-escape="false"
        append-to-body>
        <div class="login-container">
            <div class="login-deco-bar"></div>
            <div class="login-logo">
                <span class="logo-wrap">
                    <span class="logo-icon">🖥️</span>
                    <span class="logo">Web</span><span class="logo-bold">SSH</span>
                </span>
            </div>
            <p class="login-subtitle">{{ $t('loginSubtitle') }}</p>
            
            <el-form ref="loginForm" :model="form" :rules="rules" @submit.native.prevent="handleLogin" label-position="top">
                <el-form-item :label="$t('Username')" prop="username">
                    <el-input 
                        v-model="form.username" 
                        :placeholder="$t('loginUsernameTip')" 
                        prefix-icon="el-icon-user">
                    </el-input>
                </el-form-item>
                
                <el-form-item :label="$t('Password')" prop="password">
                    <el-input 
                        v-model="form.password" 
                        type="password" 
                        :placeholder="$t('loginPasswordTip')" 
                        prefix-icon="el-icon-lock"
                        show-password
                        @keyup.enter.native="handleLogin">
                    </el-input>
                </el-form-item>
                
                <el-button 
                    type="primary" 
                    class="login-btn" 
                    :loading="loading" 
                    @click="handleLogin">
                    {{ $t('loginButton') }}
                </el-button>
            </el-form>

            <div class="login-footer-tip">
                <i class="el-icon-lock"></i>
                <span>安全加密连接</span>
            </div>
        </div>
    </el-dialog>
</template>


<script>
import axios from '@/utils/request'

const loginEndpoint = process.env.NODE_ENV === 'production' ? '/api/login' : '/login'

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
.login-container {
    padding: 0 20px 10px;
    position: relative;
}

.login-deco-bar {
    height: 4px;
    background: linear-gradient(90deg, #3b82f6, #7c3aed, #ec4899, #3b82f6);
    background-size: 200% 100%;
    margin: -30px -24px 24px;
    border-radius: 0;
}

.login-logo {
    text-align: center;
    margin-bottom: 8px;
    
    .logo-wrap {
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }

    .logo-icon {
        font-size: 26px;
        line-height: 1;
    }
    
    .logo {
        font-size: 28px;
        color: #64748b;
        font-weight: 300;
        letter-spacing: -0.3px;
    }
    .logo-bold {
        font-size: 28px;
        color: #3b82f6;
        font-weight: 800;
        letter-spacing: -0.5px;
    }
}

.login-subtitle {
    text-align: center;
    color: #94a3b8;
    margin-bottom: 28px;
    font-size: 13px;
    font-weight: 500;
    letter-spacing: 0.01em;
}

.login-btn {
    width: 100%;
    margin-top: 12px;
    padding: 12px !important;
    font-size: 15px !important;
    font-weight: 600 !important;
    border-radius: 12px !important;
    letter-spacing: 0.03em;
    background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%) !important;
    box-shadow: 0 6px 20px rgba(59, 130, 246, 0.35) !important;
    border: none !important;
    
    &:hover {
        box-shadow: 0 8px 28px rgba(59, 130, 246, 0.45) !important;
    }
}

.login-footer-tip {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 5px;
    margin-top: 16px;
    color: #cbd5e1;
    font-size: 11px;
    font-weight: 500;
    
    i {
        font-size: 11px;
    }
}

::v-deep .login-dialog {
    border-radius: 20px !important;
    overflow: hidden;
    box-shadow: 0 25px 60px -15px rgba(0, 0, 0, 0.2);
    .el-dialog__header {
        display: none;
    }
    .el-dialog__body {
        padding: 30px 28px 28px;
    }
}
</style>
