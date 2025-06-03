import Vue from 'vue'
import VueRouter from 'vue-router'
import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import Dashboard from '../views/Dashboard.vue'
import Users from '../views/Users.vue'
import Accounts from '../views/Accounts.vue'
import Transactions from '../views/Transactions.vue'
import Profile from '../views/Profile.vue'
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
    path: '/accounts',
    name: 'Accounts',
    component: Accounts,
    meta: { requiresAuth: true, requiresOfficer: true }
  },
  {
    path: '/transactions',
    name: 'Transactions',
    component: Transactions,
    meta: { requiresAuth: true, requiresOfficer: true }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: Profile,
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
  const requiresAdmin = to.matched.some(record => record.meta.requiresAdmin)
  const requiresOfficer = to.matched.some(record => record.meta.requiresOfficer)

  // 如果需要身份验证且未登录，则重定向到登录页面
  if (requiresAuth && !store.getters.isLoggedIn) {
    next('/login')
  // 如果需要管理员权限但当前用户不是管理员
  } else if (requiresAdmin && !store.getters.isAdmin) {
    next('/dashboard')
  // 如果需要警员权限但当前用户不是警员或管理员
  } else if (requiresOfficer && !store.getters.isOfficer) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
