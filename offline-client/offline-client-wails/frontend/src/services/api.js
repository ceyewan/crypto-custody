import axios from 'axios'
import store from '../store'

// API基础URL
const API_URL = 'http://localhost:8080'
const MPC_URL = 'http://localhost:8088'

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
        console.error('请求拦截错误:', error)
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
                console.error('认证失败，请重新登录')
                // 可选：自动登出并跳转到登录页面
                if (store && store.dispatch) {
                    store.dispatch('logout')
                    // 使用延迟以确保状态更新完成
                    setTimeout(() => {
                        window.location.href = '/#/login'
                    }, 100)
                }
            }

            // 处理其他错误
            console.error('API错误:', error.response.status, error.response.data)
        } else if (error.request) {
            console.error('请求无响应:', error.request)
        } else {
            console.error('请求配置错误:', error.message)
        }
        return Promise.reject(error)
    }
)

// 设置请求头 - 确保发送裸token
const getAuthHeader = () => {
    const token = localStorage.getItem('token')
    return {
        headers: {
            'Authorization': token && token.startsWith('Bearer ') ? token.substring(7) : token
        }
    }
}

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

// 安全芯片API
export const seApi = {
    // 获取CPIC
    getCPIC() {
        return axios.get(`${MPC_URL}/api/v1/mpc/cplc`)
    },

    // 创建安全芯片记录
    createSE(data) {
        return apiClient.post(`/se/create`, data)
    }
}

// MPC服务API
export const mpcApi = {
    // 密钥生成
    keyGen(data) {
        return axios.post(`${MPC_URL}/api/v1/mpc/keygen`, data)
    },

    // 签名
    sign(data) {
        return axios.post(`${MPC_URL}/api/v1/mpc/sign`, data)
    }
} 