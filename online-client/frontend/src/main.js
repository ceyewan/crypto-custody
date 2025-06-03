import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import ElementUI from 'element-ui'
import 'element-ui/lib/theme-chalk/index.css'
import axios from 'axios'

Vue.config.productionTip = false

// 使用ElementUI组件库
Vue.use(ElementUI)

// 配置axios
axios.defaults.baseURL = 'http://localhost:8080'
Vue.prototype.$axios = axios

// 添加请求拦截器处理认证
axios.interceptors.request.use(
    config => {
        const token = localStorage.getItem('token')
        if (token) {
            config.headers['Authorization'] = `Bearer ${token}`
        }
        return config
    },
    error => {
        return Promise.reject(error)
    }
)

// 添加响应拦截器处理401错误
axios.interceptors.response.use(
    response => {
        return response
    },
    error => {
        if (error.response && error.response.status === 401) {
            // 清除token
            store.commit('clearToken')
            // 重定向到登录页
            router.push('/login')
            return Promise.reject(error)
        }
        return Promise.reject(error)
    }
)

new Vue({
    router,
    store,
    render: h => h(App)
}).$mount('#app')