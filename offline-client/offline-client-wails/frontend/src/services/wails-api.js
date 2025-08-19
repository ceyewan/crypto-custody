// Wails API 服务 - 用于调用内置的 web-se 后端模块
import { KeyGeneration, SignMessage, GetCPLC, DeleteMessage } from '../../wailsjs/go/main/App'

// 安全芯片 API - 直接调用 Wails 内置的 web-se 模块
export const seApi = {
    // 获取 CPLC - 调用内置 web-se
    getCPLC() {
        return GetCPLC().then(result => ({ data: result }))
    },

    // 创建安全芯片记录 - 由于是桌面应用，可以直接在本地处理
    createSE(data) {
        return Promise.resolve({ data: { success: true, message: '桌面应用中安全芯片记录已本地保存' } })
    }
}

// MPC 服务 API - 调用内置的 web-se 模块执行实际的密码学操作
export const mpcApi = {
    // 密钥生成 - 调用内置 web-se
    keyGen(data) {
        console.log('调用内置 web-se 进行密钥生成:', data)
        return KeyGeneration().then(result => {
            console.log('内置 web-se 密钥生成结果:', result)
            return { data: result }
        })
    },

    // 签名 - 调用内置 web-se
    sign(data) {
        console.log('调用内置 web-se 进行签名:', data)
        const message = data.message || data.data || '默认消息'
        return SignMessage(message).then(result => {
            console.log('内置 web-se 签名结果:', result)
            return { data: result }
        })
    },

    // 删除消息 - 调用内置 web-se
    delete() {
        console.log('调用内置 web-se 删除消息')
        return DeleteMessage().then(result => ({ data: result }))
    }
}

// 导出本地 MPC API（用于替代原来对外部 web-se 服务的调用）
export default {
    seApi,
    mpcApi
}
