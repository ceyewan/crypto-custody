import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex)

export default new Vuex.Store({
    state: {
        token: localStorage.getItem('token') || '',
        user: JSON.parse(localStorage.getItem('user')) || null,
        wsConnected: false,
        wsClient: null,
        notifications: [],
        currentSession: null,
        wsPingTimer: null
    },
    getters: {
        isLoggedIn: state => !!state.token,
        currentUser: state => state.user,
        userRole: state => state.user ? state.user.role : '',
        isAdmin: state => state.user && state.user.role === 'admin',
        isCoordinator: state => state.user && state.user.role === 'coordinator',
        isParticipant: state => state.user && state.user.role === 'participant',
        wsConnected: state => state.wsConnected,
        notifications: state => state.notifications,
        currentSession: state => state.currentSession
    },
    mutations: {
        setToken(state, token) {
            state.token = token
            localStorage.setItem('token', token)
        },
        setUser(state, user) {
            state.user = user
            localStorage.setItem('user', JSON.stringify(user))
        },
        clearToken(state) {
            state.token = ''
            state.user = null
            localStorage.removeItem('token')
            localStorage.removeItem('user')
        },
        setWsConnected(state, connected) {
            state.wsConnected = connected
        },
        setWsClient(state, client) {
            state.wsClient = client
        },
        addNotification(state, notification) {
            state.notifications.push(notification)
        },
        clearNotifications(state) {
            state.notifications = []
        },
        setCurrentSession(state, session) {
            state.currentSession = session
        },
        setWsPingTimer(state, timer) {
            state.wsPingTimer = timer
        }
    },
    actions: {
        login({ commit }, user) {
            commit('setToken', user.token)
            commit('setUser', user.user)
        },
        logout({ commit, state }) {
            // 关闭WebSocket连接
            if (state.wsClient) {
                state.wsClient.close()
                commit('setWsClient', null)
                commit('setWsConnected', false)
            }

            // 清除ping计时器
            if (state.wsPingTimer) {
                clearInterval(state.wsPingTimer)
                commit('setWsPingTimer', null)
            }

            // 清除用户数据
            commit('clearToken')
        },
        connectWebSocket({ commit, state, dispatch }) {
            // 清理现有连接和计时器
            if (state.wsClient) {
                state.wsClient.close()
            }

            if (state.wsPingTimer) {
                clearInterval(state.wsPingTimer)
                commit('setWsPingTimer', null)
            }

            // 创建WebSocket连接
            const ws = new WebSocket('ws://localhost:8081/ws')

            ws.onopen = () => {
                console.log('WebSocket连接已建立')

                // 发送注册消息
                if (state.user) {
                    ws.send(JSON.stringify({
                        type: 'register',
                        username: state.user.username,
                        role: state.user.role,
                        token: state.token
                    }))
                }

                // 设置ping定时器（每25秒发送一次，比服务器的30秒稍短）
                const pingTimer = setInterval(() => {
                    if (ws.readyState === WebSocket.OPEN) {
                        // 发送一个自定义ping消息
                        ws.send(JSON.stringify({ type: 'ping' }))
                        console.log('发送ping保持连接')
                    } else {
                        console.warn('WebSocket未连接，尝试重连...')
                        dispatch('reconnectWebSocket')
                    }
                }, 25000)

                commit('setWsPingTimer', pingTimer)
            }

            ws.onclose = (event) => {
                console.log('WebSocket连接已关闭', event.code, event.reason)
                commit('setWsConnected', false)

                // 清除ping计时器
                if (state.wsPingTimer) {
                    clearInterval(state.wsPingTimer)
                    commit('setWsPingTimer', null)
                }

                // 如果不是手动关闭，尝试在5秒后重连
                if (state.token && state.user) {
                    setTimeout(() => {
                        console.log('尝试重新连接WebSocket...')
                        dispatch('connectWebSocket')
                    }, 5000)
                }
            }

            ws.onerror = (error) => {
                console.error('WebSocket连接错误:', error)
                commit('setWsConnected', false)
            }

            // 存储WebSocket客户端
            commit('setWsClient', ws)
        },

        reconnectWebSocket({ dispatch, state }) {
            if (state.token && state.user) {
                console.log('正在重新连接WebSocket...')
                dispatch('connectWebSocket')
            }
        }
    }
}) 