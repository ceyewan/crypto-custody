# Multi-stage build for lightweight image
FROM --platform=linux/arm64 golang:1.24-alpine AS builder

# Install build dependencies for CGO
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO support
RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -a -ldflags '-linkmode external -extldflags "-static"' -o main .

# Final stage: lightweight distroless image for ARM64
FROM --platform=linux/arm64 gcr.io/distroless/base-debian12

# Set working directory
WORKDIR /

# Copy the binary from builder stage
COPY --from=builder /app/main /main

# Copy necessary files that the application might need
COPY --from=builder /app/.env* ./
COPY --from=builder /app/database ./database/
COPY --from=builder /app/logs ./logs/

# Expose port 8080
EXPOSE 8080

# Run the application
ENTRYPOINT ["/main"]
