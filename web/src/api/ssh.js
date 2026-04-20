import request from '@/utils/request'

export function checkSSHConnection(data) {
    return request.post('/ssh/check', data, {
        timeout: 20000
    })
}

export function listSavedSSH() {
    return request.get('/ssh/saved')
}

export function getSSHSystemInfo(sshInfo) {
    return request.get('/ssh/system', {
        params: {
            sshInfo
        }
    })
}

export function buildSSHSystemStreamURL(sshInfo) {
    const prefix = process.env.NODE_ENV === 'production' ? '' : '/api'
    const params = new URLSearchParams({ sshInfo })
    const token = localStorage.getItem('webssh_token')
    if (token) {
        params.set('token', token)
    }
    return `${prefix}/ssh/system/stream?${params.toString()}`
}

export function saveSavedSSH(data) {
    return request.post('/ssh/saved', data)
}

export function deleteSavedSSH(id) {
    return request.delete(`/ssh/saved/${encodeURIComponent(id)}`)
}
