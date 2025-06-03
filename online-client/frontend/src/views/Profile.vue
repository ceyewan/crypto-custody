<template>
    <div class="profile-container">
        <el-card>
            <div slot="header">
                <span>个人资料</span>
            </div>

            <el-row :gutter="20">
                <el-col :span="12">
                    <el-card class="profile-card">
                        <div slot="header">
                            <span>基本信息</span>
                        </div>

                        <el-descriptions :column="1" border>
                            <el-descriptions-item label="用户ID">{{ user.id }}</el-descriptions-item>
                            <el-descriptions-item label="用户名">{{ user.username }}</el-descriptions-item>
                            <el-descriptions-item label="邮箱">{{ user.email }}</el-descriptions-item>
                            <el-descriptions-item label="角色">
                                <el-tag :type="getRoleTagType(user.role)">{{ getRoleText(user.role) }}</el-tag>
                            </el-descriptions-item>
                        </el-descriptions>
                    </el-card>
                </el-col>

                <el-col :span="12">
                    <el-card class="password-card">
                        <div slot="header">
                            <span>修改密码</span>
                        </div>

                        <el-form :model="passwordForm" :rules="passwordRules" ref="passwordForm" label-width="100px">
                            <el-form-item label="当前密码" prop="oldPassword">
                                <el-input v-model="passwordForm.oldPassword" type="password" show-password></el-input>
                            </el-form-item>
                            <el-form-item label="新密码" prop="newPassword">
                                <el-input v-model="passwordForm.newPassword" type="password" show-password></el-input>
                            </el-form-item>
                            <el-form-item label="确认密码" prop="confirmPassword">
                                <el-input v-model="passwordForm.confirmPassword" type="password" show-password></el-input>
                            </el-form-item>
                            <el-form-item>
                                <el-button type="primary" @click="handleChangePassword" :loading="passwordLoading">
                                    修改密码
                                </el-button>
                                <el-button @click="resetPasswordForm">重置</el-button>
                            </el-form-item>
                        </el-form>
                    </el-card>
                </el-col>
            </el-row>

            <!-- 权限说明 -->
            <el-card class="permission-card" style="margin-top: 20px;">
                <div slot="header">
                    <span>权限说明</span>
                </div>

                <div class="permission-content">
                    <div v-if="isAdmin" class="permission-item">
                        <i class="el-icon-user-solid permission-icon admin"></i>
                        <div class="permission-text">
                            <h4>管理员权限</h4>
                            <p>您具有系统的最高权限，可以管理所有用户、账户和交易</p>
                            <ul>
                                <li>管理系统用户（创建、编辑、删除）</li>
                                <li>查看所有账户信息</li>
                                <li>管理所有交易记录</li>
                                <li>系统配置和维护</li>
                            </ul>
                        </div>
                    </div>

                    <div v-else-if="isOfficer" class="permission-item">
                        <i class="el-icon-s-custom permission-icon officer"></i>
                        <div class="permission-text">
                            <h4>警员权限</h4>
                            <p>您具有执法相关的权限，可以管理账户和处理交易</p>
                            <ul>
                                <li>创建和管理自己导入的账户</li>
                                <li>查询账户余额和交易记录</li>
                                <li>发起和处理交易</li>
                                <li>查看个人相关的操作记录</li>
                            </ul>
                        </div>
                    </div>

                    <div v-else class="permission-item">
                        <i class="el-icon-view permission-icon guest"></i>
                        <div class="permission-text">
                            <h4>访客权限</h4>
                            <p>您当前是访客身份，权限受限</p>
                            <ul>
                                <li>查看个人资料</li>
                                <li>修改个人密码</li>
                                <li>浏览公开信息</li>
                            </ul>
                            <p class="upgrade-hint">如需更多权限，请联系管理员</p>
                        </div>
                    </div>
                </div>
            </el-card>

            <!-- 操作统计 -->
            <el-card class="stats-card" style="margin-top: 20px;" v-if="isOfficer">
                <div slot="header">
                    <span>操作统计</span>
                </div>

                <el-row :gutter="20">
                    <el-col :span="8">
                        <div class="stat-item">
                            <div class="stat-number">{{ accountCount }}</div>
                            <div class="stat-label">管理账户数</div>
                        </div>
                    </el-col>
                    <el-col :span="8">
                        <div class="stat-item">
                            <div class="stat-number">{{ transactionCount }}</div>
                            <div class="stat-label">处理交易数</div>
                        </div>
                    </el-col>
                    <el-col :span="8">
                        <div class="stat-item">
                            <div class="stat-number">{{ Math.floor(Math.random() * 30) + 1 }}</div>
                            <div class="stat-label">在线天数</div>
                        </div>
                    </el-col>
                </el-row>
            </el-card>
        </el-card>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { userApi, accountApi } from '../services/api'

