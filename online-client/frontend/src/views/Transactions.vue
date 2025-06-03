<template>
    <div class="transactions-container">
        <el-card>
            <div slot="header" class="clearfix">
                <span>交易管理</span>
                <el-button style="float: right; padding: 3px 0" type="primary" size="small" @click="showCreateTransactionDialog">
                    发起交易
                </el-button>
            </div>

            <!-- 搜索区域 -->
            <div class="search-area">
                <el-row :gutter="20">
                    <el-col :span="6">
                        <el-input v-model="searchFromAddress" placeholder="发送方地址" clearable>
                            <template slot="prepend">发送方</template>
                        </el-input>
                    </el-col>
                    <el-col :span="6">
                        <el-input v-model="searchToAddress" placeholder="接收方地址" clearable>
                            <template slot="prepend">接收方</template>
                        </el-input>
                    </el-col>
                    <el-col :span="4">
                        <el-select v-model="statusFilter" placeholder="状态筛选" clearable>
                            <el-option label="准备中" value="prepared"></el-option>
                            <el-option label="已签名" value="signed"></el-option>
                            <el-option label="已发送" value="sent"></el-option>
                            <el-option label="已确认" value="confirmed"></el-option>
                            <el-option label="失败" value="failed"></el-option>
                        </el-select>
                    </el-col>
                    <el-col :span="4">
                        <el-button type="primary" @click="searchTransactions">搜索</el-button>
                        <el-button @click="resetSearch">重置</el-button>
                    </el-col>
                </el-row>
            </div>

            <!-- 交易列表 -->
            <el-table :data="transactionList" v-loading="loading" style="width: 100%">
                <el-table-column prop="id" label="交易ID" width="80"></el-table-column>
                <el-table-column prop="fromAddress" label="发送方" width="200">
                    <template slot-scope="scope">
                        <span :title="scope.row.fromAddress">{{ formatAddress(scope.row.fromAddress) }}</span>
                    </template>
                </el-table-column>
                <el-table-column prop="toAddress" label="接收方" width="200">
                    <template slot-scope="scope">
                        <span :title="scope.row.toAddress">{{ formatAddress(scope.row.toAddress) }}</span>
                    </template>
                </el-table-column>
                <el-table-column prop="amount" label="金额" width="120">
                    <template slot-scope="scope">
                        {{ scope.row.amount }} ETH
                    </template>
                </el-table-column>
                <el-table-column prop="status" label="状态" width="100">
                    <template slot-scope="scope">
                        <el-tag :type="getStatusTagType(scope.row.status)">{{ getStatusText(scope.row.status) }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="txHash" label="交易哈希" width="200">
                    <template slot-scope="scope">
                        <span v-if="scope.row.txHash" :title="scope.row.txHash">{{ formatAddress(scope.row.txHash) }}</span>
                        <span v-else>-</span>
                    </template>
                </el-table-column>
                <el-table-column prop="createdAt" label="创建时间" width="160">
                    <template slot-scope="scope">
                        {{ formatDateTime(scope.row.createdAt) }}
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="160">
                    <template slot-scope="scope">
                        <el-button v-if="scope.row.status === 'prepared'" type="warning" size="mini" @click="signTransaction(scope.row)">
                            签名发送
                        </el-button>
                        <el-button type="primary" size="mini" @click="viewTransactionDetail(scope.row)">
                            详情
                        </el-button>
                    </template>
                </el-table-column>
            </el-table>

            <div v-if="transactionList.length === 0 && !loading" class="empty-state">
                <p>暂无交易记录</p>
                <el-button type="primary" size="small" @click="showCreateTransactionDialog">发起交易</el-button>
            </div>
        </el-card>

        <!-- 创建交易对话框 -->
        <el-dialog title="发起交易" :visible.sync="createDialogVisible" width="500px">
            <el-form :model="createForm" :rules="createRules" ref="createForm" label-width="100px">
                <el-form-item label="发送方地址" prop="fromAddress">
                    <el-select v-model="createForm.fromAddress" placeholder="请选择发送方地址" style="width: 100%">
                        <el-option
                            v-for="account in userAccounts"
                            :key="account.address"
                            :label="account.address"
                            :value="account.address">
                            <span>{{ formatAddress(account.address) }}</span>
                            <span style="float: right; color: #8492a6; font-size: 13px">{{ account.balance || '0' }} {{ account.coinType }}</span>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="当前余额">
                    <span v-if="selectedAccountBalance">{{ selectedAccountBalance }} ETH</span>
                    <span v-else>请选择发送方地址</span>
                    <el-button v-if="createForm.fromAddress" type="text" size="mini" @click="refreshSelectedBalance">刷新</el-button>
                </el-form-item>
                <el-form-item label="接收方地址" prop="toAddress">
                    <el-input v-model="createForm.toAddress" placeholder="请输入接收方地址"></el-input>
                </el-form-item>
                <el-form-item label="转账金额" prop="amount">
                    <el-input-number
                        v-model="createForm.amount"
                        :precision="6"
                        :step="0.001"
                        :min="0.001"
                        style="width: 100%">
                    </el-input-number>
                    <span style="margin-left: 10px;">ETH</span>
                </el-form-item>
            </el-form>
            <div slot="footer" class="dialog-footer">
                <el-button @click="createDialogVisible = false">取 消</el-button>
                <el-button type="primary" @click="handleCreateTransaction" :loading="createLoading">准备交易</el-button>
            </div>
        </el-dialog>

        <!-- 签名交易对话框 -->
        <el-dialog title="签名并发送交易" :visible.sync="signDialogVisible" width="500px">
            <div v-if="selectedTransaction" class="sign-transaction">
                <el-descriptions :column="1" border>
                    <el-descriptions-item label="交易ID">{{ selectedTransaction.id }}</el-descriptions-item>
                    <el-descriptions-item label="发送方">{{ selectedTransaction.fromAddress }}</el-descriptions-item>
                    <el-descriptions-item label="接收方">{{ selectedTransaction.toAddress }}</el-descriptions-item>
                    <el-descriptions-item label="金额">{{ selectedTransaction.amount }} ETH</el-descriptions-item>
                    <el-descriptions-item label="消息哈希">{{ selectedTransaction.messageHash }}</el-descriptions-item>
                </el-descriptions>

                <el-form :model="signForm" :rules="signRules" ref="signForm" label-width="100px" style="margin-top: 20px;">
                    <el-form-item label="签名数据" prop="signature">
                        <el-input v-model="signForm.signature" type="textarea" :rows="3" placeholder="请输入签名数据"></el-input>
                    </el-form-item>
                </el-form>
            </div>
            <div slot="footer" class="dialog-footer">
                <el-button @click="signDialogVisible = false">取 消</el-button>
                <el-button type="primary" @click="handleSignTransaction" :loading="signLoading">签名发送</el-button>
            </div>
        </el-dialog>

        <!-- 交易详情对话框 -->
        <el-dialog title="交易详情" :visible.sync="detailDialogVisible" width="600px">
            <div v-if="selectedTransaction" class="transaction-detail">
                <el-descriptions :column="1" border>
                    <el-descriptions-item label="交易ID">{{ selectedTransaction.id }}</el-descriptions-item>
                    <el-descriptions-item label="发送方地址">{{ selectedTransaction.fromAddress }}</el-descriptions-item>
                    <el-descriptions-item label="接收方地址">{{ selectedTransaction.toAddress }}</el-descriptions-item>
                    <el-descriptions-item label="转账金额">{{ selectedTransaction.amount }} ETH</el-descriptions-item>
                    <el-descriptions-item label="状态">
                        <el-tag :type="getStatusTagType(selectedTransaction.status)">{{ getStatusText(selectedTransaction.status) }}</el-tag>
                    </el-descriptions-item>
                    <el-descriptions-item label="消息哈希" v-if="selectedTransaction.messageHash">
                        <span>{{ selectedTransaction.messageHash }}</span>
                        <el-button type="text" size="mini" @click="copyText(selectedTransaction.messageHash)">复制</el-button>
                    </el-descriptions-item>
                    <el-descriptions-item label="交易哈希" v-if="selectedTransaction.txHash">
                        <span>{{ selectedTransaction.txHash }}</span>
                        <el-button type="text" size="mini" @click="copyText(selectedTransaction.txHash)">复制</el-button>
                    </el-descriptions-item>
                    <el-descriptions-item label="创建时间">{{ formatDateTime(selectedTransaction.createdAt) }}</el-descriptions-item>
                    <el-descriptions-item label="更新时间" v-if="selectedTransaction.updatedAt">{{ formatDateTime(selectedTransaction.updatedAt) }}</el-descriptions-item>
                </el-descriptions>
            </div>
        </el-dialog>
    </div>
</template>

<script>
import { transactionApi, accountApi } from '../services/api'

export default {
  name: 'Transactions',
  data () {
    return {
      transactionList: [],
      userAccounts: [],
      loading: false,
      searchFromAddress: '',
      searchToAddress: '',
      statusFilter: '',
      createDialogVisible: false,
      signDialogVisible: false,
      detailDialogVisible: false,
      createLoading: false,
      signLoading: false,
      selectedTransaction: null,
      selectedAccountBalance: '',
      createForm: {
        fromAddress: '',
        toAddress: '',
        amount: null
      },
      createRules: {
        fromAddress: [
          { required: true, message: '请选择发送方地址', trigger: 'change' }
        ],
        toAddress: [
          { required: true, message: '请输入接收方地址', trigger: 'blur' },
          { min: 10, message: '地址长度不能少于10个字符', trigger: 'blur' }
        ],
        amount: [
          { required: true, message: '请输入转账金额', trigger: 'blur' },
          { type: 'number', min: 0.000001, message: '转账金额必须大于0', trigger: 'blur' }
        ]
      },
      signForm: {
        signature: ''
      },
      signRules: {
        signature: [
          { required: true, message: '请输入签名数据', trigger: 'blur' }
        ]
      }
    }
  },
  created () {
    this.fetchUserAccounts()
    this.fetchTransactions()
  },
  methods: {
    async fetchUserAccounts () {
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
          this.userAccounts = rawAccounts.map(account => ({
            id: account.ID,
            address: account.Address,
            coinType: account.CoinType,
            balance: account.Balance,
            importedBy: account.ImportedBy,
            description: account.Description,
            createdAt: account.CreatedAt,
            updatedAt: account.UpdatedAt
          }))
        }
      } catch (error) {
        console.error('获取账户列表失败:', error)
      }
    },

    async fetchTransactions () {
      this.loading = true
      try {
        // 构建查询参数
        const params = {}
        if (this.searchFromAddress.trim()) {
          params.fromAddress = this.searchFromAddress.trim()
        }
        if (this.searchToAddress.trim()) {
          params.toAddress = this.searchToAddress.trim()
        }
        if (this.statusFilter) {
          params.status = this.statusFilter
        }

        let response

        // 根据用户权限获取交易列表
        if (this.$store.getters.isAdmin) {
          response = await transactionApi.getAllTransactions(params)
        } else {
          response = await transactionApi.getTransactions(params)
        }

        if (response.data.code === 200) {
          this.transactionList = response.data.data.transactions || response.data.data || []
        } else {
          throw new Error(response.data.message || '获取交易列表失败')
        }
      } catch (error) {
        console.error('获取交易列表失败:', error)
        let errorMsg = '获取交易列表失败'
        if (error.response && error.response.data) {
          errorMsg = error.response.data.message || errorMsg
        } else if (error.message) {
          errorMsg = error.message
        }
        this.$message.error(errorMsg)
        // 如果API调用失败，设置空列表
        this.transactionList = []
      } finally {
        this.loading = false
      }
    },

    searchTransactions () {
      // 实现搜索逻辑
      this.fetchTransactions()
    },

    resetSearch () {
      this.searchFromAddress = ''
      this.searchToAddress = ''
      this.statusFilter = ''
      this.fetchTransactions()
    },

    showCreateTransactionDialog () {
      this.createForm = {
        fromAddress: '',
        toAddress: '',
        amount: null
      }
      this.selectedAccountBalance = ''
      this.createDialogVisible = true
    },

    async handleCreateTransaction () {
      this.$refs.createForm.validate(async valid => {
        if (!valid) {
          return false
        }

        this.createLoading = true

        try {
          const response = await transactionApi.prepareTransaction({
            fromAddress: this.createForm.fromAddress,
            toAddress: this.createForm.toAddress,
            amount: this.createForm.amount
          })

          if (response.data.code === 200) {
            this.$message.success('交易准备成功')
            this.createDialogVisible = false
            this.fetchTransactions()
          } else {
            throw new Error(response.data.message || '准备交易失败')
          }
        } catch (error) {
          console.error('准备交易失败:', error)
          let errorMsg = '准备交易失败'
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

    signTransaction (transaction) {
      this.selectedTransaction = transaction
      this.signForm.signature = ''
      this.signDialogVisible = true
    },

    async handleSignTransaction () {
      this.$refs.signForm.validate(async valid => {
        if (!valid) {
          return false
        }

        this.signLoading = true

        try {
          const response = await transactionApi.signAndSendTransaction({
            messageHash: this.selectedTransaction.messageHash,
            signature: this.signForm.signature
          })

          if (response.data.code === 200) {
            this.$message.success('交易签名并发送成功')
            this.signDialogVisible = false
            this.fetchTransactions()
          } else {
            throw new Error(response.data.message || '交易签名失败')
          }
        } catch (error) {
          console.error('交易签名失败:', error)
          let errorMsg = '交易签名失败'
          if (error.response && error.response.data) {
            errorMsg = error.response.data.message || errorMsg
          } else if (error.message) {
            errorMsg = error.message
          }
          this.$message.error(errorMsg)
        } finally {
          this.signLoading = false
        }
      })
    },

    viewTransactionDetail (transaction) {
      this.selectedTransaction = transaction
      this.detailDialogVisible = true
    },

    async refreshSelectedBalance () {
      if (!this.createForm.fromAddress) return

      try {
        const response = await transactionApi.getBalance(this.createForm.fromAddress)
        if (response.data.code === 200) {
          this.selectedAccountBalance = response.data.data.balance
        }
      } catch (error) {
        console.error('获取余额失败:', error)
        this.$message.error('获取余额失败')
      }
    },

    copyText (text) {
      if (navigator.clipboard) {
        navigator.clipboard.writeText(text).then(() => {
          this.$message.success('已复制到剪贴板')
        }).catch(() => {
          this.fallbackCopyTextToClipboard(text)
        })
      } else {
        this.fallbackCopyTextToClipboard(text)
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
        this.$message.success('已复制到剪贴板')
      } catch (err) {
        this.$message.error('复制失败，请手动复制')
      }
      document.body.removeChild(textArea)
    },

    formatAddress (address) {
      if (!address || address.length <= 20) return address
      return `${address.slice(0, 10)}...${address.slice(-8)}`
    },

    formatDateTime (dateString) {
      if (!dateString) return '-'
      return new Date(dateString).toLocaleString('zh-CN')
    },

    getStatusText (status) {
      const statusMap = {
        prepared: '准备中',
        signed: '已签名',
        sent: '已发送',
        confirmed: '已确认',
        failed: '失败'
      }
      return statusMap[status] || status
    },

    getStatusTagType (status) {
      const typeMap = {
        prepared: 'warning',
        signed: 'primary',
        sent: 'info',
        confirmed: 'success',
        failed: 'danger'
      }
      return typeMap[status] || 'info'
    }
  },
  watch: {
    'createForm.fromAddress' () {
      this.refreshSelectedBalance()
    }
  }
}
</script>

<style scoped>
.transactions-container {
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

.sign-transaction,
.transaction-detail {
    padding: 10px 0;
}
</style>
