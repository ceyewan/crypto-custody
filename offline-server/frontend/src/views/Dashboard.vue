<template>
    <div class="dashboard-container">
        <el-container>
            <el-header class="header">
                <div class="logo">MPC密钥管理系统</div>
                <div class="user-info">
                    <span>欢迎, {{ user.username }} ({{ roleText }})</span>
                    <el-button type="text" @click="handleLogout">退出登录</el-button>
                </div>
            </el-header>

            <el-container>
                <el-aside width="250px">
                    <el-menu :default-active="activeMenu" router class="menu" background-color="#304156"
                        text-color="#bfcbd9" active-text-color="#409EFF">
                        <el-menu-item index="keygen" @click="activeMenu = 'keygen'">
                            <i class="el-icon-key"></i>
                            <span>密钥生成</span>
                        </el-menu-item>
                        <el-menu-item index="sign" @click="activeMenu = 'sign'">
                            <i class="el-icon-edit"></i>
                            <span>交易签名</span>
                        </el-menu-item>
                        <el-menu-item index="notifications" @click="activeMenu = 'notifications'">
                            <i class="el-icon-bell"></i>
                            <span>通知消息</span>
                            <el-badge v-if="notifications.length > 0" :value="notifications.length" class="item">
                            </el-badge>
                        </el-menu-item>
                    </el-menu>
                </el-aside>

                <el-main>
                    <!-- 密钥生成面板 -->
                    <div v-if="activeMenu === 'keygen'">
                        <el-card>
                            <div slot="header">
                                <span>密钥生成</span>
                            </div>

                            <div v-if="isCoordinator || isAdmin">
                                <el-form :model="keygenForm" :rules="keygenRules" ref="keygenForm" label-width="120px">
                                    <el-form-item label="门限值" prop="threshold">
                                        <el-input-number v-model="keygenForm.threshold" :min="2"
                                            :max="keygenForm.totalParts" @change="handleThresholdChange">
                                        </el-input-number>
                                    </el-form-item>

                                    <el-form-item label="总分片数" prop="totalParts">
                                        <el-input-number v-model="keygenForm.totalParts" :min="keygenForm.threshold"
                                            :max="10">
                                        </el-input-number>
                                    </el-form-item>

                                    <el-form-item label="参与者" prop="participants">
                                        <el-select v-model="keygenForm.participants" multiple placeholder="请选择参与者"
                                            style="width: 100%">
                                            <el-option v-for="p in availableParticipants" :key="p" :label="p"
                                                :value="p">
                                            </el-option>
                                        </el-select>
                                    </el-form-item>

                                    <el-form-item>
                                        <el-button type="primary" :loading="keygenLoading" @click="handleKeyGenSubmit">
                                            发起密钥生成
                                        </el-button>
                                    </el-form-item>
                                </el-form>
                            </div>
                            <div v-else>
                                <el-alert title="您的角色没有权限发起密钥生成请求" type="info" show-icon>
                                </el-alert>
                            </div>
                        </el-card>
                    </div>

                    <!-- 签名面板 -->
                    <div v-if="activeMenu === 'sign'">
                        <el-card>
                            <div slot="header">
                                <span>交易签名</span>
                            </div>

                            <div v-if="isCoordinator || isAdmin">
                                <el-form :model="signForm" :rules="signRules" ref="signForm" label-width="120px">
                                    <el-form-item label="账户地址" prop="address">
                                        <el-input v-model="signForm.address"></el-input>
                                    </el-form-item>

                                    <el-form-item label="待签名数据" prop="data">
                                        <el-input v-model="signForm.data" type="textarea" :rows="2"></el-input>
                                    </el-form-item>

                                    <el-form-item label="门限值" prop="threshold">
                                        <el-input-number v-model="signForm.threshold" :min="2"
                                            :max="signForm.totalParts" @change="handleSignThresholdChange">
                                        </el-input-number>
                                    </el-form-item>

                                    <el-form-item label="总分片数" prop="totalParts">
                                        <el-input-number v-model="signForm.totalParts" :min="signForm.threshold"
                                            :max="10">
                                        </el-input-number>
                                    </el-form-item>

                                    <el-form-item label="参与者" prop="participants">
                                        <el-select v-model="signForm.participants" multiple placeholder="请选择参与者"
                                            style="width: 100%">
                                            <el-option v-for="p in signAvailableParticipants" :key="p" :label="p"
                                                :value="p">
                                            </el-option>
                                        </el-select>
                                    </el-form-item>

                                    <el-form-item>
                                        <el-button type="primary" :loading="signLoading" @click="handleSignSubmit">
                                            发起签名请求
                                        </el-button>
                                    </el-form-item>
                                </el-form>
                            </div>
                            <div v-else>
                                <el-alert title="您的角色没有权限发起签名请求" type="info" show-icon>
                                </el-alert>
                            </div>
                        </el-card>
                    </div>

                    <!-- 通知消息面板 -->
                    <div v-if="activeMenu === 'notifications'">
                        <el-card>
                            <div slot="header" class="clearfix">
                                <span>通知消息</span>
                                <el-button style="float: right; padding: 3px 0" type="text" @click="clearNotifications">
                                    清空通知
                                </el-button>
                            </div>

                            <el-table :data="notifications" style="width: 100%">
                                <el-table-column prop="type" label="消息类型" width="180"></el-table-column>
                                <el-table-column prop="timestamp" label="时间" width="180">
                                    <template slot-scope="scope">
                                        {{ new Date(scope.row.timestamp).toLocaleString() }}
                                    </template>
                                </el-table-column>
                                <el-table-column label="内容">
                                    <template slot-scope="scope">
                                        <el-button type="text" @click="showMessageDetail(scope.row)">
                                            查看详情
                                        </el-button>
                                    </template>
                                </el-table-column>
                            </el-table>

                            <div class="no-data" v-if="notifications.length === 0">
                                暂无通知消息
                            </div>
                        </el-card>
                    </div>
                </el-main>
            </el-container>
        </el-container>
    </div>
