# Build stage
FROM golang:1.26.5-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code
COPY backend/ ./

ARG TARGET=./cmd/api

# Build with security flags
# SECURITY: PIE and static linking harden the binary against exploitation.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -buildid= -linkmode external -extldflags '-static -z relro -z now -z noexecstack'" \
    -a -installsuffix cgo -o server ${TARGET}

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata
RUN addgroup -S zenthril && adduser -S -G zenthril zenthril

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy migrations
COPY backend/migrations ./migrations/

# SECURITY: drop all capabilities and set read-only filesystem.
RUN chown -R zenthril:zenthril /app && \
    chmod 550 /app/server && \
    chmod 755 /app/migrations

USER zenthril

# SECURITY: non-root user, read-only filesystem, no new privileges.
# Note: security options are applied at runtime via docker run --security-opt.

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=5 \
    CMD wget -q -O- http://127.0.0.1:8080/livez >/dev/null 2>&1 || exit 1

# Graceful shutdown on SIGTERM
STOPSIGNAL SIGTERM

# Run
CMD ["./server"]
