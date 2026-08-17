# 修复前故障复现（Docker）

## 项目与标准命令
json-handle-cli 是一个面向 JSON 数组、对象和 JSON Lines 的命令行处理工具，提供过滤、转换、合并和统计功能。在仓库根目录可执行：

```bash
go build ./...
go test ./...
```

本题的稳定触发命令为 Docker 容器中的 `go test ./...`；修复前该命令应失败。

## 环境构建与编译
已在以下两个平台实际构建镜像并在容器内执行编译命令：

```bash
docker build --platform linux/amd64 -f benzhi.Dockerfile -t json-handle-cli-repro:amd64 .
docker run --rm --platform linux/amd64 json-handle-cli-repro:amd64 go build ./...
docker build --platform linux/arm64 -f benzhi.Dockerfile -t json-handle-cli-repro:arm64 .
docker run --rm --platform linux/arm64 json-handle-cli-repro:arm64 go build ./...
```

两个平台的镜像构建和容器内编译均成功；故障通过下一节的测试命令触发。

## 故障触发步骤
在仓库根目录执行：

```bash
docker build --platform linux/amd64 -f benzhi.Dockerfile -t json-handle-cli-repro:amd64 .
docker run --rm --platform linux/amd64 json-handle-cli-repro:amd64 go test ./...
```

## 实际错误输出
```text
?   	json-handle-cli/cmd/cli	[no test files]
--- FAIL: TestJSONLinesMode (0.01s)
    bug_005_mode_test.go:14: mode=1
--- FAIL: TestDetectAndProcessSupportedFormats (0.01s)
    stream_test.go:34: lines.jsonl: count=1 want=2
FAIL
FAIL	json-handle-cli/internal/jsonstream	0.095s
--- FAIL: TestStatsCountsAllRecords (0.02s)
    bug_005_stats_test.go:17: count=1
--- FAIL: TestFilterTransformMergeAndStatsWorkflow (0.01s)
    processor_test.go:71: records = 1; want 3
FAIL
FAIL	json-handle-cli/internal/processor	0.100s
FAIL
```

## 期望行为
JSON Lines 与数组统计应处理全部输入记录，不能只保留首条。
