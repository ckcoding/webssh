import axios from 'axios'
import { Message } from 'element-ui'
import i18n from '@/lang/index'

let lastNetworkErrorAt = 0
const DEFAULT_RETRY_COUNT = 1
const DEFAULT_RETRY_DELAY = 400
const RETRYABLE_STATUS = new Set([408, 429, 502, 503, 504])
const RETRYABLE_METHODS = new Set(['get', 'head', 'options'])

function getStatusMessage(status) {
    switch (status) {
    case 400:
        return i18n.t('badRequest')
    case 403:
        return i18n.t('accessDenied')
    case 404:
        return i18n.t('notFound')
    case 500:
        return i18n.t('serverError')
    default:
        return ''
    }
}

function shouldTriggerLogin(config = {}) {
    const url = typeof config.url === 'string' ? config.url : ''
    return !config.skipAuthRedirect && !url.includes('/login') && !url.includes('/status')
}

function normalizeError(response) {
    if (response && response.data && typeof response.data === 'object') {
        return {
            ...response.data,
            status: response.status
        }
    }
    return {
        Msg: getStatusMessage(response ? response.status : 0),
        status: response ? response.status : 0
    }
}

function notifyNetworkError(message) {
    const now = Date.now()
    if (now-lastNetworkErrorAt < 3000) {
        return
    }
    lastNetworkErrorAt = now
    Message.error(message)
}

function retryDelay(config = {}) {
    const retryCount = Number(config.__retryCount || 0)
    const baseDelay = Number(config.retryDelay || DEFAULT_RETRY_DELAY)
    const jitter = Math.floor(Math.random() * 120)
    return baseDelay * Math.pow(2, retryCount) + jitter
}

function shouldRetry(error) {
    const config = error && error.config ? error.config : null
    if (!config || config.skipRetry) {
        return false
    }
    const method = String(config.method || 'get').toLowerCase()
    if (!RETRYABLE_METHODS.has(method)) {
        return false
    }
    const retry = config.retry === undefined ? DEFAULT_RETRY_COUNT : Number(config.retry)
    config.__retryCount = Number(config.__retryCount || 0)
    if (config.__retryCount >= retry) {
        return false
    }
    if (error.response === undefined) {
        return true
    }
    return RETRYABLE_STATUS.has(error.response.status)
}

function wait(ms) {
    return new Promise(resolve => window.setTimeout(resolve, ms))
}

var instance = axios.create({
    timeout: 12000,
    baseURL: '',
    withCredentials: true
})

instance.interceptors.request.use(
    config => {
        const token = localStorage.getItem('webssh_token')
        config.withCredentials = true
        if (token) {
            config.headers = config.headers || {}
            config.headers.Authorization = `Bearer ${token}`
        }
        return config
    },
    error => Promise.reject(error)
)

instance.interceptors.response.use(
    response => {
        return response.data
    },
    err => {
        if (axios.isCancel(err) || err.code === 'ERR_CANCELED') {
            return Promise.reject(err)
        }
        if (shouldRetry(err)) {
            err.config.__retryCount += 1
            return wait(retryDelay(err.config)).then(() => instance(err.config))
        }
        if (err.response === undefined) {
            notifyNetworkError(i18n.t('serverConnectionFailed'))
            return Promise.reject(err)
        }

        const { response } = err
        if (response.status === 401) {
            if (shouldTriggerLogin(response.config || {})) {
                localStorage.removeItem('webssh_token')
                window.dispatchEvent(new Event('show-login'))
            }
        } else {
            const message = getStatusMessage(response.status)
            if (message) {
                Message({
                    message,
                    type: response.status >= 500 || response.status === 400 ? 'error' : 'warning'
                })
            }
        }

        return Promise.reject(normalizeError(response))
    }
)

export default instance
