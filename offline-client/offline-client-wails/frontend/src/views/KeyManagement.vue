<template>
    <div class="page key-management-page">
        <el-card>
            <div slot="header" class="header">
                <span>地址管理</span>
                <el-button size="small" icon="el-icon-refresh" :loading="loading" @click="loadAll">刷新</el-button>
            </div>

            <el-form :inline="true" :model="filters" class="query">
                <el-form-item>
                    <el-input v-model="filters.address" clearable placeholder="地址"></el-input>
                </el-form-item>
                <el-form-item v-if="activeTab === 'shards'">
                    <el-select v-model="filters.username" clearable filterable placeholder="全部">
                        <el-option v-for="user in participantUsers" :key="user.username" :label="participantLabel(user)" :value="user.username"></el-option>
                    </el-select>
                </el-form-item>
                <el-form-item>
                    <el-select v-model="filters.status" clearable placeholder="状态">
                        <el-option label="active" value="active"></el-option>
                        <el-option label="transferred" value="transferred"></el-option>
                        <el-option label="destroying" value="destroying"></el-option>
                        <el-option label="destroy_failed" value="destroy_failed"></el-option>
                        <el-option label="destroyed" value="destroyed"></el-option>
                    </el-select>
                </el-form-item>
                <el-form-item>
                    <el-button type="primary" @click="searchRecords">查询</el-button>
                    <el-button @click="resetFilters">重置</el-button>
                </el-form-item>
            </el-form>

            <el-tabs v-model="activeTab" @tab-click="searchRecords">
                <el-tab-pane label="地址列表" name="keys">
                    <el-table :data="filteredKeys" v-loading="loadingKeys" style="width: 100%">
                        <el-table-column prop="address" label="地址" min-width="220"></el-table-column>
                        <el-table-column prop="case_no" label="案件编号" width="150"></el-table-column>
                        <el-table-column prop="task_no" label="任务编号" width="170"></el-table-column>
                        <el-table-column label="门限" width="100">
                            <template slot-scope="scope">
                                {{ scope.row.required_signers }} / {{ scope.row.total_parties }}
                            </template>
                        </el-table-column>
                        <el-table-column prop="offline_key_id" label="私钥编号" min-width="180"></el-table-column>
                        <el-table-column prop="status" label="状态" width="110">
                            <template slot-scope="scope">
                                <el-tag :type="statusTag(scope.row.status)">{{ scope.row.status }}</el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column label="操作" width="190">
                            <template slot-scope="scope">
                                <el-button size="mini" @click="showKey(scope.row)">私钥分片</el-button>
                                <el-button
                                    v-if="isAdmin"
                                    type="danger"
                                    size="mini"
                                    :disabled="!canDestroyKey(scope.row)"
                                    @click="destroyKey(scope.row)">
                                    销毁
                                </el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                </el-tab-pane>

                <el-tab-pane label="私钥分片" name="shards">
                    <el-table :data="shards" v-loading="loadingShards" style="width: 100%">
                        <el-table-column prop="address" label="地址" min-width="220"></el-table-column>
                        <el-table-column prop="case_no" label="案件编号" width="150"></el-table-column>
                        <el-table-column prop="shard_index" label="私钥分片" width="90"></el-table-column>
                        <el-table-column label="门限" width="100">
                            <template slot-scope="scope">
                                {{ thresholdText(scope.row) }}
                            </template>
                        </el-table-column>
                        <el-table-column prop="username" label="持有人" width="130"></el-table-column>
                        <el-table-column prop="record_id" label="安全芯片记录" min-width="220"></el-table-column>
                        <el-table-column prop="se_cplc" label="安全芯片编号" min-width="220"></el-table-column>
                        <el-table-column prop="encrypted_blob_sha256" label="密文摘要" min-width="220"></el-table-column>
                        <el-table-column prop="status" label="状态" width="110">
                            <template slot-scope="scope">
                                <el-tag :type="statusTag(scope.row.status)">{{ scope.row.status }}</el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column v-if="isAdmin" label="操作" width="120">
                            <template slot-scope="scope">
                                <el-button
                                    size="mini"
                                    type="primary"
                                    :disabled="scope.row.status !== 'active'"
                                    @click="openTransfer(scope.row)">
                                    移交
                                </el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                </el-tab-pane>
            </el-tabs>
        </el-card>

        <el-dialog title="私钥分片移交" :visible.sync="transferDialogVisible" width="560px">
            <el-descriptions v-if="transferShard" :column="1" border size="small">
                <el-descriptions-item label="地址">{{ transferShard.address }}</el-descriptions-item>
                <el-descriptions-item label="私钥分片">{{ transferShard.shard_index }}</el-descriptions-item>
                <el-descriptions-item label="当前持有人">{{ transferShard.username }}</el-descriptions-item>
                <el-descriptions-item label="安全芯片记录">{{ transferShard.record_id }}</el-descriptions-item>
            </el-descriptions>

            <el-form :model="transferForm" label-width="120px" class="transfer-form">
                <el-form-item label="接收警员">
                    <el-select v-model="transferForm.toUsername" filterable placeholder="请选择接收警员" style="width: 100%">
                        <el-option
                            v-for="user in participantUsers"
                            :key="user.username"
                            :disabled="transferShard && user.username === transferShard.username"
                            :label="participantLabel(user)"
                            :value="user.username">
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="原因">
                    <el-input v-model="transferForm.reason" placeholder="例如 人员离职交接"></el-input>
                </el-form-item>
            </el-form>

            <span slot="footer">
                <el-button @click="transferDialogVisible = false">取消</el-button>
                <el-button type="primary" :loading="transferring" @click="transferSelectedShard">确认移交</el-button>
            </span>
        </el-dialog>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { sendWSMessage } from '../services/ws'
