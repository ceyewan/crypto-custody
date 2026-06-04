# 在线系统单镜像部署

本目录用于构建在线系统 all-in-one 镜像。镜像内包含：

- Vue 前端静态文件。
- Go 在线服务端。
- NGINX，负责前端静态文件和 `/api/` 到容器内后端的反向代理。

容器只暴露一个 HTTP 端口。域名、HTTPS、证书续期和公网访问控制建议由外部 NGINX、Caddy、Ingress 或云负载均衡处理。

## 构建

```bash
./docker-build-image.sh crypto-custody-online-system:local
```

默认平台是 `linux/amd64`，可通过环境变量覆盖：

```bash
DOCKER_PLATFORM=linux/amd64 ./docker-build-image.sh
```

## 本地启动

```bash
./docker-run-local.sh crypto-custody-online-system:local
```

默认访问地址：

```text
http://127.0.0.1:8088
```

## 上传镜像

```bash
./docker-push-image.sh ceyewan/crypto-custody-online-system:latest
```

也可以一键构建并上传：

```bash
./docker-build-push.sh ceyewan/crypto-custody-online-system:latest
```

## 数据目录

Docker Compose 会挂载：

```text
database/ -> /app/database
logs/     -> /app/logs
backups/  -> /app/backups
```

SQLite 数据库和在线端备份文件都保存在宿主机目录中。
