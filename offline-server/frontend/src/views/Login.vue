<template>
    <div class="login-container">
        <el-card class="login-card">
            <div slot="header" class="card-header">
                <h2>多方门限签名系统</h2>
            </div>

            <el-form :model="loginForm" :rules="rules" ref="loginForm" label-width="0px">
                <el-form-item prop="username">
                    <el-input v-model="loginForm.username" prefix-icon="el-icon-user" placeholder="用户名">
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
                        <router-link to="/register">立即注册</router-link>
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
                username: '',
                password: ''
            },
            rules: {
                username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
                password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
            },
            loading: false
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
                        username: this.loginForm.username,
                        password: this.loginForm.password
                    })

                    // 保存用户信息和令牌
                    this.$store.dispatch('login', response.data)

                    // 连接WebSocket
                    this.$store.dispatch('connectWebSocket')

                    // 跳转到仪表板
                    this.$router.push('/dashboard')

                    this.$message.success('登录成功')
                } catch (error) {
                    console.error('登录失败:', error)
                    this.$message.error(error.response?.data?.error || '登录失败，请检查用户名和密码')
                } finally {
                    this.loading = false
                }
            })
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
    background-color: #f5f7fa;
}

.login-card {
    width: 400px;
    border-radius: 8px;
}

.card-header {
    text-align: center;
}

.card-header h2 {
    margin: 0;
    color: #409EFF;
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