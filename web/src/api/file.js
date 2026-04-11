import request from '@/utils/request'
export function fileList(path, sshInfo) {
    return request.get('/file/list', {
        params: {
            path,
            sshInfo
        }
    })
}

export function getFileContent(path, sshInfo) {
    return request.get('/file/content', {
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
