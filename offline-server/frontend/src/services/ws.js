import store from '../store'
import { mpcApi, seApi } from './api'
import { MessageBox, Message } from 'element-ui'

// 跟踪WebSocket服务初始化状态
let wsServiceInitialized = false;

// WebSocket消息类型常量
export const WS_MESSAGE_TYPES = {
    // 注册相关
    REGISTER: 'register',
    REGISTER_COMPLETE: 'register_complete',

    // 密钥生成相关
    KEYGEN_REQUEST: 'keygen_request',
    KEYGEN_INVITE: 'keygen_invite',
    KEYGEN_RESPONSE: 'keygen_response',
    KEYGEN_PARAMS: 'keygen_params',
    KEYGEN_RESULT: 'keygen_result',
    KEYGEN_COMPLETE: 'keygen_complete',

    // 签名相关
    SIGN_REQUEST: 'sign_request',
    SIGN_INVITE: 'sign_invite',
    SIGN_RESPONSE: 'sign_response',
    SIGN_PARAMS: 'sign_params',
    SIGN_RESULT: 'sign_result',
    SIGN_COMPLETE: 'sign_complete',

    // 错误消息
    ERROR: 'error'
}

// WebSocket连接参数常量
const WS_CONSTANTS = {
    RECONNECT_INTERVAL: 2000,      // 初始重连时间间隔(2秒)
    MAX_RECONNECT_INTERVAL: 30000, // 最大重连时间间隔(30秒)
    RECONNECT_DECAY: 1.5,          // 重连时间递增系数
    MAX_RECONNECT_ATTEMPTS: 10     // 最大重连尝试次数
}

// WebSocket连接状态
let wsConnectionStatus = {
    connected: false,      // 是否已连接
    connecting: false,     // 是否正在连接
    reconnectAttempts: 0   // 重连尝试次数
}

// 初始化WebSocket服务
export function initWebSocketService() {
    const ws = store.state.wsClient

    if (!ws) {
        console.error('WebSocket客户端未初始化')
        return false
    }

    // 防止重复初始化
    if (wsServiceInitialized && ws._messageHandlerInitialized) {
        console.log('WebSocket消息处理已初始化，跳过')
        return true
    }

    console.log('初始化WebSocket消息处理')

    // 处理WebSocket消息
    ws.onmessage = async (event) => {
        try {
            // 尝试解析消息内容
            let message;
            try {
                message = JSON.parse(event.data);
                console.log('收到WebSocket消息:', message);
            } catch (parseError) {
                console.error('WebSocket消息解析失败:', parseError);
                console.error('收到的原始消息数据:', event.data);

                // 尝试通过消息分隔来恢复
                if (typeof event.data === 'string') {
                    // 如果消息看起来包含多个JSON对象，尝试提取第一个有效的JSON
                    const possibleJsonStart = event.data.indexOf('{');
                    if (possibleJsonStart >= 0) {
                        let jsonDepth = 0;
                        let endPos = -1;

                        // 简单扫描找到匹配的JSON结束位置
                        for (let i = possibleJsonStart; i < event.data.length; i++) {
                            if (event.data[i] === '{') jsonDepth++;
                            else if (event.data[i] === '}') {
                                jsonDepth--;
                                if (jsonDepth === 0) {
                                    endPos = i + 1;
                                    break;
                                }
                            }
                        }

                        if (endPos > 0) {
                            const jsonPart = event.data.substring(possibleJsonStart, endPos);
                            try {
                                message = JSON.parse(jsonPart);
                                console.log('从损坏的消息中恢复JSON成功:', message);
                            } catch (e) {
                                console.error('恢复JSON失败:', e);
                                return; // 放弃处理
                            }
                        }
                    }
                }

                // 如果还是无法解析，则放弃处理此消息
                if (!message) {
                    return;
                }
            }

            // 根据消息类型处理
            switch (message.type) {
                case WS_MESSAGE_TYPES.REGISTER_COMPLETE:
                    handleRegisterComplete(message)
                    break

                case WS_MESSAGE_TYPES.KEYGEN_INVITE:
                    // 不再自动弹窗，只添加通知
                    console.log('收到密钥生成邀请:', message)
                    Message.info(`收到来自 ${message.coordinator} 的密钥生成邀请，请在通知页面处理`)
                    break

                case WS_MESSAGE_TYPES.KEYGEN_PARAMS:
                    await handleKeyGenParams(message)
                    break

                case WS_MESSAGE_TYPES.KEYGEN_COMPLETE:
                    handleKeyGenComplete(message)
                    break

                case WS_MESSAGE_TYPES.SIGN_INVITE:
                    // 不再自动弹窗，只添加通知
                    console.log('收到签名邀请:', message)
                    Message.info(`收到签名邀请，地址: ${message.address}，请在通知页面处理`)
                    break

                case WS_MESSAGE_TYPES.SIGN_PARAMS:
                    await handleSignParams(message)
                    break

                case WS_MESSAGE_TYPES.SIGN_COMPLETE:
                    handleSignComplete(message)
                    break

                case WS_MESSAGE_TYPES.ERROR:
                    handleError(message)
                    break

                default:
                    console.warn('未处理的WebSocket消息类型:', message.type)
            }

            // 将消息添加到通知列表
            store.commit('addNotification', {
                type: message.type,
                content: message,
                timestamp: new Date(),
                responded: false // 添加响应状态标识
            })
        } catch (error) {
            console.error('处理WebSocket消息出错:', error)
        }
    }

    // 标记WebSocket实例已初始化消息处理器
    ws._messageHandlerInitialized = true
    wsServiceInitialized = true

    return true
}

