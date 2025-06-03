<template>
    <div class="accounts-container">
        <el-card>
            <div slot="header" class="clearfix">
                <span>账户管理</span>
                <div style="float: right;">
                    <el-button type="primary" size="small" @click="showCreateDialog">
                        创建账户
                    </el-button>
                    <el-button type="success" size="small" @click="showImportDialog">
                        批量导入
                    </el-button>
                    <el-button type="text" @click="refreshAccountList">
                        刷新
                    </el-button>
                </div>
            </div>

            <!-- 搜索区域 -->
            <div class="search-area">
                <el-row :gutter="20">
                    <el-col :span="8">
                        <el-input v-model="searchAddress" placeholder="搜索账户地址" clearable>
                            <el-button slot="append" icon="el-icon-search" @click="searchAccount"></el-button>
                        </el-input>
                    </el-col>
                    <el-col :span="4">
                        <el-select v-model="filterCoinType" placeholder="币种筛选" clearable @change="handleFilter">
                            <el-option label="ETH" value="ETH"></el-option>
                            <el-option label="BTC" value="BTC"></el-option>
                            <el-option label="USDT" value="USDT"></el-option>
                        </el-select>
                    </el-col>
                </el-row>
            </div>

            <el-table :data="filteredAccountList" v-loading="loading" style="width: 100%">
                <el-table-column prop="id" label="ID" width="80"></el-table-column>
                <el-table-column prop="address" label="地址" width="280">
                    <template slot-scope="scope">
                        <span :title="scope.row.address">{{ formatAddress(scope.row.address) }}</span>
                        <el-button type="text" size="mini" @click="copyAddress(scope.row.address)">复制</el-button>
                    </template>
                </el-table-column>
                <el-table-column prop="coinType" label="币种" width="100">
                    <template slot-scope="scope">
                        <el-tag :type="getCoinTypeTagType(scope.row.coinType)">{{ scope.row.coinType }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="balance" label="余额" width="150">
                    <template slot-scope="scope">
                        <span>{{ scope.row.balance || '0' }} {{ scope.row.coinType }}</span>
                        <el-button type="text" size="mini" @click="refreshBalance(scope.row)">刷新</el-button>
                    </template>
                </el-table-column>
                <el-table-column prop="importedBy" label="导入者" width="120"></el-table-column>
                <el-table-column prop="description" label="描述" show-overflow-tooltip></el-table-column>
                <el-table-column label="操作" width="160">
                    <template slot-scope="scope">
                        <el-button type="primary" size="mini" @click="viewAccountDetail(scope.row)">
                            详情
                        </el-button>
                        <el-button type="danger" size="mini" @click="deleteAccount(scope.row)" style="margin-left: 5px">
                            删除
                        </el-button>
                    </template>
                </el-table-column>
            </el-table>

            <div v-if="filteredAccountList.length === 0 && !loading" class="empty-state">
                <p>暂无账户数据</p>
                <el-button type="primary" size="small" @click="showCreateDialog">创建账户</el-button>
            </div>
        </el-card>

        <!-- 创建账户对话框 -->
        <el-dialog title="创建账户" :visible.sync="createDialogVisible" width="500px">
            <el-form :model="createForm" :rules="createRules" ref="createForm" label-width="100px">
                <el-form-item label="账户地址" prop="address">
                    <el-input v-model="createForm.address" placeholder="请输入账户地址"></el-input>
                </el-form-item>
                <el-form-item label="币种类型" prop="coinType">
                    <el-select v-model="createForm.coinType" placeholder="请选择币种" style="width: 100%">
                        <el-option label="ETH" value="ETH"></el-option>
                        <el-option label="BTC" value="BTC"></el-option>
                        <el-option label="USDT" value="USDT"></el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="描述信息" prop="description">
                    <el-input v-model="createForm.description" type="textarea" :rows="3" placeholder="请输入描述信息（可选）"></el-input>
                </el-form-item>
            </el-form>
            <div slot="footer" class="dialog-footer">
                <el-button @click="createDialogVisible = false">取 消</el-button>
                <el-button type="primary" @click="handleCreateAccount" :loading="createLoading">确 定</el-button>
            </div>
        </el-dialog>

        <!-- 批量导入对话框 -->
        <el-dialog title="批量导入账户" :visible.sync="importDialogVisible" width="600px">
            <div class="import-area">
                <p>请按以下格式输入账户信息（每行一个账户）：</p>
                <p><strong>格式：地址,币种,描述</strong></p>
                <p class="example">例如：0x1234567890abcdef...,ETH,主账户</p>
                <el-input
                    v-model="importText"
                    type="textarea"
                    :rows="8"
                    placeholder="地址1,币种1,描述1&#10;地址2,币种2,描述2&#10;...">
                </el-input>
            </div>
            <div slot="footer" class="dialog-footer">
                <el-button @click="importDialogVisible = false">取 消</el-button>
                <el-button type="primary" @click="handleImportAccounts" :loading="importLoading">导 入</el-button>
            </div>
        </el-dialog>

        <!-- 账户详情对话框 -->
        <el-dialog title="账户详情" :visible.sync="detailDialogVisible" width="500px">
            <div v-if="selectedAccount" class="account-detail">
                <el-descriptions :column="1" border>
                    <el-descriptions-item label="地址">
                        <span>{{ selectedAccount.address }}</span>
                        <el-button type="text" size="mini" @click="copyAddress(selectedAccount.address)">复制</el-button>
                    </el-descriptions-item>
                    <el-descriptions-item label="币种">
                        <el-tag :type="getCoinTypeTagType(selectedAccount.coinType)">{{ selectedAccount.coinType }}</el-tag>
                    </el-descriptions-item>
                    <el-descriptions-item label="余额">
                        {{ selectedAccount.balance || '0' }} {{ selectedAccount.coinType }}
                    </el-descriptions-item>
                    <el-descriptions-item label="导入者">{{ selectedAccount.importedBy }}</el-descriptions-item>
                    <el-descriptions-item label="描述">{{ selectedAccount.description || '无' }}</el-descriptions-item>
                </el-descriptions>
            </div>
        </el-dialog>
    </div>
</template>

<script>
import { accountApi, transactionApi } from '../services/api'

export default {
  name: 'Accounts',
  data () {
    return {
      accountList: [],
      filteredAccountList: [],
      loading: false,
      searchAddress: '',
      filterCoinType: '',
      createDialogVisible: false,
      importDialogVisible: false,
      detailDialogVisible: false,
      createLoading: false,
      importLoading: false,
      selectedAccount: null,
      importText: '',
      createForm: {
        address: '',
        coinType: '',
        description: ''
      },
      createRules: {
        address: [
          { required: true, message: '请输入账户地址', trigger: 'blur' },
          { min: 10, message: '地址长度不能少于10个字符', trigger: 'blur' }
        ],
        coinType: [
          { required: true, message: '请选择币种类型', trigger: 'change' }
        ]
      }
    }
  },
  created () {
    this.fetchAccountList()
  },
  methods: {
    async fetchAccountList () {
      this.loading = true
      try {
        let response

        // 根据用户权限获取账户列表
        if (this.$store.getters.isAdmin) {
          response = await accountApi.getAllAccounts()
        } else {
          response = await accountApi.getUserAccounts()
        }

        if (response.data.code === 200) {
          const accountsData = response.data.data
          let rawAccounts = []

          if (Array.isArray(accountsData)) {
            rawAccounts = accountsData
          } else if (accountsData && Array.isArray(accountsData.accounts)) {
            rawAccounts = accountsData.accounts
          } else {
            rawAccounts = []
          }

          // 转换属性名称为小写，匹配模板中的期望格式
          this.accountList = rawAccounts.map(account => ({
            id: account.ID,
            address: account.Address,
            coinType: account.CoinType,
            balance: account.Balance,
            importedBy: account.ImportedBy,
            description: account.Description,
            createdAt: account.CreatedAt,
            updatedAt: account.UpdatedAt
          }))

          this.filteredAccountList = [...this.accountList]
        } else {
          throw new Error(response.data.message || '获取账户列表失败')
        }
      } catch (error) {
        console.error('获取账户列表失败:', error)
        let errorMsg = '获取账户列表失败'
        if (error.response && error.response.data) {
          errorMsg = error.response.data.message || errorMsg
        } else if (error.message) {
          errorMsg = error.message
        }
        this.$message.error(errorMsg)
      } finally {
        this.loading = false
      }
    },

    searchAccount () {
      if (!this.searchAddress.trim()) {
        this.applyFilters()
        return
      }

      // 尝试通过API查询单个账户
      this.searchAccountByAddress(this.searchAddress.trim())
    },

    async searchAccountByAddress (address) {
      try {
        const response = await accountApi.getAccountByAddress(address)
        if (response.data.code === 200) {
          const account = response.data.data
          // 转换属性名称为小写，匹配模板中的期望格式
          const transformedAccount = {
            id: account.ID,
            address: account.Address,
            coinType: account.CoinType,
            balance: account.Balance,
            importedBy: account.ImportedBy,
            description: account.Description,
            createdAt: account.CreatedAt,
            updatedAt: account.UpdatedAt
          }
          this.filteredAccountList = [transformedAccount]
          this.$message.success('找到匹配的账户')
        } else {
          this.$message.warning('未找到该地址的账户')
          this.filteredAccountList = []
        }
      } catch (error) {
        console.error('搜索账户失败:', error)
        this.$message.error('搜索账户失败')
        this.applyFilters()
      }
    },

    handleFilter () {
      this.applyFilters()
    },

    applyFilters () {
      let filtered = [...this.accountList]

      // 按币种筛选
      if (this.filterCoinType) {
        filtered = filtered.filter(account => account.coinType === this.filterCoinType)
      }

      // 按地址搜索
      if (this.searchAddress.trim()) {
        const searchTerm = this.searchAddress.trim().toLowerCase()
        filtered = filtered.filter(account =>
          account.address.toLowerCase().includes(searchTerm)
        )
      }

      this.filteredAccountList = filtered
    },

    showCreateDialog () {
      this.createForm = {
        address: '',
        coinType: '',
        description: ''
      }
      this.createDialogVisible = true
    },

    async handleCreateAccount () {
      this.$refs.createForm.validate(async valid => {
        if (!valid) {
          return false
        }

        this.createLoading = true

        try {
          const response = await accountApi.createAccount({
            address: this.createForm.address,
            coinType: this.createForm.coinType,
            description: this.createForm.description
          })

          if (response.data.code === 200) {
            this.$message.success('账户创建成功')
            this.createDialogVisible = false
            this.fetchAccountList()
          } else {
            throw new Error(response.data.message || '创建账户失败')
          }
        } catch (error) {
          console.error('创建账户失败:', error)
          let errorMsg = '创建账户失败'
          if (error.response && error.response.data) {
            errorMsg = error.response.data.message || errorMsg
          } else if (error.message) {
            errorMsg = error.message
          }
          this.$message.error(errorMsg)
        } finally {
          this.createLoading = false
        }
      })
    },

    showImportDialog () {
      this.importText = ''
      this.importDialogVisible = true
    },

    async handleImportAccounts () {
      if (!this.importText.trim()) {
        this.$message.warning('请输入要导入的账户信息')
        return
      }

      this.importLoading = true

      try {
        // 解析导入文本
        const lines = this.importText.trim().split('\n')
        const accounts = []

        for (const line of lines) {
          const parts = line.split(',').map(part => part.trim())
          if (parts.length >= 2) {
            accounts.push({
              address: parts[0],
              coinType: parts[1],
              description: parts[2] || ''
            })
          }
        }

        if (accounts.length === 0) {
          this.$message.warning('没有有效的账户信息')
          return
        }

        const response = await accountApi.importAccounts({ accounts })

        if (response.data.code === 200) {
          this.$message.success(`成功导入 ${accounts.length} 个账户`)
          this.importDialogVisible = false
          this.fetchAccountList()
        } else {
          throw new Error(response.data.message || '批量导入失败')
        }
      } catch (error) {
        console.error('批量导入失败:', error)
        let errorMsg = '批量导入失败'
        if (error.response && error.response.data) {
          errorMsg = error.response.data.message || errorMsg
        } else if (error.message) {
          errorMsg = error.message
        }
        this.$message.error(errorMsg)
      } finally {
        this.importLoading = false
      }
    },

    async refreshBalance (account) {
      try {
        const response = await transactionApi.getBalance(account.address)
        if (response.data.code === 200) {
          // 更新本地余额显示
          const index = this.accountList.findIndex(item => item.address === account.address)
          if (index !== -1) {
            this.$set(this.accountList[index], 'balance', response.data.data.balance)
            this.applyFilters() // 重新应用筛选
          }
          this.$message.success('余额已刷新')
        }
      } catch (error) {
        console.error('刷新余额失败:', error)
        this.$message.error('刷新余额失败')
      }
    },

    viewAccountDetail (account) {
      this.selectedAccount = account
      this.detailDialogVisible = true
    },

    copyAddress (address) {
      if (navigator.clipboard) {
        navigator.clipboard.writeText(address).then(() => {
          this.$message.success('地址已复制到剪贴板')
        }).catch(() => {
          this.fallbackCopyTextToClipboard(address)
        })
      } else {
        this.fallbackCopyTextToClipboard(address)
      }
    },

    fallbackCopyTextToClipboard (text) {
      const textArea = document.createElement('textarea')
      textArea.value = text
      document.body.appendChild(textArea)
      textArea.focus()
      textArea.select()
      try {
        document.execCommand('copy')
        this.$message.success('地址已复制到剪贴板')
      } catch (err) {
        this.$message.error('复制失败，请手动复制')
      }
      document.body.removeChild(textArea)
    },

    async deleteAccount (account) {
      try {
        const confirm = await this.$confirm(
          `确定要删除账户 "${this.formatAddress(account.address)}" 吗？此操作不可撤销。`,
          '确认删除',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )

        if (confirm) {
          const response = await accountApi.deleteAccount(account.id)
          if (response.data.code === 200) {
            this.$message.success('账户删除成功')
            this.fetchAccountList() // 刷新列表
          } else {
            throw new Error(response.data.message || '删除账户失败')
          }
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('删除账户失败:', error)
          let errorMsg = '删除账户失败'
          if (error.response && error.response.data) {
            errorMsg = error.response.data.message || errorMsg
          } else if (error.message) {
            errorMsg = error.message
          }
          this.$message.error(errorMsg)
        }
      }
    },

    refreshAccountList () {
      this.searchAddress = ''
      this.filterCoinType = ''
      this.fetchAccountList()
    },

    formatAddress (address) {
      if (!address || address.length <= 20) return address
      return `${address.slice(0, 10)}...${address.slice(-8)}`
    },

    getCoinTypeTagType (coinType) {
      const typeMap = {
        ETH: 'primary',
        BTC: 'warning',
        USDT: 'success'
      }
      return typeMap[coinType] || 'info'
    }
  },
  watch: {
    searchAddress () {
      if (!this.searchAddress.trim()) {
        this.applyFilters()
      }
    }
  }
}
</script>

<style scoped>
.accounts-container {
    padding: 20px;
}

.search-area {
    margin-bottom: 20px;
}

.empty-state {
    text-align: center;
    padding: 40px 0;
    color: #606266;
}

.dialog-footer {
    text-align: right;
}

.import-area {
    margin-bottom: 20px;
}

.import-area p {
    margin: 10px 0;
    color: #606266;
}

.import-area .example {
    color: #909399;
    font-size: 12px;
}

.account-detail {
    padding: 10px 0;
}
</style>
