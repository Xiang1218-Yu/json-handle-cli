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
--- FAIL: TestNullDocumentRemainsNullRecord (0.00s)
    bug_002_object_test.go:16: 不支持的JSON格式，首字符: n
FAIL
FAIL	json-handle-cli/internal/jsonstream	0.041s
--- FAIL: TestNestedFilterSkipsNullProfile (0.01s)
panic: nested value cannot be nil [recovered, repanicked]

goroutine 19 [running]:
testing.tRunner.func1.2({0x58fd00, 0x5d9d10})
	/usr/local/go/src/testing/testing.go:1974 +0x232
testing.tRunner.func1()
	/usr/local/go/src/testing/testing.go:1977 +0x349
panic({0x58fd00?, 0x5d9d10?})
	/usr/local/go/src/runtime/panic.go:860 +0x13a
json-handle-cli/internal/processor.getFieldValue({0x59e000, 0x3eb191a18b10}, {0x5cc8c7?, 0x0?})
	/workspace/internal/processor/filter.go:110 +0x11e
json-handle-cli/internal/processor.matchCondition({0x59e000?, 0x3eb191a18b10?}, {0x5cc8c7?, 0x589ec0?}, {0x5c8c55, 0x1}, {0x5cc8d4, 0x5})
	/workspace/internal/processor/filter.go:118 +0x46
json-handle-cli/internal/processor.RunFilter.func1({0x59e000, 0x3eb191a18b10})
	/workspace/internal/processor/filter.go:56 +0x7b
json-handle-cli/internal/jsonstream.processArray(0x3eb191a080a0, 0x3eb191a69d68)
	/workspace/internal/jsonstream/stream.go:126 +0x1c6
json-handle-cli/internal/jsonstream.StreamProcess({0x3eb191a4e0c0?, 0x3c?}, 0x0, 0x3eb191a69d68)
	/workspace/internal/jsonstream/stream.go:99 +0x1a5
json-handle-cli/internal/processor.RunFilter({{0x3eb191a4e0c0, 0x3b}, {0x3eb191a4e100, 0x3c}, {0x5cc8c7, 0x12}, 0x0, {0x5c9086, 0x5}})
	/workspace/internal/processor/filter.go:54 +0x609
json-handle-cli/internal/processor.TestNestedFilterSkipsNullProfile(0x3eb191a54248)
	/workspace/internal/processor/bug_002_filter_test.go:12 +0x331
testing.tRunner(0x3eb191a54248, 0x5d82c0)
	/usr/local/go/src/testing/testing.go:2036 +0xea
created by testing.(*T).Run in goroutine 1
	/usr/local/go/src/testing/testing.go:2101 +0x4c5
FAIL	json-handle-cli/internal/processor	0.045s
FAIL
```

## 期望行为
空文档及缺失嵌套资料应被安全处理，不能中断有效记录处理。
