<template>
    <div class="page audit-page">
        <el-card>
            <div slot="header" class="header">
                <span>审计日志</span>
                <div>
                    <el-button size="small" @click="advancedVisible = !advancedVisible">
                        {{ advancedVisible ? '收起筛选' : '更多筛选' }}
                    </el-button>
                    <el-button size="small" icon="el-icon-refresh" :loading="loadingLogs || loadingApprovals" @click="loadAll">刷新</el-button>
                </div>
            </div>

            <el-form :inline="true" :model="query" class="query">
                <el-form-item><el-input v-model="query.username" placeholder="用户" clearable /></el-form-item>
                <el-form-item><el-input v-model="query.action" placeholder="动作" clearable /></el-form-item>
                <el-form-item><el-input v-model="query.caseNo" placeholder="案件编号" clearable /></el-form-item>
                <el-form-item>
                    <el-select v-model="query.result" placeholder="结果" clearable>
                        <el-option label="成功" value="success" />
                        <el-option label="失败" value="failure" />
                    </el-select>
                </el-form-item>
                <el-form-item>
                    <el-button type="primary" :loading="loadingLogs" @click="searchLogs">查询</el-button>
                    <el-button @click="resetQuery">重置</el-button>
                </el-form-item>
            </el-form>

            <el-form v-show="advancedVisible" :inline="true" :model="query" class="query advanced-query">
                <el-form-item>
                    <el-date-picker
                        v-model="query.timeRange"
                        type="datetimerange"
                        range-separator="至"
                        start-placeholder="开始时间"
                        end-placeholder="结束时间"
                        value-format="yyyy-MM-dd HH:mm:ss"
                    />
                </el-form-item>
                <el-form-item>
                    <el-select v-model="query.role" placeholder="角色" clearable>
                        <el-option label="管理员" value="admin" />
                        <el-option label="警员" value="officer" />
                        <el-option label="审计员" value="auditor" />
                    </el-select>
                </el-form-item>
                <el-form-item><el-input v-model="query.resource" placeholder="资源类型或 ID" clearable /></el-form-item>
                <el-form-item><el-input v-model="query.address" placeholder="地址" clearable /></el-form-item>
            </el-form>

            <el-tabs v-model="activeTab" @tab-click="handleTabClick">
                <el-tab-pane label="审计日志" name="audit">
                    <el-table :data="logs" v-loading="loadingLogs" empty-text="暂无审计日志">
                        <el-table-column prop="created_at" label="时间" width="170">
                            <template slot-scope="scope">{{ formatTime(scope.row.created_at) }}</template>
                        </el-table-column>
                        <el-table-column prop="username" label="用户" width="120" />
                        <el-table-column label="角色" width="90">
                            <template slot-scope="scope">{{ roleText(scope.row.role) }}</template>
                        </el-table-column>
                        <el-table-column prop="action" label="动作" min-width="180" show-overflow-tooltip />
                        <el-table-column prop="resource_type" label="资源" width="120" show-overflow-tooltip />
                        <el-table-column prop="resource_id" label="资源ID" min-width="160" show-overflow-tooltip />
                        <el-table-column prop="redacted_detail" label="摘要" min-width="180" show-overflow-tooltip />
                        <el-table-column label="结果" width="90">
                            <template slot-scope="scope">
                                <el-tag :type="resultTag(scope.row.result)" size="small">{{ resultText(scope.row.result) }}</el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column prop="error_message" label="错误" min-width="160" show-overflow-tooltip />
                        <el-table-column label="操作" width="90" fixed="right">
                            <template slot-scope="scope">
                                <el-button type="text" @click="showDetail(scope.row)">详情</el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-pagination
                        layout="total, prev, pager, next"
                        :total="auditTotal"
                        :page-size="query.pageSize"
                        :current-page.sync="query.page"
                        @current-change="fetchLogs"
                    />
                </el-tab-pane>

                <el-tab-pane label="审批记录" name="approvals">
                    <el-table :data="approvals" v-loading="loadingApprovals" empty-text="暂无审批记录">
                        <el-table-column prop="created_at" label="时间" width="170">
                            <template slot-scope="scope">{{ formatTime(scope.row.created_at) }}</template>
                        </el-table-column>
                        <el-table-column prop="approval_id" label="审批ID" min-width="190" show-overflow-tooltip />
                        <el-table-column prop="operation" label="操作" width="160" show-overflow-tooltip />
                        <el-table-column prop="resource_id" label="资源ID" min-width="170" show-overflow-tooltip />
                        <el-table-column prop="requested_by" label="发起人" width="120" />
                        <el-table-column prop="approved_by" label="审批人" width="120" />
                        <el-table-column label="状态" width="100">
                            <template slot-scope="scope">
                                <el-tag :type="approvalTag(scope.row.status)" size="small">{{ approvalText(scope.row.status) }}</el-tag>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-pagination
                        layout="total, prev, pager, next"
                        :total="approvalTotal"
                        :page-size="approvalQuery.pageSize"
                        :current-page.sync="approvalQuery.page"
                        @current-change="fetchApprovals"
                    />
                </el-tab-pane>
            </el-tabs>
        </el-card>

        <el-dialog title="审计详情" :visible.sync="detailVisible" width="620px">
            <el-descriptions v-if="detail" :column="1" border size="small">
                <el-descriptions-item label="时间">{{ formatTime(detail.created_at) }}</el-descriptions-item>
                <el-descriptions-item label="用户">{{ detail.username || '-' }}</el-descriptions-item>
                <el-descriptions-item label="角色">{{ roleText(detail.role) }}</el-descriptions-item>
                <el-descriptions-item label="动作">{{ detail.action || '-' }}</el-descriptions-item>
                <el-descriptions-item label="资源">{{ detail.resource_type || '-' }}</el-descriptions-item>
                <el-descriptions-item label="资源ID">{{ detail.resource_id || '-' }}</el-descriptions-item>
                <el-descriptions-item label="结果">{{ resultText(detail.result) }}</el-descriptions-item>
                <el-descriptions-item label="摘要">{{ detail.redacted_detail || '-' }}</el-descriptions-item>
                <el-descriptions-item label="错误">{{ detail.error_message || '-' }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>
    </div>
</template>

<script>
export default {
    name: 'AuditLogs',
    data() {
        return {
            activeTab: 'audit',
            advancedVisible: false,
            logs: [],
            approvals: [],
            auditTotal: 0,
            approvalTotal: 0,
            query: this.defaultQuery(),
            approvalQuery: {
                page: 1,
                pageSize: 20
            },
            detail: null,
            detailVisible: false,
            loadingLogs: false,
            loadingApprovals: false
        }
    },
    created() {
        this.loadAll()
    },
    methods: {
        defaultQuery() {
            return {
                page: 1,
                pageSize: 20,
                username: '',
                action: '',
                caseNo: '',
                result: '',
                timeRange: [],
                role: '',
                resource: '',
                address: ''
            }
        },

        async loadAll() {
            await Promise.all([this.fetchLogs(), this.fetchApprovals()])
        },

        searchLogs() {
            this.query.page = 1
            this.fetchLogs()
        },

        async fetchLogs() {
            this.loadingLogs = true
            try {
                const response = await this.$offlineApi.listAudit(this.auditParams())
                const data = response.data.data || {}
                this.logs = data.items || response.data.logs || []
                this.auditTotal = data.total || this.logs.length
            } catch (error) {
                this.$message.error(this.apiError(error, '审计日志加载失败'))
            } finally {
                this.loadingLogs = false
            }
        },

        async fetchApprovals() {
            this.loadingApprovals = true
            try {
                const response = await this.$offlineApi.listApprovals(this.approvalQuery)
                const data = response.data.data || {}
                this.approvals = data.items || response.data.approvals || []
                this.approvalTotal = data.total || this.approvals.length
            } catch (error) {
                this.$message.error(this.apiError(error, '审批记录加载失败'))
            } finally {
                this.loadingApprovals = false
            }
        },

        auditParams() {
            const params = {
                page: this.query.page,
                pageSize: this.query.pageSize
            }
            if (this.query.timeRange && this.query.timeRange.length === 2) {
                params.time_from = new Date(this.query.timeRange[0]).toISOString()
                params.time_to = new Date(this.query.timeRange[1]).toISOString()
            }
            if (this.query.username) params.username = this.query.username
            if (this.query.role) params.role = this.query.role
            if (this.query.action) params.action = this.query.action
            if (this.query.resource) params.resource = this.query.resource
            if (this.query.caseNo) params.case_no = this.query.caseNo
            if (this.query.address) params.address = this.query.address
            if (this.query.result) params.result = this.query.result
            return params
        },

        resetQuery() {
            this.query = this.defaultQuery()
            this.fetchLogs()
        },

        handleTabClick() {
            if (this.activeTab === 'approvals' && this.approvals.length === 0) {
                this.fetchApprovals()
            }
        },

        showDetail(row) {
            this.detail = row
            this.detailVisible = true
        },

        formatTime(value) {
            return value ? new Date(value).toLocaleString() : '-'
        },

        roleText(role) {
            return { admin: '管理员', officer: '警员', auditor: '审计员' }[role] || role || '-'
        },

        resultText(result) {
            return { success: '成功', failure: '失败' }[result] || result || '-'
        },

        resultTag(result) {
            if (result === 'success') return 'success'
            if (result === 'failure') return 'danger'
            return 'info'
        },

        approvalText(status) {
            return { approved: '已通过', rejected: '已拒绝', pending: '待审批' }[status] || status || '-'
        },

        approvalTag(status) {
            if (status === 'approved') return 'success'
            if (status === 'rejected') return 'danger'
            return 'warning'
        },

        apiError(error, fallback) {
            return error.response?.data?.error || error.response?.data?.message || error.message || fallback
        }
    }
}
</script>

<style scoped>
.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.query {
    margin-bottom: 12px;
}

.advanced-query {
    padding: 12px 0 0;
    border-top: 1px solid #ebeef5;
}

.el-pagination {
    margin-top: 12px;
    text-align: right;
}
</style>
