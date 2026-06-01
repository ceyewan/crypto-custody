<template>
  <div id="app">
    <router-view v-if="isPublicRoute" />
    <el-container v-else class="app-shell">
      <el-header class="app-header">
        <div class="logo">在线加密货币托管系统</div>
        <div class="user-info">
          <span>{{ user.username }} / {{ roleText }}</span>
          <el-button type="text" @click="$router.push('/profile')">个人资料</el-button>
          <el-button type="text" @click="logout">退出</el-button>
        </div>
      </el-header>
      <el-container>
        <el-aside width="230px" class="aside">
          <el-menu :default-active="$route.path" router background-color="#304156" text-color="#bfcbd9" active-text-color="#409EFF">
            <el-menu-item index="/dashboard"><i class="el-icon-s-home"></i><span>仪表盘</span></el-menu-item>
            <el-menu-item v-if="isAdmin" index="/users"><i class="el-icon-user"></i><span>用户管理</span></el-menu-item>
            <el-menu-item v-if="isOfficer || isAuditor" index="/cases"><i class="el-icon-folder"></i><span>案件管理</span></el-menu-item>
            <el-menu-item v-if="isOfficer || isAuditor" index="/accounts"><i class="el-icon-wallet"></i><span>账户管理</span></el-menu-item>
            <el-menu-item v-if="isOfficer || isAuditor" index="/transactions"><i class="el-icon-s-finance"></i><span>交易管理</span></el-menu-item>
            <el-menu-item v-if="isAuditor" index="/audit-logs"><i class="el-icon-document"></i><span>审计日志</span></el-menu-item>
            <el-menu-item v-if="isAdmin" index="/backups"><i class="el-icon-box"></i><span>备份恢复</span></el-menu-item>
            <el-menu-item v-if="isAdmin" index="/test-data"><i class="el-icon-data-analysis"></i><span>测试数据</span></el-menu-item>
          </el-menu>
        </el-aside>
        <el-main class="main">
          <router-view />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { userApi } from './services/api'

export default {
  name: 'App',
  computed: {
    ...mapGetters(['currentUser', 'isAdmin', 'isOfficer', 'isAuditor']),
    isPublicRoute () {
      return this.$route.path === '/login' || this.$route.path === '/register'
    },
    user () {
      return this.currentUser || {}
    },
    roleText () {
      if (this.isAdmin) return '管理员'
      if (this.isOfficer) return '警员'
      if (this.isAuditor) return '审计员'
      return '访客'
    }
  },
  methods: {
    async logout () {
      try {
        await userApi.logout()
      } catch (e) {
        // ignore logout API failures
      }
      this.$store.dispatch('logout')
      this.$router.push('/login')
    }
  }
}
</script>

<style>
#app {
  font-family: 'Microsoft YaHei', Helvetica, Arial, sans-serif;
  color: #2c3e50;
  min-height: 100vh;
}

body,
html {
  margin: 0;
  padding: 0;
  height: 100%;
  width: 100%;
}

.app-shell {
  min-height: 100vh;
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #304156;
  color: #fff;
}

.logo {
  font-size: 18px;
  font-weight: 600;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-info .el-button {
  color: #fff;
}

.aside {
  background: #304156;
}

.aside .el-menu {
  border-right: none;
}

.main {
  min-height: calc(100vh - 60px);
  padding: 0;
  background: #f0f2f5;
}
</style>
