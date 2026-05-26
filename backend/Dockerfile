FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGET=./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o zenthril ${TARGET}

FROM alpine:3.19
WORKDIR /app
RUN addgroup -S zenthril && adduser -S -G zenthril zenthril
COPY --from=builder /app/zenthril .
RUN chown -R zenthril:zenthril /app
USER zenthril
EXPOSE 8080
CMD ["./zenthril"]
