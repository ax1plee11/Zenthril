# Zenthril backend production Dockerfile
# Multi-stage build, non-root user, minimal runtime image

FROM golang:1.26.11-alpine@sha256:ae52b5bc6ed045a9ed26ab03dc19327a802a9cc9aa8b27ed57f5c6b60579b0bb AS builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN mkdir -p /app/static
ARG TARGET=.
RUN CGO_ENABLED=0 GOOS=linux go build -o zenthril ${TARGET}

FROM alpine:3.23@sha256:469c9ea410b9215eaee7976ba6c0639039b444be8e5d638f7abbada189971eb2
WORKDIR /app

# ca-certificates: required for TLS connections to external services
RUN apk --no-cache add ca-certificates && \
    addgroup -S zenthril && \
    adduser -S -G zenthril zenthril

COPY --from=builder /app/zenthril .

# Copy pre-built frontend static files (optional)
COPY --from=builder /app/static ./static

RUN chown -R zenthril:zenthril /app
USER zenthril
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8080/livez || exit 1

ENV STATIC_DIR=/app/static
CMD ["./zenthril"]
