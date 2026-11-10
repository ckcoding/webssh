function normalizeBasePath(value = '') {
    const trimmed = String(value || '').trim()
    if (trimmed === '' || trimmed === '/') {
        return ''
    }
    return `/${trimmed.replace(/^\/+|\/+$/g, '')}`
}

function normalizePath(path = '') {
    const trimmed = String(path || '').trim()
    if (trimmed === '' || trimmed === '/') {
        return '/'
    }
    return trimmed.startsWith('/') ? trimmed : `/${trimmed}`
}

function joinPath(prefix, path) {
    const normalizedPrefix = normalizeBasePath(prefix)
    const normalizedPath = normalizePath(path)
    return normalizedPrefix ? `${normalizedPrefix}${normalizedPath}` : normalizedPath
}

function toQueryString(query) {
    if (!query) {
        return ''
    }
    if (query instanceof URLSearchParams) {
        return query.toString()
    }
    return new URLSearchParams(query).toString()
}

function configuredBasePath() {
    if (typeof document === 'undefined') {
        return ''
    }
    const meta = document.querySelector('meta[name="webssh-base-path"]')
    if (!meta) {
        return ''
    }
    const content = meta.getAttribute('content') || ''
    if (content.includes('__WEBSSH_BASE_PATH__')) {
        return ''
    }
    return normalizeBasePath(content)
}

function inferredBasePath() {
    if (typeof window === 'undefined') {
        return ''
    }
    return normalizeBasePath(window.location.pathname)
}

function getBasePath() {
    if (process.env.NODE_ENV !== 'production') {
        return ''
    }
    return configuredBasePath() || inferredBasePath()
}

export function buildAppPath(path) {
    const prefix = process.env.NODE_ENV === 'production' ? getBasePath() : '/api'
    return joinPath(prefix, path)
}

export function buildApiPath(path) {
    const prefix = process.env.NODE_ENV === 'production' ? joinPath(getBasePath(), '/api') : '/api'
    return joinPath(prefix, path)
}

export function buildAppURL(path, query) {
    const pathname = buildAppPath(path)
    const queryString = toQueryString(query)
    return queryString ? `${pathname}?${queryString}` : pathname
}

export function buildWsURL(path, query) {
    const protocol = window.location.protocol === 'http:' ? 'ws' : 'wss'
    const prefix = process.env.NODE_ENV === 'production' ? getBasePath() : '/ws'
    const pathname = joinPath(prefix, path)
    const queryString = toQueryString(query)
    return `${protocol}://${window.location.host}${pathname}${queryString ? `?${queryString}` : ''}`
}
