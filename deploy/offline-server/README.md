# Offline Server Deployment

这个目录是离线服务端的独立部署目录。服务器上只需要保留本目录、`.env`、`private_keys/ec_private_key.pem`、`data/` 和 `logs/`，不需要把整个仓库作为部署目录。

## 文件

- `docker-compose.yml`：离线服务端容器编排。
- `.env.example`：部署环境变量模板。
- `deploy.sh`：一键拉取镜像并启动服务。
- `push-image.sh`：在源码仓库里构建并推送 `offline-server` 镜像。
- `private_keys/`：运行时挂载的服务端 ECDSA 私钥目录，不提交私钥。
- `data/`：SQLite 数据目录，运行时自动创建。
- `logs/`：服务和 manager 日志目录，运行时自动创建。

## 构建并推送镜像

在源码仓库中执行：

```bash
cd deploy/offline-server
./push-image.sh ceyewan crypto-custody-offline-server latest
```

当前镜像只支持 `linux/amd64`，因为镜像内置的是 `gg20_sm_manager_linux_amd64`。

## 服务器部署

```bash
cd deploy/offline-server
cp .env.example .env
```

编辑 `.env`，至少把：

```text
OFFLINE_MANAGER_PUBLIC_HOST=192.168.1.10
```

改成桌面端可访问的离线服务器 IP 或域名。

然后放入私钥：

```bash
mkdir -p private_keys
# 将与 SE applet 公钥匹配的私钥放到这里
chmod 600 private_keys/ec_private_key.pem
```

启动：

```bash
./deploy.sh
```

也可以临时覆盖公网/内网访问地址：

```bash
OFFLINE_MANAGER_PUBLIC_HOST=192.168.1.10 ./deploy.sh
```

## 端口

需要在服务器防火墙和安全组放行：

```text
8080              HTTP API
8081              WebSocket
18001-18100       会话级 gg20 manager 端口范围
```

## 运维

```bash
docker compose logs -f offline-server
docker compose ps
docker compose down
```
