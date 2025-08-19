// Wails API 服务 - 用于调用内置的 web-se 后端模块
import { PerformKeyGeneration, PerformSignMessage, GetCPLCInfo, PerformDeleteMessage } from '../../wailsjs/go/main/App'

// 安全芯片 API - 直接调用 Wails 内置的 mpc_core 模块
export const seApi = {
    // 获取 CPLC - 调用内置 mpc_core
    getCPLC() {
        return GetCPLCInfo().then(result => ({ data: result }))
    }
}

// MPC 服务 API - 调用内置的 mpc_core 模块执行实际的密码学操作
export const mpcApi = {
    // 密钥生成 - 调用内置 mpc_core
    keyGen(data) {
        console.log('调用 Wails PerformKeyGeneration:', data)
        // Wails 会自动处理 Go struct 和 JS object 之间的转换
        return PerformKeyGeneration(data).then(result => {
            console.log('Wails 密钥生成结果:', result)
            return { data: result }
        })
    },

    // 签名 - 调用内置 mpc_core
    sign(data) {
        console.log('调用 Wails PerformSignMessage:', data)
        return PerformSignMessage(data).then(result => {
            console.log('Wails 签名结果:', result)
            return { data: result }
        })
    },

    // 删除消息 - 调用内置 mpc_core
    delete(data) {
        console.log('调用 Wails PerformDeleteMessage:', data)
        return PerformDeleteMessage(data).then(result => ({ data: result }))
    }
}

// 导出本地 MPC API（用于替代原来对外部 web-se 服务的调用）
export default {
    seApi,
    mpcApi
}
