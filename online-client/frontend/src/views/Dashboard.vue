<template>
    <div class="dashboard-container">
        <el-container>
            <el-header class="header">
                <div class="logo">在线加密货币托管系统</div>
                <div class="user-info">
                    <span>欢迎, {{ user.username }} ({{ roleText }})</span>
                    <el-dropdown @command="handleCommand">
                        <span class="el-dropdown-link">
                            <i class="el-icon-setting"></i>
                        </span>
                        <el-dropdown-menu slot="dropdown">
                            <el-dropdown-item command="profile">个人资料</el-dropdown-item>
                            <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
                        </el-dropdown-menu>
                    </el-dropdown>
                </div>
            </el-header>

            <el-container>
                <el-aside width="250px">
                    <el-menu :default-active="activeMenu" router class="menu" background-color="#304156"
                        text-color="#bfcbd9" active-text-color="#409EFF">
                        <el-menu-item v-if="isAdmin" index="/users">
                            <i class="el-icon-user"></i>
                            <span>用户管理</span>
                        </el-menu-item>
                        <el-menu-item v-if="isOfficer" index="/accounts">
                            <i class="el-icon-wallet"></i>
                            <span>账户管理</span>
                        </el-menu-item>
                        <el-menu-item v-if="isOfficer" index="/transactions">
                            <i class="el-icon-s-finance"></i>
                            <span>交易管理</span>
                        </el-menu-item>
                        <el-menu-item index="/profile">
                            <i class="el-icon-user-solid"></i>
                            <span>个人资料</span>
                        </el-menu-item>
                    </el-menu>
                </el-aside>

                <el-main>
                    <router-view></router-view>

                    <div v-if="$route.path === '/dashboard'" class="welcome-container">
                        <div class="welcome-message">
                            <i class="el-icon-s-home welcome-icon"></i>
                            <h2>欢迎使用在线加密货币托管系统</h2>
                            <p>请根据您的角色和需求，使用左侧菜单选择相应功能</p>

                            <!-- 系统统计信息 -->
                            <div class="stats-container">
                                <el-row :gutter="20">
                                    <el-col :span="8" v-if="isOfficer">
                                        <el-card class="stat-card">
                                            <div class="stat-content">
                                                <div class="stat-number">{{ accountCount }}</div>
                                                <div class="stat-label">管理账户数</div>
                                            </div>
                                            <i class="el-icon-wallet stat-icon"></i>
                                        </el-card>
                                    </el-col>
                                    <el-col :span="8" v-if="isOfficer">
                                        <el-card class="stat-card">
                                            <div class="stat-content">
                                                <div class="stat-number">{{ transactionCount }}</div>
                                                <div class="stat-label">交易次数</div>
                                            </div>
                                            <i class="el-icon-s-finance stat-icon"></i>
                                        </el-card>
                                    </el-col>
                                    <el-col :span="8" v-if="isAdmin">
                                        <el-card class="stat-card">
                                            <div class="stat-content">
                                                <div class="stat-number">{{ userCount }}</div>
                                                <div class="stat-label">系统用户数</div>
                                            </div>
                                            <i class="el-icon-user stat-icon"></i>
                                        </el-card>
                                    </el-col>
                                </el-row>
                            </div>

                            <div v-if="isAdmin" class="feature-box">
                                <h3>管理员功能</h3>
                                <el-button type="primary" @click="$router.push('/users')">用户管理</el-button>
                                <el-button type="primary" @click="$router.push('/accounts')">账户管理</el-button>
                                <el-button type="primary" @click="$router.push('/transactions')">交易管理</el-button>
                            </div>

                            <div v-else-if="isOfficer" class="feature-box">
                                <h3>警员功能</h3>
                                <el-button type="primary" @click="$router.push('/accounts')">账户管理</el-button>
                                <el-button type="primary" @click="$router.push('/transactions')">交易管理</el-button>
                            </div>

                            <div v-else class="feature-box">
                                <h3>访客功能</h3>
                                <p>您当前是访客身份，权限有限</p>
                                <el-button type="primary" @click="$router.push('/profile')">查看个人资料</el-button>
                            </div>
                        </div>
                    </div>
                </el-main>
            </el-container>
        </el-container>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { userApi, accountApi } from '../services/api'

export default {
  name: 'Dashboard',
  data () {
    return {
      activeMenu: this.$route.path,
      accountCount: 0,
      transactionCount: 0,
      userCount: 0
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
    },
    roleText () {
      if (this.isAdmin) return '管理员'
      if (this.isOfficer) return '警员'
      return '访客'
    }
  },
  created () {
    this.loadDashboardStats()
  },
  methods: {
    // 加载仪表板统计信息
    async loadDashboardStats () {
      try {
        // 加载账户数量（如果是警员或管理员）
        if (this.isOfficer) {
          const accountResponse = await accountApi.getUserAccounts()
          if (accountResponse.data.code === 200) {
            this.accountCount = accountResponse.data.data.length
          }
        }

        // 加载用户数量（如果是管理员）
        if (this.isAdmin) {
          const userResponse = await userApi.getUsers()
          if (userResponse.data.code === 200) {
            this.userCount = userResponse.data.data.length
          }
        }

        // 模拟交易数量（实际应该从API获取）
        this.transactionCount = Math.floor(Math.random() * 100)
      } catch (error) {
        console.error('Failed to load dashboard stats:', error)
      }
    },

    // 处理下拉菜单命令
    handleCommand (command) {
      switch (command) {
        case 'profile':
          this.$router.push('/profile')
          break
        case 'logout':
          this.handleLogout()
          break
      }
    },

    // 退出登录
    async handleLogout () {
      try {
        // 调用登出API
        await userApi.logout()
      } catch (error) {
        console.error('Logout API failed:', error)
        // 即使API失败也继续登出流程
      } finally {
        this.$store.dispatch('logout')
        this.$router.push('/login')
        this.$message.success('已退出登录')
      }
    }
  },
  watch: {
    // 路径变化时更新活动菜单
    '$route.path' (newPath) {
      this.activeMenu = newPath
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

.el-dropdown-link {
    cursor: pointer;
    color: white;
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

.welcome-container {
    height: 100%;
    display: flex;
    justify-content: center;
    align-items: center;
}

.welcome-message {
    text-align: center;
    padding: 40px;
    background-color: white;
    border-radius: 8px;
    box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
    max-width: 800px;
    width: 100%;
}

.welcome-icon {
    font-size: 64px;
    color: #409EFF;
    margin-bottom: 20px;
}

.stats-container {
    margin: 30px 0;
}

.stat-card {
    border-radius: 8px;
}

.stat-card .el-card__body {
    padding: 20px;
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.stat-content {
    flex: 1;
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
    margin-top: 5px;
}

.stat-icon {
    font-size: 40px;
    color: #409EFF;
    opacity: 0.3;
}

.feature-box {
    margin-top: 30px;
    padding: 20px;
    background-color: #f8f9fa;
    border-radius: 4px;
}

.feature-box h3 {
    margin-bottom: 15px;
    color: #606266;
}

.feature-box .el-button {
    margin: 10px;
}
</style>
