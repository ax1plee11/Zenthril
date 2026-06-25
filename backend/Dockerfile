FROM golang:1.26.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGET=./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o zenthril ${TARGET}

FROM alpine:3.19
WORKDIR /app

# ca-certificates: required for TLS connections to external services
# (Redis TLS, Postgres TLS, OTLP exporters, S3-compatible stores).
RUN apk --no-cache add ca-certificates && \
    addgroup -S zenthril && \
    adduser -S -G zenthril zenthril

COPY --from=builder /app/zenthril .
RUN chown -R zenthril:zenthril /app
USER zenthril
EXPOSE 8080

# Health check: /livez is a minimal unauthenticated liveness probe. Protected
# readiness/operational endpoints (/healthz, /readyz) still require bearer
# tokens in production.
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8080/livez || exit 1

CMD ["./zenthril"]
