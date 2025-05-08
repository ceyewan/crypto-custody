import store from '../store'
import { mpcApi } from './api'
import { MessageBox, Message } from 'element-ui'

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

// 初始化WebSocket服务
export function initWebSocketService() {
    const ws = store.state.wsClient

    if (!ws) {
        console.error('WebSocket客户端未初始化')
        return
    }

    // 处理WebSocket消息
    ws.onmessage = async (event) => {
        try {
            const message = JSON.parse(event.data)
            console.log('收到WebSocket消息:', message)

            // 根据消息类型处理
            switch (message.type) {
                case WS_MESSAGE_TYPES.REGISTER_COMPLETE:
                    handleRegisterComplete(message)
                    break

                case WS_MESSAGE_TYPES.KEYGEN_INVITE:
                    await handleKeyGenInvite(message)
                    break

                case WS_MESSAGE_TYPES.KEYGEN_PARAMS:
                    await handleKeyGenParams(message)
                    break

                case WS_MESSAGE_TYPES.KEYGEN_COMPLETE:
                    handleKeyGenComplete(message)
                    break

                case WS_MESSAGE_TYPES.SIGN_INVITE:
                    await handleSignInvite(message)
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
                timestamp: new Date()
            })
        } catch (error) {
            console.error('处理WebSocket消息出错:', error)
        }
    }
}

// 发送WebSocket消息
export function sendWSMessage(message) {
    const ws = store.state.wsClient

    if (!ws || ws.readyState !== WebSocket.OPEN) {
        console.error('WebSocket未连接')
        return false
    }

    ws.send(JSON.stringify(message))
    return true
}

// 处理注册完成消息
function handleRegisterComplete(message) {
    if (message.success) {
        store.commit('setWsConnected', true)
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
        const cpicResponse = await mpcApi.getCPIC()
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
            // 发送密钥生成结果
            sendWSMessage({
                type: WS_MESSAGE_TYPES.KEYGEN_RESULT,
                session_key: message.session_key,
                part_index: message.part_index,
                address: keygenResponse.data.address,
                cpic: keygenResponse.data.cpic || '',
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
        const cpicResponse = await mpcApi.getCPIC()
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