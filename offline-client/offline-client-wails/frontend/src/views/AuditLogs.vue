<template>
    <div class="page audit-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">审计日志</h2>
                <p class="page-subtitle">只读查看离线任务、密钥、分片、安全芯片和备份操作记录。</p>
            </div>
            <el-button icon="el-icon-refresh" :loading="loadingLogs || loadingApprovals" @click="loadAll">刷新</el-button>
        </div>

        <el-card>
            <el-form :inline="true" :model="filters" class="filters">
                <el-form-item label="时间">
                    <el-date-picker
                        v-model="filters.timeRange"
                        type="datetimerange"
                        range-separator="至"
                        start-placeholder="开始时间"
                        end-placeholder="结束时间">
                    </el-date-picker>
                </el-form-item>
                <el-form-item label="用户">
                    <el-input v-model="filters.username" clearable placeholder="用户"></el-input>
                </el-form-item>
                <el-form-item label="角色">
                    <el-select v-model="filters.role" clearable placeholder="全部">
                        <el-option label="管理员" value="admin"></el-option>
                        <el-option label="警员" value="officer"></el-option>
                        <el-option label="审计员" value="auditor"></el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="动作">
                    <el-input v-model="filters.action" clearable placeholder="keygen/sign/transfer"></el-input>
                </el-form-item>
                <el-form-item label="资源">
                    <el-input v-model="filters.resource" clearable placeholder="资源类型或ID"></el-input>
                </el-form-item>
                <el-form-item label="案件编号">
                    <el-input v-model="filters.caseNo" clearable placeholder="CASE-..."></el-input>
                </el-form-item>
                <el-form-item label="地址">
                    <el-input v-model="filters.address" clearable placeholder="0x..."></el-input>
                </el-form-item>
                <el-form-item label="结果">
                    <el-select v-model="filters.result" clearable placeholder="全部">
                        <el-option label="success" value="success"></el-option>
                        <el-option label="failure" value="failure"></el-option>
                    </el-select>
                </el-form-item>
                <el-form-item>
                    <el-button @click="resetFilters">重置</el-button>
                </el-form-item>
            </el-form>

            <el-tabs v-model="activeTab">
                <el-tab-pane label="审计日志" name="audit">
                    <el-table :data="filteredLogs" style="width: 100%" v-loading="loadingLogs">
                        <el-table-column prop="created_at" label="时间" width="180">
                            <template slot-scope="scope">
                                {{ formatTime(scope.row.created_at) }}
                            </template>
                        </el-table-column>
                        <el-table-column prop="username" label="用户" width="120"></el-table-column>
                        <el-table-column prop="role" label="角色" width="100"></el-table-column>
                        <el-table-column prop="action" label="动作" min-width="180"></el-table-column>
                        <el-table-column prop="resource_type" label="资源" width="130"></el-table-column>
                        <el-table-column prop="resource_id" label="资源ID" min-width="180"></el-table-column>
                        <el-table-column prop="redacted_detail" label="摘要" min-width="180"></el-table-column>
                        <el-table-column prop="result" label="结果" width="100">
                            <template slot-scope="scope">
                                <el-tag :type="scope.row.result === 'success' ? 'success' : 'danger'">
                                    {{ scope.row.result }}
                                </el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column label="操作" width="90">
                            <template slot-scope="scope">
                                <el-button type="text" @click="showDetail(scope.row)">详情</el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                </el-tab-pane>

                <el-tab-pane label="审批记录" name="approvals">
                    <el-table :data="approvals" style="width: 100%" v-loading="loadingApprovals">
                        <el-table-column prop="created_at" label="时间" width="180">
                            <template slot-scope="scope">
                                {{ formatTime(scope.row.created_at) }}
                            </template>
                        </el-table-column>
                        <el-table-column prop="approval_id" label="审批ID" min-width="220"></el-table-column>
                        <el-table-column prop="operation" label="操作" width="180"></el-table-column>
                        <el-table-column prop="resource_id" label="资源ID" min-width="180"></el-table-column>
                        <el-table-column prop="requested_by" label="发起人" width="120"></el-table-column>
                        <el-table-column prop="approved_by" label="审批人" width="120"></el-table-column>
                        <el-table-column prop="status" label="状态" width="100">
                            <template slot-scope="scope">
                                <el-tag :type="scope.row.status === 'approved' ? 'success' : 'warning'">
                                    {{ scope.row.status }}
                                </el-tag>
                            </template>
                        </el-table-column>
                    </el-table>
                </el-tab-pane>
            </el-tabs>
        </el-card>

        <el-drawer title="审计详情" :visible.sync="detailVisible" size="520px">
            <pre class="detail-json">{{ JSON.stringify(detail, null, 2) }}</pre>
        </el-drawer>
    </div>
