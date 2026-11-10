function stringToBase64(value) {
    const encoder = new TextEncoder()
    let binary = ''
    encoder.encode(value).forEach((byte) => {
        binary += String.fromCharCode(byte)
    })
    return window.btoa(binary)
}

function base64ToString(value) {
    const binary = window.atob(value)
    const bytes = Uint8Array.from(binary, char => char.charCodeAt(0))
    return new TextDecoder().decode(bytes)
}

export function normalizeSSHInfo(sshInfo = {}) {
    const normalized = {
        host: sshInfo.host || '',
        ip: sshInfo.ip || '',
        countryFlag: sshInfo.countryFlag || '',
        location: sshInfo.location || '',
        os: sshInfo.os || '',
        memory: sshInfo.memory || '',
        expiresAt: sshInfo.expiresAt || '',
        alias: sshInfo.alias || '',
        username: sshInfo.username || 'root',
        port: Number(sshInfo.port) || 22,
        logintype: sshInfo.logintype === undefined ? 0 : Number(sshInfo.logintype)
    }
    if (sshInfo.password !== undefined) {
        normalized.password = sshInfo.password
    }
    return normalized
}

export function encodeSSHInfo(sshInfo) {
    const normalized = normalizeSSHInfo(sshInfo)
    return stringToBase64(JSON.stringify({
        username: normalized.username,
        ipaddress: normalized.host,
        port: normalized.port,
        password: normalized.password || '',
        logintype: normalized.logintype
    }))
}

export function decodeSSHList(encodedList) {
    if (!encodedList) {
        return []
    }
    try {
        const parsed = JSON.parse(base64ToString(encodedList))
        return Array.isArray(parsed) ? parsed.map(item => normalizeSSHInfo(item)) : []
    } catch (error) {
        return []
    }
}

export function encodeSSHList(list = []) {
    return stringToBase64(JSON.stringify(list.map(item => normalizeSSHInfo(item))))
}

export function buildHistoryItem(sshInfo, savePass) {
    const item = normalizeSSHInfo(sshInfo)
    if (!savePass) {
        delete item.password
    }
    return item
}

export function isSameSSH(target, candidate) {
    const left = normalizeSSHInfo(target)
    const right = normalizeSSHInfo(candidate)
    return left.host === right.host &&
        left.username === right.username &&
        left.port === right.port &&
        left.logintype === right.logintype
}