import { userApi } from '../services/api'

export default {
    name: 'KeyManagement',
    data() {
        return {
            activeTab: 'keys',
            filters: {
                address: '',
                username: '',
                status: ''
            },
            appliedFilters: {
                address: '',
                username: '',
                status: ''
            },
            keys: [],
            shards: [],
            participantUsers: [],
            loading: false,
            loadingKeys: false,
            loadingShards: false,
            transferring: false,
            destroying: false,
            transferDialogVisible: false,
            transferShard: null,
            transferForm: {
                toUsername: '',
                reason: ''
            }
        }
    },
    computed: {
        ...mapGetters(['isAdmin']),
        filteredKeys() {
            return this.keys.filter(item => {
                const address = String(item.address || '').toLowerCase()
                if (this.appliedFilters.address && !address.includes(this.appliedFilters.address.toLowerCase())) return false
                if (this.appliedFilters.status && item.status !== this.appliedFilters.status) return false
                return true
            })
        }
    },
    created() {
        this.loadAll()
    },
    methods: {
        async loadAll() {
            this.loading = true
            try {
                await Promise.all([this.loadKeys(), this.loadShards(), this.loadUsers()])
            } finally {
                this.loading = false
            }
        },

        async loadKeys() {
            this.loadingKeys = true
            try {
                const response = await this.$offlineApi.listKeys()
                this.keys = response.data.keys || []
            } catch (error) {
                this.$message.error(this.apiError(error, '查询密钥列表失败'))
            } finally {
                this.loadingKeys = false
            }
        },

        async loadShards() {
            this.loadingShards = true
            try {
                const params = Object.fromEntries(Object.entries(this.appliedFilters).filter(([, value]) => value))
                const response = await this.$offlineApi.listShards(params)
                this.shards = response.data.shards || []
            } catch (error) {
                this.$message.error(this.apiError(error, '查询私钥分片列表失败'))
            } finally {
                this.loadingShards = false
            }
        },

        async loadUsers() {
            try {
                const response = await userApi.getUsers()
                const users = response.data.users || response.data.data || []
                this.participantUsers = users.filter(user => ['admin', 'officer'].includes(user.role))
            } catch {
                this.participantUsers = []
            }
        },

        searchRecords() {
            this.appliedFilters = { ...this.filters }
            if (this.activeTab === 'shards') {
                this.loadShards()
            }
        },

        resetFilters() {
            this.filters = {
                address: '',
                username: '',
                status: ''
            }
            this.appliedFilters = { ...this.filters }
            if (this.activeTab === 'shards') {
                this.loadShards()
            }
        },

        showKey(key) {
            this.filters.address = key.address
            this.appliedFilters = { ...this.filters }
            this.activeTab = 'shards'
            this.loadShards()
        },

        openTransfer(shard) {
            this.transferShard = shard
            this.transferForm = {
                toUsername: '',
                reason: ''
            }
            this.transferDialogVisible = true
        },

        async transferSelectedShard() {
            if (!this.isAdmin) {
                this.$message.error('只有管理员可以发起私钥分片移交')
                return
            }
            if (!this.transferShard || !this.transferForm.toUsername) {
                this.$message.warning('请选择接收警员')
                return
            }
            this.transferring = true
            try {
                const response = await this.$offlineApi.transferShard(this.transferShard.shard_id, {
                    to_username: this.transferForm.toUsername,
                    reason: this.transferForm.reason
                })
                const message = response.data.message
                if (!sendWSMessage(message)) {
                    throw new Error('服务连接未建立')
                }
                this.$store.commit('setCurrentSession', message.session_key)
                this.$message.success('私钥分片移交邀请已发送，等待移出和接收双方确认')
                this.transferDialogVisible = false
                this.$router.push('/notifications')
            } catch (error) {
                this.$message.error(this.apiError(error, '私钥分片移交失败'))
            } finally {
                this.transferring = false
            }
        },

        async destroyKey(key) {
            if (!this.isAdmin) {
                this.$message.error('只有管理员可以发起私钥销毁')
                return
            }
            try {
                await this.$confirm('确认发起该地址的私钥销毁流程？参与警员仍需在各自客户端确认。', '私钥销毁确认', { type: 'warning' })
            } catch {
                return
            }
            this.destroying = true
            try {
                const response = await this.$offlineApi.destroyKey(key.offline_key_id, {
                    reason: '管理员发起地址销毁'
                })
                const message = response.data.message
                if (!sendWSMessage(message)) {
                    throw new Error('服务连接未建立')
                }
                this.$store.commit('setCurrentSession', message.session_key)
                this.$message.success('私钥销毁请求已发送，请等待私钥分片持有人确认')
                this.$router.push('/notifications')
            } catch (error) {
                this.$message.error(this.apiError(error, '销毁失败'))
            } finally {
                this.destroying = false
            }
        },

        participantLabel(user) {
            const roleText = user.role === 'admin' ? '管理员' : '警员'
            return `${user.nickname || user.username} (${user.username}, ${roleText})`
        },

        thresholdText(row) {
            if (!row.required_signers || !row.total_parties) return '-'
            return `${row.required_signers} / ${row.total_parties}`
        },

        statusTag(status) {
            if (status === 'active') return 'success'
            if (status === 'destroyed') return 'danger'
            if (status === 'destroy_failed') return 'danger'
            return 'warning'
        },

        canDestroyKey(key) {
            return key && (key.status === 'active' || key.status === 'destroy_failed')
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
    align-items: center;
    justify-content: space-between;
}

.query {
    margin-bottom: 10px;
}

.transfer-form {
    margin-top: 16px;
}
</style>
