FROM golang:1.26.5-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGET=.
RUN CGO_ENABLED=0 GOOS=linux go build -o zenthril ${TARGET}

FROM alpine:3.19
WORKDIR /app

# ca-certificates: required for TLS connections to external services
RUN apk --no-cache add ca-certificates && \
    addgroup -S zenthril && \
    adduser -S -G zenthril zenthril

COPY --from=builder /app/zenthril .

# Copy pre-built frontend static files (optional — only present when
# the image is built with the client dist folder available).
# Set STATIC_DIR=/app/static in the container to serve the SPA.
COPY --from=builder /app/static ./static

RUN chown -R zenthril:zenthril /app
USER zenthril
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8080/livez || exit 1

ENV STATIC_DIR=/app/static
CMD ["./zenthril"]
