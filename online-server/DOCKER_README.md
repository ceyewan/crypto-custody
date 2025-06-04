# Docker 部署指南

本项目提供了轻量化的 Docker 镜像构建和部署方案。

## 文件说明

- `Dockerfile`: 多阶段构建配置，生成轻量化镜像
- `docker-build-push.sh`: 自动化构建和推送脚本
- `.dockerignore`: 排除不必要的文件，优化构建速度

## 快速开始

### 1. 本地构建和测试

```bash
# 构建镜像
docker build -t crypto-custody-server:latest .

# 运行容器
docker run -p 8080:8080 crypto-custody-server:latest
```

### 2. 使用自动化脚本推送到 DockerHub

```bash
# 使用默认参数
./docker-build-push.sh your-dockerhub-username

# 指定完整参数
./docker-build-push.sh your-dockerhub-username crypto-custody-server v1.0.0
```

## Docker 镜像特性

### 轻量化设计
- **多阶段构建**: 使用 `golang:1.24-alpine` 作为构建阶段
- **最终镜像**: 基于 `gcr.io/distroless/base-debian12`
- **架构支持**: 专为 ARM64/Apple Silicon 优化
- **镜像大小**: 约 20-30MB (相比普通 Go 镜像节省 90%+ 空间)

### 安全特性
- 使用 Distroless 镜像，无 shell 和包管理器
- 最小化攻击面
- 只包含运行时必需的文件

### 配置说明
- **端口**: 8080
- **工作目录**: `/`
- **入口点**: `/main`

## 环境变量和配置

容器会自动复制以下文件（如果存在）:
- `.env*` 文件
- `database/` 目录
- `logs/` 目录

确保在运行容器前准备好必要的环境配置文件。

## 部署示例

### Docker Compose
```yaml
version: '3.8'
services:
  crypto-custody-server:
    image: your-dockerhub-username/crypto-custody-server:latest
    ports:
      - "8080:8080"
    environment:
      - ENV=production
    volumes:
      - ./data:/data
    restart: unless-stopped
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: crypto-custody-server
spec:
  replicas: 2
  selector:
    matchLabels:
      app: crypto-custody-server
  template:
    metadata:
      labels:
        app: crypto-custody-server
    spec:
      containers:
      - name: server
        image: your-dockerhub-username/crypto-custody-server:latest
        ports:
        - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: crypto-custody-service
spec:
  selector:
    app: crypto-custody-server
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 故障排除

### 构建失败
1. 检查 Go 版本兼容性
2. 确认依赖项可正常下载
3. 检查 CGO 依赖 (如 SQLite)

### 运行时问题
1. 检查环境变量配置
2. 确认数据库文件权限
3. 查看容器日志: `docker logs <container-id>`

### 网络访问
确保防火墙允许 8080 端口访问，或使用端口映射:
```bash
docker run -p 3000:8080 crypto-custody-server:latest
```

## 最佳实践

1. **版本管理**: 使用语义化版本标签
2. **安全扫描**: 定期扫描镜像漏洞
3. **资源限制**: 在生产环境中设置内存和 CPU 限制
4. **健康检查**: 添加应用健康检查端点
5. **日志管理**: 配置适当的日志收集策略

## 性能优化

- 镜像大小: ~20-30MB
- 启动时间: <5秒
- 内存占用: <50MB (空载)
- 支持水平扩展
