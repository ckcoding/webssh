import axios from 'axios'
import { Message } from 'element-ui'
import i18n from '@/lang/index'

let lastNetworkErrorAt = 0

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

var instance = axios.create({
    timeout: 8000,
    baseURL: process.env.NODE_ENV === 'production' ? '/' : '/api',
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
