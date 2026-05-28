<template>
    <div class="notifications-container">
        <el-card>
            <div slot="header" class="clearfix">
                <span>通知消息</span>
                <el-button style="float: right; padding: 3px 0" type="text" @click="clearNotifications">
                    清空通知
                </el-button>
            </div>

            <el-table :data="notifications" style="width: 100%">
                <el-table-column prop="type" label="消息类型" width="180">
                    <template slot-scope="scope">
                        <el-tag :type="getTagType(scope.row.type)">{{ formatMsgType(scope.row.type) }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="timestamp" label="时间" width="180">
                    <template slot-scope="scope">
                        {{ new Date(scope.row.timestamp).toLocaleString() }}
                    </template>
                </el-table-column>
                <el-table-column label="操作">
                    <template slot-scope="scope">
                        <!-- 针对密钥生成邀请消息 -->
                        <div v-if="scope.row.type === 'keygen_invite'">
                            <el-button type="success" size="mini" :disabled="scope.row.responded"
                                @click="handleKeygenInviteAccept(scope.row)">
                                接受
                            </el-button>
                            <el-button type="danger" size="mini" :disabled="scope.row.responded"
                                @click="handleKeygenInviteReject(scope.row)">
                                拒绝
                            </el-button>
                        </div>

                        <!-- 针对签名邀请消息 -->
                        <div v-else-if="scope.row.type === 'sign_invite'">
                            <el-button type="success" size="mini" :disabled="scope.row.responded"
                                @click="handleSignInviteAccept(scope.row)">
                                接受
                            </el-button>
                            <el-button type="danger" size="mini" :disabled="scope.row.responded"
                                @click="handleSignInviteReject(scope.row)">
                                拒绝
                            </el-button>
                        </div>

                        <!-- 针对密钥销毁邀请消息 -->
                        <div v-else-if="scope.row.type === 'destroy_invite'">
                            <el-button type="danger" size="mini" :disabled="scope.row.responded"
                                @click="handleDestroyInviteAccept(scope.row)">
                                确认销毁
                            </el-button>
                            <el-button size="mini" :disabled="scope.row.responded" @click="handleDestroyInviteReject(scope.row)">
                                拒绝
                            </el-button>
                        </div>

                        <!-- 其他消息类型 -->
                        <el-button type="text" @click="showMessageDetail(scope.row)">
                            查看详情
                        </el-button>
                    </template>
                </el-table-column>
                <el-table-column label="状态" width="120">
                    <template slot-scope="scope">
                        <el-tag v-if="scope.row.responded" type="success">已响应</el-tag>
                        <el-tag v-else-if="isInviteMessage(scope.row.type)" type="warning">待响应</el-tag>
                        <el-tag v-else type="info">-</el-tag>
                    </template>
                </el-table-column>
                <el-table-column label="任务状态" width="150">
                    <template slot-scope="scope">
                        <el-tag :type="getTaskTagType(scope.row)">{{ getTaskStatusText(scope.row) }}</el-tag>
                    </template>
                </el-table-column>
            </el-table>

            <div v-if="notifications.length === 0" class="empty-state">
                暂无通知消息
            </div>
        </el-card>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { seApi } from '../services/wails-api'
import { sendWSMessage, WS_MESSAGE_TYPES, mpcTaskKey } from '../services/ws'

export default {
    name: 'Notifications',
    computed: {
        ...mapGetters(['notifications', 'mpcTasks'])
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

        // 格式化消息类型显示
        formatMsgType(type) {
            const typeMap = {
                'keygen_invite': '密钥生成邀请',
                'keygen_params': '密钥生成参数',
                'keygen_complete': '密钥生成完成',
                'sign_invite': '签名邀请',
                'sign_params': '签名参数',
                'sign_complete': '签名完成',
                'destroy_invite': '销毁邀请',
                'destroy_params': '销毁参数',
                'destroy_complete': '销毁完成',
                'error': '错误'
            }
            return typeMap[type] || type
        },

        // 获取标签类型
        getTagType(type) {
            if (type.includes('invite')) return 'warning'
            if (type.includes('complete')) return 'success'
            if (type.includes('error')) return 'danger'
            return 'info'
        },

        // 判断是否是需要响应的邀请消息
        isInviteMessage(type) {
            return type === 'keygen_invite' || type === 'sign_invite' || type === 'destroy_invite'
        },

        taskKind(type) {
            if (type.startsWith('keygen')) return 'keygen'
            if (type.startsWith('sign')) return 'sign'
            if (type.startsWith('destroy')) return 'destroy'
            return ''
        },

        getTaskStatus(notification) {
            const kind = this.taskKind(notification.type)
            if (!kind) {
                return ''
            }
            const task = this.mpcTasks[mpcTaskKey(kind, notification.content)] || {}
            return task.status || ''
        },

        getTaskStatusText(notification) {
            const statusMap = {
                invited: '已收到邀请',
                accepted: '已同意',
                rejected: '已拒绝',
                running: '执行中',
                result_ready: '待回传',
                result_sent: '结果已回传',
                completed: '已完成',
                interrupted: '已重置'
            }
            return statusMap[this.getTaskStatus(notification)] || '-'
        },

        getTaskTagType(notification) {
            const status = this.getTaskStatus(notification)
            if (status === 'running' || status === 'result_ready') return 'warning'
            if (status === 'rejected') return 'danger'
            if (status === 'result_sent' || status === 'completed' || status === 'accepted') return 'success'
            return 'info'
        },

        // 处理密钥生成邀请接受
        async handleKeygenInviteAccept(notification) {
            if (notification.responded) {
                return
            }
            try {
                // 获取当前用户的CPLC
                const cplcResponse = await seApi.getCPLC()
                const cplc = cplcResponse.data.cplc_info || ''

                // 发送接受响应
                this.sendOrThrow({
                    type: WS_MESSAGE_TYPES.KEYGEN_RESPONSE,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    cplc: cplc,
                    accept: true,
                    reason: ''
                })

                // 标记该通知已响应
                this.markNotificationResponded(notification)
                this.markTaskResponded('keygen', notification, true, '', cplc)
                this.$message.success('已接受密钥生成邀请')
            } catch (error) {
                this.$message.error('接受密钥生成邀请失败: ' + error.message)

                // 通知协调者该参与者无法接受邀请
                try {
                    this.sendOrThrow({
                        type: WS_MESSAGE_TYPES.KEYGEN_RESPONSE,
                        session_key: notification.content.session_key,
                        party_index: notification.content.party_index,
                        cplc: '',
                        accept: false,
                        reason: '获取CPLC失败: ' + error.message
                    })
                    this.markNotificationResponded(notification)
                    this.markTaskResponded('keygen', notification, false, '获取CPLC失败: ' + error.message)
                } catch {
                    // 忽略二次错误
                }
            }
        },

        // 处理密钥生成邀请拒绝
        async handleKeygenInviteReject(notification) {
            if (notification.responded) {
                return
            }
            try {
                // 发送拒绝响应
                this.sendOrThrow({
                    type: WS_MESSAGE_TYPES.KEYGEN_RESPONSE,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    cplc: '',
                    accept: false,
                    reason: '用户拒绝'
                })

                // 标记该通知已响应
                this.markNotificationResponded(notification)
                this.markTaskResponded('keygen', notification, false, '用户拒绝')
                this.$message.info('已拒绝密钥生成邀请')
            } catch (error) {
                this.$message.error('拒绝密钥生成邀请失败: ' + error.message)
            }
        },

        // 处理签名邀请接受
        async handleSignInviteAccept(notification) {
            if (notification.responded) {
                return
            }
            try {
                // 获取当前用户的CPLC
                const cplcResponse = await seApi.getCPLC()
                const cplc = cplcResponse.data.cplc_info || ''

                // 发送接受响应
                this.sendOrThrow({
                    type: WS_MESSAGE_TYPES.SIGN_RESPONSE,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    cplc: cplc,
                    accept: true,
                    reason: ''
                })

                // 标记该通知已响应
                this.markNotificationResponded(notification)
                this.markTaskResponded('sign', notification, true, '', cplc)
                this.$message.success('已接受签名邀请')
            } catch (error) {
                this.$message.error('接受签名邀请失败: ' + error.message)

                // 通知协调者该参与者无法接受邀请
                try {
                    this.sendOrThrow({
                        type: WS_MESSAGE_TYPES.SIGN_RESPONSE,
                        session_key: notification.content.session_key,
                        party_index: notification.content.party_index,
                        cplc: '',
                        accept: false,
                        reason: '获取CPLC失败: ' + error.message
                    })
                    this.markNotificationResponded(notification)
                    this.markTaskResponded('sign', notification, false, '获取CPLC失败: ' + error.message)
                } catch {
                    // 忽略二次错误
                }
            }
        },

        // 处理签名邀请拒绝
        async handleSignInviteReject(notification) {
            if (notification.responded) {
                return
            }
            try {
                // 发送拒绝响应
                this.sendOrThrow({
                    type: WS_MESSAGE_TYPES.SIGN_RESPONSE,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    cplc: '',
                    accept: false,
                    reason: '用户拒绝'
                })

                // 标记该通知已响应
                this.markNotificationResponded(notification)
                this.markTaskResponded('sign', notification, false, '用户拒绝')
                this.$message.info('已拒绝签名邀请')
            } catch (error) {
                this.$message.error('拒绝签名邀请失败: ' + error.message)
            }
        },

        // 处理密钥销毁邀请接受
        async handleDestroyInviteAccept(notification) {
            if (notification.responded) {
                return
            }
            try {
                await this.$confirm('确认对当前安全芯片执行密钥记录删除？', '销毁确认', { type: 'warning' })
            } catch {
                return
            }
            try {
                const cplcResponse = await seApi.getCPLC()
                const cplc = cplcResponse.data.cplc_info || ''

                this.sendOrThrow({
                    type: WS_MESSAGE_TYPES.DESTROY_RESPONSE,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    cplc: cplc,
                    accept: true,
                    reason: ''
                })

                this.markNotificationResponded(notification)
                this.markTaskResponded('destroy', notification, true, '', cplc)
                this.$message.success('已确认密钥销毁邀请')
            } catch (error) {
                this.$message.error('确认销毁邀请失败: ' + error.message)
                try {
                    this.sendOrThrow({
                        type: WS_MESSAGE_TYPES.DESTROY_RESPONSE,
                        session_key: notification.content.session_key,
                        party_index: notification.content.party_index,
                        cplc: '',
                        accept: false,
                        reason: '获取CPLC失败: ' + error.message
                    })
                    this.markNotificationResponded(notification)
                    this.markTaskResponded('destroy', notification, false, '获取CPLC失败: ' + error.message)
                } catch {
                    // 忽略二次错误
                }
            }
        },

        // 处理密钥销毁邀请拒绝
        async handleDestroyInviteReject(notification) {
            if (notification.responded) {
                return
            }
            try {
                this.sendOrThrow({
                    type: WS_MESSAGE_TYPES.DESTROY_RESPONSE,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    cplc: '',
                    accept: false,
                    reason: '用户拒绝'
                })

                this.markNotificationResponded(notification)
                this.markTaskResponded('destroy', notification, false, '用户拒绝')
                this.$message.info('已拒绝密钥销毁邀请')
            } catch (error) {
                this.$message.error('拒绝销毁邀请失败: ' + error.message)
            }
        },

        // 标记通知已响应
        markNotificationResponded(notification) {
            // 使用 Vuex mutation 来更新状态，遵循单向数据流原则
            this.$store.commit('updateNotificationResponse', {
                timestamp: notification.timestamp,
                type: notification.type,
                responded: true
            })
        },

        sendOrThrow(message) {
            if (!sendWSMessage(message)) {
                throw new Error('WebSocket未连接，无法回传响应')
            }
        },

        markTaskResponded(kind, notification, accepted, reason = '', cplc = '') {
            this.$store.commit('setMpcTask', {
                key: mpcTaskKey(kind, notification.content),
                patch: {
                    kind,
                    session_key: notification.content.session_key,
                    party_index: notification.content.party_index,
                    status: accepted ? 'accepted' : 'rejected',
                    phase: notification.type,
                    success: accepted,
                    cplc,
                    message: accepted ? '用户已确认邀请' : reason
                }
            })
        }
    }
}
</script>

<style scoped>
.notifications-container {
    padding: 20px;
}

.empty-state {
    text-align: center;
    padding: 50px 0;
    color: #909399;
}
</style>
