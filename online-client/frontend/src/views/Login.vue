<template>
    <div class="login-container">
        <el-card class="login-card">
            <div slot="header" class="card-header">
                <h2>在线存管提控系统</h2>
            </div>

            <el-form :model="loginForm" :rules="rules" ref="loginForm" label-width="0px">
                <el-form-item prop="identifier">
                    <el-input v-model="loginForm.identifier" prefix-icon="el-icon-user" placeholder="手机号 / 警号 / 身份证号">
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
  data () {
    return {
      loginForm: {
        identifier: localStorage.getItem('last_username') || '',
        password: ''
      },
      rules: {
        identifier: [{ required: true, message: '请输入登录标识', trigger: 'blur' }],
        password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
      },
      loading: false
    }
  },
  methods: {
    handleLogin () {
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

          if (!response.data || response.data.code !== 200) {
            throw new Error(response.data?.message || '登录失败')
          }

          const userData = response.data.data

          this.$store.dispatch('login', {
            token: userData.token,
            user: userData.user
          })

          this.$router.push('/dashboard')
          this.$message.success('登录成功')
        } catch (error) {
          let errorMsg = '登录失败，请检查登录标识和密码'

          if (error.response && error.response.data) {
            errorMsg = error.response.data.message || errorMsg
          } else if (error.message) {
            errorMsg = error.message
          }

          this.$message.error(errorMsg)
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
    background-color: #f0f2f5;
}

.login-card {
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
