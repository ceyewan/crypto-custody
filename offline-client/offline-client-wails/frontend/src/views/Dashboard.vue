<template>
    <div class="page">
        <div class="page-header">
            <div>
                <h2 class="page-title">{{ title }}</h2>
                <p class="page-subtitle">{{ subtitle }}</p>
            </div>
            <el-button v-if="isAdmin" type="primary" icon="el-icon-upload2" @click="$router.push('/offline-tasks')">
                导入任务包
            </el-button>
        </div>

        <el-row :gutter="16">
            <el-col :span="6">
                <el-card class="metric-card">
                    <div class="metric-value">{{ pendingInvitations.length }}</div>
                    <div class="metric-label">待处理邀请</div>
                </el-card>
            </el-col>
            <el-col :span="6">
                <el-card class="metric-card">
                    <div class="metric-value">{{ taskCount }}</div>
                    <div class="metric-label">本机任务记录</div>
                </el-card>
            </el-col>
            <el-col :span="6">
                <el-card class="metric-card">
                    <div class="metric-value">{{ wsConnected ? '正常' : '断开' }}</div>
                    <div class="metric-label">WebSocket</div>
                </el-card>
            </el-col>
            <el-col :span="6">
                <el-card class="metric-card">
                    <div class="metric-value small">{{ currentUser && currentUser.username }}</div>
                    <div class="metric-label">登录标识</div>
                </el-card>
            </el-col>
        </el-row>

        <el-row v-if="isOfficer" :gutter="16" class="section-row">
            <el-col :span="12">
                <el-card>
                    <div slot="header" class="card-header">
                        <span>我的私钥分片</span>
                        <el-button type="text" @click="$router.push('/my-shards')">查看全部</el-button>
                    </div>
                    <el-table :data="myShardSummary" v-loading="loadingOfficerSnapshot" size="mini" style="width: 100%">
                        <el-table-column prop="address" label="地址" min-width="160">
                            <template slot-scope="scope">
                                {{ short(scope.row.address) }}
                            </template>
                        </el-table-column>
                        <el-table-column prop="case_no" label="案件" width="120"></el-table-column>
                        <el-table-column prop="shard_index" label="私钥分片" width="90"></el-table-column>
                        <el-table-column label="门限" width="80">
                            <template slot-scope="scope">
                                {{ thresholdText(scope.row) }}
                            </template>
                        </el-table-column>
                        <el-table-column prop="status" label="状态" width="90"></el-table-column>
                    </el-table>
                    <el-empty v-if="!loadingOfficerSnapshot && myShardSummary.length === 0" description="暂无私钥分片" :image-size="70"></el-empty>
                </el-card>
            </el-col>

            <el-col :span="12">
                <el-card>
                    <div slot="header" class="card-header">
                        <span>最近参与记录</span>
                        <el-button type="text" @click="$router.push('/participation')">查看全部</el-button>
                    </div>
                    <el-table :data="participationSummary" v-loading="loadingOfficerSnapshot" size="mini" style="width: 100%">
                        <el-table-column prop="type" label="类型" width="90">
                            <template slot-scope="scope">
                                {{ participationTypeText(scope.row.type) }}
                            </template>
                        </el-table-column>
                        <el-table-column prop="resource_id" label="资源" min-width="150">
                            <template slot-scope="scope">
                                {{ short(scope.row.resource_id) }}
                            </template>
                        </el-table-column>
                        <el-table-column prop="result" label="结果" width="90"></el-table-column>
                    </el-table>
                    <el-empty v-if="!loadingOfficerSnapshot && participationSummary.length === 0" description="暂无参与记录" :image-size="70"></el-empty>
                </el-card>
            </el-col>
        </el-row>

        <el-row :gutter="16" class="section-row">
            <el-col :span="isAdmin ? 14 : 24">
                <el-card>
                    <div slot="header" class="card-header">
                        <span>{{ isOfficer ? '待处理邀请' : '最近任务状态' }}</span>
                        <el-button type="text" @click="$router.push(statusRoute)">查看全部</el-button>
                    </div>
                    <div v-if="pendingInvitations.length" class="invite-list">
                        <div v-for="item in pendingInvitations.slice(0, 4)" :key="notificationKey(item)" class="invite-item">
                            <div>
                                <strong>{{ inviteTitle(item) }}</strong>
                                <p>{{ inviteSummary(item) }}</p>
                            </div>
                            <el-button size="mini" type="primary" @click="$router.push(statusRoute)">处理</el-button>
                        </div>
                    </div>
                    <el-empty v-else description="暂无待处理邀请"></el-empty>
                </el-card>
            </el-col>

            <el-col v-if="isAdmin" :span="10">
                <el-card>
                    <div slot="header">管理员快捷操作</div>
                    <div class="quick-actions">
                        <el-button icon="el-icon-upload2" @click="$router.push('/offline-tasks')">导入 JSON 任务包</el-button>
                        <el-button icon="el-icon-key" @click="$router.push('/keygen')">生成私钥</el-button>
                        <el-button icon="el-icon-s-finance" @click="$router.push('/sign')">交易签名</el-button>
                        <el-button icon="el-icon-cpu" @click="$router.push('/security-elements')">安全芯片管理</el-button>
                    </div>
                </el-card>
            </el-col>
        </el-row>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'

