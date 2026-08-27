# BENZHI_README

基于 Go 实现的湿地修复苗木投放核验工作台 Web 项目，一款后端服务，用于支持湿地修复苗木投放核验工作台的核心业务流程。

## 项目说明
- 项目：benzhi-project-32ce4119-1585-4b65-bd75-c29a1f6c81a6
- 项目用途：用于支持湿地修复苗木投放核验工作台的核心业务流程。
- Go 工具链：`golang:1.22`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-32ce4119-1585-4b65-bd75-c29a1f6c81a6-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-32ce4119-1585-4b65-bd75-c29a1f6c81a6-arm64 linux/arm64
docker run -it benzhi-project-32ce4119-1585-4b65-bd75-c29a1f6c81a6-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -selfcheck -addr=127.0.0.1:19081`
