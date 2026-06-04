import Vue from 'vue'
import VueRouter from 'vue-router'
import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import Dashboard from '../views/Dashboard.vue'
import Users from '../views/Users.vue'
import Notifications from '../views/Notifications.vue'
import ImportSE from '../views/ImportSE.vue'
import OfflineTasks from '../views/OfflineTasks.vue'
import KeyManagement from '../views/KeyManagement.vue'
import AuditLogs from '../views/AuditLogs.vue'
import ClientSettings from '../views/ClientSettings.vue'
import ServerSettings from '../views/ServerSettings.vue'
import MyShards from '../views/MyShards.vue'
import ParticipationHistory from '../views/ParticipationHistory.vue'
import BackupRestore from '../views/BackupRestore.vue'
import store from '../store'

Vue.use(VueRouter)

const routes = [
    {
        path: '/',
        redirect: '/dashboard'
    },
    {
        path: '/login',
        name: 'Login',
        component: Login,
        meta: { requiresAuth: false }
    },
    {
        path: '/register',
        name: 'Register',
        component: Register,
        meta: { requiresAuth: false }
    },
    {
        path: '/server-settings',
        name: 'ServerSettings',
        component: ServerSettings,
        meta: { requiresAuth: false }
    },
    {
        path: '/dashboard',
        name: 'Dashboard',
        component: Dashboard,
        meta: { requiresAuth: true }
    },
    {
        path: '/users',
        name: 'Users',
        component: Users,
        meta: { requiresAuth: true, requiresAdmin: true }
    },
    {
        path: '/import-se',
        redirect: '/security-elements'
    },
    {
        path: '/security-elements',
        name: 'SecurityElements',
        component: ImportSE,
        meta: { requiresAuth: true, requiresAdmin: true }
    },
    { path: '/keygen', redirect: '/offline-tasks' },
    { path: '/sign', redirect: '/offline-tasks' },
    {
        path: '/offline-tasks',
        name: 'OfflineTasks',
        component: OfflineTasks,
        meta: { requiresAuth: true, requiresAuditAccess: true }
    },
    {
        path: '/keys',
        name: 'KeyManagement',
        component: KeyManagement,
        meta: { requiresAuth: true, requiresAuditAccess: true }
    },
    {
        path: '/audit',
        name: 'AuditLogs',
        component: AuditLogs,
        meta: { requiresAuth: true, requiresAuditAccess: true }
    },
    {
        path: '/notifications',
        name: 'Notifications',
        component: Notifications,
        meta: { requiresAuth: true, requiresOfficerOrAdmin: true }
    },
    {
        path: '/my-shards',
        name: 'MyShards',
        component: MyShards,
        meta: { requiresAuth: true, requiresOfficerOrAdmin: true }
    },
    {
        path: '/participation',
        name: 'ParticipationHistory',
        component: ParticipationHistory,
        meta: { requiresAuth: true, requiresOfficerOrAdmin: true }
    },
    {
        path: '/backup',
        name: 'BackupRestore',
        component: BackupRestore,
        meta: { requiresAuth: true, requiresAdmin: true }
    },
    {
        path: '/settings',
        name: 'ClientSettings',
        component: ClientSettings,
        meta: { requiresAuth: true }
    }
]

const router = new VueRouter({
    mode: 'hash', // Electron 环境下使用 hash 模式
    base: process.env.BASE_URL,
    routes
})

// 全局路由守卫
router.beforeEach((to, from, next) => {
    const requiresAuth = to.matched.some(record => record.meta.requiresAuth)
    const requiresAdmin = to.matched.some(record => record.meta.requiresAdmin)
    const requiresOfficerOrAdmin = to.matched.some(record => record.meta.requiresOfficerOrAdmin)
    const requiresAuditAccess = to.matched.some(record => record.meta.requiresAuditAccess)

    // 如果需要身份验证且未登录，则重定向到登录页面
    if (requiresAuth && !store.getters.isLoggedIn) {
        next('/login')
    }
    else if (requiresAdmin && !store.getters.isAdmin) {
        next('/dashboard')
    }
    else if (requiresOfficerOrAdmin && !(store.getters.isOfficer || store.getters.isAdmin)) {
        next('/dashboard')
    }
    else if (requiresAuditAccess && !(store.getters.isAuditor || store.getters.isAdmin)) {
        next('/dashboard')
    }
    else {
        next()
    }
})

export default router
