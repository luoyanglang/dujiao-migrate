# 多阶段构建
FROM golang:1.21 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o dujiao-migrate main.go

# 运行镜像
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/dujiao-migrate /usr/local/bin/dujiao-migrate

ENTRYPOINT ["dujiao-migrate"]
