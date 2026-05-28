import Vue from 'vue'
import VueRouter from 'vue-router'
import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import Dashboard from '../views/Dashboard.vue'
import Users from '../views/Users.vue'
import KeyGen from '../views/KeyGen.vue'
import Sign from '../views/Sign.vue'
import Notifications from '../views/Notifications.vue'
import Test from '../views/Test.vue'
import ImportSE from '../views/ImportSE.vue'
import OfflineTasks from '../views/OfflineTasks.vue'
import KeyManagement from '../views/KeyManagement.vue'
import AuditLogs from '../views/AuditLogs.vue'
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
        name: 'ImportSE',
        component: ImportSE,
        meta: { requiresAuth: true, requiresAdmin: true }
    },
    {
        path: '/keygen',
        name: 'KeyGen',
        component: KeyGen,
        meta: { requiresAuth: true, requiresCoordinatorOrAdmin: true }
    },
    {
        path: '/sign',
        name: 'Sign',
        component: Sign,
        meta: { requiresAuth: true, requiresCoordinatorOrAdmin: true }
    },
    {
        path: '/offline-tasks',
        name: 'OfflineTasks',
        component: OfflineTasks,
        meta: { requiresAuth: true, requiresCoordinatorOrAdmin: true }
    },
    {
        path: '/keys',
        name: 'KeyManagement',
        component: KeyManagement,
        meta: { requiresAuth: true, requiresCoordinatorOrAdmin: true }
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
        meta: { requiresAuth: true }
    },
    {
        path: '/test',
        name: 'Test',
        component: Test,
        meta: { requiresAuth: false } // 测试页面不需要认证
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
    const requiresCoordinatorOrAdmin = to.matched.some(record => record.meta.requiresCoordinatorOrAdmin)
    const requiresAuditAccess = to.matched.some(record => record.meta.requiresAuditAccess)

    // 如果需要身份验证且未登录，则重定向到登录页面
    if (requiresAuth && !store.getters.isLoggedIn) {
        next('/login')
    }
    // 如果需要管理员权限但当前用户不是管理员
    else if (requiresAdmin && !store.getters.isAdmin) {
        next('/dashboard')
    }
    // 如果需要协调者或管理员权限，但当前用户既不是协调者也不是管理员
    else if (requiresCoordinatorOrAdmin && !(store.getters.isCoordinator || store.getters.isAdmin)) {
        next('/dashboard')
    }
    // 如果需要审计查询权限，但当前用户不是审计员、协调者或管理员
    else if (requiresAuditAccess && !(store.getters.isAuditor || store.getters.isCoordinator || store.getters.isAdmin)) {
        next('/dashboard')
    }
    else {
        next()
    }
})

export default router
