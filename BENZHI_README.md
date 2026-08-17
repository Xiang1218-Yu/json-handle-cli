# JSON Handle CLI Docker 交付说明

JSON Handle CLI 是一个面向数据预处理与 ETL 的本地命令行工具，可对 JSON 数组、单对象和 JSON Lines 输入执行过滤、字段转换、合并和统计。项目不依赖外部服务，容器用于在固定 Go 环境中从源码构建和运行该工具。

## 本地构建与使用

在仓库根目录执行：

```bash
go build ./...
go run ./cmd/cli help
go test ./...
```

例如可使用随仓库提供的示例文件查看字段统计：

```bash
go run ./cmd/cli stats -i test_users.json --json
```

## 固定的构建环境

实际 Dockerfile 为 `benzhi.Dockerfile`。`go.mod` 固定 Go 语言版本为 `1.26.5`（本项目只使用标准库，因此无需依赖校验文件），镜像使用 `golang:1.26.5-bookworm`，并设置 `GOTOOLCHAIN=local`，避免构建过程中自动切换工具链。镜像会复制源码、执行 `go mod download`，随后在镜像构建阶段执行 `go build ./...`；不会复制宿主机编译出的二进制。

## 双架构标准验收

仓库中的 `build_benzhi_docker.sh` 会依次为 `linux/amd64` 和 `linux/arm64` 构建镜像 `json-handle-cli-benzhi:<架构>`，随后分别在容器中执行 `go build ./...`，并启动默认命令 `go run ./cmd/cli help`。

```bash
./build_benzhi_docker.sh
```

也可以按下面步骤手动验证单个平台：

```bash
docker build --platform linux/amd64 -f benzhi.Dockerfile -t json-handle-cli-benzhi:amd64 .
docker run --rm --platform linux/amd64 json-handle-cli-benzhi:amd64 go build ./...
docker run --rm --platform linux/amd64 json-handle-cli-benzhi:amd64
docker run --rm --platform linux/amd64 json-handle-cli-benzhi:amd64 go test ./...
```

将 `linux/amd64` 与镜像标签中的 `amd64` 分别替换为 `linux/arm64` 和 `arm64`，即可验证另一种标准架构。

## 通过标准

本地或容器内的 `go build ./...` 与 `go test ./...` 必须以退出码 0 结束。容器默认启动后应输出 CLI 帮助文本并以退出码 0 结束；这表示工具可以在固定环境中从当前源码构建并正常启动。
