<template>
    <div class="register-container">
        <el-card class="register-card">
            <div slot="header" class="card-header">
                <h2>用户注册</h2>
            </div>

            <el-form :model="registerForm" :rules="rules" ref="registerForm" label-width="0px">
                <el-form-item prop="identifier">
                    <el-input
                        v-model="registerForm.identifier"
                        prefix-icon="el-icon-user"
                        placeholder="手机号 / 警号 / 身份证号，用于登录">
                    </el-input>
                </el-form-item>

                <el-form-item prop="nickname">
                    <el-input
                        v-model="registerForm.nickname"
                        prefix-icon="el-icon-postcard"
                        placeholder="昵称，可填姓名或常用称呼">
                    </el-input>
                </el-form-item>

                <el-form-item prop="password">
                    <el-input v-model="registerForm.password" prefix-icon="el-icon-lock" placeholder="密码" show-password>
                    </el-input>
                </el-form-item>

                <el-form-item prop="confirmPassword">
                    <el-input
                        v-model="registerForm.confirmPassword"
                        prefix-icon="el-icon-lock"
                        placeholder="确认密码"
                        show-password>
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
        const validateConfirmPassword = (rule, value, callback) => {
            if (value !== this.registerForm.password) {
                callback(new Error('两次输入的密码不一致'))
            } else {
                callback()
            }
        }

        return {
            registerForm: {
                identifier: '',
                nickname: '',
                password: '',
                confirmPassword: ''
            },
            rules: {
                identifier: [
                    { required: true, message: '请输入手机号、警号或身份证号', trigger: 'blur' },
                    { min: 3, max: 40, message: '登录标识长度应为 3-40 个字符', trigger: 'blur' }
                ],
                nickname: [
                    { min: 2, max: 30, message: '昵称长度应为 2-30 个字符', trigger: 'blur' }
                ],
                password: [
                    { required: true, message: '请输入密码', trigger: 'blur' },
                    { min: 6, message: '密码长度至少为 6 个字符', trigger: 'blur' }
                ],
                confirmPassword: [
                    { required: true, message: '请确认密码', trigger: 'blur' },
                    { validator: validateConfirmPassword, trigger: 'blur' }
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
                    await userApi.register({
                        identifier: this.registerForm.identifier,
                        username: this.registerForm.identifier,
                        nickname: this.registerForm.nickname,
                        password: this.registerForm.password
                    })
                    this.$message.success('注册成功，请登录')
                    this.$router.push('/login')
                } catch (error) {
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
    background-color: #f0f2f5;
}

.register-card {
    width: 420px;
    border-radius: 8px;
}

.card-header {
    text-align: center;
}

.card-header h2 {
    margin: 0;
    color: #304156;
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
