<template>
    <div class="login-container">
        <el-card class="login-card">
            <div slot="header" class="card-header">
                <h2>离线存管提控系统</h2>
                <el-button type="text" icon="el-icon-setting" @click="$router.push('/server-settings')">
                    服务器设置
                </el-button>
            </div>

            <el-alert
                type="info"
                :closable="false"
                :title="serverSummary"
                class="server-alert">
            </el-alert>

            <el-form :model="loginForm" :rules="rules" ref="loginForm" label-width="0px">
                <el-form-item prop="identifier">
                    <el-input
                        v-model="loginForm.identifier"
                        prefix-icon="el-icon-user"
                        placeholder="手机号 / 警号 / 身份证号">
                    </el-input>
                </el-form-item>

                <el-form-item prop="password">
                    <el-input v-model="loginForm.password" prefix-icon="el-icon-lock" placeholder="密码" show-password>
                    </el-input>
                </el-form-item>

                <el-form-item>
                    <el-button type="primary" :loading="loading" @click="handleLogin" style="width: 100%">
                        登录
                    </el-button>
                </el-form-item>

                <el-form-item>
                    <div class="register-link">
                        <span>没有账号?</span>
                        <a href="#" @click.prevent="$router.push('/register')">立即注册</a>
                    </div>
                </el-form-item>
            </el-form>
        </el-card>
    </div>
</template>

<script>
import { userApi } from '../services/api'

export default {
    name: 'Login',
    data() {
        return {
            loginForm: {
                identifier: localStorage.getItem('last_username') || '',
                password: ''
            },
            rules: {
                identifier: [{ required: true, message: '请输入手机号、警号或身份证号', trigger: 'blur' }],
                password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
            },
            loading: false
        }
    },
    computed: {
        serverSummary() {
            return `当前服务器：${this.$store.state.clientSettings.serverHttpUrl}`
        }
    },
    methods: {
        handleLogin() {
            this.$refs.loginForm.validate(async valid => {
                if (!valid) {
                    return false
                }

                this.loading = true
                try {
                    const response = await userApi.login({
                        identifier: this.loginForm.identifier,
                        username: this.loginForm.identifier,
                        password: this.loginForm.password
                    })

                    if (!response.data || !response.data.token) {
                        throw new Error('服务器响应异常：缺少认证令牌')
                    }

                    const token = response.data.token.startsWith('Bearer ')
                        ? response.data.token
                        : `Bearer ${response.data.token}`

                    this.$store.dispatch('login', {
                        token,
                        user: response.data.user
                    })

                    this.$store.dispatch('connectWebSocket')
                    setTimeout(() => {
                        this.$router.push('/dashboard')
                        this.$message.success('登录成功')
                    }, 300)
                } catch (error) {
                    this.$message.error(this.apiError(error, '登录失败，请检查账号和密码'))
                } finally {
                    this.loading = false
                }
            })
        },
        apiError(error, fallback) {
            return error.response?.data?.error || error.response?.data?.message || error.message || fallback
        }
    }
}
</script>

<style scoped>
.login-container {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100vh;
    background-color: #f0f2f5;
}

.login-card {
    width: 420px;
    border-radius: 8px;
}

.card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.card-header h2 {
    margin: 0;
    color: #304156;
    font-size: 20px;
}

.server-alert {
    margin-bottom: 18px;
}

.register-link {
    text-align: center;
    font-size: 14px;
    color: #606266;
}

.register-link a {
    color: #409EFF;
    text-decoration: none;
    margin-left: 5px;
}
</style>
