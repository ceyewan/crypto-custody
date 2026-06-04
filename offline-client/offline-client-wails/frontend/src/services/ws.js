import store from '../store'
import { mpcApi, seApi } from './wails-api'
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

    // 销毁相关
    DESTROY_REQUEST: 'destroy_request',
    DESTROY_INVITE: 'destroy_invite',
    DESTROY_RESPONSE: 'destroy_response',
    DESTROY_PARAMS: 'destroy_params',
    DESTROY_RESULT: 'destroy_result',
    DESTROY_COMPLETE: 'destroy_complete',

    // 分片移交相关
    TRANSFER_REQUEST: 'transfer_request',
    TRANSFER_INVITE: 'transfer_invite',
    TRANSFER_RESPONSE: 'transfer_response',
    TRANSFER_COMPLETE: 'transfer_complete',

    // 错误消息
    ERROR: 'error'
}

// WebSocket连接状态
let wsConnectionStatus = {
    connected: false,      // 是否已连接
    connecting: false,     // 是否正在连接
    reconnectAttempts: 0   // 重连尝试次数
}

export function mpcTaskKey(kind, message) {
    return `${kind}:${message.session_key || ''}`
}

function taskStatus(key) {
    return store.state.mpcTasks[key] ? store.state.mpcTasks[key].status : ''
}

function commitTask(key, patch) {
    store.commit('setMpcTask', { key, patch })
}

function isTerminalStatus(status) {
    return status === 'result_sent' || status === 'rejected' || status === 'completed'
}

function markInvited(kind, message, patch) {
    const key = mpcTaskKey(kind, message)
    const status = taskStatus(key)
    if (status && status !== 'invited' && status !== 'interrupted') {
        console.log(`任务 ${key} 当前状态为 ${status}，保留已有状态`)
        return
    }
    commitTask(key, patch)
}

function beginTask(kind, message) {
    const key = mpcTaskKey(kind, message)
    const status = taskStatus(key)

    if (status === 'result_ready') {
        resendPendingResult(key)
        return { key, started: false }
    }
    if (status === 'running' || isTerminalStatus(status)) {
        console.log(`任务 ${key} 当前状态为 ${status}，跳过重复执行`)
        return { key, started: false }
    }

    commitTask(key, {
        kind,
        session_key: message.session_key,
        party_index: message.party_index,
        signing_index: message.signing_index,
        status: 'running',
        phase: `${kind}_params`,
        message: '协同任务执行中'
    })
    return { key, started: true }
}

function sendTaskResult(key, resultMessage, successMessage, failureMessage) {
    if (sendWSMessage(resultMessage)) {
        commitTask(key, {
            status: 'result_sent',
            phase: resultMessage.type,
            success: !!resultMessage.success,
            result_message: null,
            message: successMessage || resultMessage.message || '结果已回传'
        })
        return true
    }

    commitTask(key, {
        status: 'result_ready',
        phase: resultMessage.type,
        success: !!resultMessage.success,
        result_message: resultMessage,
        message: failureMessage || '结果已生成，但服务连接未建立，等待重连后回传'
    })
    Message.warning(failureMessage || '结果已生成，但服务连接未建立，等待重连后回传')
    return false
}

function resendPendingResult(key) {
    const task = store.state.mpcTasks[key]
    if (!task || task.status !== 'result_ready' || !task.result_message) {
        return false
    }
    return sendTaskResult(key, task.result_message, '缓存结果已重新回传', '缓存结果回传失败，等待下次重连')
}

function resendPendingResults() {
    Object.keys(store.state.mpcTasks).forEach(key => {
        resendPendingResult(key)
    })
}

function errorMessage(error) {
    return error && error.message ? error.message : String(error)
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
                    handleKeyGenInvite(message)
                    break

                case WS_MESSAGE_TYPES.KEYGEN_PARAMS:
                    await handleKeyGenParams(message)
                    break

                case WS_MESSAGE_TYPES.KEYGEN_COMPLETE:
                    handleKeyGenComplete(message)
                    break

                case WS_MESSAGE_TYPES.SIGN_INVITE:
                    handleSignInvite(message)
                    break

                case WS_MESSAGE_TYPES.SIGN_PARAMS:
                    await handleSignParams(message)
                    break

                case WS_MESSAGE_TYPES.SIGN_COMPLETE:
                    handleSignComplete(message)
                    break

                case WS_MESSAGE_TYPES.DESTROY_INVITE:
                    handleDestroyInvite(message)
                    break

                case WS_MESSAGE_TYPES.DESTROY_PARAMS:
                    await handleDestroyParams(message)
                    break

                case WS_MESSAGE_TYPES.DESTROY_COMPLETE:
                    handleDestroyComplete(message)
                    break

                case WS_MESSAGE_TYPES.TRANSFER_INVITE:
                    handleTransferInvite(message)
                    break

                case WS_MESSAGE_TYPES.TRANSFER_COMPLETE:
                    handleTransferComplete(message)
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
        resendPendingResults()
    } else {
        console.error('WebSocket注册失败:', message.message)
        Message.error('WebSocket注册失败: ' + message.message)
    }
}