export default {
    name: 'Dashboard',
    data() {
        return {
            myShardSummary: [],
            participationSummary: [],
            loadingOfficerSnapshot: false
        }
    },
    computed: {
        ...mapGetters(['currentUser', 'isAdmin', 'isOfficer', 'isAuditor', 'notifications', 'mpcTasks', 'wsConnected']),
        title() {
            if (this.isOfficer) return '我的工作台'
            if (this.isAuditor) return '审计工作台'
            return '离线工作台'
        },
        subtitle() {
            if (this.isOfficer) return '处理私钥生成、签名等邀请，查看自己的私钥分片和参与记录。'
            if (this.isAuditor) return '查看离线系统任务、密钥和安全操作审计。'
            return '通过 JSON 任务包生成托管地址和私钥分片，完成签名后导出结果包回传在线系统。'
        },
        pendingInvitations() {
            return this.notifications.filter(n =>
                ['keygen_invite', 'sign_invite', 'destroy_invite', 'transfer_invite'].includes(n.type) &&
                !n.responded
            )
        },
        taskCount() {
            return Object.keys(this.mpcTasks || {}).length
        },
        statusRoute() {
            if (this.isAuditor && !this.isOfficer && !this.isAdmin) {
                return '/audit'
            }
            return '/notifications'
        }
    },
    created() {
        if (this.isOfficer) {
            this.loadOfficerSnapshot()
        }
    },
    methods: {
        async loadOfficerSnapshot() {
            this.loadingOfficerSnapshot = true
            try {
                const [shards, records] = await Promise.all([
                    this.$offlineApi.listMyShards(),
                    this.$offlineApi.listMyParticipation(20)
                ])
                this.myShardSummary = (shards.data.shards || []).slice(0, 5)
                this.participationSummary = (records.data.records || []).slice(0, 5)
            } catch (error) {
                this.$message.error(error.response?.data?.error || error.message || '加载警员工作台失败')
            } finally {
                this.loadingOfficerSnapshot = false
            }
        },
        notificationKey(item) {
            return `${item.type}:${item.content && item.content.session_key}:${item.timestamp}`
        },
        inviteTitle(item) {
            const map = {
                keygen_invite: '私钥生成邀请',
                sign_invite: '交易签名邀请',
                destroy_invite: '私钥销毁确认',
                transfer_invite: '私钥分片移交确认'
            }
            return map[item.type] || item.type
        },
        inviteSummary(item) {
            const content = item.content || {}
            return [
                content.case_no ? `案件 ${content.case_no}` : '',
                content.address ? `地址 ${this.short(content.address)}` : '',
                content.message_hash ? `哈希 ${this.short(content.message_hash)}` : '',
                content.initiator ? `发起人 ${content.initiator}` : ''
            ].filter(Boolean).join(' / ') || '等待确认'
        },
        short(value) {
            if (!value || value.length <= 18) return value || ''
            return `${value.slice(0, 10)}...${value.slice(-6)}`
        },
        thresholdText(row) {
            if (!row.required_signers || !row.total_parties) return '-'
            return `${row.required_signers}/${row.total_parties}`
        },
        participationTypeText(type) {
            const map = {
                keygen: '私钥生成',
                sign: '交易签名',
                transfer: '移交',
                destroy: '销毁'
            }
            return map[type] || type || '-'
        }
    }
}
</script>

<style scoped>
.metric-card {
    min-height: 96px;
}

.metric-value {
    font-size: 28px;
    font-weight: 600;
    color: #303133;
}

.metric-value.small {
    font-size: 18px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.metric-label {
    margin-top: 8px;
    color: #606266;
}

.section-row {
    margin-top: 16px;
}

.card-header,
.invite-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.invite-item {
    padding: 12px 0;
    border-bottom: 1px solid #ebeef5;
}

.invite-item:last-child {
    border-bottom: none;
}

.invite-item p {
    margin: 6px 0 0;
    color: #606266;
    font-size: 13px;
}

.quick-actions {
    display: grid;
    grid-template-columns: 1fr;
    gap: 10px;
}

.quick-actions .el-button {
    margin-left: 0;
}
</style>
