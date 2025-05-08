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
        wsPingTimer: null,
        wsConnecting: false,
        wsReconnectTimer: null,
        wsReconnectAttempts: 0
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
        },
        setWsConnecting(state, connecting) {
            state.wsConnecting = connecting
        },
        setWsReconnectTimer(state, timer) {
            state.wsReconnectTimer = timer
        },
        setWsReconnectAttempts(state, attempts) {
            state.wsReconnectAttempts = attempts
        },
        incrementWsReconnectAttempts(state) {
            state.wsReconnectAttempts++
        }
    },
    actions: {
        login({ commit }, user) {
            commit('setToken', user.token)
            commit('setUser', user.user)
        },
        logout({ commit, state }) {
            if (state.wsClient) {
                state.wsClient.close(1000, "用户登出")
                commit('setWsClient', null)
                commit('setWsConnected', false)
            }

            if (state.wsPingTimer) {
                clearInterval(state.wsPingTimer)
                commit('setWsPingTimer', null)
            }

            if (state.wsReconnectTimer) {
                clearTimeout(state.wsReconnectTimer)
                commit('setWsReconnectTimer', null)
            }

            commit('setWsReconnectAttempts', 0)
            commit('setWsConnecting', false)

            commit('clearToken')
        },
        connectWebSocket({ commit, state, dispatch }) {
            if (state.wsConnected || state.wsConnecting) {
                console.log('WebSocket已连接或正在连接中，跳过连接请求')
                return
            }

            commit('setWsConnecting', true)

            if (state.wsClient) {
                try {
                    state.wsClient.close(1000, "主动关闭以重新连接")
                } catch (error) {
                    console.error('关闭WebSocket连接出错:', error)
                }
            }

            if (state.wsPingTimer) {
                clearInterval(state.wsPingTimer)
                commit('setWsPingTimer', null)
            }

            if (state.wsReconnectTimer) {
                clearTimeout(state.wsReconnectTimer)
                commit('setWsReconnectTimer', null)
            }

            try {
                console.log('正在创建新的WebSocket连接...')
                const ws = new WebSocket('ws://localhost:8081/ws')

                ws.onopen = () => {
                    console.log('WebSocket连接已建立')
                    commit('setWsConnected', true)
                    commit('setWsConnecting', false)
                    commit('setWsReconnectAttempts', 0)

                    if (state.user) {
                        ws.send(JSON.stringify({
                            type: 'register',
                            username: state.user.username,
                            role: state.user.role,
                            token: state.token
                        }))
                    }

                    const pingTimer = setInterval(() => {
                        if (ws.readyState === WebSocket.OPEN) {
                            ws.send(JSON.stringify({ type: 'ping' }))
                            console.log('发送ping保持连接')
                        } else if (ws.readyState !== WebSocket.CONNECTING) {
                            console.warn('WebSocket连接已断开，清除ping定时器')
                            clearInterval(pingTimer)
                        }
                    }, 25000)

                    commit('setWsPingTimer', pingTimer)
                }

                ws.onclose = (event) => {
                    const wasConnected = state.wsConnected;
                    console.log('WebSocket连接已关闭', event.code, event.reason)
                    commit('setWsConnected', false)
                    commit('setWsConnecting', false)

                    if (state.wsPingTimer) {
                        clearInterval(state.wsPingTimer)
                        commit('setWsPingTimer', null)
                    }

                    const isNormalClosure = event.code === 1000 && event.reason.includes("用户登出");
                    const isManualClose = event.code === 1000 && event.reason.includes("主动关闭");

                    if (!isNormalClosure && !isManualClose && state.token && state.user) {
                        const attempts = state.wsReconnectAttempts;
                        const delay = Math.min(1000 * Math.pow(1.5, attempts), 30000);

                        console.log(`将在 ${delay / 1000} 秒后尝试第 ${attempts + 1} 次重连...`);

                        const reconnectTimer = setTimeout(() => {
                            commit('incrementWsReconnectAttempts');
                            dispatch('connectWebSocket');
                        }, delay);

                        commit('setWsReconnectTimer', reconnectTimer);
                    }
                }

                ws.onerror = (error) => {
                    console.error('WebSocket连接错误:', error)
                }

                commit('setWsClient', ws)
            } catch (error) {
                console.error('创建WebSocket连接失败:', error)
                commit('setWsConnecting', false)

                if (state.token && state.user) {
                    const reconnectTimer = setTimeout(() => {
                        commit('incrementWsReconnectAttempts')
                        dispatch('connectWebSocket')
                    }, 5000)

                    commit('setWsReconnectTimer', reconnectTimer)
                }
            }
        },

        resetWebSocketConnection({ commit, dispatch, state }) {
            console.log('重置WebSocket连接')

            if (state.wsClient) {
                try {
                    state.wsClient.close(1000, "重置连接")
                } catch (error) {
                    console.error('关闭WebSocket连接出错:', error)
                }
                commit('setWsClient', null)
            }

            if (state.wsPingTimer) {
                clearInterval(state.wsPingTimer)
                commit('setWsPingTimer', null)
            }

            if (state.wsReconnectTimer) {
                clearTimeout(state.wsReconnectTimer)
                commit('setWsReconnectTimer', null)
            }

            commit('setWsConnected', false)
            commit('setWsConnecting', false)
            commit('setWsReconnectAttempts', 0)

            setTimeout(() => {
                dispatch('connectWebSocket')
            }, 1000)
        }
    }
}) 