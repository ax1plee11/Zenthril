# Build stage
FROM golang:1.23-alpine AS builder

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
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/server .

# Copy migrations
COPY backend/migrations ./migrations/

# Expose port
EXPOSE 8080

# Run
CMD ["./server"]
