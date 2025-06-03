import axios from 'axios'
import store from '../store'

// API基础URL
const API_URL = 'http://localhost:8080'

// 创建axios实例
const apiClient = axios.create({
  baseURL: API_URL,
  timeout: 10000
})

// axios请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = token // 添加认证Token
    }
    return config
  },
  (error) => {
    console.error('Request interceptor error:', error)
    return Promise.reject(error)
  }
)

// axios响应拦截器
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      // 处理401认证错误
      if (error.response.status === 401) {
        console.error('Authentication failed, please login again')
        // 自动登出并跳转到登录页面
        if (store && store.dispatch) {
          store.dispatch('logout')
          // 避免循环重定向：只有当前不在登录页时才跳转
          if (window.location.hash !== '#/login') {
            setTimeout(() => {
              window.location.hash = '#/login'
            }, 100)
          }
        }
      }

      // 处理其他错误
      console.error('API error:', error.response.status, error.response.data)
    } else if (error.request) {
      console.error('Request timeout or no response:', error.request)
    } else {
      console.error('Request configuration error:', error.message)
    }
    return Promise.reject(error)
  }
)

// 用户API
export const userApi = {
  // 用户登录
  login (credentials) {
    return apiClient.post('/api/login', credentials)
  },

  // 用户注册
  register (userData) {
    return apiClient.post('/api/register', userData)
  },

  // 验证Token
  checkAuth (token) {
    return apiClient.post('/api/check-auth', { token })
  },

  // 获取当前用户信息
  getProfile () {
    return apiClient.get('/api/users/profile')
  },

  // 用户登出
  logout () {
    return apiClient.post('/api/users/logout')
  },

  // 修改密码
  changePassword (passwordData) {
    return apiClient.post('/api/users/change-password', passwordData)
  },

  // 获取所有用户 (管理员)
  getUsers () {
    return apiClient.get('/api/users/admin/users')
  },

  // 获取指定用户信息 (管理员)
  getUserById (id) {
    return apiClient.get(`/api/users/admin/users/${id}`)
  },

  // 更新用户角色 (管理员)
  updateUserRole (id, role) {
    return apiClient.put(`/api/users/admin/users/${id}/role`, { role })
  },

  // 更新用户名 (管理员)
  updateUsername (id, username) {
    return apiClient.put(`/api/users/admin/users/${id}/username`, { username })
  },

  // 管理员修改用户密码
  adminUpdatePassword (id, newPassword) {
    return apiClient.put(`/api/users/admin/users/${id}/password`, {
      newPassword
    })
  },

  // 删除用户 (管理员)
  deleteUser (id) {
    return apiClient.delete(`/api/users/admin/users/${id}`)
  }
}

// 账户管理API
export const accountApi = {
  // 根据地址查询账户
  getAccountByAddress (address) {
    return apiClient.get(`/api/accounts/address/${address}`)
  },

  // 获取用户账户列表 (警员+)
  getUserAccounts () {
    return apiClient.get('/api/accounts/officer/')
  },

  // 创建账户 (警员+)
  createAccount (accountData) {
    return apiClient.post('/api/accounts/officer/create', accountData)
  },

  // 批量导入账户 (警员+)
  importAccounts (accountsData) {
    return apiClient.post('/api/accounts/officer/import', accountsData)
  },

  // 获取所有账户 (管理员)
  getAllAccounts () {
    return apiClient.get('/api/accounts/admin/all')
  }
}

// 交易管理API
export const transactionApi = {
  // 获取账户余额
  getBalance (address) {
    return apiClient.get(`/api/transaction/balance/${address}`)
  },

  // 准备交易 (警员+)
  prepareTransaction (transactionData) {
    return apiClient.post('/api/transaction/tx/prepare', transactionData)
  },

  // 签名并发送交易 (警员+)
  signAndSendTransaction (signData) {
    return apiClient.post('/api/transaction/tx/sign-send', signData)
  },

  // 获取交易列表 (警员+)
  getTransactions (params = {}) {
    return apiClient.get('/api/transaction/list', { params })
  },

  // 获取所有交易 (管理员)
  getAllTransactions (params = {}) {
    return apiClient.get('/api/transaction/admin/all', { params })
  },

  // 获取交易详情
  getTransactionById (id) {
    return apiClient.get(`/api/transaction/${id}`)
  },

  // 获取交易统计 (警员+)
  getTransactionStats () {
    return apiClient.get('/api/transaction/stats')
  },

  // 获取所有交易统计 (管理员)
  getAllTransactionStats () {
    return apiClient.get('/api/transaction/admin/stats')
  }
}

// 导出apiClient供其他模块使用
export { apiClient }
