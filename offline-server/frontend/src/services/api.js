import axios from 'axios'

// API基础URL
const API_URL = 'http://localhost:8080'
const MPC_URL = 'http://localhost:8088'

// 设置请求头
const getAuthHeader = () => {
    const token = localStorage.getItem('token')
    return {
        headers: {
            'Authorization': token
        }
    }
}

// 用户API
export const userApi = {
    // 用户注册
    register(userData) {
        return axios.post(`${API_URL}/user/register`, userData)
    },

    // 用户登录
    login(credentials) {
        return axios.post(`${API_URL}/user/login`, credentials)
    },

    // 获取用户列表 (仅管理员)
    getUsers() {
        return axios.get(`${API_URL}/user/admin/users`, getAuthHeader())
    },

    // 更新用户角色 (仅管理员)
    updateUserRole(username, role) {
        return axios.put(`${API_URL}/user/admin/users/${username}/role`, { role }, getAuthHeader())
    }
}

// 密钥生成API
export const keygenApi = {
    // 创建密钥生成会话
    createSession(initiator) {
        return axios.get(`${API_URL}/keygen/create/${initiator}`, getAuthHeader())
    },

    // 获取可用参与者
    getAvailableUsers() {
        return axios.get(`${API_URL}/keygen/users`, getAuthHeader())
    }
}

// 签名API
export const signApi = {
    // 创建签名会话
    createSession(initiator) {
        return axios.get(`${API_URL}/sign/create/${initiator}`, getAuthHeader())
    },

    // 获取可用签名参与者
    getAvailableUsers(address) {
        return axios.get(`${API_URL}/sign/users/${address}`, getAuthHeader())
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
        return axios.post(`${API_URL}/se/create`, data, getAuthHeader())
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