// 重置WebSocket消息处理服务
export function resetWebSocketService() {
    wsServiceInitialized = false;

    // 重置连接状态
    wsConnectionStatus = {
        connected: false,
        connecting: false,
        reconnectAttempts: 0
    }
}

// 发送WebSocket消息
export function sendWSMessage(message) {
    const ws = store.state.wsClient

    if (!ws || ws.readyState !== WebSocket.OPEN) {
        console.error('WebSocket未连接')
        return false
    }

    try {
        ws.send(JSON.stringify(message))
        return true
    } catch (error) {
        console.error('发送WebSocket消息出错:', error)
        return false
    }
}

// 处理注册完成消息
function handleRegisterComplete(message) {
    if (message.success) {
        store.commit('setWsConnected', true)
        wsConnectionStatus.connected = true
        wsConnectionStatus.reconnectAttempts = 0
        console.log('WebSocket注册成功')
        Message.success('WebSocket连接成功')
    } else {
        console.error('WebSocket注册失败:', message.message)
        Message.error('WebSocket注册失败: ' + message.message)
    }
}

// 处理密钥生成邀请
async function handleKeyGenInvite(message) {
    try {
        // 显示确认对话框
        const confirm = await MessageBox.confirm(
            `您收到密钥生成邀请，参与者索引: ${message.part_index}, 会话: ${message.session_key}, 发起者: ${message.coordinator}。是否接受?`,
            '密钥生成邀请',
            {
                confirmButtonText: '接受',
                cancelButtonText: '拒绝',
                type: 'info'
            }
        ).catch(() => false)

        // 获取当前用户的CPIC
        const cpicResponse = await seApi.getCPIC()
        const cpic = cpicResponse.data.cpic

        // 发送响应
        sendWSMessage({
            type: WS_MESSAGE_TYPES.KEYGEN_RESPONSE,
            session_key: message.session_key,
            part_index: message.part_index,
            cpic: cpic,
            accept: confirm !== false,
            reason: confirm === false ? '用户拒绝' : ''
        })
    } catch (error) {
        console.error('处理密钥生成邀请出错:', error)
        // 发送拒绝响应
        sendWSMessage({
            type: WS_MESSAGE_TYPES.KEYGEN_RESPONSE,
            session_key: message.session_key,
            part_index: message.part_index,
            cpic: '',
            accept: false,
            reason: '处理邀请出错: ' + error.message
        })
    }
}

