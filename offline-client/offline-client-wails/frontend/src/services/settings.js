const HTTP_URL_KEY = 'offline_client_server_http_url'
const WS_URL_KEY = 'offline_client_server_ws_url'
const CARD_READER_KEY = 'offline_client_card_reader_name'
const DEFAULT_CARD_READER_NAME = 'GOODIX GSE SmartCard Reader'

function trimTrailingSlash(value) {
    return value.replace(/\/+$/, '')
}

export function getDefaultServerHttpUrl() {
    if (process.env.VUE_APP_OFFLINE_SERVER_HTTP_URL) {
        return trimTrailingSlash(process.env.VUE_APP_OFFLINE_SERVER_HTTP_URL.trim())
    }
    return process.env.NODE_ENV === 'development' ? '/api' : 'http://127.0.0.1:8080'
}

export function getDefaultServerWsUrl(httpUrl) {
    if (process.env.VUE_APP_OFFLINE_SERVER_WS_URL) {
        return trimTrailingSlash(process.env.VUE_APP_OFFLINE_SERVER_WS_URL.trim())
    }
    if (process.env.NODE_ENV === 'development') {
        return deriveWsUrl(httpUrl || getDefaultServerHttpUrl())
    }
    return 'ws://127.0.0.1:8081/ws'
}

export function normalizeHttpUrl(value) {
    const url = (value || '').trim()
    if (!url) {
        return getDefaultServerHttpUrl()
    }
    if (url.startsWith('/')) {
        return trimTrailingSlash(url)
    }
    if (/^https?:\/\//i.test(url)) {
        return trimTrailingSlash(url)
    }
    const parsed = new URL(`http://${url}`)
    if (!parsed.port) {
        parsed.port = '8080'
    }
    return trimTrailingSlash(parsed.toString())
}

export function deriveWsUrl(httpUrl) {
    const normalized = normalizeHttpUrl(httpUrl)
    if (normalized.startsWith('/')) {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        return `${protocol}//${window.location.host}/ws`
    }

    const parsed = new URL(normalized)
    parsed.protocol = parsed.protocol === 'https:' ? 'wss:' : 'ws:'
    if (parsed.port === '8080') {
        parsed.port = '8081'
    }
    parsed.pathname = '/ws'
    parsed.search = ''
    parsed.hash = ''
    return trimTrailingSlash(parsed.toString())
}

export function normalizeWsUrl(value, httpUrl) {
    const url = (value || '').trim()
    if (!url) {
        return getDefaultServerWsUrl(httpUrl)
    }
    if (/^wss?:\/\//i.test(url)) {
        return trimTrailingSlash(url)
    }
    if (/^https?:\/\//i.test(url)) {
        return deriveWsUrl(url)
    }
    return trimTrailingSlash(`ws://${url}`)
}

export function loadClientSettings() {
    const serverHttpUrl = normalizeHttpUrl(localStorage.getItem(HTTP_URL_KEY) || getDefaultServerHttpUrl())
    const serverWsUrl = normalizeWsUrl(localStorage.getItem(WS_URL_KEY) || getDefaultServerWsUrl(serverHttpUrl), serverHttpUrl)
    const savedCardReaderName = (localStorage.getItem(CARD_READER_KEY) || '').trim()
    const cardReaderName = savedCardReaderName === 'GOODIX GSE SmartCard Reader 01'
        ? DEFAULT_CARD_READER_NAME
        : (savedCardReaderName || DEFAULT_CARD_READER_NAME)
    return {
        serverHttpUrl,
        serverWsUrl,
        cardReaderName
    }
}

export function saveClientSettings(settings) {
    const serverHttpUrl = normalizeHttpUrl(settings.serverHttpUrl)
    const serverWsUrl = normalizeWsUrl(settings.serverWsUrl, serverHttpUrl)
    const cardReaderName = (settings.cardReaderName || '').trim()

    localStorage.setItem(HTTP_URL_KEY, serverHttpUrl)
    localStorage.setItem(WS_URL_KEY, serverWsUrl)
    localStorage.setItem(CARD_READER_KEY, cardReaderName)

    return {
        serverHttpUrl,
        serverWsUrl,
        cardReaderName
    }
}

export function getServerHttpUrl() {
    return loadClientSettings().serverHttpUrl
}

export function getServerWsUrl() {
    return loadClientSettings().serverWsUrl
}
