<template>
  <div class="page">
    <el-card>
      <div slot="header" class="header"><span>测试数据</span><el-button size="small" @click="loadSummary">刷新</el-button></div>
      <el-descriptions :column="3" border>
        <el-descriptions-item label="测试案件">{{ summary.cases || 0 }}</el-descriptions-item>
        <el-descriptions-item label="测试账户">{{ summary.accounts || 0 }}</el-descriptions-item>
        <el-descriptions-item label="测试交易">{{ summary.transactions || 0 }}</el-descriptions-item>
      </el-descriptions>
      <el-form :model="form" label-width="100px" class="form">
        <el-form-item label="生成用户"><el-switch v-model="form.users" /></el-form-item>
        <el-form-item label="案件数"><el-input-number v-model="form.caseCount" :min="1" /></el-form-item>
        <el-form-item label="账户数"><el-input-number v-model="form.accounts" :min="1" /></el-form-item>
        <el-form-item label="交易数"><el-input-number v-model="form.transactions" :min="1" /></el-form-item>
        <el-form-item label="币种"><el-input v-model="form.coinType" /></el-form-item>
        <el-form-item>
          <el-button type="primary" @click="seed">生成测试数据</el-button>
          <el-button type="danger" @click="clear">清理测试数据</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { testDataApi } from '../services/api'

export default {
  name: 'TestData',
  data () {
    return { summary: {}, form: { users: true, caseCount: 10, accounts: 1000, transactions: 500, coinType: 'ETH' } }
  },
  created () { this.loadSummary() },
  methods: {
    async loadSummary () {
      const res = await testDataApi.summary()
      this.summary = res.data.data || {}
    },
    async seed () {
      await testDataApi.seed(this.form)
      this.$message.success('测试数据已生成')
      this.loadSummary()
    },
    async clear () {
      await this.$confirm('确认清理测试数据？', '清理确认')
      await testDataApi.clear()
      this.$message.success('测试数据已清理')
      this.loadSummary()
    }
  }
}
</script>

<style scoped>
.page { padding: 20px; }
.header { display: flex; justify-content: space-between; align-items: center; }
.form { margin-top: 20px; max-width: 420px; }
</style>
