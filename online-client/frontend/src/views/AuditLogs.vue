<template>
  <div class="page">
    <el-card>
      <div slot="header" class="header"><span>审计日志</span><el-button size="small" @click="fetchLogs">刷新</el-button></div>
      <el-form :inline="true" :model="query" class="query">
        <el-form-item><el-input v-model="query.username" placeholder="用户" clearable /></el-form-item>
        <el-form-item><el-input v-model="query.action" placeholder="动作" clearable /></el-form-item>
        <el-form-item><el-input v-model="query.caseNo" placeholder="案件编号" clearable /></el-form-item>
        <el-form-item><el-button type="primary" @click="fetchLogs">查询</el-button></el-form-item>
      </el-form>
      <el-table :data="logs" v-loading="loading">
        <el-table-column prop="CreatedAt" label="时间" width="170" />
        <el-table-column prop="Username" label="用户" width="120" />
        <el-table-column prop="Role" label="角色" width="90" />
        <el-table-column prop="Action" label="动作" width="190" />
        <el-table-column prop="ResourceType" label="资源" width="120" />
        <el-table-column prop="CaseNo" label="案件编号" width="140" />
        <el-table-column prop="Result" label="结果" width="90" />
        <el-table-column prop="ErrorMessage" label="错误" show-overflow-tooltip />
      </el-table>
      <el-pagination layout="total, prev, pager, next" :total="total" :page-size="query.pageSize" :current-page.sync="query.page" @current-change="fetchLogs" />
    </el-card>
  </div>
</template>

<script>
import { auditApi } from '../services/api'

export default {
  name: 'AuditLogs',
  data () {
    return { loading: false, logs: [], total: 0, query: { page: 1, pageSize: 20, username: '', action: '', caseNo: '' } }
  },
  created () { this.fetchLogs() },
  methods: {
    async fetchLogs () {
      this.loading = true
      try {
        const res = await auditApi.list(this.query)
        const data = res.data.data || {}
        this.logs = data.items || []
        this.total = data.total || 0
      } finally {
        this.loading = false
      }
    }
  }
}
</script>

<style scoped>
.page { padding: 20px; }
.header { display: flex; justify-content: space-between; align-items: center; }
.query { margin-bottom: 12px; }
</style>
