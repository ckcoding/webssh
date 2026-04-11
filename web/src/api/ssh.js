import request from '@/utils/request'

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

export function saveSavedSSH(data) {
    return request.post('/ssh/saved', data)
}

export function deleteSavedSSH(id) {
    return request.delete(`/ssh/saved/${encodeURIComponent(id)}`)
}
