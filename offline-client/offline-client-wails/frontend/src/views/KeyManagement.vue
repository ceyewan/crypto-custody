<template>
    <div class="key-management-container">
        <el-card>
            <div slot="header">
                <span>密钥管理</span>
            </div>

            <el-form :model="queryForm" label-width="120px">
                <el-form-item label="密钥ID或地址">
                    <el-input v-model="queryForm.id"></el-input>
                </el-form-item>
                <el-form-item>
                    <el-button type="primary" :loading="loading" @click="queryKey">
                        查询
                    </el-button>
                </el-form-item>
            </el-form>

            <el-descriptions v-if="keyInfo" :column="2" border>
                <el-descriptions-item label="离线密钥ID">{{ keyInfo.offline_key_id }}</el-descriptions-item>
                <el-descriptions-item label="地址">{{ keyInfo.address }}</el-descriptions-item>
                <el-descriptions-item label="币种">{{ keyInfo.coin_type }}</el-descriptions-item>
                <el-descriptions-item label="算法">{{ keyInfo.algorithm }}</el-descriptions-item>
                <el-descriptions-item label="门限">{{ keyInfo.required_signers }} / {{ keyInfo.total_parties }}</el-descriptions-item>
                <el-descriptions-item label="归属">{{ keyInfo.logical_owner }}</el-descriptions-item>
                <el-descriptions-item label="状态">
                    <el-tag :type="statusTag(keyInfo.status)">{{ keyInfo.status }}</el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="任务编号">{{ keyInfo.task_no }}</el-descriptions-item>
            </el-descriptions>

            <el-table v-if="keyInfo && keyInfo.shards" :data="keyInfo.shards" style="width: 100%; margin-top: 20px">
                <el-table-column prop="shard_index" label="分片" width="80"></el-table-column>
                <el-table-column prop="username" label="参与者" width="120"></el-table-column>
                <el-table-column prop="se_cplc" label="SE CPLC"></el-table-column>
                <el-table-column prop="record_id" label="Record ID"></el-table-column>
                <el-table-column prop="encrypted_blob_sha256" label="密文摘要"></el-table-column>
                <el-table-column prop="status" label="状态" width="110">
                    <template slot-scope="scope">
                        <el-tag :type="statusTag(scope.row.status)">{{ scope.row.status }}</el-tag>
                    </template>
                </el-table-column>
            </el-table>

            <el-divider v-if="keyInfo && isAdmin"></el-divider>

            <el-form v-if="keyInfo && isAdmin" :model="transferForm" label-width="120px">
                <el-form-item label="新归属">
                    <el-input v-model="transferForm.newOwner"></el-input>
                </el-form-item>
                <el-form-item label="原因">
                    <el-input v-model="transferForm.reason"></el-input>
                </el-form-item>
                <el-form-item>
                    <el-button type="primary" :loading="transferring" @click="transferKey">
                        移交
                    </el-button>
                    <el-button type="danger" :loading="destroying" @click="destroyKey">
                        销毁
                    </el-button>
                </el-form-item>
            </el-form>
        </el-card>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { sendWSMessage } from '../services/ws'

export default {
    name: 'KeyManagement',
    data() {
        return {
            queryForm: {
                id: ''
            },
            transferForm: {
                newOwner: '',
                reason: ''
            },
            keyInfo: null,
            loading: false,
            transferring: false,
            destroying: false
        }
    },
    computed: {
        ...mapGetters(['isAdmin'])
    },
    methods: {
        async queryKey() {
            if (!this.queryForm.id) {
                this.$message.warning('请输入密钥ID或地址')
                return
            }
            this.loading = true
            try {
                const response = await this.$offlineApi.getKey(this.queryForm.id)
                this.keyInfo = response.data.key
                this.transferForm.newOwner = this.keyInfo.logical_owner || ''
            } catch (error) {
                this.keyInfo = null
                this.$message.error(this.apiError(error, '查询失败'))
            } finally {
                this.loading = false
            }
        },

        async transferKey() {
            if (!this.keyInfo || !this.transferForm.newOwner) {
                this.$message.warning('请输入新归属')
                return
            }
            this.transferring = true
            try {
                await this.$offlineApi.transferKey(this.keyInfo.offline_key_id, {
                    new_owner: this.transferForm.newOwner,
                    reason: this.transferForm.reason
                })
                this.$message.success('移交完成')
                await this.queryKey()
            } catch (error) {
                this.$message.error(this.apiError(error, '移交失败'))
            } finally {
                this.transferring = false
            }
        },

        async destroyKey() {
            if (!this.keyInfo) {
                return
            }
            try {
                await this.$confirm('确认销毁该离线密钥？', '销毁确认', { type: 'warning' })
            } catch {
                return
            }
            this.destroying = true
            try {
                const response = await this.$offlineApi.destroyKey(this.keyInfo.offline_key_id, {
                    reason: this.transferForm.reason
                })
                const message = response.data.message
                if (!sendWSMessage(message)) {
                    throw new Error('WebSocket未连接')
                }
                this.$store.commit('setCurrentSession', message.session_key)
                this.$message.success('销毁请求已发送，请等待参与方执行SE删除')
                this.$router.push('/notifications')
            } catch (error) {
                this.$message.error(this.apiError(error, '销毁失败'))
            } finally {
                this.destroying = false
            }
        },

        statusTag(status) {
            if (status === 'active') return 'success'
            if (status === 'destroyed') return 'danger'
            return 'warning'
        },

        apiError(error, fallback) {
            return error.response?.data?.error || error.response?.data?.message || error.message || fallback
        }
    }
}
</script>

<style scoped>
.key-management-container {
    padding: 20px;
}
</style>
