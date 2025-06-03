<template>
    <div class="transactions-container">
        <el-card>
            <div slot="header" class="clearfix">
                <span>交易管理</span>
                <el-button style="float: right; padding: 3px 0" type="text" @click="fetchTransactions">
                    刷新
                </el-button>
            </div>

            <!-- 发起交易区域 -->
            <div class="action-area">
                <el-row :gutter="20">
                    <el-col :span="18">
                        <span class="transaction-summary">共 {{ transactionList.length }} 笔交易</span>
                    </el-col>
                    <el-col :span="6" style="text-align: right;">
                        <el-button type="primary" @click="showCreateTransactionDialog">
                            <i class="el-icon-plus"></i> 发起交易
                        </el-button>
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
                        {{ scope.row.amount }}
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
                <el-table-column label="操作" width="220">
                    <template slot-scope="scope">
                        <el-button v-if="scope.row.status === 'prepared'" type="warning" size="mini" @click="signTransaction(scope.row)">
                            操作
                        </el-button>
                        <el-button type="primary" size="mini" @click="viewTransactionDetail(scope.row)">
                            详情
                        </el-button>
                        <el-button type="danger" size="mini" @click="deleteTransaction(scope.row)" style="margin-left: 5px">
                            删除
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
                    <el-descriptions-item label="发送方地址">
                        <div class="address-row">
                            <span class="address-text">{{ selectedTransaction.fromAddress }}</span>
                            <div class="address-actions">
                                <el-button type="text" size="mini" @click="copyText(selectedTransaction.fromAddress)">
                                    <i class="el-icon-document-copy"></i> 复制
                                </el-button>
                            </div>
                        </div>
                    </el-descriptions-item>
                    <el-descriptions-item label="接收方地址">
                        <div class="address-row">
                            <span class="address-text">{{ selectedTransaction.toAddress }}</span>
                            <div class="address-actions">
                                <el-button type="text" size="mini" @click="copyText(selectedTransaction.toAddress)">
                                    <i class="el-icon-document-copy"></i> 复制
                                </el-button>
                            </div>
                        </div>
                    </el-descriptions-item>
                    <el-descriptions-item label="转账金额">{{ selectedTransaction.amount }}</el-descriptions-item>
                    <el-descriptions-item label="状态">
                        <el-tag :type="getStatusTagType(selectedTransaction.status)">{{ getStatusText(selectedTransaction.status) }}</el-tag>
                    </el-descriptions-item>
                    <el-descriptions-item label="消息哈希" v-if="selectedTransaction.messageHash">
                        <div class="hash-row">
                            <span class="hash-text">{{ selectedTransaction.messageHash }}</span>
                            <div class="hash-actions">
                                <el-button type="text" size="mini" @click="copyText(selectedTransaction.messageHash)">
                                    <i class="el-icon-document-copy"></i> 复制
                                </el-button>
                            </div>
                        </div>
                    </el-descriptions-item>
                    <el-descriptions-item label="交易哈希" v-if="selectedTransaction.txHash">
                        <div class="hash-row">
                            <span class="hash-text">{{ selectedTransaction.txHash }}</span>
                            <div class="hash-actions">
                                <el-button type="text" size="mini" @click="copyText(selectedTransaction.txHash)">
                                    <i class="el-icon-document-copy"></i> 复制
                                </el-button>
                            </div>
                        </div>
                    </el-descriptions-item>
                    <el-descriptions-item label="创建时间">{{ formatDateTime(selectedTransaction.createdAt) }}</el-descriptions-item>
                    <el-descriptions-item label="更新时间" v-if="selectedTransaction.updatedAt">{{ formatDateTime(selectedTransaction.updatedAt) }}</el-descriptions-item>
                </el-descriptions>

                <!-- 操作按钮区域 -->
                <div class="detail-actions">
                    <el-button type="primary" @click="downloadTransactionInfo">
                        <i class="el-icon-download"></i> 下载交易信息
                    </el-button>
                    <el-button @click="copyAllTransactionInfo">
                        <i class="el-icon-document-copy"></i> 复制全部信息
                    </el-button>
                </div>
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
    },

    // 复制全部交易信息
    copyAllTransactionInfo () {
      if (!this.selectedTransaction) return

      const info = this.formatTransactionInfo(this.selectedTransaction)
      this.copyText(info)
    },

    // 下载交易信息
    downloadTransactionInfo () {
      if (!this.selectedTransaction) return

      const info = this.formatTransactionInfo(this.selectedTransaction)
      const filename = `transaction_${this.selectedTransaction.id}_${new Date().getTime()}.txt`

      // 创建下载链接
      const blob = new Blob([info], { type: 'text/plain;charset=utf-8' })
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      this.$message.success('交易信息已下载')
    },

    // 删除交易
    async deleteTransaction (transaction) {
      try {
        const confirm = await this.$confirm(
          `确定要删除交易 ID "${transaction.id}" 吗？此操作不可撤销。`,
          '确认删除',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )

        if (confirm) {
          const response = await transactionApi.deleteTransaction(transaction.id)
          if (response.data.code === 200) {
            this.$message.success('交易删除成功')
            this.fetchTransactions() // 刷新列表
          } else {
            throw new Error(response.data.message || '删除交易失败')
          }
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('删除交易失败:', error)
          let errorMsg = '删除交易失败'
          if (error.response && error.response.data) {
            errorMsg = error.response.data.message || errorMsg
          } else if (error.message) {
            errorMsg = error.message
          }
          this.$message.error(errorMsg)
        }
      }
    },

    // 格式化交易信息
    formatTransactionInfo (transaction) {
      let info = '交易详情信息\n'
      info += '================\n\n'
      info += `交易ID: ${transaction.id}\n`
      info += `发送方地址: ${transaction.fromAddress}\n`
      info += `接收方地址: ${transaction.toAddress}\n`
      info += `转账金额: ${transaction.amount}\n`
      info += `状态: ${this.getStatusText(transaction.status)}\n`
      if (transaction.messageHash) {
        info += `消息哈希: ${transaction.messageHash}\n`
      }
      if (transaction.txHash) {
        info += `交易哈希: ${transaction.txHash}\n`
      }
      info += `创建时间: ${this.formatDateTime(transaction.createdAt)}\n`
      if (transaction.updatedAt) {
        info += `更新时间: ${this.formatDateTime(transaction.updatedAt)}\n`
      }
      info += '\n生成时间: ' + new Date().toLocaleString('zh-CN')

      return info
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

.action-area {
    margin-bottom: 20px;
    padding: 15px 0;
    border-bottom: 1px solid #f0f0f0;
}

.transaction-summary {
    font-size: 14px;
    color: #666;
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

.address-row,
.hash-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
}

.address-text,
.hash-text {
    flex: 1;
    word-break: break-all;
    margin-right: 10px;
    font-family: 'Courier New', monospace;
    font-size: 13px;
}

.address-actions,
.hash-actions {
    flex-shrink: 0;
}

.detail-actions {
    margin-top: 20px;
    text-align: center;
    padding-top: 15px;
    border-top: 1px solid #f0f0f0;
}

.detail-actions .el-button {
    margin: 0 10px;
}
</style>
