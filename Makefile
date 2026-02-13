.PHONY: build run clean test install

# 编译
build:
	go build -o dujiao-migrate main.go

# 编译所有平台
build-all:
	GOOS=linux GOARCH=amd64 go build -o dujiao-migrate-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build -o dujiao-migrate-linux-arm64 main.go
	GOOS=darwin GOARCH=amd64 go build -o dujiao-migrate-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o dujiao-migrate-darwin-arm64 main.go
	GOOS=windows GOARCH=amd64 go build -o dujiao-migrate-windows-amd64.exe main.go

# 运行
run:
	go run main.go

# 清理
clean:
	rm -f dujiao-migrate dujiao-migrate-*

# 测试
test:
	go test -v ./...

# 安装依赖
install:
	go mod download
	go mod tidy

# 生成配置文件
config:
	go run main.go --generate-config > config.yaml

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run
