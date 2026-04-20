import request from '@/utils/request'
export function fileList(path, sshInfo, config = {}) {
    return request.get('/file/list', {
        ...config,
        params: {
            path,
            sshInfo
        }
    })
}

export function getFileContent(path, sshInfo, config = {}) {
    return request.get('/file/content', {
        ...config,
        params: {
            path,
            sshInfo
        }
    })
}

export function saveFileContent(data) {
    return request.post('/file/save', data)
}

export function createFile(data) {
    return request.post('/file/create', data)
}

export function createDirectory(data) {
    return request.post('/file/mkdir', data)
}
