import request from '@/utils/request'
import { buildAppPath } from '@/utils/base'

function normalizeConnectionParams(identity) {
    if (identity && typeof identity === 'object') {
        return identity.sessionId ? { sessionId: identity.sessionId } : { sshInfo: identity.sshInfo || '' }
    }
    return { sshInfo: identity || '' }
}

export function fileList(path, identity, config = {}) {
    return request.get(buildAppPath('/file/list'), {
        ...config,
        timeout: config.timeout || 20000,
        params: {
            path,
            ...normalizeConnectionParams(identity)
        }
    })
}

export function getFileContent(path, identity, config = {}) {
    return request.get(buildAppPath('/file/content'), {
        ...config,
        timeout: config.timeout || 30000,
        params: {
            path,
            ...normalizeConnectionParams(identity)
        }
    })
}

export function saveFileContent(data) {
    return request.post(buildAppPath('/file/save'), data, {
        timeout: 30000
    })
}

export function createFile(data) {
    return request.post(buildAppPath('/file/create'), data, {
        timeout: 20000
    })
}

export function createDirectory(data) {
    return request.post(buildAppPath('/file/mkdir'), data, {
        timeout: 20000
    })
}

export function renameEntry(data) {
    return request.post(buildAppPath('/file/rename'), data, {
        timeout: 20000
    })
}
