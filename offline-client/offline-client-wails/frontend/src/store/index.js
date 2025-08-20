import Vue from 'vue'
import Vuex from 'vuex'
import { Environment } from '../../wailsjs/runtime/runtime'

Vue.use(Vuex)

export default new Vuex.Store({
    state: {
        token: localStorage.getItem('token') || '',
        user: JSON.parse(localStorage.getItem('user')) || null,
        wsConnected: false,
        wsClient: null,
        notifications: [],
        currentSession: null,
        wsConnecting: false,
        wsReconnectTimer: null,
        wsReconnectAttempts: 0,
        wsLastError: null,
        wsConnectionLostTime: null
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
        updateNotificationResponse(state, { timestamp, type, responded = true }) {
            const index = state.notifications.findIndex(n =>
                n.timestamp.getTime() === timestamp.getTime() &&
                n.type === type
            )

            if (index !== -1) {
                // 使用Vue.set确保响应式更新
                Vue.set(state.notifications[index], 'responded', responded)
            }
        },
        setCurrentSession(state, session) {
            state.currentSession = session
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
        },
        setWsLastError(state, error) {
            state.wsLastError = error
        },
        setWsConnectionLostTime(state, timestamp) {
            state.wsConnectionLostTime = timestamp
        }
    },
    actions: {
        login({ commit }, user) {
            const token = user.token && user.token.startsWith('Bearer ')
                ? user.token.substring(7)
                : user.token

            commit('setToken', token)
            commit('setUser', user.user)
        },
        logout({ commit, state }) {
            if (state.wsClient) {
                state.wsClient.close(1000, "用户登出")
                commit('setWsClient', null)
                commit('setWsConnected', false)
            }

            commit('setWsReconnectAttempts', 0)
            commit('setWsConnecting', false)

            commit('clearToken')
        },
        async connectWebSocket({ commit, state, dispatch }) {
            if (state.wsConnected || state.wsConnecting) {
                console.log('WebSocket已连接或正在连接中，跳过连接请求')
                return
            }

            commit('setWsConnecting', true)

            if (state.wsClient) {
                try {
                    if (state.wsClient.readyState === WebSocket.OPEN ||
                        state.wsClient.readyState === WebSocket.CONNECTING) {
                        state.wsClient.close(1000, "主动关闭以重新连接")
                    }
                } catch (error) {
                    console.error('关闭WebSocket连接出错:', error)
                }
            }

            if (state.wsReconnectTimer) {
                clearTimeout(state.wsReconnectTimer)
                commit('setWsReconnectTimer', null)
            }

            try {
                console.log('正在创建新的WebSocket连接...')
                
                // 使用 Wails runtime.Environment() 来可靠地检测环境
                let wsURL = `ws://localhost:8090/ws`; // 默认使用代理
                try {
                    const envInfo = await Environment();
                    // 在 Wails 环境中直接连接远程服务器
                    wsURL = 'wss://crypto-custody-offline-server.ceyewan.icu/ws';
                    console.log(`[WS Debug] Wails Environment detected - buildType: ${envInfo.buildType}, platform: ${envInfo.platform}, WS URL: ${wsURL}`);
                } catch (error) {
                    // 不是 Wails 环境，使用代理
                    console.log(`[WS Debug] Non-Wails Environment detected, using proxy: ${wsURL}`);
                }
                
                const ws = new WebSocket(wsURL)

                const connectionTimeout = setTimeout(() => {
                    if (ws.readyState !== WebSocket.OPEN) {
                        console.error('WebSocket连接超时')
                        ws.close(3000, "连接超时")
                        commit('setWsConnecting', false)
                        commit('setWsLastError', "连接超时")

                        if (state.token && state.user) {
                            const attempts = state.wsReconnectAttempts;
                            const delay = Math.min(1000 * Math.pow(1.5, attempts), 30000);

                            console.log(`连接超时，将在 ${delay / 1000} 秒后重新连接...`);

                            const reconnectTimer = setTimeout(() => {
                                commit('incrementWsReconnectAttempts');
                                dispatch('connectWebSocket');
                            }, delay);

                            commit('setWsReconnectTimer', reconnectTimer);
                        }
                    }
                }, 10000);

                ws.onopen = () => {
                    clearTimeout(connectionTimeout);
                    console.log('WebSocket连接已建立')
                    commit('setWsConnected', true)
                    commit('setWsConnecting', false)
                    commit('setWsReconnectAttempts', 0)
                    commit('setWsLastError', null)

                    if (state.user) {
                        const token = state.token && state.token.startsWith('Bearer ')
                            ? state.token.substring(7)
                            : state.token

                        ws.send(JSON.stringify({
                            type: 'register',
                            username: state.user.username,
                            role: state.user.role,
                            token: token
                        }))
                    }
                }

                ws.onclose = (event) => {
                    clearTimeout(connectionTimeout);
                    const wasConnected = state.wsConnected;
                    console.log('WebSocket连接已关闭', event.code, event.reason)
                    commit('setWsConnected', false)
                    commit('setWsConnecting', false)

                    if (wasConnected) {
                        commit('setWsConnectionLostTime', Date.now())
                    }

                    const isNormalClosure = event.code === 1000 && (
                        event.reason.includes("用户登出") ||
                        event.reason.includes("主动关闭")
                    );

                    if (!isNormalClosure && state.token && state.user) {
                        const attempts = state.wsReconnectAttempts;
                        const delay = Math.min(1000 * Math.pow(1.5, attempts), 30000);

                        console.log(`WebSocket连接非正常关闭，将在 ${delay / 1000} 秒后尝试第 ${attempts + 1} 次重连...`);

                        commit('setWsLastError', `连接关闭: 代码 ${event.code}, 原因: ${event.reason || "未知"}`);

                        const reconnectTimer = setTimeout(() => {
                            commit('incrementWsReconnectAttempts');
                            dispatch('connectWebSocket');
                        }, delay);

                        commit('setWsReconnectTimer', reconnectTimer);
                    }
                }

                ws.onerror = (error) => {
                    console.error('WebSocket连接错误:', error)
                    commit('setWsLastError', "连接错误")
                }

                commit('setWsClient', ws)
            } catch (error) {
                console.error('创建WebSocket连接失败:', error)
                commit('setWsConnecting', false)
                commit('setWsLastError', `连接创建失败: ${error.message || "未知错误"}`)

                if (state.token && state.user) {
                    const attempts = state.wsReconnectAttempts;
                    const delay = Math.min(1000 * Math.pow(1.5, attempts), 30000);

                    console.log(`创建连接失败，将在 ${delay / 1000} 秒后重新尝试...`);

                    const reconnectTimer = setTimeout(() => {
                        commit('incrementWsReconnectAttempts')
                        dispatch('connectWebSocket')
                    }, delay)

                    commit('setWsReconnectTimer', reconnectTimer)
                }
            }
        },

        resetWebSocketConnection({ commit, dispatch, state }) {
            console.log('重置WebSocket连接')

            if (state.wsClient) {
                try {
                    if (state.wsClient.readyState === WebSocket.OPEN ||
                        state.wsClient.readyState === WebSocket.CONNECTING) {
                        state.wsClient.close(1000, "重置连接")
                    }
                } catch (error) {
                    console.error('关闭WebSocket连接出错:', error)
                }
                commit('setWsClient', null)
            }

            if (state.wsReconnectTimer) {
                clearTimeout(state.wsReconnectTimer)
                commit('setWsReconnectTimer', null)
            }

            commit('setWsConnected', false)
            commit('setWsConnecting', false)
            commit('setWsReconnectAttempts', 0)

            if (state.token && state.user) {
                setTimeout(() => {
                    dispatch('connectWebSocket')
                }, 1000)
            }
        },

        checkWebSocketHealth({ dispatch, state }) {
            if (!state.user || !state.token) {
                return false
            }

            const ws = state.wsClient
            if (!ws || ws.readyState !== WebSocket.OPEN) {
                console.warn('WebSocket连接不正常，尝试重置...')
                dispatch('resetWebSocketConnection')
                return false
            }

            return true
        }
    }
}) 