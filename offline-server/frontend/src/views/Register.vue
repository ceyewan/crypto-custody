<template>
    <div class="register-container">
        <el-card class="register-card">
            <div slot="header" class="card-header">
                <h2>用户注册</h2>
            </div>

            <el-form :model="registerForm" :rules="rules" ref="registerForm" label-width="0px">
                <el-form-item prop="username">
                    <el-input v-model="registerForm.username" prefix-icon="el-icon-user" placeholder="用户名">
                    </el-input>
                </el-form-item>

                <el-form-item prop="password">
                    <el-input v-model="registerForm.password" prefix-icon="el-icon-lock" placeholder="密码" show-password>
                    </el-input>
                </el-form-item>

                <el-form-item prop="confirmPassword">
                    <el-input v-model="registerForm.confirmPassword" prefix-icon="el-icon-lock" placeholder="确认密码"
                        show-password>
                    </el-input>
                </el-form-item>

                <el-form-item prop="email">
                    <el-input v-model="registerForm.email" prefix-icon="el-icon-message" placeholder="邮箱">
                    </el-input>
                </el-form-item>

                <el-form-item>
                    <el-button type="primary" :loading="loading" @click="handleRegister" style="width: 100%">
                        注册
                    </el-button>
                </el-form-item>

                <el-form-item>
                    <div class="login-link">
                        <span>已有账号?</span>
                        <router-link to="/login">立即登录</router-link>
                    </div>
                </el-form-item>
            </el-form>
        </el-card>
    </div>
</template>

<script>
import { userApi } from '../services/api'

export default {
    name: 'Register',
    data() {
        // 密码一致性校验
        const validateConfirmPassword = (rule, value, callback) => {
            if (value !== this.registerForm.password) {
                callback(new Error('两次输入的密码不一致'))
            } else {
                callback()
            }
        }

        return {
            registerForm: {
                username: '',
                password: '',
                confirmPassword: '',
                email: ''
            },
            rules: {
                username: [
                    { required: true, message: '请输入用户名', trigger: 'blur' },
                    { min: 3, max: 20, message: '用户名长度应为3-20个字符', trigger: 'blur' }
                ],
                password: [
                    { required: true, message: '请输入密码', trigger: 'blur' },
                    { min: 6, message: '密码长度至少为6个字符', trigger: 'blur' }
                ],
                confirmPassword: [
                    { required: true, message: '请确认密码', trigger: 'blur' },
                    { validator: validateConfirmPassword, trigger: 'blur' }
                ],
                email: [
                    { required: true, message: '请输入邮箱', trigger: 'blur' },
                    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
                ]
            },
            loading: false
        }
    },
    methods: {
        handleRegister() {
            this.$refs.registerForm.validate(async valid => {
                if (!valid) {
                    return false
                }

                this.loading = true

                try {
                    const response = await userApi.register({
                        username: this.registerForm.username,
                        password: this.registerForm.password,
                        email: this.registerForm.email
                    })

                    // 注册成功
                    this.$message.success('注册成功，请登录')

                    // 跳转到登录页面
                    this.$router.push('/login')
                } catch (error) {
                    console.error('注册失败:', error)
                    this.$message.error(error.response?.data?.error || '注册失败，请稍后重试')
                } finally {
                    this.loading = false
                }
            })
        }
    }
}
</script>

<style scoped>
.register-container {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100vh;
    background-color: #f5f7fa;
}

.register-card {
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

.login-link {
    text-align: center;
    font-size: 14px;
    color: #606266;
}

.login-link a {
    color: #409EFF;
    text-decoration: none;
    margin-left: 5px;
}
</style>