# Wails 网络连接问题解决方案文档

## 概述

本文档记录了 Wails 离线客户端应用中网络连接问题的解决过程，包括当前的临时解决方案和未来的优化建议。

## 问题描述

### 原始问题
- **开发环境** (`wails dev`): 网络连接正常工作
- **生产环境** (`wails build`): 出现 "network error"，无法连接远程服务器
- **目标服务器**: `https://crypto-custody-offline-server.ceyewan.icu`
- **WebSocket**: `wss://crypto-custody-offline-server.ceyewan.icu/ws`

### 错误信息
```
Refused to connect to https://crypto-custody-offline-server.ceyewan.icu/user/login 
because it appears in neither the connect-src directive nor the default-src directive 
of the Content Security Policy.
```

## 根本原因分析

### 1. CSP (Content Security Policy) 限制
- **位置**: `frontend/public/index.html` 第7行
- **原始策略**: 缺少 `connect-src` 指令，仅允许本地资源连接
- **影响**: 阻止了所有外部 HTTPS/WSS 连接

### 2. 环境检测问题
- **问题**: 简单的 `window.location.protocol` 检测在 Wails 环境中不可靠
- **原因**: Wails 应用使用 `wails://` 协议，导致环境判断失效

## 当前解决方案 (临时性/取巧方式)

### 1. CSP 策略修改
**文件**: `frontend/public/index.html`

```html
<!-- 修改前 -->
<meta http-equiv="Content-Security-Policy" content="default-src 'self' file: data: blob: 'unsafe-inline' 'unsafe-eval'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline';">

<!-- 修改后 -->
<meta http-equiv="Content-Security-Policy" content="default-src 'self' file: data: blob: 'unsafe-inline' 'unsafe-eval'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' https://crypto-custody-offline-server.ceyewan.icu wss://crypto-custody-offline-server.ceyewan.icu;">
```

**问题**: 
- 硬编码了外部域名，缺乏灵活性
- 安全策略过于宽松
- 不利于多环境部署

### 2. 环境检测优化
**文件**: `frontend/src/services/api.js`

```javascript
// 使用 Wails Runtime API 进行环境检测
async function detectEnvironmentAndSetURL() {
    try {
        const envInfo = await Environment();
        isWailsEnvironment = true;
        API_URL = 'https://crypto-custody-offline-server.ceyewan.icu';
        apiClient.defaults.baseURL = API_URL;
    } catch (error) {
        isWailsEnvironment = false;
        API_URL = '/api';
    }
}
```

**问题**: 
- 仍然硬编码了生产环境 URL
- 缺乏配置文件管理
- 没有区分不同的部署环境

## 未来优化方案

### 1. 架构重构：Go 后端代理方案 (推荐)

**优势**:
- 完全避免 CSP 问题
- 更好的安全控制
- 符合离线客户端架构理念
- 支持请求缓存和离线处理

**实现步骤**:
```go
// 在 Go 后端添加代理 API
type App struct {
    httpClient *http.Client
    config     *Config
}

func (a *App) ProxyAPIRequest(method, path string, data interface{}) (interface{}, error) {
    // 代理 HTTP 请求到远程服务器
    url := a.config.RemoteServerURL + path
    // ... 实现代理逻辑
}

func (a *App) ProxyWebSocket() error {
    // 代理 WebSocket 连接
    // ... 实现 WebSocket 代理
}
```

**前端调用**:
```javascript
// 通过 Wails bindings 调用 Go 方法
import { ProxyAPIRequest } from '../wailsjs/go/main/App';

const response = await ProxyAPIRequest('POST', '/user/login', loginData);
```

### 2. 动态 CSP 配置方案

**实现方案**:
```javascript
// 根据构建环境动态生成 CSP
const buildCSP = () => {
    const isDev = process.env.NODE_ENV === 'development';
    const isWails = window.__WAILS__;
    
    let connectSrc = "'self'";
    
    if (isWails) {
        connectSrc += ` ${process.env.VUE_APP_API_URL} ${process.env.VUE_APP_WS_URL}`;
    } else if (isDev) {
        connectSrc += " http://localhost:*";
    }
    
    return `default-src 'self' file: data: blob: 'unsafe-inline' 'unsafe-eval'; connect-src ${connectSrc};`;
};
```

### 3. 配置文件管理方案

**目录结构**:
```
config/
├── development.json
├── production.json
└── staging.json
```

**配置文件示例**:
```json
{
  "apiConfig": {
    "baseURL": "https://api.example.com",
    "wsURL": "wss://ws.example.com",
    "timeout": 30000
  },
  "security": {
    "allowedDomains": [
      "https://crypto-custody-offline-server.ceyewan.icu"
    ],
    "cspPolicy": "restrictive"
  }
}
```

### 4. 环境检测增强方案

```javascript
class EnvironmentDetector {
    constructor() {
        this.isWails = false;
        this.environment = 'development';
        this.config = null;
    }
    
    async detect() {
        try {
            // 检测 Wails 环境
            const envInfo = await Environment();
            this.isWails = true;
            this.environment = envInfo.buildType || 'production';
        } catch (error) {
            this.isWails = false;
            this.environment = process.env.NODE_ENV || 'development';
        }
        
        // 加载对应环境配置
        this.config = await this.loadConfig(this.environment);
        return this.config;
    }
    
    async loadConfig(env) {
        // 动态加载配置文件
        const configModule = await import(`@/config/${env}.json`);
        return configModule.default;
    }
}
```

## 安全考虑

### 当前方案的安全风险
1. **CSP 策略过于宽松**: 允许任意的 `unsafe-inline` 和 `unsafe-eval`
2. **硬编码域名**: 无法灵活应对域名变更
3. **缺乏请求验证**: 直接暴露外部 API 调用

### 推荐安全增强
1. **最小权限原则**: 仅允许必要的网络连接
2. **请求代理**: 通过 Go 后端统一处理外部请求
3. **证书验证**: 确保 HTTPS 连接的安全性
4. **请求签名**: 对关键 API 请求进行签名验证

## 迁移计划

### 阶段一：配置文件重构 (1-2 天)
- [ ] 创建环境配置文件
- [ ] 实现动态配置加载
- [ ] 移除硬编码 URL

### 阶段二：Go 后端代理 (3-5 天)
- [ ] 设计代理 API 接口
- [ ] 实现 HTTP 请求代理
- [ ] 实现 WebSocket 代理
- [ ] 前端调用迁移

### 阶段三：安全增强 (2-3 天)
- [ ] 优化 CSP 策略
- [ ] 添加请求验证
- [ ] 实现证书管理

### 阶段四：测试和部署 (1-2 天)
- [ ] 全面功能测试
- [ ] 性能测试
- [ ] 生产环境验证

## 注意事项

1. **向后兼容性**: 确保迁移过程中不影响现有功能
2. **性能影响**: Go 代理可能引入额外延迟，需要优化
3. **错误处理**: 代理层需要完善的错误处理和重试机制
4. **日志记录**: 添加详细的网络请求日志便于调试

## 总结

当前的临时解决方案虽然能够解决问题，但存在安全性和可维护性的隐患。建议在时间允许的情况下，按照上述优化方案进行重构，以获得更好的架构设计和安全保障。

---
**文档创建时间**: 2025-08-20  
**最后更新**: 2025-08-20  
**状态**: 临时解决方案已实施，等待优化重构