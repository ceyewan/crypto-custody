# 在线加密货币托管系统 - 前端

## 项目简介

这是在线加密货币托管系统的前端界面，基于 Vue.js 2.x 开发，使用 Element UI 组件库，提供用户友好的 Web 界面来管理加密货币账户和交易。

## 功能特性

### 用户管理
- 用户注册和登录
- JWT Token 认证
- 角色权限控制（管理员、警员、访客）
- 密码修改功能

### 账户管理
- 创建和导入加密货币账户
- 批量导入账户功能
- 账户余额查询
- 账户信息查看和管理

### 交易管理
- 发起转账交易
- 交易签名和发送
- 交易状态跟踪
- 交易历史记录

### 权限系统
- **管理员**: 管理所有用户、账户和交易
- **警员**: 管理自己的账户和交易
- **访客**: 只能查看个人信息

## 技术栈

- **框架**: Vue.js 2.6.x
- **UI组件库**: Element UI 2.15.x
- **状态管理**: Vuex 3.6.x
- **路由**: Vue Router 3.5.x
- **HTTP客户端**: Axios
- **构建工具**: Vue CLI 5.x

## 项目结构

```
frontend/
├── public/                 # 静态文件
├── src/
│   ├── components/         # 公共组件
│   ├── views/             # 页面组件
│   │   ├── Login.vue      # 登录页面
│   │   ├── Register.vue   # 注册页面
│   │   ├── Dashboard.vue  # 仪表板
│   │   ├── Users.vue      # 用户管理
│   │   ├── Accounts.vue   # 账户管理
│   │   ├── Transactions.vue # 交易管理
│   │   └── Profile.vue    # 个人资料
│   ├── router/            # 路由配置
│   ├── store/             # Vuex状态管理
│   ├── services/          # API服务
│   │   └── api.js         # API接口封装
│   ├── App.vue           # 根组件
│   └── main.js           # 入口文件
├── package.json          # 项目配置
├── vue.config.js         # Vue配置
└── README.md            # 项目文档
```

## 安装和运行

### 环境要求

- Node.js >= 14.x
- npm >= 6.x 或 yarn >= 1.22.x

### 安装依赖

```bash
cd frontend
npm install
# 或者
yarn install
```

### 开发环境运行

```bash
npm run serve
# 或者
yarn serve
```

默认访问地址: http://localhost:8090

### 生产环境构建

```bash
npm run build
# 或者
yarn build
```

## API 配置

前端通过 axios 与后端 API 通信，API 基础地址配置：

- 开发环境: `http://localhost:8080`
- 生产环境: 根据实际部署调整

### API 接口

#### 用户相关
- `POST /api/login` - 用户登录
- `POST /api/register` - 用户注册
- `GET /api/users/profile` - 获取用户信息
- `POST /api/users/change-password` - 修改密码

#### 账户相关
- `GET /api/accounts/officer/` - 获取用户账户列表
- `POST /api/accounts/officer/create` - 创建账户
- `POST /api/accounts/officer/import` - 批量导入账户
- `GET /api/accounts/address/{address}` - 根据地址查询账户

#### 交易相关
- `GET /api/transaction/balance/{address}` - 获取账户余额
- `POST /api/transaction/tx/prepare` - 准备交易
- `POST /api/transaction/tx/sign-send` - 签名并发送交易

## 权限说明

### 路由权限
- 公开路由: 登录、注册页面
- 需要认证: 仪表板、个人资料
- 需要警员权限: 账户管理、交易管理
- 需要管理员权限: 用户管理

### 功能权限
- **管理员**: 所有功能权限
- **警员**: 账户和交易管理权限
- **访客**: 仅个人信息查看权限

## 开发指南

### 添加新页面

1. 在 `src/views/` 创建新的 Vue 组件
2. 在 `src/router/index.js` 添加路由配置
3. 在仪表板菜单中添加导航链接

### API 调用示例

```javascript
import { userApi } from '@/services/api'

// 登录
const response = await userApi.login({
  username: 'admin',
  password: 'password'
})

// 获取账户列表
const accounts = await accountApi.getUserAccounts()
```

### 状态管理

使用 Vuex 管理全局状态:

```javascript
// 获取当前用户
this.$store.getters.currentUser

// 检查权限
this.$store.getters.isAdmin
this.$store.getters.isOfficer

// 更新用户信息
this.$store.dispatch('login', userData)
```

## 样式指南

- 使用 Element UI 组件的默认样式
- 自定义样式使用 scoped CSS
- 响应式设计适配移动端

## 部署说明

### Nginx 配置示例

```nginx
server {
    listen 80;
    server_name your-domain.com;
    root /path/to/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 注意事项

1. **Token 安全**: JWT Token 存储在 localStorage，请确保 HTTPS 部署
2. **CORS**: 开发环境已配置代理，生产环境需要后端配置 CORS
3. **错误处理**: 已集成全局错误处理和用户友好的错误提示
4. **权限检查**: 前端权限检查仅用于 UI 显示，真正的权限控制在后端

## 常见问题

### 1. 登录后Token失效
检查后端 JWT 配置和过期时间设置

### 2. API 请求失败
检查后端服务是否正常运行，确认 API 地址配置正确

### 3. 路由权限问题
确认用户角色和路由权限配置匹配

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交变更
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License