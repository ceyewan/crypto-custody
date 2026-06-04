ARG TARGETPLATFORM=linux/amd64
ARG BUILDPLATFORM

FROM --platform=$BUILDPLATFORM node:18-alpine AS frontend-builder

WORKDIR /src/online-client/frontend

COPY online-client/frontend/package*.json ./
RUN npm ci --legacy-peer-deps && npm cache clean --force

COPY online-client/frontend/ ./
ARG VUE_APP_API_BASE_URL=
ENV VUE_APP_API_BASE_URL=$VUE_APP_API_BASE_URL
RUN npm run build

FROM --platform=$TARGETPLATFORM golang:1.24-alpine AS backend-builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

ENV CGO_ENABLED=1 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH

RUN apk add --no-cache git gcc musl-dev

WORKDIR /src/online-server

COPY online-server/go.mod online-server/go.sum ./
RUN go mod download

COPY online-server/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/online-server .

FROM --platform=$TARGETPLATFORM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata nginx gettext curl \
    && mkdir -p /app/database /app/logs /app/backups /run/nginx /usr/share/nginx/html

WORKDIR /app

COPY --from=backend-builder /out/online-server /app/online-server
COPY --from=frontend-builder /src/online-client/frontend/dist/ /usr/share/nginx/html/
COPY deploy/online-system/nginx.conf.template /etc/nginx/http.d/default.conf.template
COPY deploy/online-system/entrypoint.sh /app/entrypoint.sh

RUN chmod 755 /app/online-server /app/entrypoint.sh

ENV TZ=Asia/Singapore \
    WEB_PORT=8088 \
    BACKEND_BIND_ADDR=127.0.0.1 \
    BACKEND_PORT=22221 \
    DATABASE_PATH=/app/database/crypto-custody.db

VOLUME ["/app/database", "/app/logs", "/app/backups"]

EXPOSE 8088

ENTRYPOINT ["/app/entrypoint.sh"]
