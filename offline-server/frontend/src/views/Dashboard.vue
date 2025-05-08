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
                        <el-menu-item v-if="isAdmin" index="/users">
                            <i class="el-icon-user"></i>
                            <span>用户管理</span>
                        </el-menu-item>
                        <el-menu-item v-if="isAdmin || isCoordinator" index="/keygen">
                            <i class="el-icon-key"></i>
                            <span>密钥生成</span>
                        </el-menu-item>
                        <el-menu-item v-if="isAdmin || isCoordinator" index="/sign">
                            <i class="el-icon-edit"></i>
                            <span>交易签名</span>
                        </el-menu-item>
                        <el-menu-item index="/notifications">
                            <i class="el-icon-bell"></i>
                            <span>通知消息</span>
                            <el-badge v-if="notifications.length > 0" :value="notifications.length" class="item">
                            </el-badge>
                        </el-menu-item>
                    </el-menu>
                </el-aside>

                <el-main>
                    <router-view></router-view>

                    <div v-if="$route.path === '/dashboard'" class="welcome-container">
                        <div class="welcome-message">
                            <i class="el-icon-s-home welcome-icon"></i>
                            <h2>欢迎使用多方门限签名系统</h2>
                            <p>请根据您的角色和需求，使用左侧菜单选择相应功能</p>

                            <div v-if="isAdmin" class="feature-box">
                                <h3>管理员功能</h3>
                                <el-button type="primary" @click="$router.push('/users')">用户管理</el-button>
                                <el-button type="primary" @click="$router.push('/keygen')">密钥生成</el-button>
                                <el-button type="primary" @click="$router.push('/sign')">交易签名</el-button>
                            </div>

                            <div v-else-if="isCoordinator" class="feature-box">
                                <h3>协调者功能</h3>
                                <el-button type="primary" @click="$router.push('/keygen')">密钥生成</el-button>
                                <el-button type="primary" @click="$router.push('/sign')">交易签名</el-button>
                            </div>

                            <div v-else-if="isParticipant" class="feature-box">
                                <h3>参与者功能</h3>
                                <p>作为参与者，您将收到密钥生成和签名请求的邀请</p>
                                <el-button type="primary" @click="$router.push('/notifications')">查看通知</el-button>
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

export default {
    name: 'Dashboard',
    data() {
        return {
            activeMenu: this.$route.path
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
    },
    methods: {
        // 退出登录
        handleLogout() {
            this.$store.dispatch('logout')
            this.$router.push('/login')
        }
    },
    watch: {
        // 路径变化时更新活动菜单
        '$route.path'(newPath) {
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
    max-width: 600px;
}

.welcome-icon {
    font-size: 64px;
    color: #409EFF;
    margin-bottom: 20px;
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