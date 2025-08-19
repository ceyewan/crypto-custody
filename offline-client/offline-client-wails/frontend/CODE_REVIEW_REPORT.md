# Frontend Code Review 报告 (已修订)

## 概述

本次 Code Review 的目标是分析 `frontend` 目录下的代码，找出在将本地 mpc 服务从 HTTP 调用迁移到 Wails 调用过程中存在的问题，并根据最新反馈进行调整。

**架构说明 (已澄清)**：
- **云端后端**: 通过 WebSocket (`ws.js`) 和 HTTP (`api.js`) 与前端通信，处理核心业务逻辑和协调。
- **本地 MPC 服务**: 通过 Wails 提供，负责执行客户端的密码学操作（`keygen`, `sign`, `delete`）和访问安全芯片（`get-cplc`）。`wails-api.js` 是其接口。

审查发现，用户描述的 bug 源于一个在 Wails 迁移过程中产生的、非常具体的 API 调用错误。同时，也存在一些代码冗余和可维护性问题。

## 发现的主要问题

### 1. 关键的 API 调用错误 (Bug 的直接原因)

- **问题描述**:
    1.  **错误的 API 导入**: `Notifications.vue` 文件在需要调用本地 MPC 服务时，错误地从 `../services/api` (云端后端 HTTP 接口) 导入了 API，而它本应从 `../services/wails-api` (本地 Wails 接口) 导入。
    2.  **调用了不存在的函数**: 代码尝试调用 `seApi.getCPIC()`，但在 `wails-api.js` 中，该功能由 `seApi.getCPLC()` 提供。这个拼写错误导致调用必然失败。
- **影响**:
    - **运行时错误**: 这是导致用户在接受 keygen/sign 邀请时功能失败的直接原因。
    - **迁移不彻底**: 表明 Wails 的迁移工作只完成了一部分，调用端的代码没有同步更新。

### 2. 逻辑冗余与职责划分 (可维护性问题)

- **问题描述**:
    - **死代码**: `ws.js` 中包含 `handleKeyGenInvite` 和 `handleSignInvite` 函数，但它们从未被项目调用。
    - **逻辑位置**: 邀请处理的核心逻辑（包括调用本地 MPC 服务）直接实现在 `Notifications.vue` 组件中。根据您的反馈，这套逻辑在之前使用 HTTP 连接 MPC 服务时工作良好。因此，这并非一个 bug，但将业务逻辑放在视图层是一种潜在的维护风险。
- **影响**:
    - **维护陷阱**: `ws.js` 中的死代码容易误导开发者，以为邀请逻辑在此处理。
    - **可选的优化**: 将逻辑保留在组件中是可行的，但将其提取到服务层（例如 `wails-api.js`）可以使代码职责更清晰，更易于测试和复用。**这应被视为一个可选的优化，而非强制修复**。

### 3. 状态管理 (Vuex) 问题

- **问题描述**:
    1.  **状态更新不规范**: `Notifications.vue` 直接通过 `this.$set` 修改数组中的对象，绕过了 Vuex 的 mutation，这破坏了 Vuex 的单向数据流原则。
    2.  **状态定义混杂**: Vuex store 中包含了 WebSocket 连接管理的内部实现细节（如 `wsConnecting`），与核心业务状态混杂在一起。
- **影响**:
    - **难以调试**: 状态变更的来源不清晰，增加了调试难度。
    - **可维护性差**: 状态管理层职责不清。

### 4. 错误处理和配置问题

- **问题描述**:
    1.  **错误处理不完整**: 在 `Notifications.vue` 的 `handleKeygenInviteAccept` 和 `handleSignInviteAccept` 中，如果对本地 Wails 服务的调用失败（例如 `getCPLC` 失败），错误只会在前端 `console.log` 中打印，但没有通过 WebSocket 通知协调者该用户已拒绝。这会导致协调者的工作流卡死。
    2.  **配置残留**: `vue.config.js` 中包含一个为之前 HTTP 代理设置的 `devServer.proxy`，这在 Wails 环境下是无效的，应被清理。
    3.  **技术债务**: 项目依赖的 Vue 2 已于 2023 年底停止维护 (EOL)。

## 修复建议 (按优先级排序)

### P0: 核心 Bug 修复 (高优先级)

- **目标**: 解决邀请功能无法使用的核心问题。
- **步骤**:
    1.  **修正 API 导入**: 在 [`Notifications.vue`](offline-client-wails/frontend/src/views/Notifications.vue)，找到 `import { userApi, keygenApi, signApi, seApi } from '../services/api'` 这一行，将其修改为从 `../services/wails-api` 导入 `seApi` 和 `mpcApi`。云端 API `userApi` 等的导入保持不变。
    2.  **修正函数调用**: 将所有 `seApi.getCPIC()` 的调用修改为 `seApi.getCPLC()`。
    3.  **完善错误处理**: 在 `handleKeygenInviteAccept` 和 `handleSignInviteAccept` 的 `catch` 块中，增加通过 WebSocket 发送拒绝/失败消息的逻辑，通知协调者。

### P1: 代码清理与可维护性提升 (中优先级)

- **目标**: 提高代码质量，降低未来维护成本。
- **步骤**:
    1.  **删除死代码**: 从 [`ws.js`](offline-client-wails/frontend/src/services/ws.js) 中删除未被使用的 `handleKeyGenInvite` 和 `handleSignInvite` 函数。
    2.  **规范状态更新**: 在 `store/index.js` 中添加一个 mutation (例如 `updateNotificationResponse`)，并在 [`Notifications.vue`](offline-client-wails/frontend/src/views/Notifications.vue) 中使用 `this.$store.commit(...)` 来代替 `this.$set` 更新通知状态。
    3.  **清理配置文件**: 移除 `vue.config.js` 中无效的 `devServer.proxy` 配置。

### P2: 可选的重构与技术债务 (低优先级)

- **目标**: 优化架构，确保项目长期健康。
- **步骤**:
    1.  **可选重构**: 考虑将 [`Notifications.vue`](offline-client-wails/frontend/src/views/Notifications.vue) 中的邀请处理逻辑封装到 `wails-api.js` 的新函数中，以改善职责分离。
    2.  **制定升级计划**: 规划将项目从 Vue 2 升级到 Vue 3，并更新其他过时的依赖。