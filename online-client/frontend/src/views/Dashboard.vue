<template>
  <div class="page">
    <el-row :gutter="16">
      <el-col :span="6">
        <el-card><div class="num">{{ accountTotal }}</div><div class="label">账户</div></el-card>
      </el-col>
      <el-col :span="6">
        <el-card><div class="num">{{ transactionTotal }}</div><div class="label">交易</div></el-card>
      </el-col>
      <el-col :span="6" v-if="isAdmin">
        <el-card><div class="num">{{ userTotal }}</div><div class="label">用户</div></el-card>
      </el-col>
    </el-row>
    <el-card class="welcome">
      <h2>虚拟货币存管提控在线系统</h2>
      <p>在线端负责案件、账户、交易、审计、备份和测试数据；私钥、安全芯片和门限签名在离线端完成。</p>
    </el-card>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { accountApi, transactionApi, userApi } from '../services/api'

export default {
  name: 'Dashboard',
  data () {
    return { accountTotal: 0, transactionTotal: 0, userTotal: 0 }
  },
  computed: {
    ...mapGetters(['isAdmin'])
  },
  created () {
    this.load()
  },
  methods: {
    async load () {
      const accounts = await accountApi.getAccounts({ page: 1, pageSize: 1 })
      this.accountTotal = accounts.data.data.total || 0
      const txs = await transactionApi.getTransactionPage({ page: 1, pageSize: 1 })
      this.transactionTotal = txs.data.data.total || 0
      if (this.isAdmin) {
        const users = await userApi.getUsers()
        this.userTotal = (users.data.data || []).length
      }
    }
  }
}
</script>

<style scoped>
.page { padding: 20px; }
.num { font-size: 28px; font-weight: 600; }
.label { margin-top: 8px; color: #606266; }
.welcome { margin-top: 16px; }
</style>