// 处理密钥生成参数
async function handleKeyGenParams(message) {
    try {
        const user = store.state.user.username
        // 调用MPC服务进行密钥生成
        const keygenResponse = await mpcApi.keyGen({
            threshold: message.threshold,
            parties: message.total_parts,
            index: message.part_index,
            filename: message.filename,
            username: user
        })

        if (keygenResponse.data.success) {
            // 尝试获取CPIC，即使出错也继续处理
            let cpic = ''
            try {
                const cpicResponse = await seApi.getCPIC()
                cpic = cpicResponse.data.cpic
            } catch (cpicError) {
                console.error('获取CPIC失败:', cpicError)
            }

            // 发送密钥生成结果
            sendWSMessage({
                type: WS_MESSAGE_TYPES.KEYGEN_RESULT,
                session_key: message.session_key,
                part_index: message.part_index,
                address: keygenResponse.data.address,
                cpic: cpic,
                encrypted_shard: keygenResponse.data.encryptedKey,
                success: true,
                message: '密钥生成成功'
            })

            Message.success('密钥生成成功')
        } else {
            throw new Error('MPC服务密钥生成失败')
        }
    } catch (error) {
        console.error('密钥生成失败:', error)
        // 发送失败结果
        sendWSMessage({
            type: WS_MESSAGE_TYPES.KEYGEN_RESULT,
            session_key: message.session_key,
            part_index: message.part_index,
            address: '',
            cpic: '',
            encrypted_shard: '',
            success: false,
            message: '密钥生成失败: ' + error.message
        })

        Message.error('密钥生成失败: ' + error.message)
    }
}

// 处理密钥生成完成消息
function handleKeyGenComplete(message) {
    if (message.success) {
        MessageBox.alert(
            `密钥生成成功! 地址: ${message.address}`,
            '密钥生成完成',
            { type: 'success' }
        )
    } else {
        MessageBox.alert(
            `密钥生成失败: ${message.message}`,
            '密钥生成失败',
            { type: 'error' }
        )
    }
}

// 处理签名邀请
async function handleSignInvite(message) {
    try {
        // 显示确认对话框
        const confirm = await MessageBox.confirm(
            `您收到签名邀请，参与者索引: ${message.part_index}, 会话: ${message.session_key}, 地址: ${message.address}。是否接受?`,
            '签名邀请',
            {
                confirmButtonText: '接受',
                cancelButtonText: '拒绝',
                type: 'info'
            }
        ).catch(() => false)

        // 获取当前用户的CPIC
        const cpicResponse = await seApi.getCPIC()
        const cpic = cpicResponse.data.cpic

        // 发送响应
        sendWSMessage({
            type: WS_MESSAGE_TYPES.SIGN_RESPONSE,
            session_key: message.session_key,
            part_index: message.part_index,
            cpic: cpic,
            accept: confirm !== false,
            reason: confirm === false ? '用户拒绝' : ''
        })
    } catch (error) {
        console.error('处理签名邀请出错:', error)
        // 发送拒绝响应
        sendWSMessage({
            type: WS_MESSAGE_TYPES.SIGN_RESPONSE,
            session_key: message.session_key,
            part_index: message.part_index,
            cpic: '',
            accept: false,
            reason: '处理邀请出错: ' + error.message
        })
    }
}

// 处理签名参数
async function handleSignParams(message) {
    try {
        // 调用MPC签名服务
        const signResponse = await mpcApi.sign({
            parties: message.parties,
            data: message.data,
            filename: message.filename,
            encryptedKey: message.encrypted_shard,
            userName: store.state.user.username,
            address: message.address,
            signature: message.signature
        })

        if (signResponse.data.success) {
            // 发送签名结果
            sendWSMessage({
                type: WS_MESSAGE_TYPES.SIGN_RESULT,
                session_key: message.session_key,
                part_index: message.part_index,
                success: true,
                signature: signResponse.data.signature,
                message: '签名成功'
            })

            Message.success('签名成功')
        } else {
            throw new Error('MPC服务签名失败')
        }
    } catch (error) {
        console.error('签名失败:', error)
        // 发送失败结果
        sendWSMessage({
            type: WS_MESSAGE_TYPES.SIGN_RESULT,
            session_key: message.session_key,
            part_index: message.part_index,
            success: false,
            signature: '',
            message: '签名失败: ' + error.message
        })

        Message.error('签名失败: ' + error.message)
    }
}

// 处理签名完成消息
function handleSignComplete(message) {
    if (message.success) {
        MessageBox.alert(
            `签名成功! 签名结果: ${message.signature}`,
            '签名完成',
            { type: 'success' }
        )
    } else {
        MessageBox.alert(
            `签名失败: ${message.message}`,
            '签名失败',
            { type: 'error' }
        )
    }
}

// 处理错误消息
function handleError(message) {
    console.error('收到错误消息:', message)
    Message.error(`错误: ${message.message}`)
} 