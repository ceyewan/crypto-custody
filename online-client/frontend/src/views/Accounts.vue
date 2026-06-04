<template>
  <div class="page">
    <el-card>
      <div slot="header" class="header">
        <span>账户管理</span>
        <div>
          <el-button v-if="canWrite" size="small" type="primary" @click="openCreate">新增账户</el-button>
          <el-button v-if="canWrite" size="small" type="success" @click="openImport">导入账户 JSON/CSV</el-button>
          <el-button v-if="canWrite" size="small" :loading="batchSyncing" @click="syncAllBalances">同步全部余额</el-button>
          <el-button size="small" :loading="loading" @click="fetchAccounts">刷新</el-button>
        </div>
      </div>

      <el-form :inline="true" :model="query" class="query">
        <el-form-item><el-input v-model="query.address" placeholder="地址" clearable /></el-form-item>
        <el-form-item><el-input v-model="query.caseNo" placeholder="案件编号" clearable /></el-form-item>
        <el-form-item>
          <el-select v-model="query.accountType" placeholder="账户类型" clearable>
            <el-option label="收缴原始账户" value="seized_original" />
            <el-option label="案件托管钱包" value="custody_wallet" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchAccounts">查询</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="accounts" v-loading="loading">
        <el-table-column prop="Address" label="地址" min-width="220">
          <template slot-scope="scope">{{ short(scope.row.Address) }}</template>
        </el-table-column>
        <el-table-column prop="CaseNo" label="案件编号" width="150" />
        <el-table-column prop="AccountType" label="类型" width="150">
          <template slot-scope="scope">{{ accountTypeText(scope.row.AccountType) }}</template>
        </el-table-column>
        <el-table-column prop="CoinType" label="币种" width="90" />
        <el-table-column label="余额" width="220">
          <template slot-scope="scope">
            <div>{{ scope.row.Balance || '0' }} {{ scope.row.CoinType }}</div>
            <div class="muted">
              {{ scope.row.BalanceSource || 'manual' }}
              <span v-if="scope.row.LastBalanceSyncAt"> · {{ formatTime(scope.row.LastBalanceSyncAt) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="KeyMaterialHint" label="密钥状态" width="150" />
        <el-table-column prop="Description" label="描述" show-overflow-tooltip />
        <el-table-column label="操作" width="240">
          <template slot-scope="scope">
            <el-button size="mini" @click="view(scope.row)">详情</el-button>
            <el-button v-if="canWrite" size="mini" type="primary" :loading="isSyncing(scope.row.ID)" @click="syncBalance(scope.row)">同步余额</el-button>
            <el-button v-if="$store.getters.isAdmin" size="mini" type="danger" @click="remove(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination layout="total, prev, pager, next" :total="total" :page-size="query.pageSize" :current-page.sync="query.page" @current-change="fetchAccounts" />
    </el-card>

    <el-dialog title="新增账户" :visible.sync="formDialog" width="560px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="地址"><el-input v-model="form.address" /></el-form-item>
        <el-form-item label="案件编号">
          <el-select v-model="form.caseNo" filterable clearable allow-create default-first-option style="width:100%" placeholder="请选择案件，或手动输入案件编号">
            <el-option v-for="item in caseOptions" :key="item.CaseNo" :label="caseLabel(item)" :value="item.CaseNo" />
          </el-select>
        </el-form-item>
        <el-form-item label="账户类型">
          <el-select v-model="form.accountType" style="width:100%">
            <el-option label="收缴原始账户" value="seized_original" />
            <el-option label="案件托管钱包" value="custody_wallet" />
          </el-select>
        </el-form-item>
        <el-form-item label="币种"><el-input v-model="form.coinType" /></el-form-item>
        <el-form-item label="余额"><el-input v-model="form.balance" /></el-form-item>
        <el-form-item label="密钥状态">
          <el-select v-model="form.keyMaterialHint" style="width:100%">
            <el-option label="仅地址" value="none" />
            <el-option label="线下掌握私钥" value="has_private_key" />
            <el-option label="离线生成" value="offline_generated" />
            <el-option label="可离线签名" value="offline_signed" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" /></el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="formDialog=false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createAccount">保存账户</el-button>
      </span>
    </el-dialog>

    <el-dialog title="导入账户 JSON/CSV" :visible.sync="importDialog" width="760px">
      <div class="import-tools">
        <input ref="importFile" type="file" accept=".json,.csv,application/json,text/csv" @change="handleImportFile">
        <el-button size="mini" @click="clearImport">清空</el-button>
      </div>
      <el-alert
        class="import-hint"
        type="info"
        :closable="false"
        title="支持 JSON 数组、{ accounts: [...] } 或 CSV。JSON 字段可用 address/coinType/accountType/balance/caseNo/keyMaterialHint/description。"
      />
      <el-form label-width="100px">
        <el-form-item label="默认案件">
          <el-select v-model="importDefaultCaseNo" filterable clearable allow-create default-first-option style="width:100%" placeholder="未写 caseNo 的账户会使用这里选择的案件">
            <el-option v-for="item in caseOptions" :key="item.CaseNo" :label="caseLabel(item)" :value="item.CaseNo" />
          </el-select>
        </el-form-item>
      </el-form>
      <el-input
        v-model="importText"
        type="textarea"
        :rows="12"
        placeholder='例如: {"accounts":[{"address":"0x...","coinType":"ETH","accountType":"seized_original","balance":"0","caseNo":"CASE-001"}]}'
      />
      <span slot="footer">
        <el-button @click="importDialog=false">取消</el-button>
        <el-button type="primary" :loading="importing" @click="batchImport">导入账户</el-button>
      </span>
    </el-dialog>

    <el-dialog title="账户详情" :visible.sync="detailDialog" width="620px">
      <el-descriptions v-if="selected" :column="1" border>
        <el-descriptions-item label="地址">{{ selected.Address }}</el-descriptions-item>
        <el-descriptions-item label="案件编号">{{ selected.CaseNo || '-' }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ accountTypeText(selected.AccountType) }}</el-descriptions-item>
        <el-descriptions-item label="余额">{{ selected.Balance }} {{ selected.CoinType }}</el-descriptions-item>
        <el-descriptions-item label="密钥状态">{{ selected.KeyMaterialHint }}</el-descriptions-item>
        <el-descriptions-item label="离线引用">{{ selected.OfflineRefNo || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script>
import { accountApi, caseApi } from '../services/api'

export default {
  name: 'Accounts',
  data () {
    return {
      loading: false,
      accounts: [],
      caseOptions: [],
      total: 0,
      query: { page: 1, pageSize: 20, address: '', caseNo: '', accountType: '' },
      formDialog: false,
      importDialog: false,
      detailDialog: false,
      importText: '',
      importDefaultCaseNo: '',
      selected: null,
      form: {},
      creating: false,
      importing: false,
      batchSyncing: false,
      syncingIds: {}
    }
  },
  created () {
    this.fetchAccounts()
    this.fetchCases()
  },
  computed: {
    canWrite () {
      return this.$store.getters.isOfficer
    }
  },
  methods: {
    async fetchAccounts () {
      this.loading = true
      try {
        const res = await accountApi.getAccounts(this.query)
        const data = res.data.data || {}
        this.accounts = data.items || []
        this.total = data.total || 0
      } catch (error) {
        this.$message.error(this.apiError(error, '查询账户列表失败'))
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
    openCreate () {
      this.form = { address: '', coinType: 'ETH', accountType: 'seized_original', balance: '0', caseNo: '', keyMaterialHint: 'none', description: '' }
      this.formDialog = true
    },
    async createAccount () {
      this.creating = true
      try {
        await accountApi.createAccount(this.form)
        this.$message.success('账户已保存')
        this.formDialog = false
        this.fetchAccounts()
      } catch (error) {
        this.$message.error(this.apiError(error, '账户导入失败'))
      } finally {
        this.creating = false
      }
    },
    openImport () {
      this.importText = ''
      this.importDefaultCaseNo = ''
      this.importDialog = true
    },
    clearImport () {
      this.importText = ''
      if (this.$refs.importFile) this.$refs.importFile.value = ''
    },
    handleImportFile (event) {
      const file = event.target.files && event.target.files[0]
      if (!file) return
      const reader = new FileReader()
      reader.onload = () => {
        this.importText = String(reader.result || '')
        this.$message.success(`已读取文件: ${file.name}`)
      }
      reader.onerror = () => {
        this.$message.error('读取文件失败')
      }
      reader.readAsText(file)
    },
    async batchImport () {
      let accounts
      try {
        accounts = this.parseImportAccounts(this.importText)
      } catch (error) {
        this.$message.error(error.message)
        return
      }
      if (!accounts.length) {
        this.$message.warning('没有可导入的账户')
        return
      }
      this.importing = true
      try {
        const response = await accountApi.importAccounts({ accounts })
        const data = response.data.data || {}
        this.$message.success(`已导入 ${data.success || accounts.length} 条账户`)
        this.importDialog = false
        this.fetchAccounts()
      } catch (error) {
        this.$message.error(this.apiError(error, '批量导入失败'))
      } finally {
        this.importing = false
      }
    },
    async syncBalance (row) {
      this.setSyncing(row.ID, true)
      try {
        const response = await accountApi.syncBalance(row.ID)
        const account = response.data.data || {}
        this.$message.success(`余额已同步: ${account.Balance || row.Balance || '0'} ${account.CoinType || row.CoinType}`)
        this.fetchAccounts()
      } catch (error) {
        this.$message.error(this.apiError(error, '同步余额失败，请检查 ETH_RPC_URL / Ganache 是否运行'))
      } finally {
        this.setSyncing(row.ID, false)
      }
    },
    async syncAllBalances () {
      this.batchSyncing = true
      try {
        const response = await accountApi.syncBalances()
        const data = response.data.data || {}
        const failed = data.failed || 0
        if (failed > 0) {
          this.$message.warning(`余额同步完成: 成功 ${data.success || 0} 条，失败 ${failed} 条`)
        } else {
          this.$message.success(`余额同步完成: 成功 ${data.success || 0} 条`)
        }
        this.fetchAccounts()
      } catch (error) {
        this.$message.error(this.apiError(error, '一键同步余额失败，请检查 ETH_RPC_URL / Ganache 是否运行'))
      } finally {
        this.batchSyncing = false
      }
    },
    view (row) {
      this.selected = row
      this.detailDialog = true
    },
    async remove (row) {
      await this.$confirm('确认删除该账户？', '删除确认')
      await accountApi.deleteAccount(row.ID)
      this.$message.success('已删除')
      this.fetchAccounts()
    },
    short (v) {
      if (!v) return ''
      return v.length > 20 ? `${v.slice(0, 10)}...${v.slice(-8)}` : v
    },
    accountTypeText (v) {
      return { seized_original: '收缴原始账户', custody_wallet: '案件托管钱包' }[v] || v
    },
    parseImportAccounts (text) {
      const raw = String(text || '').trim()
      if (!raw) return []
      if (raw[0] === '{' || raw[0] === '[') {
        let parsed
        try {
          parsed = JSON.parse(raw)
        } catch (error) {
          throw new Error(`JSON 格式错误: ${error.message}`)
        }
        const list = Array.isArray(parsed) ? parsed : parsed.accounts
        if (!Array.isArray(list)) {
          throw new Error('JSON 需要是数组，或包含 accounts 数组')
        }
        return list.map(this.normalizeAccount).filter(Boolean)
      }
      return raw.split(/\r?\n/).map(line => {
        const trimmed = line.trim()
        if (!trimmed || trimmed.startsWith('#')) return null
        if (/^address\s*,/i.test(trimmed)) return null
        const p = trimmed.split(',').map(v => v.trim())
        return this.normalizeAccount({
          address: p[0],
          coinType: p[1],
          accountType: p[2],
          balance: p[3],
          caseNo: p[4],
          keyMaterialHint: p[5],
          description: p.slice(6).join(','),
          source: 'csv'
        })
      }).filter(Boolean)
    },
    normalizeAccount (item) {
      if (!item) return null
      const account = {
        address: item.address || item.Address,
        coinType: item.coinType || item.CoinType || 'ETH',
        accountType: item.accountType || item.AccountType || 'seized_original',
        balance: String(item.balance || item.Balance || '0'),
        caseNo: item.caseNo || item.CaseNo || this.importDefaultCaseNo || '',
        keyMaterialHint: item.keyMaterialHint || item.KeyMaterialHint || 'none',
        description: item.description || item.Description || '',
        source: item.source || item.Source || 'json',
        balanceSource: item.balanceSource || item.BalanceSource || 'manual',
        offlineRefNo: item.offlineRefNo || item.OfflineRefNo || ''
      }
      if (!account.address) return null
      return account
    },
    caseLabel (item) {
      return item.Name ? `${item.CaseNo} - ${item.Name}` : item.CaseNo
    },
    isSyncing (id) {
      return Boolean(this.syncingIds[id])
    },
    setSyncing (id, syncing) {
      this.$set(this.syncingIds, id, syncing)
    },
    formatTime (unixSeconds) {
      if (!unixSeconds) return '-'
      return new Date(Number(unixSeconds) * 1000).toLocaleString()
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
.muted { color: #909399; font-size: 12px; line-height: 18px; }
.import-tools { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.import-hint { margin-bottom: 10px; }
</style>
