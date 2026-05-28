<template>
    <div class="sign-container">
        <el-card>
            <div slot="header">
                <span>交易签名</span>
            </div>

            <el-form :model="signForm" :rules="signRules" ref="signForm" label-width="120px">
                <el-form-item label="账户地址" prop="address">
                    <el-input v-model="signForm.address"></el-input>
                    <el-button type="text" @click="fetchSignAvailableParticipants" style="margin-top: 5px">
                        获取可用参与者
                    </el-button>
                </el-form-item>

                <el-form-item label="待签名数据" prop="data">
                    <el-input v-model="signForm.data" type="textarea" :rows="2"></el-input>
                </el-form-item>

                <el-form-item label="参与者" prop="participants">
                    <el-select v-model="signForm.participants" multiple placeholder="请选择参与者" style="width: 100%">
                        <el-option v-for="p in signAvailableParticipants" :key="p" :label="p" :value="p">
                        </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item>
                    <el-button type="primary" :loading="signLoading" @click="handleSignSubmit">
                        发起签名请求
                    </el-button>
                </el-form-item>
            </el-form>

        </el-card>
    </div>
</template>

<script>
import { sendWSMessage, WS_MESSAGE_TYPES } from '../services/ws'

export default {
    name: 'Sign',
    data() {
        return {
            signForm: {
                address: '',
                data: '0x0000000000000000000000000000000000000000000000000000000000000000',
                participants: []
            },
            signRules: {
                address: [
                    { required: true, message: '请输入账户地址', trigger: 'blur' }
                ],
                data: [
                    { required: true, message: '请输入待签名数据', trigger: 'blur' }
                ],
                participants: [
                    { required: true, message: '请选择参与者', trigger: 'change' },
                    { validator: this.validateSignParticipants, trigger: 'change' }
                ]
            },
            signAvailableParticipants: [],
            signLoading: false
        }
    },
    methods: {
        // 验证签名参与者选择是否满足门限要求
        validateSignParticipants(rule, value, callback) {
            if (!value || value.length === 0) {
                callback(new Error('至少需要选择1个参与者'))
            } else {
                callback()
            }
        },

        // 获取签名可用参与者（从云端服务器）
        async fetchSignAvailableParticipants() {
            if (!this.signForm.address) {
                this.$message.warning('请先输入账户地址')
                return
            }

            try {
                const response = await this.$signApi.getAvailableUsers(this.signForm.address)
                this.signAvailableParticipants = response.data.data.map(user => user.username)

                // 默认选择前两个参与者；实际门限由离线密钥元数据在服务端校验
                if (this.signAvailableParticipants.length > 0) {
                    this.signForm.participants = this.signAvailableParticipants.slice(0, Math.min(2, this.signAvailableParticipants.length))
                }

                this.$message.success('已获取可用参与者')
            } catch (error) {
                console.error('获取签名参与者列表失败:', error)
                this.$message.error('获取签名参与者列表失败')
            }
        },

        // 发起签名请求（通过云端服务器协调）
        handleSignSubmit() {
            this.$refs.signForm.validate(async valid => {
                if (!valid) {
                    return false
                }

                this.signLoading = true

                try {
                    // 创建签名会话（与云端服务器通信）
                    const response = await this.$signApi.createSession(this.$store.getters.currentUser.username)
                    const sessionKey = response.data.session_key

                    // 存储当前会话
                    this.$store.commit('setCurrentSession', sessionKey)

                    // 发送WebSocket消息给云端服务器
                    sendWSMessage({
                        type: WS_MESSAGE_TYPES.SIGN_REQUEST,
                        session_key: sessionKey,
                        message_hash: this.signForm.data,
                        address: this.signForm.address,
                        participants: this.signForm.participants
                    })

                    this.$message.success('签名请求已发送到云端服务器')
                    // 自动跳转到通知页面等待云端服务器响应
                    this.$router.push('/notifications')
                } catch (error) {
                    console.error('发起签名请求失败:', error)
                    this.$message.error('发起签名请求失败')
                } finally {
                    this.signLoading = false
                }
            })
        }
    }
}
</script>

<style scoped>
.sign-container {
    padding: 20px;
}
</style>
