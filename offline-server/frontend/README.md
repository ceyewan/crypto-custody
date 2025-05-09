# 多方门限签名系统前端

这是一个基于Vue.js的多方门限签名系统前端应用，用于协调者、参与者和管理员之间的密钥生成和签名操作。

## 功能特性

- 用户注册与登录
- 基于角色的权限控制（管理员、协调者、参与者）
- 密钥生成流程
- 交易签名流程
- WebSocket实时通信
- 本地MPC服务集成

## 系统架构

前端应用与三个后端服务交互：

1. **Web服务** (端口8080)：提供HTTP API，如用户认证、会话创建等
2. **WebSocket服务** (端口8081)：提供实时通信，处理密钥生成和签名流程
3. **MPC服务** (端口8088)：提供本地密钥生成和签名计算功能

## 快速开始

### 安装依赖

```bash
cd frontend
npm install
```

### 本地开发

```bash
npm run serve
```

应用将在 http://localhost:8090 运行

### 构建生产版本

```bash
npm run build
```

## 使用方法

1. 系统需要四种角色的用户：
   - 管理员 (admin)
   - 协调者 (coordinator)
   - 参与者 (participant) - 至少需要3个

2. 首先使用管理员账户登录，为其他用户设置正确的角色

3. 使用协调者账户发起密钥生成或签名请求：
   - 设置门限值和总分片数
   - 选择参与者
   - 点击发起请求

4. 参与者将收到请求通知，可以选择接受或拒绝

5. 所有参与者接受后，将自动进行密钥生成或签名计算

6. 协调者将收到操作结果通知

## 注意事项

- 确保后端服务（Web服务、WebSocket服务和MPC服务）已启动
- 后端服务的默认URL已在配置中设置，如有变更请修改相应配置
- 参与者数量必须大于等于门限值 