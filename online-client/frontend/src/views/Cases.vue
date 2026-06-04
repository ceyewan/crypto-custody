<template>
  <div class="page">
    <el-card>
      <div slot="header" class="header">
        <span>案件管理</span>
        <div>
          <el-button v-if="canWrite" size="small" type="primary" @click="openCreate">新建案件</el-button>
          <el-button v-if="canWrite" size="small" type="success" @click="openCaseImport">导入案件 JSON</el-button>
          <el-button size="small" @click="fetchCases">刷新</el-button>
        </div>
      </div>

      <el-form :inline="true" :model="query" class="query">
        <el-form-item>
          <el-input v-model="query.caseNo" placeholder="案件编号" clearable />
        </el-form-item>
        <el-form-item>
          <el-input v-model="query.keyword" placeholder="关键词" clearable />
        </el-form-item>
        <el-form-item>
          <el-select v-model="query.status" placeholder="状态" clearable>
            <el-option label="进行中" value="active" />
            <el-option label="已关闭" value="closed" />
            <el-option label="已归档" value="archived" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchCases">查询</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="cases" v-loading="loading">
        <el-table-column prop="CaseNo" label="案件编号" width="160" />
        <el-table-column prop="Name" label="案件名称" width="180" />
        <el-table-column prop="Status" label="状态" width="100">
          <template slot-scope="scope">
            <el-tag>{{ statusText(scope.row.Status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="CustodyAddress" label="托管钱包" min-width="220">
          <template slot-scope="scope">{{ short(scope.row.CustodyAddress) || '-' }}</template>
        </el-table-column>
        <el-table-column prop="Description" label="描述" show-overflow-tooltip />
        <el-table-column label="操作" width="360">
          <template slot-scope="scope">
            <el-button size="mini" @click="viewAccounts(scope.row)">账户</el-button>
            <el-button v-if="canWrite" size="mini" type="primary" @click="createTask(scope.row)">生成钱包任务</el-button>
            <el-button v-if="canWrite" size="mini" type="success" @click="openImport(scope.row)">导入托管钱包结果</el-button>
            <el-button v-if="canWrite" size="mini" type="warning" @click="openEdit(scope.row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination layout="total, prev, pager, next" :total="total" :page-size="query.pageSize" :current-page.sync="query.page" @current-change="fetchCases" />
    </el-card>

    <el-dialog :title="form.ID ? '编辑案件' : '新建案件'" :visible.sync="caseDialog" width="520px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="案件编号"><el-input v-model="form.caseNo" placeholder="留空自动生成" /></el-form-item>
        <el-form-item label="案件名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width:100%">
            <el-option label="进行中" value="active" />
            <el-option label="已关闭" value="closed" />
            <el-option label="已归档" value="archived" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" /></el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="caseDialog=false">取消</el-button>
        <el-button type="primary" @click="saveCase">保存</el-button>
      </span>
    </el-dialog>

    <el-dialog title="导入案件 JSON" :visible.sync="caseImportDialog" width="760px">
      <div class="import-tools">
        <div>
          <input ref="caseImportFile" type="file" accept="application/json,.json" @change="handleCaseImportFile">
          <div class="file-name" v-if="caseImportFileName">{{ caseImportFileName }}</div>
        </div>
        <el-button size="mini" @click="clearCaseImport">清空</el-button>
      </div>
      <el-alert
        class="import-hint"
        type="info"
        :closable="false"
        title='支持 JSON 数组或 { "cases": [...] }。字段可用 caseNo/name/status/description；caseNo 留空时后端自动生成。'
      />
      <el-input
        v-model="caseImportText"
        type="textarea"
        :rows="12"
        placeholder='例如: {"cases":[{"caseNo":"CASE-2026-001","name":"涉案资产处置案件","status":"active","description":"案件基础信息"}]}'
        @blur="updateCaseImportPreview"
      />
      <div class="import-summary">{{ caseImportSummary }}</div>
      <span slot="footer">
        <el-button @click="caseImportDialog=false">取消</el-button>
        <el-button type="primary" :loading="caseImporting" @click="batchImportCases">导入案件</el-button>
      </span>
    </el-dialog>

    <el-dialog title="导入离线托管钱包结果" :visible.sync="walletDialog" width="680px">
      <el-form :model="walletForm" label-width="110px">
        <el-form-item label="结果包文件">
          <input ref="walletResultFile" type="file" accept="application/json,.json" @change="handleWalletResultFile">
          <div class="file-name" v-if="walletResultFileName">{{ walletResultFileName }}</div>
        </el-form-item>
        <el-form-item label="结果包 JSON">
          <el-input v-model="walletResultText" type="textarea" :rows="8" placeholder="可粘贴离线端导出的 offline_result JSON，系统会自动识别托管地址和离线结果编号" @blur="parseWalletResult" />
        </el-form-item>
        <el-descriptions :column="1" border size="small" class="result-summary">
          <el-descriptions-item label="任务编号">{{ walletForm.taskNo || '-' }}</el-descriptions-item>
          <el-descriptions-item label="托管地址">{{ walletForm.custodyAddress || '-' }}</el-descriptions-item>
          <el-descriptions-item label="离线结果编号">{{ walletForm.offlineRefNo || '-' }}</el-descriptions-item>
          <el-descriptions-item label="币种">{{ walletForm.coinType || '-' }}</el-descriptions-item>
        </el-descriptions>
      </el-form>
      <span slot="footer">
        <el-button @click="walletDialog=false">取消</el-button>
        <el-button type="primary" :loading="walletImporting" @click="importWallet">导入结果包</el-button>
      </span>
    </el-dialog>

    <el-dialog title="案件账户" :visible.sync="accountsDialog" width="760px">
      <el-table :data="caseAccounts">
        <el-table-column prop="Address" label="地址" min-width="220">
          <template slot-scope="scope">{{ short(scope.row.Address) }}</template>
        </el-table-column>
        <el-table-column prop="AccountType" label="类型" width="150" />
        <el-table-column prop="Balance" label="余额" width="160" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script>
import { caseApi, offlineTaskApi } from '../services/api'

export default {
  name: 'Cases',
  data () {
    return {
      loading: false,
      cases: [],
      total: 0,
      query: { page: 1, pageSize: 20, caseNo: '', keyword: '', status: '' },
      caseDialog: false,
      walletDialog: false,
      accountsDialog: false,
      caseImportDialog: false,
      selectedCase: null,
      caseAccounts: [],
      caseImportText: '',
      caseImportFileName: '',
      caseImportCount: 0,
      caseImporting: false,
      walletResultText: '',
      walletResultFileName: '',
      walletImporting: false,
      form: { caseNo: '', name: '', status: 'active', description: '' },
      walletForm: { taskNo: '', custodyAddress: '', offlineRefNo: '', coinType: 'ETH' }
    }
  },
  created () {
    this.fetchCases()
  },
  computed: {
    canWrite () {
      return this.$store.getters.isOfficer
    },
    caseImportSummary () {
      return this.caseImportCount > 0 ? `已解析 ${this.caseImportCount} 条案件` : '尚未解析案件'
    }
  },
  methods: {
    async fetchCases () {
      this.loading = true
      try {
        const res = await caseApi.list(this.query)
        const data = res.data.data || {}
        this.cases = data.items || []
        this.total = data.total || 0
      } finally {
        this.loading = false
      }
    },
    openCreate () {
      this.form = { caseNo: '', name: '', status: 'active', description: '' }
      this.caseDialog = true
    },
    openCaseImport () {
      this.caseImportText = ''
      this.caseImportFileName = ''
      this.caseImportCount = 0
      if (this.$refs.caseImportFile) this.$refs.caseImportFile.value = ''
      this.caseImportDialog = true
    },
    clearCaseImport () {
      this.caseImportText = ''
      this.caseImportFileName = ''
      this.caseImportCount = 0
      if (this.$refs.caseImportFile) this.$refs.caseImportFile.value = ''
    },
    handleCaseImportFile (event) {
      const file = event.target.files && event.target.files[0]
      if (!file) return
      const reader = new FileReader()
      reader.onload = () => {
        this.caseImportFileName = file.name
        this.caseImportText = String(reader.result || '')
        this.updateCaseImportPreview()
      }
      reader.onerror = () => {
        this.$message.error('读取案件 JSON 文件失败')
      }
      reader.readAsText(file)
    },
    updateCaseImportPreview () {
      try {
        this.caseImportCount = this.parseImportCases(this.caseImportText).length
      } catch (error) {
        this.caseImportCount = 0
      }
    },
    async batchImportCases () {
      let cases
      try {
        cases = this.parseImportCases(this.caseImportText)
      } catch (error) {
        this.$message.error(error.message)
        return
      }
      if (!cases.length) {
        this.$message.warning('没有可导入的案件')
        return
      }
      this.caseImporting = true
      try {
        const response = await caseApi.importCases({ cases })
        const data = response.data.data || {}
        this.$message.success(`已导入 ${data.success || cases.length} 条案件`)
        this.caseImportDialog = false
        this.fetchCases()
      } catch (error) {
        this.$message.error(this.apiError(error, '批量导入案件失败'))
      } finally {
        this.caseImporting = false
      }
    },
    openEdit (row) {
      this.form = { ID: row.ID, caseNo: row.CaseNo, name: row.Name, status: row.Status, description: row.Description }
      this.caseDialog = true
    },
    async saveCase () {
      if (!this.form.name || !this.form.name.trim()) {
        this.$message.warning('请输入案件名称')
        return
      }
      if (this.form.ID) {
        await caseApi.update(this.form.ID, this.form)
      } else {
        await caseApi.create(this.form)
      }
      this.caseDialog = false
      this.$message.success('保存成功')
      this.fetchCases()
    },
    async createTask (row) {
      const res = await offlineTaskApi.createCustodyKeygen({ caseId: row.ID, coinType: 'ETH', thresholdPolicy: '2_of_3' })
      const pkg = res.data.data.package || res.data.data.payload
      this.downloadJson(pkg, `offline_task_${pkg.task_no || res.data.data.task.TaskNo}.json`)
      this.$alert(JSON.stringify(pkg, null, 2), '离线任务包已下载')
    },
    openImport (row) {
      this.selectedCase = row
      this.walletResultText = ''
      this.walletResultFileName = ''
      if (this.$refs.walletResultFile) this.$refs.walletResultFile.value = ''
      this.walletForm = { taskNo: '', custodyAddress: '', offlineRefNo: '', coinType: 'ETH' }
      this.walletDialog = true
    },
    handleWalletResultFile (event) {
      const file = event.target.files && event.target.files[0]
      if (!file) return
      const reader = new FileReader()
      reader.onload = () => {
        this.walletResultFileName = file.name
        this.walletResultText = String(reader.result || '')
        this.parseWalletResult()
      }
      reader.onerror = () => {
        this.$message.error('读取结果包文件失败')
      }
      reader.readAsText(file)
    },
    parseWalletResult () {
      if (!this.walletResultText.trim()) return
      try {
        const pkg = JSON.parse(this.walletResultText)
        if (pkg.package_type && pkg.package_type !== 'offline_result') {
          this.$message.warning('请选择离线系统导出的 offline_result 结果包')
          return
        }
        if (pkg.task_type && pkg.task_type !== 'custody_keygen_result') {
          this.$message.warning('当前按钮只导入 custody_keygen_result 托管钱包结果包')
          return
        }
        const payload = pkg.payload || pkg
        this.walletForm = {
          taskNo: pkg.task_no || payload.task_no || this.walletForm.taskNo,
          custodyAddress: payload.custody_address || payload.custodyAddress || this.walletForm.custodyAddress,
          offlineRefNo: payload.offline_ref_no || payload.offlineRefNo || this.walletForm.offlineRefNo,
          coinType: payload.coin_type || payload.coinType || this.walletForm.coinType || 'ETH'
        }
      } catch (error) {
        this.$message.warning('结果包 JSON 格式不正确')
      }
    },
    async importWallet () {
      this.parseWalletResult()
      if (!this.walletForm.custodyAddress) {
        this.$message.warning('结果包中没有托管地址')
        return
      }
      this.walletImporting = true
      try {
        await caseApi.importCustodyWallet(this.selectedCase.ID, { ...this.walletForm, caseNo: this.selectedCase.CaseNo })
        this.walletDialog = false
        this.$message.success('托管钱包结果包已导入')
        this.fetchCases()
      } catch (error) {
        this.$message.error(this.apiError(error, '导入托管钱包结果失败'))
      } finally {
        this.walletImporting = false
      }
    },
    async viewAccounts (row) {
      const res = await caseApi.accounts(row.ID)
      this.caseAccounts = res.data.data || []
      this.accountsDialog = true
    },
    short (v) {
      if (!v) return ''
      return v.length > 20 ? `${v.slice(0, 10)}...${v.slice(-8)}` : v
    },
    downloadJson (data, filename) {
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      link.click()
      URL.revokeObjectURL(url)
    },
    statusText (v) {
      return { active: '进行中', closed: '已关闭', archived: '已归档' }[v] || v
    },
    parseImportCases (text) {
      const raw = String(text || '').trim()
      if (!raw) return []
      let parsed
      try {
        parsed = JSON.parse(raw)
      } catch (error) {
        throw new Error(`JSON 格式错误: ${error.message}`)
      }
      const list = Array.isArray(parsed) ? parsed : parsed.cases
      if (!Array.isArray(list)) {
        throw new Error('JSON 需要是数组，或包含 cases 数组')
      }
      return list.map(this.normalizeCase).filter(Boolean)
    },
    normalizeCase (item) {
      if (!item) return null
      const name = item.name || item.Name || ''
      if (!name) return null
      return {
        caseNo: item.caseNo || item.CaseNo || '',
        name,
        status: item.status || item.Status || 'active',
        description: item.description || item.Description || ''
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
.import-tools { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 10px; }
.import-hint { margin-bottom: 10px; }
.import-summary { color: #606266; font-size: 12px; margin-top: 8px; }
.file-name { color: #606266; font-size: 12px; margin-top: 6px; }
.result-summary { margin-top: 12px; }
</style>
