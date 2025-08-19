# MPC Core - 开发者指南

本指南旨在帮助开发者理解 `mpc_core` 模块的内部工作原理，特别是从前端发起一个请求直到与硬件安全芯片交互的完整流程。

## 核心架构与请求流程

本模块采用分层架构，以确保职责分离和代码的可维护性。一个来自前端的请求会经过以下几个主要层级：

```mermaid
graph TD
    A[外部协调服务器] -->|WebSocket| B[ws.js];
    B --> C[wails-api.js];
    C --> D{Wails Bridge};
    D --> E[app.go (Wails绑定层)];
    E --> F[wails_services.go (适配器层)];
    F --> G[services/*.go (核心业务逻辑层)];
    G --> H[utils/*.go & seclient/*.go];
    H --> I[硬件安全芯片];
```

*   **`ws.js`**: 负责与外部协调服务器进行 WebSocket 通信，接收 MPC 流程的指令。
*   **`wails-api.js`**: **前端的 Wails API 网关**。它封装了所有对 Go 后端的原生调用，供 `ws.js` 使用。
*   **Wails 绑定层 (`app.go`)**: 直接暴露给 Wails 前端的方法。它负责接收前端调用，并将其转发给适配器层。
*   **适配器层 (`wails_services.go`)**: 充当应用层和核心业务逻辑之间的桥梁，将前端请求转换为内部服务调用。
*   **核心业务逻辑层 (`services/*.go`)**: 包含 `MPCService` 和 `SecurityService`，负责编排复杂的业务流程。
*   **辅助/硬件通信层 (`utils/*.go`, `seclient/*.go`)**: 提供原子化的辅助功能和与硬件的底层通信。

---

## API 方法详解

以下是所有通过 Wails 暴露给前端的公开方法的详细流程和参数说明。

### 1. `PerformKeyGeneration`

此方法执行 MPC 密钥生成，并将加密密钥存储到安全芯片中。

**流程**: `外部服务器 -> ws.js -> wails-api.js -> App -> WailsServices -> MPCService -> ... -> SE`

**参数详解**:

| 层级 | 方法/对象 | 输入参数 | 输出参数 (成功时) |
| :--- | :--- | :--- | :--- |
| **前端调用层** | `wails-api.js` | `data: { threshold, parties, user_name, index }` | `Promise<{ data: { address, encrypted_key } }>` |
| **Wails 绑定层** | `(a *App) PerformKeyGeneration` | `req KeyGenerationRequest` | `map[string]interface{}`, `error` |
| **适配器层** | `(ws *WailsServices) PerformKeyGeneration` | `req KeyGenerationRequest` | `map[string]interface{}`, `error` |
| **核心业务逻辑层** | `(s *MPCService) KeyGeneration` | `ctx, threshold, parties, index, filename, userName` | `string` (地址), `[]byte` (加密密钥), `error` |

### 2. `PerformSignMessage`

使用之前生成的密钥分片和安全芯片中的解密密钥来对消息进行签名。

**流程**: `外部服务器 -> ws.js -> wails-api.js -> App -> WailsServices -> MPCService -> ... -> SE`

**参数详解**:

| 层级 | 方法/对象 | 输入参数 | 输出参数 (成功时) |
| :--- | :--- | :--- | :--- |
| **前端调用层** | `wails-api.js` | `data: { parties, message, user_name, address, encrypted_key, signature }` | `Promise<{ data: { signature, message } }>` |
| **Wails 绑定层** | `(a *App) PerformSignMessage` | `req SignMessageRequest` | `map[string]interface{}`, `error` |
| **适配器层** | `(ws *WailsServices) PerformSignMessage` | `req SignMessageRequest` | `map[string]interface{}`, `error` |
| **核心业务逻辑层** | `(s *MPCService) SignMessage` | `ctx, parties, data, filename, userName, address, encryptedKey, signature` | `string` (签名), `error` |

### 3. `PerformDeleteMessage`

从安全芯片中删除一个已存储的密钥记录。

**流程**: `(前端调用) -> wails-api.js -> App -> WailsServices -> SecurityService -> ... -> SE`

**参数详解**:

| 层级 | 方法/对象 | 输入参数 | 输出参数 (成功时) |
| :--- | :--- | :--- | :--- |
| **前端调用层** | `wails-api.js` | `data: { user_name, address, signature }` | `Promise<{ data: result }>` |
| **Wails 绑定层** | `(a *App) PerformDeleteMessage` | `req DeleteMessageRequest` | `error` |
| **适配器层** | `(ws *WailsServices) PerformDeleteMessage` | `req DeleteMessageRequest` | `error` |
| **核心业务逻辑层** | `(s *SecurityService) DeleteData` | `username, addr, signature` | `error` |

### 4. `GetCPLCInfo`

从硬件安全芯片中读取 CPLC (Card Production Life Cycle) 信息。

**流程**: `(前端调用) -> wails-api.js -> App -> WailsServices -> SecurityService -> ... -> SE`

**参数详解**:

| 层级 | 方法/对象 | 输入参数 | 输出参数 (成功时) |
| :--- | :--- | :--- | :--- |
| **前端调用层** | `wails-api.js` | (无) | `Promise<{ data: { cplc_info } }>` |
| **Wails 绑定层** | `(a *App) GetCPLCInfo` | (无) | `map[string]interface{}`, `error` |
| **适配器层** | `(ws *WailsServices) GetCPLCInfo` | (无) | `map[string]interface{}`, `error` |
| **核心业务逻辑层** | `(s *SecurityService) GetCPLC` | (无) | `[]byte` (CPLC 数据), `error` |