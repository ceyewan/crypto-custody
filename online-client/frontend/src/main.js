import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import ElementUI from 'element-ui'
import 'element-ui/lib/theme-chalk/index.css'
import { apiClient } from './services/api'

Vue.config.productionTip = false

// 使用ElementUI组件库
Vue.use(ElementUI)

// 使用统一配置的axios实例
Vue.prototype.$axios = apiClient

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