function handleKeyGenInvite(message) {
    console.log('收到私钥生成邀请:', message)
    markInvited('keygen', message, {
        kind: 'keygen',
        session_key: message.session_key,
        task_no: message.task_no,
        case_no: message.case_no,
        party_index: message.party_index,
        status: 'invited',
        phase: WS_MESSAGE_TYPES.KEYGEN_INVITE,
        message: '等待用户确认私钥生成邀请'
    })
    Message.info(`收到来自 ${message.initiator || '管理员'} 的私钥生成邀请，请在通知页面处理`)
}

function handleSignInvite(message) {
    console.log('收到签名邀请:', message)
    markInvited('sign', message, {
        kind: 'sign',
        session_key: message.session_key,
        task_no: message.task_no,
        case_no: message.case_no,
        address: message.address,
        party_index: message.party_index,
        status: 'invited',
        phase: WS_MESSAGE_TYPES.SIGN_INVITE,
        message: '等待用户确认签名邀请'
    })
    Message.info(`收到签名邀请，地址: ${message.address}，请在通知页面处理`)
}

function handleDestroyInvite(message) {
    console.log('收到私钥销毁邀请:', message)
    markInvited('destroy', message, {
        kind: 'destroy',
        session_key: message.session_key,
        case_no: message.case_no,
        address: message.address,
        party_index: message.party_index,
        status: 'invited',
        phase: WS_MESSAGE_TYPES.DESTROY_INVITE,
        message: '等待用户确认私钥销毁邀请'
    })
    Message.warning(`收到私钥销毁邀请，地址: ${message.address}，请在通知页面处理`)
}

function handleTransferInvite(message) {
    console.log('收到私钥分片移交邀请:', message)
    markInvited('transfer', message, {
        kind: 'transfer',
        session_key: message.session_key,
        shard_id: message.shard_id,
        address: message.address,
        case_no: message.case_no,
        status: 'invited',
        phase: WS_MESSAGE_TYPES.TRANSFER_INVITE,
        message: '等待用户确认私钥分片移交邀请'
    })
    Message.warning(`收到私钥分片移交邀请，地址: ${message.address}，请在通知页面处理`)
}


// 处理密钥生成参数
async function handleKeyGenParams(message) {
    const task = beginTask('keygen', message)
    if (!task.started) {
        return
    }

    try {
        // 调用MPC服务进行密钥生成，使用与 models 匹配的字段名
        const keygenResponse = await mpcApi.keyGen({
            manager_addr: message.manager_addr,
            room: message.room,
            threshold: message.threshold,
            parties: message.total_parties,
            party_index: message.party_index,
            record_id: message.record_id,
            filename: message.filename || "keygen_temp.json"
        })

        if (keygenResponse.data.success) {
            let cplc = (store.state.mpcTasks[task.key] || {}).cplc || ''
            if (!cplc) {
                try {
                    const cplcResponse = await seApi.getCPLC()
                    cplc = cplcResponse.data.cplc_info || ''
                } catch (cplcError) {
                    throw new Error('获取CPLC失败: ' + errorMessage(cplcError))
                }
            }
            if (!cplc) {
                throw new Error('获取CPLC失败: 返回为空')
            }

            // 发送密钥生成结果
            const resultMessage = {
                type: WS_MESSAGE_TYPES.KEYGEN_RESULT,
                session_key: message.session_key,
                party_index: message.party_index,
                address: keygenResponse.data.address,
                public_key: keygenResponse.data.public_key,
                cplc: cplc,
                record_id: message.record_id,
                encrypted_shard: keygenResponse.data.encrypted_shard,
                success: true,
                message: '私钥生成成功'
            }
            sendTaskResult(task.key, resultMessage, '私钥生成成功，结果已回传')

            Message.success('私钥生成成功')
        } else {
            throw new Error('私钥生成失败')
        }
    } catch (error) {
        console.error('私钥生成失败:', error)
        // 发送失败结果
        const resultMessage = {
            type: WS_MESSAGE_TYPES.KEYGEN_RESULT,
            session_key: message.session_key,
            party_index: message.party_index,
            address: '',
            public_key: '',
            cplc: '',
            record_id: message.record_id,
            encrypted_shard: '',
            success: false,
            message: '私钥生成失败: ' + errorMessage(error)
        }
        sendTaskResult(task.key, resultMessage, '私钥生成失败，错误已回传')

        Message.error('私钥生成失败: ' + errorMessage(error))
    }
}