</template>

<script>
import { keygenApi, signApi } from '../services/api'
import { sendWSMessage, WS_MESSAGE_TYPES, initWebSocketService } from '../services/ws'
import { mapGetters } from 'vuex'

export default {
    name: 'Dashboard',
    data() {
        return {
            activeMenu: 'keygen',
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
            availableParticipants: [],
            signAvailableParticipants: [],
            keygenLoading: false,
            signLoading: false
        }
    },
    computed: {
        ...mapGetters([
            'currentUser',
            'isAdmin',
            'isCoordinator',
            'isParticipant',
            'notifications',
            'wsConnected'
        ]),
        user() {
            return this.currentUser || {}
        },
        roleText() {
            if (this.isAdmin) return '管理员'
            if (this.isCoordinator) return '协调者'
            if (this.isParticipant) return '参与者'
            return '访客'
        }
    },
    created() {
        // 初始化WebSocket
        if (!this.wsConnected) {
            this.$store.dispatch('connectWebSocket')
        }

        // 初始化WebSocket服务
        setTimeout(() => {
            initWebSocketService()
        }, 1000)

        // 获取可用参与者列表
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

        // 验证签名参与者选择是否满足门限要求
        validateSignParticipants(rule, value, callback) {
            if (!value || value.length < this.signForm.threshold) {
                callback(new Error(`至少需要选择${this.signForm.threshold}个参与者`))
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

        // 签名门限值变更处理
        handleSignThresholdChange(val) {
            if (val > this.signForm.totalParts) {
                this.signForm.totalParts = val
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
                this.$message.error('获取参与者列表失败')
            }
        },

        // 获取签名可用参与者
        async fetchSignAvailableParticipants() {
            if (!this.signForm.address) {
                this.$message.warning('请先输入账户地址')
                return
            }

            try {
                const response = await signApi.getAvailableUsers(this.signForm.address)
                this.signAvailableParticipants = response.data.data.map(user => user.username)

                // 默认选择前n个参与者
                if (this.signAvailableParticipants.length >= this.signForm.threshold) {
                    this.signForm.participants = this.signAvailableParticipants.slice(0, this.signForm.threshold)
                }
            } catch (error) {
                console.error('获取签名参与者列表失败:', error)
                this.$message.error('获取签名参与者列表失败')
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
                    const response = await keygenApi.createSession(this.user.username)
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
                } catch (error) {
                    console.error('发起密钥生成请求失败:', error)
                    this.$message.error('发起密钥生成请求失败')
                } finally {
                    this.keygenLoading = false
                }
            })
        },

        // 发起签名请求
        handleSignSubmit() {
            this.$refs.signForm.validate(async valid => {
                if (!valid) {
                    return false
                }

                this.signLoading = true

                try {
                    // 创建签名会话
                    const response = await signApi.createSession(this.user.username)
                    const sessionKey = response.data.session_key

                    // 存储当前会话
                    this.$store.commit('setCurrentSession', sessionKey)

                    // 发送WebSocket消息
                    sendWSMessage({
                        type: WS_MESSAGE_TYPES.SIGN_REQUEST,
                        session_key: sessionKey,
                        threshold: this.signForm.threshold,
                        total_parts: this.signForm.totalParts,
                        data: this.signForm.data,
                        address: this.signForm.address,
                        participants: this.signForm.participants
                    })

                    this.$message.success('签名请求已发送')
                } catch (error) {
                    console.error('发起签名请求失败:', error)
                    this.$message.error('发起签名请求失败')
                } finally {
                    this.signLoading = false
                }
            })
        },

        // 退出登录
        handleLogout() {
            this.$store.dispatch('logout')
            this.$router.push('/login')
        },

        // 清空通知
        clearNotifications() {
            this.$store.commit('clearNotifications')
        },

        // 显示消息详情
        showMessageDetail(message) {
            this.$alert(JSON.stringify(message.content, null, 2), '消息详情', {
                closeOnClickModal: true
            })
        }
    },
    watch: {
        activeMenu(val) {
            if (val === 'sign' && this.signAvailableParticipants.length === 0) {
                this.fetchSignAvailableParticipants()
            }
        }
    }
}
</script>

<style scoped>
.dashboard-container {
    height: 100vh;
    display: flex;
    flex-direction: column;
}

.header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background-color: #304156;
    color: white;
    padding: 0 20px;
    height: 60px;
}

.logo {
    font-size: 20px;
    font-weight: bold;
}

.user-info {
    display: flex;
    align-items: center;
}

.user-info span {
    margin-right: 15px;
}

.menu {
    height: calc(100vh - 60px);
}

.el-menu-item {
    font-size: 14px;
}

.el-aside {
    background-color: #304156;
}

.el-main {
    padding: 20px;
    background-color: #f0f2f5;
}

.no-data {
    text-align: center;
    color: #909399;
    padding: 50px 0;
}
</style>