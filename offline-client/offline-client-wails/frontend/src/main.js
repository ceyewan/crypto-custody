import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import ElementUI from 'element-ui'
import 'element-ui/lib/theme-chalk/index.css'
import axios from 'axios'

// 引入原有的云端 API 服务（用于与云端服务器通信）
import { userApi, keygenApi, signApi } from './services/api.js'

// 引入 Wails 本地 MPC API 服务（用于调用内置 web-se 模块）
import wailsMpcAPI from './services/wails-api.js'

Vue.config.productionTip = false

// 使用 ElementUI 组件库
Vue.use(ElementUI)

// 配置 axios（用于与云端服务器通信）
axios.defaults.baseURL = 'http://localhost:8080'
Vue.prototype.$axios = axios

// 挂载云端 API 服务（用户认证、会话管理等）
Vue.prototype.$userApi = userApi
Vue.prototype.$keygenApi = keygenApi
Vue.prototype.$signApi = signApi

// 挂载本地 MPC API 服务（替代原来的外部 web-se 调用）
Vue.prototype.$localMpcApi = wailsMpcAPI.mpcApi
Vue.prototype.$localSeApi = wailsMpcAPI.seApi

// 添加响应拦截器处理401错误（仅对云端服务器通信）
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