// 处理私钥生成完成消息
function handleKeyGenComplete(message) {
    commitTask(mpcTaskKey('keygen', message), {
        status: 'completed',
        phase: WS_MESSAGE_TYPES.KEYGEN_COMPLETE,
        success: !!message.success,
        message: message.message || (message.success ? '私钥生成完成' : '私钥生成失败')
    })
    if (message.success) {
        MessageBox.alert(
            `私钥生成成功，托管地址: ${message.address}`,
            '私钥生成完成',
            { type: 'success' }
        )
    } else {
        MessageBox.alert(
            `私钥生成失败: ${message.message}`,
            '私钥生成失败',
            { type: 'error' }
        )
    }
}


// 处理签名参数
async function handleSignParams(message) {
    const task = beginTask('sign', message)
    if (!task.started) {
        return
    }

    try {
        // 调用MPC签名服务，使用与 models 匹配的字段名
        const signResponse = await mpcApi.sign({
            manager_addr: message.manager_addr,
            room: message.room,
            parties: message.parties,
            signing_index: message.signing_index,
            message_hash: message.message_hash,
            filename: message.filename || "sign_temp.json",
            record_id: message.record_id,
            address: message.address,
            encrypted_shard: message.encrypted_shard,
            signature: message.signature
        })

        if (signResponse.data.success) {
            // 发送签名结果
            const resultMessage = {
                type: WS_MESSAGE_TYPES.SIGN_RESULT,
                session_key: message.session_key,
                signing_index: message.signing_index,
                success: true,
                signature: signResponse.data.signature,
                message: '签名成功'
            }
            sendTaskResult(task.key, resultMessage, '签名成功，结果已回传')

            Message.success('签名成功')
        } else {
            throw new Error('MPC服务签名失败')
        }
    } catch (error) {
        console.error('签名失败:', error)
        // 发送失败结果
        const resultMessage = {
            type: WS_MESSAGE_TYPES.SIGN_RESULT,
            session_key: message.session_key,
            signing_index: message.signing_index,
            success: false,
            signature: '',
            message: '签名失败: ' + errorMessage(error)
        }
        sendTaskResult(task.key, resultMessage, '签名失败，错误已回传')

        Message.error('签名失败: ' + errorMessage(error))
    }
}

// 处理签名完成消息
function handleSignComplete(message) {
    commitTask(mpcTaskKey('sign', message), {
        status: 'completed',
        phase: WS_MESSAGE_TYPES.SIGN_COMPLETE,
        success: !!message.success,
        message: message.message || (message.success ? '签名完成' : '签名失败')
    })
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

// 处理私钥销毁参数
async function handleDestroyParams(message) {
    const task = beginTask('destroy', message)
    if (!task.started) {
        return
    }

    try {
        await mpcApi.delete({
            record_id: message.record_id,
            address: message.address,
            signature: message.signature
        })

        const resultMessage = {
            type: WS_MESSAGE_TYPES.DESTROY_RESULT,
            session_key: message.session_key,
            party_index: message.party_index,
            success: true,
            message: '安全芯片记录已删除并验证不可读取'
        }
        sendTaskResult(task.key, resultMessage, '私钥销毁成功，结果已回传')

        Message.success('私钥销毁成功')
    } catch (error) {
        const resultMessage = {
            type: WS_MESSAGE_TYPES.DESTROY_RESULT,
            session_key: message.session_key,
            party_index: message.party_index,
            success: false,
            message: '私钥销毁失败: ' + errorMessage(error)
        }
        sendTaskResult(task.key, resultMessage, '私钥销毁失败，错误已回传')

        Message.error('私钥销毁失败: ' + errorMessage(error))
    }
}

// 处理私钥销毁完成消息
function handleDestroyComplete(message) {
    commitTask(mpcTaskKey('destroy', message), {
        status: 'completed',
        phase: WS_MESSAGE_TYPES.DESTROY_COMPLETE,
        success: !!message.success,
        message: message.message || (message.success ? '私钥销毁完成' : '私钥销毁失败')
    })
    if (message.success) {
        MessageBox.alert(
            `私钥销毁完成，已销毁私钥分片数: ${message.destroyed}`,
            '私钥销毁完成',
            { type: 'success' }
        )
    } else {
        MessageBox.alert(
            `私钥销毁失败: ${message.message}`,
            '私钥销毁失败',
            { type: 'error' }
        )
    }
}

function handleTransferComplete(message) {
    commitTask(mpcTaskKey('transfer', message), {
        status: 'completed',
        phase: WS_MESSAGE_TYPES.TRANSFER_COMPLETE,
        success: !!message.success,
        message: message.message || (message.success ? '私钥分片移交完成' : '私钥分片移交失败')
    })
    if (message.success) {
        Message.success(message.message || '私钥分片移交完成')
    } else {
        Message.error(message.message || '私钥分片移交失败')
    }
}

// 处理错误消息
function handleError(message) {
    console.error('收到错误消息:', message)
    Message.error(`错误: ${message.message}`)
} 
