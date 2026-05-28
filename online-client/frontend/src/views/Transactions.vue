<template>
  <div class="page">
    <el-card>
      <div slot="header" class="header">
        <span>交易管理</span>
        <div>
          <el-button v-if="canWrite" size="small" type="primary" @click="openCreate">新建交易</el-button>
          <el-button size="small" @click="fetchTransactions">刷新</el-button>
        </div>
      </div>

      <el-form :inline="true" class="query">
        <el-form-item><el-input v-model="query.address" placeholder="地址" clearable /></el-form-item>
        <el-form-item>
          <el-select v-model="query.status" placeholder="状态" clearable>
            <el-option label="草稿" value="draft" />
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
        <el-table-column prop="TxNo" label="交易编号" width="170" />
        <el-table-column prop="CaseNo" label="案件编号" width="140" />
        <el-table-column prop="TxType" label="类型" width="90" />
        <el-table-column prop="FromAddress" label="发送方" min-width="170"><template slot-scope="s">{{ short(s.row.FromAddress) }}</template></el-table-column>
        <el-table-column prop="ToAddress" label="接收方" min-width="170"><template slot-scope="s">{{ short(s.row.ToAddress) }}</template></el-table-column>
        <el-table-column prop="Value" label="金额" width="120" />
        <el-table-column prop="Status" label="状态" width="110"><template slot-scope="s"><el-tag>{{ statusText(s.row.Status) }}</el-tag></template></el-table-column>
        <el-table-column label="操作" width="360">
          <template slot-scope="scope">
            <el-button v-if="canWrite" size="mini" @click="prepare(scope.row)">生成哈希</el-button>
            <el-button v-if="canWrite" size="mini" type="primary" @click="exportTask(scope.row)">导出任务</el-button>
            <el-button v-if="canWrite" size="mini" type="success" @click="openSignature(scope.row)">导入签名</el-button>
            <el-button v-if="canWrite" size="mini" type="warning" @click="broadcast(scope.row)">广播</el-button>
            <el-button size="mini" @click="view(scope.row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination layout="total, prev, pager, next" :total="total" :page-size="query.pageSize" :current-page.sync="query.page" @current-change="fetchTransactions" />
    </el-card>

    <el-dialog title="新建交易" :visible.sync="createDialog" width="560px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="案件编号"><el-input v-model="form.caseNo" /></el-form-item>
        <el-form-item label="交易类型">
          <el-select v-model="form.txType" style="width:100%">
            <el-option label="归集" value="collect" />
            <el-option label="提取" value="withdraw" />
            <el-option label="测试" value="test" />
          </el-select>
        </el-form-item>
        <el-form-item label="发送方"><el-input v-model="form.fromAddress" /></el-form-item>
        <el-form-item label="接收方"><el-input v-model="form.toAddress" /></el-form-item>
        <el-form-item label="金额"><el-input v-model="form.value" /></el-form-item>
        <el-form-item label="币种"><el-input v-model="form.coinType" /></el-form-item>
        <el-form-item label="事由"><el-input v-model="form.reason" type="textarea" /></el-form-item>
      </el-form>
      <span slot="footer"><el-button @click="createDialog=false">取消</el-button><el-button type="primary" @click="createTx">保存</el-button></span>
    </el-dialog>

    <el-dialog title="导入签名结果" :visible.sync="signatureDialog" width="620px">
      <el-form :model="signatureForm" label-width="100px">
        <el-form-item label="任务编号"><el-input v-model="signatureForm.taskNo" /></el-form-item>
        <el-form-item label="消息哈希"><el-input v-model="signatureForm.messageHash" /></el-form-item>
        <el-form-item label="签名"><el-input v-model="signatureForm.signature" type="textarea" :rows="5" /></el-form-item>
      </el-form>
      <span slot="footer"><el-button @click="signatureDialog=false">取消</el-button><el-button type="primary" @click="importSignature">导入</el-button></span>
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
import { transactionApi } from '../services/api'

export default {
  name: 'Transactions',
  data () {
    return {
      loading: false,
      transactions: [],
      total: 0,
      query: { page: 1, pageSize: 20, address: '', status: '' },
      createDialog: false,
      signatureDialog: false,
      detailDialog: false,
      selected: null,
      form: {},
      signatureForm: {}
    }
  },
  created () {
    this.fetchTransactions()
  },
  computed: {
    canWrite () {
      return this.$store.getters.isOfficer
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
    openCreate () {
      this.form = { caseNo: '', txType: 'withdraw', fromAddress: '', toAddress: '', value: '0.01 ETH', coinType: 'ETH', reason: '' }
      this.createDialog = true
    },
    async createTx () {
      await transactionApi.createDraft(this.form)
      this.$message.success('交易已创建')
      this.createDialog = false
      this.fetchTransactions()
    },
    async prepare (row) {
      await transactionApi.prepareById(row.ID)
      this.$message.success('已生成待签名哈希')
      this.fetchTransactions()
    },
    async exportTask (row) {
      const res = await transactionApi.exportSignTask(row.ID)
      this.$alert(JSON.stringify(res.data.data.payload, null, 2), '签名任务包')
    },
    openSignature (row) {
      this.selected = row
      this.signatureForm = { taskNo: '', messageHash: row.MessageHash || '', signature: '' }
      this.signatureDialog = true
    },
    async importSignature () {
      await transactionApi.importSignature(this.selected.ID, this.signatureForm)
      this.$message.success('签名已导入')
      this.signatureDialog = false
      this.fetchTransactions()
    },
    async broadcast (row) {
      await transactionApi.broadcast(row.ID)
      this.$message.success('广播请求完成')
      this.fetchTransactions()
    },
    view (row) {
      this.selected = row
      this.detailDialog = true
    },
    short (v) {
      if (!v) return ''
      return v.length > 20 ? `${v.slice(0, 10)}...${v.slice(-8)}` : v
    },
    statusText (v) {
      const map = { 0: '待签名', 1: '已签名', 2: '已提交', 3: '已确认', 4: '失败', 5: '草稿', 6: '已导出', 7: '已广播', 8: '已取消' }
      return map[v] || v
    }
  }
}
</script>

<style scoped>
.page { padding: 20px; }
.header { display: flex; justify-content: space-between; align-items: center; }
.query { margin-bottom: 12px; }
</style>
