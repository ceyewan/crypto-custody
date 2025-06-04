# 多阶段构建 - 使用 Nginx 静态文件服务器 (最小镜像大小 ~15MB)
# 第一阶段：构建应用
FROM node:18-alpine AS build-stage

WORKDIR /app

# 复制 package 文件并安装依赖
COPY package*.json ./
RUN npm ci --legacy-peer-deps && npm cache clean --force

# 复制源代码并构建
COPY . .
RUN npm run build

# 第二阶段：Nginx 静态文件服务器
FROM nginx:alpine AS production-stage

# 复制自定义 nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 从构建阶段复制静态文件
COPY --from=build-stage /app/dist /usr/share/nginx/html

# 暴露端口
EXPOSE 80

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1

# 启动 Nginx
CMD ["nginx", "-g", "daemon off;"]
