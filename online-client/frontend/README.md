# 在线系统前端

该前端是在线加密资产托管系统的 Web 界面，负责用户认证、账户管理、交易管理、统计展示和个人资料维护。

## 核心功能

- 用户登录、注册和登录态检查。
- 基于角色的页面访问控制。
- 管理员用户管理。
- 账户查询、创建、导入和删除。
- 交易准备、签名提交、交易详情、交易列表和统计。
- 个人资料查看和密码修改。

## 技术栈

- Vue.js 2.6.14
- Element UI 2.15.14
- Vue Router 3.5.1
- Vuex 3.6.2
- Axios 0.24.0
- Vue CLI 5.0.0

## 快速开始

### 开发环境

```bash
npm install
npm run serve
```

默认访问地址：

```text
http://localhost:8090
```

### 生产环境构建

```bash
npm run build
```

生成的静态文件位于 `dist/`。

## 容器运行

构建镜像：

```bash
docker build -t crypto-custody-frontend .
```

运行容器：

```bash
docker run -d -p 80:80 --name crypto-custody-frontend crypto-custody-frontend
```

自定义 API 地址：

```bash
docker run -d -p 80:80 \
  -e VUE_APP_API_BASE_URL=https://your-api.example.com:22221 \
  --name crypto-custody-frontend \
  crypto-custody-frontend
```

## 目录结构

```text
src/
├── components/          # 公共组件
├── router/              # 路由配置
├── services/            # API 服务封装
├── store/               # Vuex 状态管理
├── views/               # 页面组件
│   ├── Login.vue        # 登录页面
│   ├── Register.vue     # 注册页面
│   ├── Dashboard.vue    # 仪表板
│   ├── Users.vue        # 用户管理
│   ├── Accounts.vue     # 账户管理
│   ├── Transactions.vue # 交易管理
│   └── Profile.vue      # 个人资料
└── main.js              # 应用入口
```

## 环境配置

开发环境：

```bash
VUE_APP_API_BASE_URL=http://192.168.192.1:22221
```

生产环境：

```bash
VUE_APP_API_BASE_URL=https://your-api.example.com:22221
```

## 部署说明

### SPA 路由

使用 `nginx.conf` 可以处理 Vue Router history 模式下的刷新和直接访问问题。

如需避免服务器重写配置，也可以将 `src/router/index.js` 中的路由模式改为 `hash`。

### 生产环境检查清单

- [ ] 修改 API 地址为生产环境
- [ ] 移除开发调试代码
- [ ] 配置 HTTPS
- [ ] 设置适当的 CORS 策略
- [ ] 检查 Nginx 静态资源缓存策略

## 故障排除

### 路由 404

确认 Nginx 已配置 SPA 路由重写，或改用 Hash 路由模式。

### API 连接失败

检查 `VUE_APP_API_BASE_URL` 是否正确，并确认在线服务端可访问。

### 容器构建失败

确认容器运行环境可用，必要时清理构建缓存：

```bash
docker builder prune
```
