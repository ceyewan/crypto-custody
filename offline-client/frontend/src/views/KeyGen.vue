<template>
    <div class="keygen-container">
        <el-card>
            <div slot="header">
                <span>密钥生成</span>
            </div>

            <el-form :model="keygenForm" :rules="keygenRules" ref="keygenForm" label-width="120px">
                <el-form-item label="门限值" prop="threshold">
                    <el-input-number v-model="keygenForm.threshold" :min="1" :max="keygenForm.totalParts"
                        @change="handleThresholdChange">
                    </el-input-number>
                </el-form-item>

                <el-form-item label="总分片数" prop="totalParts">
                    <el-input-number v-model="keygenForm.totalParts" :min="keygenForm.threshold" :max="10">
                    </el-input-number>
                </el-form-item>

                <el-form-item label="参与者" prop="participants">
                    <el-select v-model="keygenForm.participants" multiple placeholder="请选择参与者" style="width: 100%">
                        <el-option v-for="p in availableParticipants" :key="p" :label="p" :value="p">
                        </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item>
                    <el-button type="primary" :loading="keygenLoading" @click="handleKeyGenSubmit">
                        发起密钥生成
                    </el-button>
                </el-form-item>
            </el-form>
        </el-card>
    </div>
</template>

<script>
import { keygenApi } from '../services/api'
import { sendWSMessage, WS_MESSAGE_TYPES } from '../services/ws'

export default {
    name: 'KeyGen',
    data() {
        return {
            keygenForm: {
                threshold: 2,
                totalParts: 3,
                participants: []
            },
            keygenRules: {
                threshold: [
                    { required: true, message: '请输入门限值', trigger: 'blur' }
                ],
                totalParts: [
                    { required: true, message: '请输入总分片数', trigger: 'blur' }
                ],
                participants: [
                    { required: true, message: '请选择参与者', trigger: 'change' },
                    { validator: this.validateParticipants, trigger: 'change' }
                ]
            },
            availableParticipants: [],
            keygenLoading: false
        }
    },
    created() {
        this.fetchAvailableParticipants()
    },
    methods: {
        // 验证参与者选择是否满足门限要求
        validateParticipants(rule, value, callback) {
            if (!value || value.length < this.keygenForm.threshold) {
                callback(new Error(`至少需要选择${this.keygenForm.threshold}个参与者`))
            } else {
                callback()
            }
        },

        // 门限值变更处理
        handleThresholdChange(val) {
            if (val > this.keygenForm.totalParts) {
                this.keygenForm.totalParts = val
            }
        },

        // 获取可用参与者列表
        async fetchAvailableParticipants() {
            try {
                const response = await keygenApi.getAvailableUsers()
                this.availableParticipants = response.data.data.filter(user =>
                    user.role === 'participant'
                ).map(user => user.username)

                // 默认选择前n个参与者
                if (this.availableParticipants.length >= this.keygenForm.threshold) {
                    this.keygenForm.participants = this.availableParticipants.slice(0, this.keygenForm.totalParts)
                }
            } catch (error) {
                console.error('获取参与者列表失败:', error)

                // 检查错误类型
                let errorMsg = '获取参与者列表失败'
                if (error.response && error.response.status === 401) {
                    errorMsg = '认证已过期，请重新登录'

                    // 延迟1秒再登出，让用户有时间看到错误信息
                    setTimeout(() => {
                        this.$store.dispatch('logout')
                        this.$router.push('/login')
                    }, 1000)
                } else if (error.response) {
                    errorMsg = error.response.data?.error || error.response.data?.message || errorMsg
                }

                this.$message.error(errorMsg)
            }
        },

        // 发起密钥生成请求
        handleKeyGenSubmit() {
            this.$refs.keygenForm.validate(async valid => {
                if (!valid) {
                    return false
                }

                this.keygenLoading = true

                try {
                    // 创建密钥生成会话
                    const response = await keygenApi.createSession(this.$store.getters.currentUser.username)
                    const sessionKey = response.data.session_key

                    // 存储当前会话
                    this.$store.commit('setCurrentSession', sessionKey)

                    // 发送WebSocket消息
                    sendWSMessage({
                        type: WS_MESSAGE_TYPES.KEYGEN_REQUEST,
                        session_key: sessionKey,
                        threshold: this.keygenForm.threshold,
                        total_parts: this.keygenForm.totalParts,
                        participants: this.keygenForm.participants
                    })

                    this.$message.success('密钥生成请求已发送')
                    // 自动跳转到通知页面
                    this.$router.push('/notifications')
                } catch (error) {
                    console.error('发起密钥生成请求失败:', error)
                    this.$message.error('发起密钥生成请求失败')
                } finally {
                    this.keygenLoading = false
                }
            })
        }
    }
}
</script>

<style scoped>
.keygen-container {
    padding: 20px;
}
</style>