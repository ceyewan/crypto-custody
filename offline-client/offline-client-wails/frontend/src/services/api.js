import axios from 'axios'
import store from '../store'
import { Environment } from '../../wailsjs/runtime/runtime'

// API基础URL
// 使用 Wails runtime.Environment() 来可靠地检测环境
let API_URL = '/api'; // 默认使用代理

// 异步检测 Wails 环境并设置正确的 API URL
async function detectEnvironmentAndSetURL() {
    try {
        await Environment();
        // 在 Wails 环境中直接连接远程服务器
        API_URL = 'https://crypto-custody-offline-server.ceyewan.icu';

        // 更新 axios 实例的 baseURL
        apiClient.defaults.baseURL = API_URL;
    } catch {
        // 不是 Wails 环境，使用代理
        API_URL = '/api';
        apiClient.defaults.baseURL = API_URL;
    }
}

// 初始化环境检测
detectEnvironmentAndSetURL();

// 创建axios实例
const apiClient = axios.create({
    baseURL: API_URL,
    timeout: 10000
})

// axios请求拦截器
apiClient.interceptors.request.use(
    config => {
        const token = localStorage.getItem('token')
        if (token) {
            // 如果token以Bearer开头，则去掉前缀，只发送裸token
            config.headers['Authorization'] = token.startsWith('Bearer ') ? token.substring(7) : token
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
                // 可选：自动登出并跳转到登录页面
                if (store && store.dispatch) {
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
    // 创建安全芯片记录
    createSecurityElement(seid, cplc) {
        return apiClient.post(`/se/create`, { se_id: seid, cplc: cplc })
    }
}

// 离线任务和密钥API
export const offlineApi = {
    importTask(taskPackage) {
        return apiClient.post(`/offline/tasks/import`, taskPackage)
    },

    getTask(taskNo) {
        return apiClient.get(`/offline/tasks/${taskNo}`)
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

    getKey(offlineKeyID) {
        return apiClient.get(`/offline/keys/${offlineKeyID}`)
    },

    transferKey(offlineKeyID, data) {
        return apiClient.post(`/offline/keys/${offlineKeyID}/transfer`, data)
    },

    destroyKey(offlineKeyID, data = {}) {
        return apiClient.post(`/offline/keys/${offlineKeyID}/destroy`, data)
    },

    listAudit(limit = 100) {
        return apiClient.get(`/offline/audit`, { params: { limit } })
    },

    listApprovals(limit = 100) {
        return apiClient.get(`/offline/approvals`, { params: { limit } })
    }
}
