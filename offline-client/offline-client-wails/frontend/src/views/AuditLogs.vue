<template>
    <div class="audit-logs-container">
        <el-card>
            <div slot="header">
                <span>审计记录</span>
                <el-button style="float: right; padding: 3px 0" type="text" @click="loadAll">
                    刷新
                </el-button>
            </div>

            <el-tabs v-model="activeTab">
                <el-tab-pane label="审计日志" name="audit">
                    <el-table :data="logs" style="width: 100%" v-loading="loadingLogs">
                        <el-table-column prop="created_at" label="时间" width="180">
                            <template slot-scope="scope">
                                {{ formatTime(scope.row.created_at) }}
                            </template>
                        </el-table-column>
                        <el-table-column prop="username" label="用户" width="120"></el-table-column>
                        <el-table-column prop="role" label="角色" width="100"></el-table-column>
                        <el-table-column prop="action" label="动作" width="180"></el-table-column>
                        <el-table-column prop="resource_type" label="资源" width="120"></el-table-column>
                        <el-table-column prop="resource_id" label="资源ID"></el-table-column>
                        <el-table-column prop="result" label="结果" width="100">
                            <template slot-scope="scope">
                                <el-tag :type="scope.row.result === 'success' ? 'success' : 'danger'">
                                    {{ scope.row.result }}
                                </el-tag>
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
                        <el-table-column prop="approval_id" label="审批ID"></el-table-column>
                        <el-table-column prop="operation" label="操作" width="180"></el-table-column>
                        <el-table-column prop="resource_id" label="资源ID"></el-table-column>
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
            loadingLogs: false,
            loadingApprovals: false
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
                const response = await this.$offlineApi.listAudit(200)
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
                const response = await this.$offlineApi.listApprovals(200)
                this.approvals = response.data.approvals || []
            } catch (error) {
                this.$message.error(this.apiError(error, '审批记录加载失败'))
            } finally {
                this.loadingApprovals = false
            }
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
.audit-logs-container {
    padding: 20px;
}
</style>
