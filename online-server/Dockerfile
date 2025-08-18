# 阶段 1: 构建器
# 使用基于 Alpine Linux 的特定 Go 版本作为构建器镜像。
FROM golang:1.22-alpine AS builder

# 设置构建所需的必要环境变量
# GOOS=linux 和 GOARCH=amd64 是目标平台
# CGO_ENABLED=1 是 sqlite3 驱动所必需的
ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=1

# 安装 CGO 构建依赖。
# git 用于从仓库获取 Go 模块。
# gcc 和 musl-dev 用于 CGO 支持。
RUN apk add --no-cache git gcc musl-dev

# 设置容器内的工作目录
WORKDIR /app

# 首先复制 go.mod 和 go.sum 以下载依赖
# 这可以利用 Docker 的层缓存机制。
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码
COPY . .

# 将 Go 应用程序构建为静态二进制文件。
# -ldflags '-w -s' 去除调试信息，减小二进制文件大小。
# -a 安装构建所依赖的包。
# -o 指定输出文件名。
RUN go build -ldflags '-w -s' -a -o /main .

# 阶段 2: 最终镜像
# 为最终应用程序使用一个轻量级的、非 root 用户的基本镜像。
# distroless/static 是静态 Go 二进制文件的理想选择。
FROM gcr.io/distroless/static-debian12

# 从构建器阶段复制编译好的二进制文件。
COPY --from=builder /main /main

# 暴露应用程序将运行的端口。
EXPOSE 8080

# 设置为非 root 用户以提高安全性。
USER nonroot:nonroot

# 定义容器的入口点。
ENTRYPOINT ["/main"]