export default {
  name: 'Profile',
  data () {
    // 确认密码验证
    const validateConfirmPassword = (rule, value, callback) => {
      if (value !== this.passwordForm.newPassword) {
        callback(new Error('两次输入的密码不一致'))
      } else {
        callback()
      }
    }

    return {
      passwordForm: {
        oldPassword: '',
        newPassword: '',
        confirmPassword: ''
      },
      passwordRules: {
        oldPassword: [
          { required: true, message: '请输入当前密码', trigger: 'blur' }
        ],
        newPassword: [
          { required: true, message: '请输入新密码', trigger: 'blur' },
          { min: 6, message: '新密码长度至少为6个字符', trigger: 'blur' }
        ],
        confirmPassword: [
          { required: true, message: '请确认新密码', trigger: 'blur' },
          { validator: validateConfirmPassword, trigger: 'blur' }
        ]
      },
      passwordLoading: false,
      accountCount: 0,
      transactionCount: 0
    }
  },
  computed: {
    ...mapGetters([
      'currentUser',
      'isAdmin',
      'isOfficer',
      'isGuest'
    ]),
    user () {
      return this.currentUser || {}
    }
  },
  created () {
    this.loadUserStats()
  },
  methods: {
    async loadUserStats () {
      if (this.isOfficer) {
        try {
          // 获取账户数量
          const accountResponse = await accountApi.getUserAccounts()
          if (accountResponse.data.code === 200) {
            this.accountCount = accountResponse.data.data.length
          }

          // 模拟交易数量
          this.transactionCount = Math.floor(Math.random() * 50)
        } catch (error) {
          console.error('加载用户统计失败:', error)
        }
      }
    },

    async handleChangePassword () {
      this.$refs.passwordForm.validate(async valid => {
        if (!valid) {
          return false
        }

        this.passwordLoading = true

        try {
          const response = await userApi.changePassword({
            oldPassword: this.passwordForm.oldPassword,
            newPassword: this.passwordForm.newPassword
          })

          if (response.data.code === 200) {
            this.$message.success('密码修改成功')
            this.resetPasswordForm()
          } else {
            throw new Error(response.data.message || '密码修改失败')
          }
        } catch (error) {
          console.error('修改密码失败:', error)
          let errorMsg = '密码修改失败'
          if (error.response && error.response.data) {
            errorMsg = error.response.data.message || errorMsg
          } else if (error.message) {
            errorMsg = error.message
          }
          this.$message.error(errorMsg)
        } finally {
          this.passwordLoading = false
        }
      })
    },

    resetPasswordForm () {
      this.passwordForm = {
        oldPassword: '',
        newPassword: '',
        confirmPassword: ''
      }
      this.$refs.passwordForm && this.$refs.passwordForm.resetFields()
    },

    getRoleText (role) {
      const roleMap = {
        admin: '管理员',
        officer: '警员',
        guest: '访客'
      }
      return roleMap[role] || role
    },

    getRoleTagType (role) {
      const typeMap = {
        admin: 'danger',
        officer: 'warning',
        guest: 'info'
      }
      return typeMap[role] || 'info'
    }
  }
}
</script>

<style scoped>
.profile-container {
    padding: 20px;
}

.profile-card,
.password-card,
.permission-card,
.stats-card {
    margin-bottom: 20px;
}

.permission-content {
    padding: 10px 0;
}

.permission-item {
    display: flex;
    align-items: flex-start;
    gap: 15px;
}

.permission-icon {
    font-size: 48px;
    margin-top: 5px;
}

.permission-icon.admin {
    color: #F56C6C;
}

.permission-icon.officer {
    color: #E6A23C;
}

.permission-icon.guest {
    color: #909399;
}

.permission-text h4 {
    margin: 0 0 10px 0;
    color: #303133;
}

.permission-text p {
    margin: 0 0 10px 0;
    color: #606266;
    line-height: 1.5;
}

.permission-text ul {
    margin: 0;
    padding-left: 20px;
    color: #606266;
}

.permission-text li {
    margin-bottom: 5px;
}

.upgrade-hint {
    color: #F56C6C !important;
    font-weight: bold;
    margin-top: 10px !important;
}

.stat-item {
    text-align: center;
    padding: 20px;
    background-color: #f8f9fa;
    border-radius: 8px;
}

.stat-number {
    font-size: 32px;
    font-weight: bold;
    color: #409EFF;
    line-height: 1;
}

.stat-label {
    font-size: 14px;
    color: #606266;
    margin-top: 8px;
}
</style>
