import Vue from 'vue'
import VueRouter from 'vue-router'
import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import Dashboard from '../views/Dashboard.vue'
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
    }
]

const router = new VueRouter({
    mode: 'history',
    base: process.env.BASE_URL,
    routes
})

// 全局路由守卫
router.beforeEach((to, from, next) => {
    const requiresAuth = to.matched.some(record => record.meta.requiresAuth)

    // 如果需要身份验证且未登录，则重定向到登录页面
    if (requiresAuth && !store.getters.isLoggedIn) {
        next('/login')
    } else {
        next()
    }
})

export default router 