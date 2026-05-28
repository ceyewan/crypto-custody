FROM --platform=$TARGETPLATFORM golang:1.24-alpine AS builder

ENV CGO_ENABLED=1

RUN apk add --no-cache git gcc musl-dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags '-w -s' -o /main .

FROM --platform=$TARGETPLATFORM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=builder /main /app/main
RUN mkdir -p /app/database /app/logs \
    && chown -R app:app /app

EXPOSE 8080

USER app

ENTRYPOINT ["/app/main"]
