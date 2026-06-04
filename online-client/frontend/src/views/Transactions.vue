<template>
  <div class="page">
    <el-card>
      <div slot="header" class="header">
        <span>交易管理</span>
        <div>
          <el-button v-if="canWrite" size="small" type="primary" @click="openCreate">新建交易</el-button>
          <el-button v-if="canWrite" size="small" type="success" @click="openTransactionImport">导入交易 JSON</el-button>
          <el-button size="small" @click="fetchTransactions">刷新</el-button>
        </div>
      </div>

      <el-form :inline="true" class="query">
        <el-form-item>
          <el-select v-model="query.caseNo" filterable clearable placeholder="案件编号" @change="fetchTransactions">
            <el-option v-for="item in caseOptions" :key="item.CaseNo" :label="caseLabel(item)" :value="item.CaseNo" />
          </el-select>
        </el-form-item>
        <el-form-item><el-input v-model="query.address" placeholder="地址" clearable /></el-form-item>
        <el-form-item>
          <el-select v-model="query.status" placeholder="状态" clearable>
            <el-option label="待签名" value="pending_signature" />
            <el-option label="已导出" value="signature_exported" />
            <el-option label="已签名" value="signed" />
            <el-option label="已广播" value="broadcasted" />
            <el-option label="已确认" value="confirmed" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item><el-button type="primary" @click="fetchTransactions">查询</el-button></el-form-item>
      </el-form>

      <el-table :data="transactions" v-loading="loading">
        <el-table-column prop="TxNo" label="交易编号" width="145" />
        <el-table-column prop="CaseNo" label="案件编号" width="115" show-overflow-tooltip />
        <el-table-column prop="TxType" label="类型" width="60">
          <template slot-scope="s">{{ txTypeText(s.row.TxType) }}</template>
        </el-table-column>
        <el-table-column prop="FromAddress" label="发送方" width="125"><template slot-scope="s">{{ short(s.row.FromAddress) }}</template></el-table-column>
        <el-table-column prop="ToAddress" label="接收方" width="125"><template slot-scope="s">{{ short(s.row.ToAddress) }}</template></el-table-column>
        <el-table-column prop="Value" label="金额" width="90" show-overflow-tooltip />
        <el-table-column prop="Status" label="状态" width="75"><template slot-scope="s"><el-tag size="small">{{ statusText(s.row.Status) }}</el-tag></template></el-table-column>
        <el-table-column label="操作" width="190" fixed="right">
          <template slot-scope="scope">
            <div class="operation-actions">
              <el-button size="mini" @click="view(scope.row)">详情</el-button>
              <el-button v-if="canWrite" size="mini" type="primary" :loading="exportingTaskId === scope.row.ID" @click="exportTask(scope.row)">导出任务</el-button>
              <el-button v-if="canWrite" size="mini" type="success" @click="openSignature(scope.row)">导入签名</el-button>
              <el-button v-if="canWrite" size="mini" type="warning" :loading="broadcastingId === scope.row.ID" @click="broadcast(scope.row)">广播交易</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination layout="total, prev, pager, next" :total="total" :page-size="query.pageSize" :current-page.sync="query.page" @current-change="fetchTransactions" />
    </el-card>

    <el-dialog title="新建交易" :visible.sync="createDialog" width="560px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="案件编号">
          <el-select v-model="form.caseNo" filterable clearable allow-create default-first-option style="width:100%" placeholder="请选择案件，或手动输入案件编号" @change="handleFormCaseChange">
            <el-option v-for="item in caseOptions" :key="item.CaseNo" :label="caseLabel(item)" :value="item.CaseNo" />
          </el-select>
        </el-form-item>
        <el-form-item label="交易类型">
          <el-select v-model="form.txType" style="width:100%">
            <el-option label="归集" value="collect" />
            <el-option label="提取" value="withdraw" />
            <el-option label="测试" value="test" />
          </el-select>
        </el-form-item>
        <el-form-item label="发送方">
          <el-autocomplete
            v-model="form.fromAddress"
            :fetch-suggestions="queryAccountSuggestions"
            value-key="value"
            placeholder="请选择账户，或手动输入地址"
            style="width:100%"
            @select="selectFromAccount"
          />
        </el-form-item>
        <el-form-item label="接收方">
          <el-autocomplete
            v-model="form.toAddress"
            :fetch-suggestions="queryAccountSuggestions"
            value-key="value"
            placeholder="请选择账户，或手动输入地址"
            style="width:100%"
            @select="selectToAccount"
          />
        </el-form-item>
        <el-form-item label="金额"><el-input v-model="form.value" /></el-form-item>
        <el-form-item label="币种"><el-input v-model="form.coinType" /></el-form-item>
        <el-form-item label="事由"><el-input v-model="form.reason" type="textarea" /></el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="createDialog=false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createTx">创建并下载离线签名任务</el-button>
      </span>
    </el-dialog>

    <el-dialog title="导入交易 JSON" :visible.sync="transactionImportDialog" width="780px">
      <div class="import-tools">
        <div>
          <input ref="transactionImportFile" type="file" accept="application/json,.json" @change="handleTransactionImportFile">
          <div class="file-name" v-if="transactionImportFileName">{{ transactionImportFileName }}</div>
        </div>
        <el-button size="mini" @click="clearTransactionImport">清空</el-button>
      </div>
      <el-alert
        class="import-hint"
        type="info"
        :closable="false"
        title='支持 JSON 数组或 { "transactions": [...] }。导入交易只写入记录，不生成离线签名任务。'
      />
      <el-input
        v-model="transactionImportText"
        type="textarea"
        :rows="12"
        placeholder='例如: {"transactions":[{"caseNo":"CASE-DEMO-001","txType":"test","fromAddress":"0x...","toAddress":"0x...","value":"0.01 ETH","coinType":"ETH","reason":"本地 JSON 导入样例"}]}'
        @blur="updateTransactionImportPreview"
      />
      <div class="import-summary">{{ transactionImportSummary }}</div>
      <span slot="footer">
        <el-button @click="transactionImportDialog=false">取消</el-button>
        <el-button type="primary" :loading="transactionImporting" @click="batchImportTransactions">导入交易</el-button>
      </span>
    </el-dialog>

    <el-dialog title="导入离线签名结果" :visible.sync="signatureDialog" width="720px">
      <el-form :model="signatureForm" label-width="100px">
        <el-form-item label="结果包文件">
          <input ref="signatureResultFile" type="file" accept="application/json,.json" @change="handleSignatureResultFile">
          <div class="file-name" v-if="signatureResultFileName">{{ signatureResultFileName }}</div>
        </el-form-item>
        <el-form-item label="结果包 JSON">
          <el-input v-model="signatureResultText" type="textarea" :rows="8" placeholder="可粘贴离线端导出的 offline_result JSON，系统会自动识别消息哈希和签名" @blur="parseSignatureResult" />
        </el-form-item>
        <el-descriptions :column="1" border size="small" class="result-summary">
          <el-descriptions-item label="任务编号">{{ signatureForm.taskNo || '-' }}</el-descriptions-item>
          <el-descriptions-item label="消息哈希">{{ signatureForm.messageHash || '-' }}</el-descriptions-item>
          <el-descriptions-item label="签名">{{ short(signatureForm.signature) || '-' }}</el-descriptions-item>
        </el-descriptions>
      </el-form>
      <span slot="footer"><el-button @click="signatureDialog=false">取消</el-button><el-button type="primary" :loading="importingSignature" @click="importSignature">导入结果包</el-button></span>
    </el-dialog>

    <el-dialog title="交易详情" :visible.sync="detailDialog" width="720px">
      <el-descriptions v-if="selected" :column="1" border>
        <el-descriptions-item label="交易编号">{{ selected.TxNo }}</el-descriptions-item>
        <el-descriptions-item label="消息哈希">{{ selected.MessageHash || '-' }}</el-descriptions-item>
        <el-descriptions-item label="链上哈希">{{ selected.TxHash || '-' }}</el-descriptions-item>
        <el-descriptions-item label="事由">{{ selected.Reason || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script>
import { accountApi, caseApi, transactionApi } from '../services/api'

export default {
  name: 'Transactions',
  data () {
    return {
      loading: false,
      creating: false,
      exportingTaskId: 0,
      broadcastingId: 0,
      importingSignature: false,
      transactions: [],
      caseOptions: [],
      accountOptions: [],
      caseAccountOptions: [],
      total: 0,
      query: { page: 1, pageSize: 20, caseNo: '', address: '', status: '' },
      createDialog: false,
      transactionImportDialog: false,
      signatureDialog: false,
      detailDialog: false,
      selected: null,
      form: {},
      signatureForm: {},
      transactionImportText: '',
      transactionImportFileName: '',
      transactionImportCount: 0,
      transactionImporting: false,
      signatureResultText: '',
      signatureResultFileName: ''
    }
  },
  created () {
    this.fetchCases()
    this.fetchAccounts()
    this.fetchTransactions()
  },
  computed: {
    canWrite () {
      return this.$store.getters.isOfficer
    },
    transactionImportSummary () {
      return this.transactionImportCount > 0 ? `已解析 ${this.transactionImportCount} 条交易` : '尚未解析交易'
    }
  },
  methods: {
    async fetchTransactions () {
      this.loading = true
      try {
        const res = await transactionApi.getTransactionPage(this.query)
        const data = res.data.data || {}
        this.transactions = data.items || []
        this.total = data.total || 0
      } finally {
        this.loading = false
      }
    },
    async fetchCases () {
      try {
        const res = await caseApi.list({ page: 1, pageSize: 100, status: 'active' })
        const data = res.data.data || {}
        this.caseOptions = data.items || []
      } catch (error) {
        this.$message.error(this.apiError(error, '查询案件列表失败'))
      }
    },
    async fetchAccounts (caseId) {
      try {
        if (caseId) {
          const res = await caseApi.accounts(caseId)
          this.caseAccountOptions = res.data.data || []
          return
        }
        const res = await accountApi.getAccounts({ page: 1, pageSize: 100 })
        const data = res.data.data || {}
        this.accountOptions = data.items || []
      } catch (error) {
        this.$message.error(this.apiError(error, '查询账户列表失败'))
      }
    },
    openCreate () {
      this.form = { caseId: 0, caseNo: '', txType: 'withdraw', fromAccountId: 0, fromAddress: '', toAddress: '', value: '0.01 ETH', coinType: 'ETH', reason: '' }
      this.caseAccountOptions = []
      this.createDialog = true
    },
    openTransactionImport () {
      this.transactionImportText = ''
      this.transactionImportFileName = ''
      this.transactionImportCount = 0
      if (this.$refs.transactionImportFile) this.$refs.transactionImportFile.value = ''
      this.transactionImportDialog = true
    },
    clearTransactionImport () {
      this.transactionImportText = ''
      this.transactionImportFileName = ''
      this.transactionImportCount = 0
      if (this.$refs.transactionImportFile) this.$refs.transactionImportFile.value = ''
    },
    handleTransactionImportFile (event) {
      const file = event.target.files && event.target.files[0]
      if (!file) return
      const reader = new FileReader()
      reader.onload = () => {
        this.transactionImportFileName = file.name
        this.transactionImportText = String(reader.result || '')
        this.updateTransactionImportPreview()
      }
      reader.onerror = () => {
        this.$message.error('读取交易 JSON 文件失败')
      }
      reader.readAsText(file)
    },
    updateTransactionImportPreview () {
      try {
        this.transactionImportCount = this.parseImportTransactions(this.transactionImportText).length
      } catch (error) {
        this.transactionImportCount = 0
      }
    },
    async batchImportTransactions () {
      let transactions
      try {
        transactions = this.parseImportTransactions(this.transactionImportText)
      } catch (error) {
        this.$message.error(error.message)
        return
      }
      if (!transactions.length) {
        this.$message.warning('没有可导入的交易')
        return
      }
      this.transactionImporting = true
      try {
        const response = await transactionApi.importTransactions({ transactions })
        const data = response.data.data || {}
        this.$message.success(`已导入 ${data.success || transactions.length} 条交易`)
        this.transactionImportDialog = false
        this.fetchTransactions()
      } catch (error) {
        this.$message.error(this.apiError(error, '批量导入交易失败'))
      } finally {
        this.transactionImporting = false
      }
    },
    async createTx () {
      this.creating = true
      try {
        const createRes = await transactionApi.createTransaction(this.form)
        const tx = createRes.data.data
        this.$message.success(`已生成待签名哈希: ${tx.MessageHash || ''}`)
        await this.downloadSignTaskById(tx.ID)
        this.createDialog = false
        this.fetchTransactions()
      } catch (error) {
        this.$message.error(this.apiError(error, '创建交易失败'))
      } finally {
        this.creating = false
      }
    },
    async exportTask (row) {
      this.exportingTaskId = row.ID
      try {
        await this.downloadSignTaskById(row.ID)
        await this.fetchTransactions()
      } catch (error) {
        this.$message.error(this.apiError(error, '导出离线签名任务失败'))
      } finally {
        this.exportingTaskId = 0
      }
    },
    async downloadSignTaskById (id) {
      const res = await transactionApi.exportSignTask(id)
      const pkg = res.data.data.package || res.data.data.payload
      const taskNo = pkg.task_no || (res.data.data.task && res.data.data.task.TaskNo) || `transaction_${id}`
      this.downloadJson(pkg, `offline_task_${taskNo}.json`)
      this.$message.success('离线签名任务 JSON 已下载，可导入离线系统签名')
    },
    openSignature (row) {
      this.selected = row
      this.signatureResultText = ''
      this.signatureResultFileName = ''
      if (this.$refs.signatureResultFile) this.$refs.signatureResultFile.value = ''
      this.signatureForm = { taskNo: '', messageHash: row.MessageHash || '', signature: '' }
      this.signatureDialog = true
    },
    handleSignatureResultFile (event) {
      const file = event.target.files && event.target.files[0]
      if (!file) return
      const reader = new FileReader()
      reader.onload = () => {
        this.signatureResultFileName = file.name
        this.signatureResultText = String(reader.result || '')
        this.parseSignatureResult()
      }
      reader.onerror = () => {
        this.$message.error('读取签名结果包失败')
      }
      reader.readAsText(file)
    },
    parseSignatureResult () {
      if (!this.signatureResultText.trim()) return
      try {
        const pkg = JSON.parse(this.signatureResultText)
        if (pkg.package_type && pkg.package_type !== 'offline_result') {
          this.$message.warning('请选择离线系统导出的 offline_result 结果包')
          return
        }
        if (pkg.task_type && pkg.task_type !== 'sign_result') {
          this.$message.warning('当前按钮只导入 sign_result 签名结果包')
          return
        }
        const payload = pkg.payload || pkg
        this.signatureForm = {
          taskNo: pkg.task_no || payload.task_no || this.signatureForm.taskNo,
          messageHash: payload.message_hash || payload.messageHash || this.signatureForm.messageHash,
          signature: payload.signature || this.signatureForm.signature
        }
      } catch (error) {
        this.$message.warning('结果包 JSON 格式不正确')
      }
    },
    async importSignature () {
      this.parseSignatureResult()
      if (!this.signatureForm.signature) {
        this.$message.warning('结果包中没有签名')
        return
      }
      this.importingSignature = true
      try {
        await transactionApi.importSignature(this.selected.ID, this.signatureForm)
        this.$message.success('签名结果包已导入')
        this.signatureDialog = false
        this.fetchTransactions()
      } catch (error) {
        this.$message.error(this.apiError(error, '导入签名结果失败'))
      } finally {
        this.importingSignature = false
      }
    },
    async broadcast (row) {
      this.broadcastingId = row.ID
      try {
        await transactionApi.broadcast(row.ID)
        this.$message.success('广播请求完成')
        this.fetchTransactions()
      } catch (error) {
        this.$message.error(this.apiError(error, '广播失败'))
      } finally {
        this.broadcastingId = 0
      }
    },
    view (row) {
      this.selected = row
      this.detailDialog = true
    },
    short (v) {
      if (!v) return ''
      return v.length > 20 ? `${v.slice(0, 10)}...${v.slice(-8)}` : v
    },
    handleFormCaseChange (caseNo) {
      const selected = this.caseOptions.find(item => item.CaseNo === caseNo)
      this.form.caseId = selected ? selected.ID : 0
      this.form.caseNo = caseNo || ''
      this.form.fromAccountId = 0
      this.form.fromAddress = ''
      this.form.toAddress = ''
      this.caseAccountOptions = []
      if (selected) this.fetchAccounts(selected.ID)
    },
    queryAccountSuggestions (queryString, cb) {
      const source = this.caseAccountOptions.length ? this.caseAccountOptions : this.accountOptions
      const q = String(queryString || '').toLowerCase()
      const suggestions = source.filter(account => {
        const address = String(account.Address || '').toLowerCase()
        const caseNo = String(account.CaseNo || '').toLowerCase()
        return !q || address.includes(q) || caseNo.includes(q)
      }).slice(0, 20).map(account => ({
        value: account.Address,
        account,
        label: `${this.short(account.Address)} ${account.CaseNo ? `(${account.CaseNo})` : ''}`
      }))
      cb(suggestions)
    },
    selectFromAccount (item) {
      this.form.fromAddress = item.value
      this.form.fromAccountId = item.account.ID || 0
      if (!this.form.caseNo && item.account.CaseNo) {
        this.form.caseNo = item.account.CaseNo
      }
    },
    selectToAccount (item) {
      this.form.toAddress = item.value
    },
    caseLabel (item) {
      return item.Name ? `${item.CaseNo} - ${item.Name}` : item.CaseNo
    },
    downloadJson (data, filename) {
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      link.style.display = 'none'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    },
    statusText (v) {
      const map = { 0: '待签名', 1: '已签名', 2: '已提交', 3: '已确认', 4: '失败', 5: '草稿', 6: '已导出', 7: '已广播', 8: '已取消', draft: '草稿', pending_signature: '待签名', signature_exported: '已导出', signed: '已签名', broadcasted: '已广播', confirmed: '已确认', failed: '失败', cancelled: '已取消' }
      return map[v] || v
    },
    txTypeText (v) {
      return { collect: '归集', withdraw: '提取', test: '测试' }[v] || v
    },
    parseImportTransactions (text) {
      const raw = String(text || '').trim()
      if (!raw) return []
      let parsed
      try {
        parsed = JSON.parse(raw)
      } catch (error) {
        throw new Error(`JSON 格式错误: ${error.message}`)
      }
      const list = Array.isArray(parsed) ? parsed : parsed.transactions
      if (!Array.isArray(list)) {
        throw new Error('JSON 需要是数组，或包含 transactions 数组')
      }
      return list.map(this.normalizeTransaction).filter(Boolean)
    },
    normalizeTransaction (item) {
      if (!item) return null
      const fromAddress = item.fromAddress || item.FromAddress || ''
      const toAddress = item.toAddress || item.ToAddress || ''
      const value = item.value || item.Value || ''
      if (!fromAddress || !toAddress || !value) return null
      return {
        caseNo: item.caseNo || item.CaseNo || '',
        txNo: item.txNo || item.TxNo || '',
        txType: item.txType || item.TxType || 'test',
        fromAccountId: item.fromAccountId || item.FromAccountID || 0,
        fromAddress,
        toAddress,
        value: String(value),
        coinType: item.coinType || item.CoinType || 'ETH',
        reason: item.reason || item.Reason || '',
        messageHash: item.messageHash || item.MessageHash || '',
        txHash: item.txHash || item.TxHash || '',
        status: item.status || item.Status || 'draft'
      }
    },
    apiError (error, fallback) {
      const data = error && error.response && error.response.data
      return (data && (data.message || data.error)) || (error && error.message) || fallback
    }
  }
}
</script>

<style scoped>
.page { padding: 20px; }
.header { display: flex; justify-content: space-between; align-items: center; }
.query { margin-bottom: 12px; }
.operation-actions { display: grid; grid-template-columns: repeat(2, max-content); gap: 6px; align-items: center; }
.operation-actions .el-button + .el-button { margin-left: 0; }
.import-tools { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 10px; }
.import-hint { margin-bottom: 10px; }
.import-summary { color: #606266; font-size: 12px; margin-top: 8px; }
.file-name { color: #606266; font-size: 12px; margin-top: 6px; }
.result-summary { margin-top: 12px; }
</style>
