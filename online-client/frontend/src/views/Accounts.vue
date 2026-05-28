<template>
  <div class="page">
    <el-card>
      <div slot="header" class="header">
        <span>账户管理</span>
        <div>
          <el-button v-if="canWrite" size="small" type="primary" @click="openCreate">导入账户</el-button>
          <el-button v-if="canWrite" size="small" type="success" @click="openImport">批量导入</el-button>
          <el-button size="small" @click="fetchAccounts">刷新</el-button>
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
        <el-table-column prop="Balance" label="余额" width="170" />
        <el-table-column prop="KeyMaterialHint" label="密钥状态" width="150" />
        <el-table-column prop="Description" label="描述" show-overflow-tooltip />
        <el-table-column label="操作" width="210">
          <template slot-scope="scope">
            <el-button size="mini" @click="view(scope.row)">详情</el-button>
            <el-button v-if="canWrite" size="mini" type="primary" @click="syncBalance(scope.row)">余额</el-button>
            <el-button v-if="$store.getters.isAdmin" size="mini" type="danger" @click="remove(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination layout="total, prev, pager, next" :total="total" :page-size="query.pageSize" :current-page.sync="query.page" @current-change="fetchAccounts" />
    </el-card>

    <el-dialog title="导入账户" :visible.sync="formDialog" width="560px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="地址"><el-input v-model="form.address" /></el-form-item>
        <el-form-item label="案件编号"><el-input v-model="form.caseNo" /></el-form-item>
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
        <el-button type="primary" @click="createAccount">保存</el-button>
      </span>
    </el-dialog>

    <el-dialog title="批量导入账户" :visible.sync="importDialog" width="720px">
      <el-input v-model="importText" type="textarea" :rows="10" placeholder="address,coinType,accountType,balance,caseNo,keyMaterialHint,description" />
      <span slot="footer">
        <el-button @click="importDialog=false">取消</el-button>
        <el-button type="primary" @click="batchImport">导入</el-button>
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
import { accountApi } from '../services/api'

export default {
  name: 'Accounts',
  data () {
    return {
      loading: false,
      accounts: [],
      total: 0,
      query: { page: 1, pageSize: 20, address: '', caseNo: '', accountType: '' },
      formDialog: false,
      importDialog: false,
      detailDialog: false,
      importText: '',
      selected: null,
      form: {}
    }
  },
  created () {
    this.fetchAccounts()
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
      } finally {
        this.loading = false
      }
    },
    openCreate () {
      this.form = { address: '', coinType: 'ETH', accountType: 'seized_original', balance: '0', caseNo: '', keyMaterialHint: 'none', description: '' }
      this.formDialog = true
    },
    async createAccount () {
      await accountApi.createAccount(this.form)
      this.$message.success('账户已导入')
      this.formDialog = false
      this.fetchAccounts()
    },
    openImport () {
      this.importText = ''
      this.importDialog = true
    },
    async batchImport () {
      const accounts = this.importText.split('\n').map(line => {
        const p = line.split(',').map(v => v.trim())
        return p[0] ? { address: p[0], coinType: p[1] || 'ETH', accountType: p[2] || 'seized_original', balance: p[3] || '0', caseNo: p[4] || '', keyMaterialHint: p[5] || 'none', description: p[6] || '' } : null
      }).filter(Boolean)
      await accountApi.importAccounts({ accounts })
      this.$message.success(`已导入 ${accounts.length} 条账户`)
      this.importDialog = false
      this.fetchAccounts()
    },
    async syncBalance (row) {
      await accountApi.syncBalance(row.ID)
      this.$message.success('余额已同步')
      this.fetchAccounts()
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
    }
  }
}
</script>

<style scoped>
.page { padding: 20px; }
.header { display: flex; justify-content: space-between; align-items: center; }
.query { margin-bottom: 12px; }
</style>
