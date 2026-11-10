import request from '@/utils/request'
import { buildAppPath, buildAppURL } from '@/utils/base'

export function checkSSHConnection(data) {
    return request.post(buildAppPath('/ssh/check'), data, {
        timeout: 25000
    })
}

function normalizeSSHParams(identity) {
    if (identity && typeof identity === 'object') {
        return identity.sessionId ? { sessionId: identity.sessionId } : { sshInfo: identity.sshInfo || '' }
    }
    return { sshInfo: identity || '' }
}

export function createSSHSession(data) {
    return request.post(buildAppPath('/ssh/session'), data, {
        timeout: 15000
    })
}

export function listSavedSSH() {
    return request.get(buildAppPath('/ssh/saved'), {
        timeout: 15000
    })
}

export function getSSHSystemInfo(identity) {
    return request.get(buildAppPath('/ssh/system'), {
        timeout: 30000,
        params: normalizeSSHParams(identity)
    })
}

export function buildSSHSystemStreamURL(identity) {
    const params = normalizeSSHParams(identity)
    const token = localStorage.getItem('webssh_token')
    if (token) {
        params.token = token
    }
    return buildAppURL('/ssh/system/stream', params)
}

export function saveSavedSSH(data) {
    return request.post(buildAppPath('/ssh/saved'), data, {
        timeout: 20000
    })
}

export function deleteSavedSSH(id) {
    return request.delete(buildAppPath(`/ssh/saved/${encodeURIComponent(id)}`), {
        timeout: 15000
    })
}

export function refreshSavedSSHInfo(id) {
    return request.post(buildAppPath(`/ssh/saved/${encodeURIComponent(id)}/refresh`), {}, { timeout: 35000 })
}
