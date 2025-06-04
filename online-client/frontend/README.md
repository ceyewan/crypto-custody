# Crypto Custody Frontend

在线加密货币托管系统前端界面

## 🚀 快速开始

### 开发环境
```bash
# 安装依赖
npm install

# 启动开发服务器
npm run serve

# 访问 http://localhost:8090
```

### 生产环境构建
```bash
# 构建静态文件
npm run build

# 生成的文件在 dist/ 目录
```

## 🐳 Docker 部署 (推荐)

### 一键构建并推送到 Docker Hub
```bash
# 使用默认配置 (ceyewan/crypto-custody-frontend:latest)
./docker-build-push.sh

# 或指定自定义参数
./docker-build-push.sh your-username your-image-name v1.0.0
```

### 从 Docker Hub 拉取使用
```bash
# 拉取镜像
docker pull ceyewan/crypto-custody-frontend:latest

# 运行容器
docker run -d -p 80:80 --name crypto-custody-frontend ceyewan/crypto-custody-frontend:latest

# 访问 http://localhost
```

### 自定义 API 地址
```bash
docker run -d -p 80:80 \
  -e VUE_APP_API_BASE_URL=https://your-api.com:22221 \
  --name crypto-custody-frontend \
  ceyewan/crypto-custody-frontend:latest
```

## 📋 项目结构

```
src/
├── components/          # 公共组件
├── router/             # 路由配置
├── services/           # API 服务
├── store/              # Vuex 状态管理
├── views/              # 页面组件
│   ├── Login.vue       # 登录页面
│   ├── Register.vue    # 注册页面
│   ├── Dashboard.vue   # 仪表板
│   ├── Users.vue       # 用户管理 (管理员)
│   ├── Accounts.vue    # 账户管理 (警员+)
│   ├── Transactions.vue # 交易管理 (警员+)
│   └── Profile.vue     # 个人资料
└── main.js             # 应用入口
```

## 🔧 技术栈

- **框架**: Vue.js 2.6.14
- **UI库**: Element UI 2.15.14
- **路由**: Vue Router 3.5.1
- **状态管理**: Vuex 3.6.2
- **HTTP客户端**: Axios 0.24.0
- **构建工具**: Vue CLI 5.0.0

## 🌐 功能特性

### 用户角色权限
- **普通用户**: 查看个人资料
- **警员**: 账户管理、交易操作
- **管理员**: 用户管理、全局数据查看

### 核心功能
- 用户认证与授权
- 账户创建与管理
- 交易准备与签名
- 实时数据统计
- 响应式设计

## 🔒 安全特性

- JWT Token 认证
- 角色权限控制
- API 请求拦截
- 自动登录状态检查
- 安全头部配置

## 📊 Docker 镜像信息

- **基础镜像**: nginx:alpine
- **镜像大小**: ~15MB
- **构建方式**: 多阶段构建
- **特性**: SPA 路由支持、Gzip 压缩、安全头部

## 🛠️ 环境配置

### 开发环境
```bash
# .env.development (自动使用)
VUE_APP_API_BASE_URL=http://192.168.192.1:22221
```

### 生产环境
```bash
# .env.production
VUE_APP_API_BASE_URL=https://your-api-domain.com:22221
```

## 📝 部署说明

### SPA 路由问题解决方案

1. **Nginx 配置** (推荐)
   - 使用提供的 `nginx.conf` 配置
   - 自动处理 Vue Router history 模式

2. **Hash 模式**
   - 修改 `src/router/index.js` 中的 `mode: 'hash'`
   - 无需服务器配置

### 生产环境检查清单

- [ ] 修改 API 地址为生产环境
- [ ] 移除开发调试代码
- [ ] 配置 HTTPS (推荐)
- [ ] 设置适当的 CORS 策略
- [ ] 配置 CDN (可选)

## 🚨 故障排除

### 常见问题

1. **路由 404 错误**
   - 确保 Nginx 配置了 SPA 路由重写
   - 或使用 Hash 模式路由

2. **API 连接失败**
   - 检查 `VUE_APP_API_BASE_URL` 环境变量
   - 确认后端服务正常运行

3. **Docker 构建失败**
   - 检查 Docker 是否运行
   - 清理构建缓存: `docker builder prune`

## 📄 License

This project is licensed under the MIT License.
