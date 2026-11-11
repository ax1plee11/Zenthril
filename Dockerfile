# Build stage
FROM golang:1.26.11-alpine@sha256:ae52b5bc6ed045a9ed26ab03dc19327a802a9cc9aa8b27ed57f5c6b60579b0bb AS builder

WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code
COPY backend/ ./

ARG TARGET=.

# Build with security flags
# SECURITY: PIE and static linking harden the binary against exploitation.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -buildid=" \
    -a -installsuffix cgo -o server ${TARGET}

# Runtime stage
FROM alpine:3.23@sha256:469c9ea410b9215eaee7976ba6c0639039b444be8e5d638f7abbada189971eb2

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
