ARG TARGETPLATFORM=linux/amd64

FROM --platform=$TARGETPLATFORM golang:1.24-bookworm AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

ENV CGO_ENABLED=1 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH

RUN test "$TARGETOS/$TARGETARCH" = "linux/amd64" || \
    (echo "offline-server Docker image currently supports linux/amd64 only because gg20_sm_manager_linux_amd64 is bundled." && exit 1)

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/offline-server .

FROM --platform=$TARGETPLATFORM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/offline-server /app/offline-server
COPY bin/gg20_sm_manager_linux_amd64 /app/bin/gg20_sm_manager_linux_amd64

RUN chmod 755 /app/offline-server /app/bin/gg20_sm_manager_linux_amd64 \
    && mkdir -p /app/data/offline_tasks /app/logs /app/private_keys

ENV TZ=Asia/Singapore \
    OFFLINE_MANAGER_BIN=/app/bin/gg20_sm_manager_linux_amd64 \
    OFFLINE_MANAGER_BIND_ADDRESS=0.0.0.0 \
    OFFLINE_MANAGER_PUBLIC_HOST=127.0.0.1 \
    OFFLINE_MANAGER_PORT_START=18001 \
    OFFLINE_MANAGER_PORT_END=18100 \
    OFFLINE_MANAGER_LOG_DIR=/app/logs/managers

VOLUME ["/app/data", "/app/logs", "/app/private_keys"]

EXPOSE 8080 8081

ENTRYPOINT ["/app/offline-server"]
CMD ["-web-port", "8080", "-ws-port", "8081", "-ws-host", "0.0.0.0"]
