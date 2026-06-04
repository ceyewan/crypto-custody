import axios from 'axios'
import store from '../store'
import { getServerHttpUrl } from './settings'

const apiClient = axios.create({
    baseURL: getServerHttpUrl(),
    timeout: 10000
})

function normalizeToken(token) {
    if (!token) {
        return ''
    }

    const value = String(token).trim()
    return value.startsWith('Bearer ') ? value.substring(7).trim() : value
}

function currentToken() {
    return normalizeToken(store.state.token) || normalizeToken(localStorage.getItem('token'))
}

function isPublicAuthRequest(config) {
    const url = config.url || ''
    return url === '/user/login' || url === '/user/register'
}

// axios请求拦截器
apiClient.interceptors.request.use(
    config => {
        config.baseURL = getServerHttpUrl()
        config.headers = config.headers || {}
        config.__requiresAuth = !isPublicAuthRequest(config)

        const token = config.__requiresAuth ? currentToken() : ''
        if (token) {
            config.headers['Authorization'] = token
            config.__authToken = token
        }
        return config
    },
    error => {
        return Promise.reject(error)
    }
)

// axios响应拦截器
apiClient.interceptors.response.use(
    response => response,
    error => {
        if (error.response) {
            // 处理401认证错误
            if (error.response.status === 401) {
                const requestToken = normalizeToken(error.config && error.config.__authToken)
                const latestToken = currentToken()

                // 只让当前登录态对应的401触发退出，避免旧请求返回后踢掉刚登录的新会话。
                if (error.config && error.config.__requiresAuth && latestToken && requestToken === latestToken && store && store.dispatch) {
                    store.dispatch('logout')
                    // 使用延迟以确保状态更新完成
                    setTimeout(() => {
                        window.location.href = '/#/login'
                    }, 100)
                }
            }

        }
        return Promise.reject(error)
    }
)

// 用户API
export const userApi = {
    // 用户注册
    register(userData) {
        return apiClient.post(`/user/register`, userData)
    },

    // 用户登录
    login(credentials) {
        return apiClient.post(`/user/login`, credentials)
    },

    // 获取用户列表 (仅管理员)
    getUsers() {
        return apiClient.get(`/user/admin/users`)
    },

    // 更新用户角色 (仅管理员)
    updateUserRole(username, role) {
        return apiClient.put(`/user/admin/users/${username}/role`, { role })
    },

    // 更新用户状态 (仅管理员)
    updateUserStatus(username, status) {
        return apiClient.put(`/user/admin/users/${username}/status`, { status })
    }
}

// 密钥生成API
export const keygenApi = {
    // 创建密钥生成会话
    createSession(initiator) {
        return apiClient.get(`/keygen/create/${initiator}`)
    },

    // 获取可用参与者
    getAvailableUsers() {
        return apiClient.get(`/keygen/users`)
    }
}

// 签名API
export const signApi = {
    // 创建签名会话
    createSession(initiator) {
        return apiClient.get(`/sign/create/${initiator}`)
    },

    // 获取可用签名参与者
    getAvailableUsers(address) {
        return apiClient.get(`/sign/users/${address}`)
    }
}

// 安全芯片API (云端)
export const seApi = {
    // 查询安全芯片列表
    listSecurityElements() {
        return apiClient.get(`/se/list`)
    },

    // 创建安全芯片记录
    createSecurityElement(seid, cplc, custodyLocation = '') {
        return apiClient.post(`/se/create`, {
            se_id: seid,
            cplc: cplc,
            custody_location: custodyLocation
        })
    }
}

// 离线任务和密钥API
export const offlineApi = {
    importTask(taskPackage) {
        return apiClient.post(`/offline/tasks/import`, taskPackage)
    },

    importTaskFile(file) {
        const formData = new FormData()
        formData.append('file', file)
        return apiClient.post(`/offline/tasks/import`, formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        })
    },

    getTask(taskNo) {
        return apiClient.get(`/offline/tasks/${taskNo}`)
    },

    listTasks() {
        return apiClient.get(`/offline/tasks`)
    },

    buildKeygenRequest(taskNo, data) {
        return apiClient.post(`/offline/tasks/${taskNo}/keygen/start`, data)
    },

    buildSignRequest(taskNo, data) {
        return apiClient.post(`/offline/tasks/${taskNo}/sign/start`, data)
    },

    downloadResult(taskNo) {
        return apiClient.get(`/offline/results/${taskNo}/download`)
    },

    listKeys() {
        return apiClient.get(`/offline/keys`)
    },

    getKey(offlineKeyID) {
        return apiClient.get(`/offline/keys/${offlineKeyID}`)
    },

    listShards(params = {}) {
        return apiClient.get(`/offline/shards`, { params })
    },

    listMyShards() {
        return apiClient.get(`/offline/shards/mine`)
    },

    listMyParticipation(limit = 200) {
        return apiClient.get(`/offline/participation/mine`, { params: { limit } })
    },

    transferShard(shardID, data) {
        return apiClient.post(`/offline/shards/${encodeURIComponent(shardID)}/transfer`, data)
    },

    destroyKey(offlineKeyID, data = {}) {
        return apiClient.post(`/offline/keys/${offlineKeyID}/destroy`, data)
    },

    listAudit(params = {}) {
        const query = typeof params === 'number' ? { limit: params } : params
        return apiClient.get(`/offline/audit`, { params: query })
    },

    listApprovals(limit = 100) {
        return apiClient.get(`/offline/approvals`, { params: { limit } })
    },

    downloadBackup() {
        return apiClient.get(`/offline/backup/download`, { responseType: 'blob' })
    }
}
