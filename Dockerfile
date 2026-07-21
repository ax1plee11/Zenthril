# Build stage
FROM golang:1.26.5-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code
COPY backend/ ./

ARG TARGET=./cmd/api

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ${TARGET}

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata
RUN addgroup -S zenthril && adduser -S -G zenthril zenthril

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy migrations
COPY backend/migrations ./migrations/

RUN chown -R zenthril:zenthril /app
USER zenthril

# Expose port
EXPOSE 8080

# Run
CMD ["./server"]
