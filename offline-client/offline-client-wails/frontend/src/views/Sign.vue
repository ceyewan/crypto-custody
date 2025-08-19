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

                <el-form-item label="门限值" prop="threshold">
                    <el-input-number v-model="signForm.threshold" :min="2" :max="signForm.totalParts"
                        @change="handleSignThresholdChange">
                    </el-input-number>
                </el-form-item>

                <el-form-item label="总分片数" prop="totalParts">
                    <el-input-number v-model="signForm.totalParts" :min="signForm.threshold" :max="10">
                    </el-input-number>
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

            <!-- 本地测试区域 -->
            <el-divider content-text="本地测试（直接调用内置MPC）"></el-divider>
            <el-form label-width="120px">
                <el-form-item>
                    <el-button type="success" :loading="localSignLoading" @click="handleLocalSignTest">
                        本地签名测试
                    </el-button>
                </el-form-item>
            </el-form>

            <!-- 结果显示区域 -->
            <el-card v-if="signResult" style="margin-top: 20px;">
                <div slot="header">
                    <span>签名结果</span>
                </div>
                <el-row>
                    <el-col :span="24">
                        <p><strong>状态:</strong> 
                            <el-tag :type="signResult.success ? 'success' : 'danger'">
                                {{ signResult.success ? '成功' : '失败' }}
                            </el-tag>
                        </p>
                        <p v-if="signResult.success && signResult.data">
                            <strong>签名结果:</strong>
                            <el-input 
                                v-model="signResult.data.signature" 
                                type="textarea" 
                                :rows="3" 
                                readonly
                                style="margin-top: 10px;">
                            </el-input>
                        </p>
                        <p v-if="!signResult.success">
                            <strong>错误信息:</strong> {{ signResult.error }}
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
    name: 'Sign',
    data() {
        return {
            signForm: {
                address: '',
                data: '0x1234abcd5678efgh9012ijkl3456mnop7890qrst', // 测试数据
                threshold: 2,
                totalParts: 3,
                participants: []
            },
            signRules: {
                address: [
                    { required: true, message: '请输入账户地址', trigger: 'blur' }
                ],
                data: [
                    { required: true, message: '请输入待签名数据', trigger: 'blur' }
                ],
                threshold: [
                    { required: true, message: '请输入门限值', trigger: 'blur' }
                ],
                totalParts: [
                    { required: true, message: '请输入总分片数', trigger: 'blur' }
                ],
                participants: [
                    { required: true, message: '请选择参与者', trigger: 'change' },
                    { validator: this.validateSignParticipants, trigger: 'change' }
                ]
            },
            signAvailableParticipants: [],
            signLoading: false,
            localSignLoading: false,
            signResult: null
        }
    },
    methods: {
        // 验证签名参与者选择是否满足门限要求
        validateSignParticipants(rule, value, callback) {
            if (!value || value.length < this.signForm.threshold) {
                callback(new Error(`至少需要选择${this.signForm.threshold}个参与者`))
            } else {
                callback()
            }
        },

        // 签名门限值变更处理
        handleSignThresholdChange(val) {
            if (val > this.signForm.totalParts) {
                this.signForm.totalParts = val
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

                // 默认选择前n个参与者
                if (this.signAvailableParticipants.length >= this.signForm.threshold) {
                    this.signForm.participants = this.signAvailableParticipants.slice(0, this.signForm.threshold)
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
                        threshold: this.signForm.threshold,
                        total_parts: this.signForm.totalParts,
                        data: this.signForm.data,
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
        },

        // 本地签名测试（直接调用内置MPC模块）
        async handleLocalSignTest() {
            this.localSignLoading = true
            this.signResult = null

            try {
                console.log('开始本地签名测试...')
                // 直接调用 Wails 内置的 MPC 模块
                const result = await this.$localMpcApi.sign({
                    address: this.signForm.address,
                    message: this.signForm.data,
                    data: this.signForm.data,
                    threshold: this.signForm.threshold,
                    total_parts: this.signForm.totalParts,
                    participants: this.signForm.participants
                })

                console.log('本地签名结果:', result)
                this.signResult = result.data

                if (result.data.success) {
                    this.$message.success('本地签名成功！')
                } else {
                    this.$message.error('本地签名失败: ' + (result.data.error || '未知错误'))
                }
            } catch (error) {
                console.error('本地签名失败:', error)
                this.signResult = {
                    success: false,
                    error: error.message || '本地签名过程中发生错误'
                }
                this.$message.error('本地签名失败: ' + (error.message || '未知错误'))
            } finally {
                this.localSignLoading = false
            }
        }
    }
}
</script>

<style scoped>
.sign-container {
    padding: 20px;
}
</style>