import axios from 'axios'
import store from '../store'

// API基础URL - 支持环境变量配置
const API_URL = process.env.VUE_APP_API_BASE_URL || 'http://192.168.192.1:22221'

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
  // 账户分页查询
  getAccounts (params = {}) {
    return apiClient.get('/api/accounts', { params })
  },

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
    return apiClient.post('/api/accounts', accountData)
  },

  // 批量导入账户 (警员+)
  importAccounts (accountsData) {
    return apiClient.post('/api/accounts/import', accountsData)
  },

  // 获取所有账户 (管理员)
  getAllAccounts () {
    return apiClient.get('/api/accounts/admin/all')
  },

  // 删除账户 (管理员)
  deleteAccount (id) {
    return apiClient.delete(`/api/accounts/admin/${id}`)
  },

  syncBalance (id) {
    return apiClient.post(`/api/accounts/${id}/sync-balance`)
  },

  exportAccounts () {
    return apiClient.get('/api/accounts/export')
  },

  getTemplate () {
    return apiClient.get('/api/accounts/template')
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
  },

  // 删除交易 (管理员)
  deleteTransaction (id) {
    return apiClient.delete(`/api/transaction/admin/${id}`)
  },

  createDraft (data) {
    return apiClient.post('/api/transactions', data)
  },

  getTransactionPage (params = {}) {
    return apiClient.get('/api/transactions', { params })
  },

  prepareById (id) {
    return apiClient.post(`/api/transactions/${id}/prepare`)
  },

  exportSignTask (id) {
    return apiClient.get(`/api/transactions/${id}/export-sign-task`)
  },

  importSignature (id, data) {
    return apiClient.post(`/api/transactions/${id}/import-signature`, data)
  },

  broadcast (id) {
    return apiClient.post(`/api/transactions/${id}/broadcast`)
  },

  checkReceipt (id) {
    return apiClient.post(`/api/transactions/${id}/check-receipt`)
  }
}

export const caseApi = {
  list (params = {}) {
    return apiClient.get('/api/cases', { params })
  },
  create (data) {
    return apiClient.post('/api/cases', data)
  },
  update (id, data) {
    return apiClient.put(`/api/cases/${id}`, data)
  },
  remove (id) {
    return apiClient.delete(`/api/cases/${id}`)
  },
  accounts (id) {
    return apiClient.get(`/api/cases/${id}/accounts`)
  },
  linkAccount (id, accountId) {
    return apiClient.post(`/api/cases/${id}/accounts`, { accountId })
  },
  unlinkAccount (id, accountId) {
    return apiClient.delete(`/api/cases/${id}/accounts/${accountId}`)
  },
  importCustodyWallet (id, data) {
    return apiClient.post(`/api/cases/${id}/custody-wallet/import-result`, data)
  }
}

export const offlineTaskApi = {
  list (params = {}) {
    return apiClient.get('/api/offline-tasks', { params })
  },
  createCustodyKeygen (data) {
    return apiClient.post('/api/offline-tasks/custody-keygen', data)
  },
  exportTask (id) {
    return apiClient.get(`/api/offline-tasks/${id}/export`)
  },
  importResult (id, result) {
    return apiClient.post(`/api/offline-tasks/${id}/import-result`, { result })
  }
}

export const auditApi = {
  list (params = {}) {
    return apiClient.get('/api/audit-logs', { params })
  },
  export () {
    return apiClient.get('/api/audit-logs/export')
  }
}

export const backupApi = {
  list () {
    return apiClient.get('/api/backups')
  },
  createHot () {
    return apiClient.post('/api/backups/hot')
  },
  createCold (password) {
    return apiClient.post('/api/backups/cold/export', { password })
  },
  verify (id) {
    return apiClient.post(`/api/backups/${id}/verify`)
  },
  download (id) {
    return apiClient.get(`/api/backups/${id}/download`, { responseType: 'blob' })
  },
  restore (id, password = '') {
    return apiClient.post(`/api/backups/${id}/restore`, { password })
  }
}

export const testDataApi = {
  seed (data) {
    return apiClient.post('/api/test-data/seed', data)
  },
  clear () {
    return apiClient.post('/api/test-data/clear')
  },
  summary () {
    return apiClient.get('/api/test-data/summary')
  }
}

// 导出apiClient供其他模块使用
export { apiClient }
