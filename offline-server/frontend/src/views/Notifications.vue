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
                            <el-button type="success" size="mini" @click="handleKeygenInviteAccept(scope.row)">
                                接受
                            </el-button>
                            <el-button type="danger" size="mini" @click="handleKeygenInviteReject(scope.row)">
                                拒绝
                            </el-button>
                        </div>

                        <!-- 针对签名邀请消息 -->
                        <div v-else-if="scope.row.type === 'sign_invite'">
                            <el-button type="success" size="mini" @click="handleSignInviteAccept(scope.row)">
                                接受
                            </el-button>
                            <el-button type="danger" size="mini" @click="handleSignInviteReject(scope.row)">
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
            </el-table>

            <div v-if="notifications.length === 0" class="empty-state">
                暂无通知消息
            </div>
        </el-card>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { mpcApi, seApi } from '../services/api'
import { sendWSMessage, WS_MESSAGE_TYPES } from '../services/ws'

export default {
    name: 'Notifications',
    computed: {
        ...mapGetters(['notifications'])
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
            return type === 'keygen_invite' || type === 'sign_invite'
        },

        // 处理密钥生成邀请接受
        async handleKeygenInviteAccept(notification) {
            try {
                // 获取当前用户的CPIC
                const cpicResponse = await seApi.getCPIC()
                const cpic = cpicResponse.data.cpic

                // 发送接受响应
                sendWSMessage({
                    type: WS_MESSAGE_TYPES.KEYGEN_RESPONSE,
                    session_key: notification.content.session_key,
                    part_index: notification.content.part_index,
                    cpic: cpic,
                    accept: true,
                    reason: ''
                })

                // 标记该通知已响应
                this.markNotificationResponded(notification)
                this.$message.success('已接受密钥生成邀请')
            } catch (error) {
                console.error('接受密钥生成邀请失败:', error)
                this.$message.error('接受密钥生成邀请失败: ' + error.message)
            }
        },

        // 处理密钥生成邀请拒绝
        async handleKeygenInviteReject(notification) {
            try {
                // 发送拒绝响应
                sendWSMessage({
                    type: WS_MESSAGE_TYPES.KEYGEN_RESPONSE,
                    session_key: notification.content.session_key,
                    part_index: notification.content.part_index,
                    cpic: '',
                    accept: false,
                    reason: '用户拒绝'
                })

                // 标记该通知已响应
                this.markNotificationResponded(notification)
                this.$message.info('已拒绝密钥生成邀请')
            } catch (error) {
                console.error('拒绝密钥生成邀请失败:', error)
                this.$message.error('拒绝密钥生成邀请失败: ' + error.message)
            }
        },

        // 处理签名邀请接受
        async handleSignInviteAccept(notification) {
            try {
                // 获取当前用户的CPIC
                const cpicResponse = await seApi.getCPIC()
                const cpic = cpicResponse.data.cpic

                // 发送接受响应
                sendWSMessage({
                    type: WS_MESSAGE_TYPES.SIGN_RESPONSE,
                    session_key: notification.content.session_key,
                    part_index: notification.content.part_index,
                    cpic: cpic,
                    accept: true,
                    reason: ''
                })

                // 标记该通知已响应
                this.markNotificationResponded(notification)
                this.$message.success('已接受签名邀请')
            } catch (error) {
                console.error('接受签名邀请失败:', error)
                this.$message.error('接受签名邀请失败: ' + error.message)
            }
        },

        // 处理签名邀请拒绝
        async handleSignInviteReject(notification) {
            try {
                // 发送拒绝响应
                sendWSMessage({
                    type: WS_MESSAGE_TYPES.SIGN_RESPONSE,
                    session_key: notification.content.session_key,
                    part_index: notification.content.part_index,
                    cpic: '',
                    accept: false,
                    reason: '用户拒绝'
                })

                // 标记该通知已响应
                this.markNotificationResponded(notification)
                this.$message.info('已拒绝签名邀请')
            } catch (error) {
                console.error('拒绝签名邀请失败:', error)
                this.$message.error('拒绝签名邀请失败: ' + error.message)
            }
        },

        // 标记通知已响应
        markNotificationResponded(notification) {
            const index = this.notifications.findIndex(n =>
                n.timestamp === notification.timestamp &&
                n.type === notification.type
            )

            if (index !== -1) {
                // Vue 无法直接修改数组元素的属性，需要用 this.$set
                this.$set(this.notifications[index], 'responded', true)
            }
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