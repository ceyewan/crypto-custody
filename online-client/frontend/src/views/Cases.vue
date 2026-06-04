<template>
  <div class="page">
    <el-card>
      <div slot="header" class="header">
        <span>案件管理</span>
        <div>
          <el-button v-if="canWrite" size="small" type="primary" @click="openCreate">新建案件</el-button>
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
            <el-button v-if="canWrite" size="mini" type="success" @click="openImport(scope.row)">导入钱包</el-button>
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

    <el-dialog title="导入离线托管钱包结果" :visible.sync="walletDialog" width="680px">
      <el-form :model="walletForm" label-width="110px">
        <el-form-item label="结果包 JSON">
          <el-input v-model="walletResultText" type="textarea" :rows="6" placeholder="可粘贴离线端导出的 offline_result JSON，系统会自动识别托管地址和离线结果编号" @blur="parseWalletResult" />
        </el-form-item>
        <el-form-item label="任务编号"><el-input v-model="walletForm.taskNo" /></el-form-item>
        <el-form-item label="托管地址"><el-input v-model="walletForm.custodyAddress" /></el-form-item>
        <el-form-item label="离线结果编号"><el-input v-model="walletForm.offlineRefNo" /></el-form-item>
        <el-form-item label="币种"><el-input v-model="walletForm.coinType" /></el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="walletDialog=false">取消</el-button>
        <el-button type="primary" @click="importWallet">导入</el-button>
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
      selectedCase: null,
      caseAccounts: [],
      walletResultText: '',
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
      this.walletForm = { taskNo: '', custodyAddress: '', offlineRefNo: '', coinType: 'ETH' }
      this.walletDialog = true
    },
    parseWalletResult () {
      if (!this.walletResultText.trim()) return
      try {
        const pkg = JSON.parse(this.walletResultText)
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
      await caseApi.importCustodyWallet(this.selectedCase.ID, { ...this.walletForm, caseNo: this.selectedCase.CaseNo })
      this.walletDialog = false
      this.$message.success('导入成功')
      this.fetchCases()
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
    }
  }
}
</script>

<style scoped>
.page { padding: 20px; }
.header { display: flex; justify-content: space-between; align-items: center; }
.query { margin-bottom: 12px; }
</style>
