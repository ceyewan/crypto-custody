import Vue from 'vue'
import Vuex from 'vuex'
import { clientConfigApi } from '../services/wails-api'
import {
    getServerWsUrl,
    loadClientSettings,
    saveClientSettings as persistClientSettings
} from '../services/settings'

Vue.use(Vuex)

function loadMpcTasks() {
    try {
        const tasks = JSON.parse(localStorage.getItem('offline_client_mpc_tasks') || '{}')
        Object.keys(tasks).forEach(key => {
            if (tasks[key].status === 'running') {
                tasks[key].status = 'interrupted'
                tasks[key].message = '客户端重启后任务执行状态已重置'
            }
        })
        return tasks
    } catch {
        return {}
    }
}

function persistMpcTasks(tasks) {
    localStorage.setItem('offline_client_mpc_tasks', JSON.stringify(tasks))
}

function notificationIdentity(notification) {
    const content = notification.content || {}
    return [
        notification.type,
        content.session_key || '',
        content.party_index || content.signing_index || ''
    ].join(':')
}

export default new Vuex.Store({
    state: {
        token: localStorage.getItem('token') || '',
        user: JSON.parse(localStorage.getItem('user')) || null,
        clientSettings: loadClientSettings(),
        mpcTasks: loadMpcTasks(),
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
        isOfficer: state => state.user && state.user.role === 'officer',
        isAuditor: state => state.user && state.user.role === 'auditor',
        wsConnected: state => state.wsConnected,
        notifications: state => state.notifications,
        currentSession: state => state.currentSession,
        clientSettings: state => state.clientSettings,
        mpcTasks: state => state.mpcTasks
    },
    mutations: {
        setToken(state, token) {
            state.token = token
            localStorage.setItem('token', token)
        },
        setUser(state, user) {
            state.user = user
            localStorage.setItem('user', JSON.stringify(user))
            if (user && user.username) {
                localStorage.setItem('last_username', user.username)
            }
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
            const identity = notificationIdentity(notification)
            const index = state.notifications.findIndex(item => notificationIdentity(item) === identity)
            if (index !== -1) {
                Vue.set(state.notifications, index, {
                    ...state.notifications[index],
                    ...notification,
                    responded: state.notifications[index].responded || notification.responded
                })
                return
            }
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
        },
        setClientSettings(state, settings) {
            state.clientSettings = settings
        },
        setMpcTask(state, { key, patch }) {
            const previous = state.mpcTasks[key] || {}
            Vue.set(state.mpcTasks, key, {
                ...previous,
                ...patch,
                updated_at: new Date().toISOString()
            })
            persistMpcTasks(state.mpcTasks)
        },
        clearMpcTasks(state) {
            state.mpcTasks = {}
            persistMpcTasks(state.mpcTasks)
        }
    },
    actions: {
        async saveClientSettings({ commit, state, dispatch }, settings) {
            const oldWsUrl = state.clientSettings.serverWsUrl
            const saved = persistClientSettings(settings)
            commit('setClientSettings', saved)

            try {
                await clientConfigApi.setCardReaderName(saved.cardReaderName)
            } catch (error) {
                console.warn('设置读卡器名称失败，可能不在 Wails 环境中:', error)
            }

            if (oldWsUrl !== saved.serverWsUrl && state.wsClient) {
                dispatch('resetWebSocketConnection')
            }
            return saved
        },
        async applyClientRuntimeSettings({ state }) {
            try {
                await clientConfigApi.setCardReaderName(state.clientSettings.cardReaderName)
            } catch (error) {
                console.warn('应用读卡器名称失败，可能不在 Wails 环境中:', error)
            }
        },
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
                const wsURL = state.clientSettings.serverWsUrl || getServerWsUrl()
                console.log(`[WS Debug] 使用 WebSocket 地址: ${wsURL}`);
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

                ws.onopen = async () => {
                    clearTimeout(connectionTimeout);
                    console.log('WebSocket连接已建立')
                    commit('setWsConnected', true)
                    commit('setWsConnecting', false)
                    commit('setWsReconnectAttempts', 0)
                    commit('setWsLastError', null)

                    try {
                        const { initWebSocketService } = await import('../services/ws')
                        initWebSocketService()
                    } catch (error) {
                        console.error('初始化WebSocket消息处理失败:', error)
                    }

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
                        event.reason.includes("主动关闭") ||
                        event.reason.includes("重置连接")
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
