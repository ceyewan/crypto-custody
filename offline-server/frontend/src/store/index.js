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
        currentSession: null
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

            // 清除用户数据
            commit('clearToken')
        },
        connectWebSocket({ commit, state }) {
            if (state.wsClient) {
                state.wsClient.close()
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
            }

            ws.onclose = () => {
                console.log('WebSocket连接已关闭')
                commit('setWsConnected', false)
            }

            // 存储WebSocket客户端
            commit('setWsClient', ws)
        }
    }
}) 