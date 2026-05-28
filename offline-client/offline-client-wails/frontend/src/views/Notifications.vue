<template>
    <div class="page notifications-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">待处理邀请</h2>
                <p class="page-subtitle">确认前请插入对应安全芯片；同意时客户端会读取当前 SE CPLC 并回传服务端。</p>
            </div>
            <el-button icon="el-icon-delete" @click="clearNotifications">清空通知</el-button>
        </div>

        <div v-if="inviteNotifications.length" class="invite-grid">
            <el-card
                v-for="item in inviteNotifications"
                :key="notificationKey(item)"
                class="invite-card"
                :class="{ responded: item.responded }">
                <div slot="header" class="invite-header">
                    <div>
                        <el-tag :type="getTagType(item.type)">{{ formatMsgType(item.type) }}</el-tag>
                        <strong>{{ item.content.summary || inviteTitle(item) }}</strong>
                    </div>
                    <el-tag :type="getTaskTagType(item)">{{ getTaskStatusText(item) }}</el-tag>
                </div>

                <el-descriptions :column="1" size="small" border>
                    <el-descriptions-item label="任务编号">{{ item.content.task_no || '-' }}</el-descriptions-item>
                    <el-descriptions-item label="案件编号">{{ item.content.case_no || '-' }}</el-descriptions-item>
                    <el-descriptions-item label="地址">{{ short(item.content.address) || '-' }}</el-descriptions-item>
                    <el-descriptions-item v-if="item.content.message_hash" label="消息哈希">
                        {{ short(item.content.message_hash) }}
                    </el-descriptions-item>
                    <el-descriptions-item v-if="item.content.required_signers" label="门限">
                        {{ item.content.required_signers }} / {{ item.content.total_parties }}
                    </el-descriptions-item>
                    <el-descriptions-item label="发起人">
                        {{ item.content.initiator || '-' }}
                    </el-descriptions-item>
                    <el-descriptions-item label="分片序号">{{ item.content.party_index || '-' }}</el-descriptions-item>
                    <el-descriptions-item v-if="item.content.from_username" label="移出警员">
                        {{ item.content.from_username }}
                    </el-descriptions-item>
                    <el-descriptions-item v-if="item.content.to_username" label="接收警员">
                        {{ item.content.to_username }}
                    </el-descriptions-item>
                    <el-descriptions-item label="指定 SE">{{ item.content.se_id || '-' }}</el-descriptions-item>
                    <el-descriptions-item v-if="item.content.reason" label="原因">{{ item.content.reason }}</el-descriptions-item>
                </el-descriptions>

                <div v-if="item.content.display" class="display-block">
                    <div v-for="(value, key) in item.content.display" :key="key" class="display-item">
                        <span>{{ key }}</span>
                        <strong>{{ value }}</strong>
                    </div>
                </div>

                <div class="card-actions">
                    <template v-if="item.type === 'keygen_invite'">
                        <el-button type="success" size="small" :disabled="item.responded" @click="handleKeygenInviteAccept(item)">
                            读取 SE 并同意
                        </el-button>
                        <el-button size="small" :disabled="item.responded" @click="handleKeygenInviteReject(item)">拒绝</el-button>
                    </template>

                    <template v-else-if="item.type === 'sign_invite'">
                        <el-button type="success" size="small" :disabled="item.responded" @click="handleSignInviteAccept(item)">
                            读取 SE 并同意
                        </el-button>
                        <el-button size="small" :disabled="item.responded" @click="handleSignInviteReject(item)">拒绝</el-button>
                    </template>

                    <template v-else-if="item.type === 'destroy_invite'">
                        <el-button type="danger" size="small" :disabled="item.responded" @click="handleDestroyInviteAccept(item)">
                            确认销毁
                        </el-button>
                        <el-button size="small" :disabled="item.responded" @click="handleDestroyInviteReject(item)">拒绝</el-button>
                    </template>

                    <template v-else-if="item.type === 'transfer_invite'">
                        <el-button type="success" size="small" :disabled="item.responded" @click="handleTransferInviteAccept(item)">
                            同意移交
                        </el-button>
                        <el-button size="small" :disabled="item.responded" @click="handleTransferInviteReject(item)">拒绝</el-button>
                    </template>

                    <el-button type="text" size="small" @click="showMessageDetail(item)">详情</el-button>
                </div>
            </el-card>
        </div>

        <el-card v-if="otherNotifications.length" class="history-card">
            <div slot="header">最近状态</div>
            <el-table :data="otherNotifications" style="width: 100%">
                <el-table-column prop="type" label="类型" width="160">
                    <template slot-scope="scope">
                        <el-tag :type="getTagType(scope.row.type)">{{ formatMsgType(scope.row.type) }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column label="摘要">
                    <template slot-scope="scope">
                        {{ statusSummary(scope.row) }}
                    </template>
                </el-table-column>
                <el-table-column prop="timestamp" label="时间" width="180">
                    <template slot-scope="scope">
                        {{ formatTime(scope.row.timestamp) }}
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                    <template slot-scope="scope">
                        <el-button type="text" @click="showMessageDetail(scope.row)">详情</el-button>
                    </template>
                </el-table-column>
            </el-table>
        </el-card>

        <el-empty v-if="notifications.length === 0" description="暂无通知消息" :image-size="100"></el-empty>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { seApi } from '../services/wails-api'
import { sendWSMessage, WS_MESSAGE_TYPES, mpcTaskKey } from '../services/ws'

export default {
    name: 'Notifications',
    computed: {
        ...mapGetters(['notifications', 'mpcTasks']),
        inviteNotifications() {
            return this.notifications.filter(item => this.isInviteMessage(item.type))
        },
        otherNotifications() {
            return this.notifications.filter(item => !this.isInviteMessage(item.type)).slice().reverse().slice(0, 20)
        }
    },
    methods: {
        clearNotifications() {
            this.$store.commit('clearNotifications')
        },

        showMessageDetail(message) {
            this.$alert(JSON.stringify(message.content, null, 2), '消息详情', {
                closeOnClickModal: true
            })
        },

        notificationKey(item) {
            return `${item.type}:${item.content?.session_key || ''}:${item.content?.party_index || ''}:${this.formatTime(item.timestamp)}`
        },

        formatMsgType(type) {
            const typeMap = {
                keygen_invite: '密钥生成邀请',
                keygen_params: '密钥生成执行',
                keygen_complete: '密钥生成完成',
                sign_invite: '交易签名邀请',
                sign_params: '交易签名执行',
                sign_complete: '签名完成',
                destroy_invite: '销毁邀请',
                destroy_params: '销毁执行',
                destroy_complete: '销毁完成',
                transfer_invite: '分片移交确认',
                transfer_complete: '分片移交完成',
                error: '错误'
            }
            return typeMap[type] || type
        },

        inviteTitle(item) {
            const map = {
                keygen_invite: '密钥生成邀请',
                sign_invite: '交易签名邀请',
                destroy_invite: '分片销毁确认',
                transfer_invite: '分片移交确认'
            }
            return map[item.type] || item.type
        },

        getTagType(type) {
            if (type.includes('invite')) return 'warning'
            if (type.includes('complete')) return 'success'
            if (type.includes('error')) return 'danger'
            return 'info'
        },

        isInviteMessage(type) {
            return type === 'keygen_invite' || type === 'sign_invite' || type === 'destroy_invite' || type === 'transfer_invite'
        },

        taskKind(type) {
            if (type.startsWith('keygen')) return 'keygen'
            if (type.startsWith('sign')) return 'sign'
            if (type.startsWith('destroy')) return 'destroy'
            if (type.startsWith('transfer')) return 'transfer'
            return ''
        },

        getTaskStatus(notification) {
            const kind = this.taskKind(notification.type)
            if (!kind) return ''
            const task = this.mpcTasks[mpcTaskKey(kind, notification.content)] || {}
            return task.status || ''
        },

        getTaskStatusText(notification) {
            const statusMap = {
                invited: '待确认',
                accepted: '已同意',
                rejected: '已拒绝',
                running: '执行中',
                result_ready: '待回传',
                result_sent: '结果已回传',
                completed: '已完成',
                interrupted: '已重置'
            }
            if (notification.responded && !this.getTaskStatus(notification)) return '已响应'
            return statusMap[this.getTaskStatus(notification)] || (notification.responded ? '已响应' : '待确认')
        },

        getTaskTagType(notification) {
            const status = this.getTaskStatus(notification)
            if (status === 'running' || status === 'result_ready') return 'warning'
            if (status === 'rejected') return 'danger'
            if (status === 'result_sent' || status === 'completed' || status === 'accepted') return 'success'
            return notification.responded ? 'success' : 'warning'
        },

        statusSummary(notification) {
            const content = notification.content || {}
            return content.message || content.details || content.address || content.session_key || '-'
        },

        async handleKeygenInviteAccept(notification) {
            if (notification.responded) return
            try {
                const cplc = await this.readCPLC()
                this.sendOrThrow({
                    type: WS_MESSAGE_TYPES.KEYGEN_RESPONSE,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    cplc,
                    accept: true,
                    reason: ''
                })
                this.markNotificationResponded(notification)
                this.markTaskResponded('keygen', notification, true, '', cplc)
                this.$message.success('已接受密钥生成邀请')
            } catch (error) {
                await this.rejectAfterError(notification, WS_MESSAGE_TYPES.KEYGEN_RESPONSE, 'keygen', error)
            }
        },

        async handleKeygenInviteReject(notification) {
            await this.rejectInvite(notification, WS_MESSAGE_TYPES.KEYGEN_RESPONSE, 'keygen')
        },

        async handleSignInviteAccept(notification) {
            if (notification.responded) return
            try {
                const cplc = await this.readCPLC()
                this.sendOrThrow({
                    type: WS_MESSAGE_TYPES.SIGN_RESPONSE,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    cplc,
                    accept: true,
                    reason: ''
                })
                this.markNotificationResponded(notification)
                this.markTaskResponded('sign', notification, true, '', cplc)
                this.$message.success('已接受签名邀请')
            } catch (error) {
                await this.rejectAfterError(notification, WS_MESSAGE_TYPES.SIGN_RESPONSE, 'sign', error)
            }
        },

        async handleSignInviteReject(notification) {
            await this.rejectInvite(notification, WS_MESSAGE_TYPES.SIGN_RESPONSE, 'sign')
        },

        async handleDestroyInviteAccept(notification) {
            if (notification.responded) return
            try {
                await this.$confirm('确认对当前安全芯片执行密钥记录删除？', '销毁确认', { type: 'warning' })
            } catch {
                return
            }
            try {
                const cplc = await this.readCPLC()
                this.sendOrThrow({
                    type: WS_MESSAGE_TYPES.DESTROY_RESPONSE,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    cplc,
                    accept: true,
                    reason: ''
                })
                this.markNotificationResponded(notification)
                this.markTaskResponded('destroy', notification, true, '', cplc)
                this.$message.success('已确认密钥销毁邀请')
            } catch (error) {
                await this.rejectAfterError(notification, WS_MESSAGE_TYPES.DESTROY_RESPONSE, 'destroy', error)
            }
        },

        async handleDestroyInviteReject(notification) {
            await this.rejectInvite(notification, WS_MESSAGE_TYPES.DESTROY_RESPONSE, 'destroy')
        },

        async handleTransferInviteAccept(notification) {
            if (notification.responded) return
            try {
                this.sendOrThrow({
                    type: WS_MESSAGE_TYPES.TRANSFER_RESPONSE,
                    session_key: notification.content.session_key,
                    shard_id: notification.content.shard_id,
                    accept: true,
                    reason: ''
                })
                this.markNotificationResponded(notification)
                this.markTaskResponded('transfer', notification, true)
                this.$message.success('已同意分片移交')
            } catch (error) {
                this.$message.error('同意分片移交失败: ' + error.message)
            }
        },

        async handleTransferInviteReject(notification) {
            await this.rejectInvite(notification, WS_MESSAGE_TYPES.TRANSFER_RESPONSE, 'transfer')
        },

        async readCPLC() {
            const cplcResponse = await seApi.getCPLC()
            const cplc = cplcResponse.data.cplc_info || ''
            if (!cplc) {
                throw new Error('未读取到当前 SE CPLC')
            }
            return cplc
        },

        async rejectAfterError(notification, responseType, kind, error) {
            const reason = '获取CPLC失败: ' + error.message
            this.$message.error(reason)
            try {
                this.sendOrThrow({
                    type: responseType,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    shard_id: notification.content.shard_id,
                    cplc: '',
                    accept: false,
                    reason
                })
                this.markNotificationResponded(notification)
                this.markTaskResponded(kind, notification, false, reason)
            } catch {
                // 二次回传失败时保留本地错误提示。
            }
        },

        async rejectInvite(notification, responseType, kind) {
            if (notification.responded) return
            try {
                this.sendOrThrow({
                    type: responseType,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    shard_id: notification.content.shard_id,
                    cplc: '',
                    accept: false,
                    reason: '用户拒绝'
                })
                this.markNotificationResponded(notification)
                this.markTaskResponded(kind, notification, false, '用户拒绝')
                this.$message.info('已拒绝邀请')
            } catch (error) {
                this.$message.error('拒绝邀请失败: ' + error.message)
            }
        },

        markNotificationResponded(notification) {
            this.$store.commit('updateNotificationResponse', {
                timestamp: notification.timestamp,
                type: notification.type,
                responded: true
            })
        },

        sendOrThrow(message) {
            if (!sendWSMessage(message)) {
                throw new Error('WebSocket 未连接，无法回传响应')
            }
        },

        markTaskResponded(kind, notification, accepted, reason = '', cplc = '') {
            this.$store.commit('setMpcTask', {
                key: mpcTaskKey(kind, notification.content),
                patch: {
                    kind,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    shard_id: notification.content.shard_id,
                    status: accepted ? 'accepted' : 'rejected',
                    phase: notification.type,
                    success: accepted,
                    cplc,
                    message: accepted ? '用户已确认邀请' : reason
                }
            })
        },

        short(value) {
            if (!value) return ''
            if (value.length <= 24) return value
            return `${value.slice(0, 12)}...${value.slice(-8)}`
        },

        formatTime(value) {
            return value ? new Date(value).toLocaleString() : ''
        }
    }
}
</script>

<style scoped>
.invite-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
    gap: 16px;
}

.invite-card.responded {
    opacity: 0.82;
}

.invite-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
}

.invite-header > div {
    display: flex;
    align-items: center;
    gap: 8px;
}

.display-block {
    margin-top: 12px;
    padding: 10px;
    background: #f5f7fa;
    border: 1px solid #ebeef5;
    border-radius: 4px;
}

.display-item {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    line-height: 1.8;
    font-size: 13px;
}

.display-item span {
    color: #606266;
}

.card-actions {
    margin-top: 14px;
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
}

.history-card {
    margin-top: 16px;
}
</style>