</template>

<script>
export default {
    name: 'AuditLogs',
    data() {
        return {
            activeTab: 'audit',
            logs: [],
            approvals: [],
            filters: {
                timeRange: [],
                username: '',
                role: '',
                action: '',
                resource: '',
                caseNo: '',
                address: '',
                result: ''
            },
            detail: null,
            detailVisible: false,
            loadingLogs: false,
            loadingApprovals: false
        }
    },
    computed: {
        filteredLogs() {
            return this.logs.filter(log => {
                if (this.filters.username && !String(log.username || '').includes(this.filters.username)) return false
                if (this.filters.role && log.role !== this.filters.role) return false
                if (this.filters.action && !String(log.action || '').includes(this.filters.action)) return false
                const haystack = `${log.resource_type || ''} ${log.resource_id || ''} ${log.redacted_detail || ''}`
                if (this.filters.resource && !haystack.includes(this.filters.resource)) return false
                if (this.filters.caseNo && !haystack.includes(this.filters.caseNo)) return false
                if (this.filters.address && !haystack.toLowerCase().includes(this.filters.address.toLowerCase())) return false
                if (this.filters.result && log.result !== this.filters.result) return false
                if (this.filters.timeRange && this.filters.timeRange.length === 2) {
                    const time = new Date(log.created_at).getTime()
                    const start = new Date(this.filters.timeRange[0]).getTime()
                    const end = new Date(this.filters.timeRange[1]).getTime()
                    if (time < start || time > end) return false
                }
                return true
            })
        }
    },
    created() {
        this.loadAll()
    },
    methods: {
        async loadAll() {
            await Promise.all([this.loadAudit(), this.loadApprovals()])
        },

        async loadAudit() {
            this.loadingLogs = true
            try {
                const response = await this.$offlineApi.listAudit(300)
                this.logs = response.data.logs || []
            } catch (error) {
                this.$message.error(this.apiError(error, '审计日志加载失败'))
            } finally {
                this.loadingLogs = false
            }
        },

        async loadApprovals() {
            this.loadingApprovals = true
            try {
                const response = await this.$offlineApi.listApprovals(300)
                this.approvals = response.data.approvals || []
            } catch (error) {
                this.$message.error(this.apiError(error, '审批记录加载失败'))
            } finally {
                this.loadingApprovals = false
            }
        },

        resetFilters() {
            this.filters = {
                timeRange: [],
                username: '',
                role: '',
                action: '',
                resource: '',
                caseNo: '',
                address: '',
                result: ''
            }
        },

        showDetail(row) {
            this.detail = row
            this.detailVisible = true
        },

        formatTime(value) {
            return value ? new Date(value).toLocaleString() : ''
        },

        apiError(error, fallback) {
            return error.response?.data?.error || error.response?.data?.message || error.message || fallback
        }
    }
}
</script>

<style scoped>
.filters {
    margin-bottom: 10px;
}

.detail-json {
    margin: 0;
    padding: 0 20px 20px;
    white-space: pre-wrap;
    word-break: break-word;
    color: #303133;
}
</style>
