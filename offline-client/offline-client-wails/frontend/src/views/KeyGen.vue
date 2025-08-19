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

            <!-- 本地测试区域 -->
            <el-divider content-text="本地测试（直接调用内置MPC）"></el-divider>
            <el-form label-width="120px">
                <el-form-item>
                    <el-button type="success" :loading="localKeygenLoading" @click="handleLocalKeyGenTest">
                        本地密钥生成测试
                    </el-button>
                </el-form-item>
            </el-form>

            <!-- 结果显示区域 -->
            <el-card v-if="keygenResult" style="margin-top: 20px;">
                <div slot="header">
                    <span>密钥生成结果</span>
                </div>
                <el-row>
                    <el-col :span="24">
                        <p><strong>状态:</strong> 
                            <el-tag :type="keygenResult.success ? 'success' : 'danger'">
                                {{ keygenResult.success ? '成功' : '失败' }}
                            </el-tag>
                        </p>
                        <p v-if="keygenResult.success && keygenResult.data">
                            <strong>以太坊地址:</strong> {{ keygenResult.data.address }}
                        </p>
                        <p v-if="!keygenResult.success">
                            <strong>错误信息:</strong> {{ keygenResult.error }}
                        </p>
                    </el-col>
                </el-row>
            </el-card>
        </el-card>
    </div>
</template>

<script>
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
            keygenLoading: false,
            localKeygenLoading: false,
            keygenResult: null
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

        // 获取可用参与者列表（从云端服务器）
        async fetchAvailableParticipants() {
            try {
                const response = await this.$keygenApi.getAvailableUsers()
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

        // 发起密钥生成请求（通过云端服务器协调）
        handleKeyGenSubmit() {
            this.$refs.keygenForm.validate(async valid => {
                if (!valid) {
                    return false
                }

                this.keygenLoading = true

                try {
                    // 创建密钥生成会话（与云端服务器通信）
                    const response = await this.$keygenApi.createSession(this.$store.getters.currentUser.username)
                    const sessionKey = response.data.session_key

                    // 存储当前会话
                    this.$store.commit('setCurrentSession', sessionKey)

                    // 发送WebSocket消息给云端服务器
                    sendWSMessage({
                        type: WS_MESSAGE_TYPES.KEYGEN_REQUEST,
                        session_key: sessionKey,
                        threshold: this.keygenForm.threshold,
                        total_parts: this.keygenForm.totalParts,
                        participants: this.keygenForm.participants
                    })

                    this.$message.success('密钥生成请求已发送到云端服务器')
                    // 自动跳转到通知页面等待云端服务器响应
                    this.$router.push('/notifications')
                } catch (error) {
                    console.error('发起密钥生成请求失败:', error)
                    this.$message.error('发起密钥生成请求失败')
                } finally {
                    this.keygenLoading = false
                }
            })
        },

        // 本地密钥生成测试（直接调用内置MPC模块）
        async handleLocalKeyGenTest() {
            this.localKeygenLoading = true
            this.keygenResult = null

            try {
                console.log('开始本地密钥生成测试...')
                // 直接调用 Wails 内置的 MPC 模块
                const result = await this.$localMpcApi.keyGen({
                    threshold: this.keygenForm.threshold,
                    total_parts: this.keygenForm.totalParts,
                    participants: this.keygenForm.participants
                })

                console.log('本地密钥生成结果:', result)
                this.keygenResult = result.data

                if (result.data.success) {
                    this.$message.success('本地密钥生成成功！')
                } else {
                    this.$message.error('本地密钥生成失败: ' + (result.data.error || '未知错误'))
                }
            } catch (error) {
                console.error('本地密钥生成失败:', error)
                this.keygenResult = {
                    success: false,
                    error: error.message || '本地密钥生成过程中发生错误'
                }
                this.$message.error('本地密钥生成失败: ' + (error.message || '未知错误'))
            } finally {
                this.localKeygenLoading = false
            }
        }
    }
}
</script>

<style scoped>
.keygen-container {
    padding: 20px;
}
</style>