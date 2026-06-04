<template>
    <div id="app">
        <router-view v-if="isPublicRoute" />

        <el-container v-else class="app-shell">
            <el-header class="app-header">
                <div class="brand">
                    <span class="brand-title">离线虚拟货币存管提控系统</span>
                    <span class="brand-subtitle">离线协同工作台</span>
                </div>
                <div class="header-actions">
                    <ws-status-indicator class="header-ws"></ws-status-indicator>
                    <span class="user-chip">{{ userLabel }} / {{ roleText }}</span>
                    <el-button type="text" icon="el-icon-setting" @click="$router.push('/settings')">设置</el-button>
                    <el-button type="text" icon="el-icon-switch-button" @click="logout">退出</el-button>
                </div>
            </el-header>

            <el-container>
                <el-aside width="236px" class="app-aside">
                    <el-menu
                        :default-active="$route.path"
                        router
                        background-color="#304156"
                        text-color="#bfcbd9"
                        active-text-color="#409EFF">
                        <el-menu-item index="/dashboard">
                            <i class="el-icon-s-home"></i>
                            <span>仪表盘</span>
                        </el-menu-item>

                        <template v-if="isAdmin">
                            <el-menu-item index="/offline-tasks">
                                <i class="el-icon-upload2"></i>
                                <span>离线任务</span>
                            </el-menu-item>
                            <el-menu-item index="/keys">
                                <i class="el-icon-wallet"></i>
                                <span>地址管理</span>
                            </el-menu-item>
                            <el-menu-item index="/security-elements">
                                <i class="el-icon-cpu"></i>
                                <span>SE 管理</span>
                            </el-menu-item>
                            <el-menu-item index="/users">
                                <i class="el-icon-user"></i>
                                <span>用户管理</span>
                            </el-menu-item>
                            <el-menu-item index="/backup">
                                <i class="el-icon-folder-checked"></i>
                                <span>备份恢复</span>
                            </el-menu-item>
                        </template>

                        <template v-if="canParticipateMpc">
                            <el-menu-item index="/notifications">
                                <i class="el-icon-bell"></i>
                                <span>待处理邀请</span>
                                <el-badge v-if="pendingInvitations.length" :value="pendingInvitations.length" class="nav-badge" />
                            </el-menu-item>
                            <el-menu-item index="/my-shards">
                                <i class="el-icon-collection-tag"></i>
                                <span>我的私钥分片</span>
                            </el-menu-item>
                            <el-menu-item index="/participation">
                                <i class="el-icon-time"></i>
                                <span>参与记录</span>
                            </el-menu-item>
                        </template>

                        <template v-if="isAuditor && !isAdmin">
                            <el-menu-item index="/offline-tasks">
                                <i class="el-icon-document"></i>
                                <span>离线任务</span>
                            </el-menu-item>
                            <el-menu-item index="/keys">
                                <i class="el-icon-wallet"></i>
                                <span>地址管理</span>
                            </el-menu-item>
                        </template>

                        <el-menu-item v-if="isAdmin || isAuditor" index="/audit">
                            <i class="el-icon-document-checked"></i>
                            <span>审计日志</span>
                        </el-menu-item>

                        <el-menu-item index="/settings">
                            <i class="el-icon-setting"></i>
                            <span>客户端设置</span>
                        </el-menu-item>
                    </el-menu>
                </el-aside>

                <el-main class="app-main">
                    <router-view />
                </el-main>
            </el-container>
        </el-container>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'
import WsStatusIndicator from './components/WsStatusIndicator.vue'

export default {
    name: 'App',
    components: { WsStatusIndicator },
    computed: {
        ...mapGetters(['isLoggedIn', 'currentUser', 'isAdmin', 'isOfficer', 'isAuditor', 'notifications', 'canParticipateMpc']),
        isPublicRoute() {
            return this.$route.path === '/login' || this.$route.path === '/register' || this.$route.path === '/server-settings'
        },
        userLabel() {
            const user = this.currentUser || {}
            return user.nickname || user.username || '未登录'
        },
        roleText() {
            if (this.isAdmin) return '管理员'
            if (this.isOfficer) return '警员'
            if (this.isAuditor) return '审计员'
            return '访客'
        },
        pendingInvitations() {
            return this.notifications.filter(n =>
                ['keygen_invite', 'sign_invite', 'destroy_invite', 'transfer_invite'].includes(n.type) &&
                !n.responded
            )
        }
    },
    mounted() {
        if (this.isLoggedIn) {
            this.$store.dispatch('connectWebSocket')
        }
    },
    methods: {
        logout() {
            this.$store.dispatch('logout')
            this.$router.push('/login')
        }
    }
}
</script>

<style>
#app {
    font-family: 'Microsoft YaHei', Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
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

.brand {
    display: flex;
    align-items: baseline;
    gap: 12px;
}

.brand-title {
    font-size: 18px;
    font-weight: 600;
}

.brand-subtitle {
    color: #bfcbd9;
    font-size: 12px;
}

.header-actions {
    display: flex;
    align-items: center;
    gap: 12px;
}

.header-actions .el-button {
    color: #fff;
}

.header-actions .ws-status-indicator {
    position: static;
    opacity: 1;
}

.user-chip {
    color: #e5eaf3;
    font-size: 13px;
}

.app-aside {
    background: #304156;
}

.app-aside .el-menu {
    border-right: none;
}

.nav-badge {
    margin-left: 8px;
}

.app-main {
    min-height: calc(100vh - 60px);
    padding: 0;
    background: #f0f2f5;
}

.page {
    padding: 20px;
}

.page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
}

.page-title {
    margin: 0;
    font-size: 20px;
    font-weight: 600;
}

.page-subtitle {
    margin: 6px 0 0;
    color: #606266;
    font-size: 13px;
}
</style>
