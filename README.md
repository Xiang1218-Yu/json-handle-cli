# json-handle-cli

> 用 Go 开发的高性能大型 JSON 文件命令行处理工具，面向数据预处理与 ETL 场景。

基于流式（streaming）解析与写入，支持 GB 级 JSON 文件而不会 OOM，零第三方依赖，单二进制分发。

---

## ✨ 特性

- **流式处理**：JSON 数组 / JSON 对象 / JSON Lines（每行一条）三种输入格式自动识别；边读边写不加载全量数据
- **四大操作**：`filter` 过滤、`transform` 转换、`merge` 合并、`stats` 统计，覆盖常见 ETL 需求
- **嵌套路径**：所有支持字段的子命令均支持 `a.b.c` 点路径访问嵌套对象
- **内存友好**：过滤/转换/合并且均采用流式写入器，输出数组或 JSON Lines 可选
- **零依赖**：纯 Go 标准库实现，`go build` 即可得到静态编译的单文件可执行程序

---

## 🚀 安装 / 构建

```bash
# 需要 Go 1.18+
go build -o json-handle-cli ./cmd/cli

# 或直接运行
go run ./cmd/cli --help
```

构建成功后将二进制放到 `PATH` 中即可。

---

## 📦 支持的输入格式

自动识别以下三种，无需手动指定：

| 格式        | 说明                                          |
| ----------- | --------------------------------------------- |
| JSON 数组   | 顶层是 `[ {...}, {...} ]`，最常见             |
| JSON 对象   | 顶层是单个 `{...}`，会被当作 1 条记录处理     |
| JSON Lines  | 每行一个 JSON，常用于日志 / Hadoop 生态导出   |

输出格式可通过 `-f array`（默认）或 `-f lines`（JSON Lines）手动指定。

---

## 🛠 命令参考

### 全局

```
json-handle-cli <子命令> [选项]

子命令:
  filter      按条件过滤JSON元素
  transform   批量转换/清洗字段（增删改/类型转换/大小写）
  merge       合并多个JSON文件（拼接/去重/对象合并）
  stats       统计字段分布与数值特征
  help        显示帮助
  version     显示版本号
```

每个子命令均支持 `-h` / `--help` 查看详细用法。

---

### 1. `filter` — 按条件过滤

```
json-handle-cli filter -i <input> -o <output> -e "<表达式>" [选项]
```

**比较操作符**：

| 操作符 | 含义                     | 示例                       |
| ------ | ------------------------ | -------------------------- |
| `=` / `==` | 等于              | `status=active`            |
| `!=`   | 不等于                   | `country!=CN`              |
| `>` `>=` `<` `<=` | 数值 / 字典序比较 | `age>=18`, `price<99.9`    |
| `~`    | 包含（字符串子串匹配）   | `tags~go`, `name~A`        |

**常用选项**：

| 参数             | 说明                                           |
| ---------------- | ---------------------------------------------- |
| `-i / --input`   | 输入文件                                       |
| `-o / --output`  | 输出文件                                       |
| `-e / --expr`    | 过滤表达式（必填）                             |
| `-v / --invert`  | 反转：保留**不匹配**的记录                     |
| `-f / --format`  | `array`（默认）或 `lines`                      |

**示例**：

```bash
# 保留 age>=18 的成年人
json-handle-cli filter -i users.json -o adults.json -e "age>=18"

# 嵌套路径：只保留北京的记录
json-handle-cli filter -i users.json -o beijing.json -e "address.city=Beijing"

# 剔除 status=deleted（反转匹配）
json-handle-cli filter -i data.json -o valid.json -e "status=deleted" -v

# 输出为 JSON Lines 方便下游并行处理
json-handle-cli filter -i big.json -o chunk.jsonl -e "level~error" -f lines
```

---

### 2. `transform` — 批量字段转换 / 清洗

```
json-handle-cli transform -i <input> -o <output> -r "<规则>" [-r "<规则>" ...]
```

规则格式：`action:field[=value]`，可多次指定 `-r`，按顺序执行。

**支持的 action**：

| Action                       | 作用                                              |
| ---------------------------- | ------------------------------------------------- |
| `delete:field`               | 删除字段（别名 `del` / `remove`）                 |
| `rename:field=newName`       | 重命名字段（别名 `mv`），可移动到嵌套路径         |
| `set:field=value`            | 新增 / 覆盖字段值（别名 `add`）                   |
| `type:field=int\|float\|string\|bool` | 强制类型转换（别名 `cast`）             |
| `upper:field`                | 字符串转大写（别名 `uppercase`）                  |
| `lower:field`                | 字符串转小写（别名 `lowercase`）                  |
| `trim:field`                 | 去除字符串首尾空白                                |

`set` 的 value 会自动识别类型：`true/false` → bool，`123` / `3.14` → number，`null` → null，其余为字符串。

**示例**：

```bash
json-handle-cli transform -i raw.json -o clean.json \
    -r "delete:password" \
    -r "rename:city=location" \
    -r "set:source=etl_batch_01" \
    -r "type:id=string" \
    -r "type:vip=bool" \
    -r "upper:status"
```

---

### 3. `merge` — 合并多个 JSON 文件

```
json-handle-cli merge -i <a.json> -i <b.json> [-i ...] -o <output> [选项]
```

**合并策略 `-s / --strategy`**：

| 策略          | 说明                                                         |
| ------------- | ------------------------------------------------------------ |
| `concat`      | （默认）直接拼接所有输入中的每一条记录                       |
| `deduplicate` | 去重：需配合 `-k key_field` 指定主键，相同 key 仅保留首次出现 |
| `union`       | 顶层按键合并对象（后者覆盖前者），或按 `-k` 分组合并数组元素 |

**示例**：

```bash
# 1) 拼接两批数据（同一结构）
json-handle-cli merge -i jan.json -i feb.json -o h1.json

# 2) 按 user_id 去重，首次出现优先
json-handle-cli merge -i batch1.json -i batch2.json -o users.json \
    -s deduplicate -k user_id

# 3) 合并两个配置 JSON 对象（后者覆盖前者）
json-handle-cli merge -i base.json -i override.json -o config.json -s union
```

---

### 4. `stats` — 字段统计与数据概览

```
json-handle-cli stats -i <input> [选项]
```

**输出内容**：

- 总记录数、字段总数
- 每个字段：出现次数、空值数、类型分布（字符串 / 数值 / 布尔 / 对象 / 数组）、唯一值数
- 数值字段：最小 / 最大 / 和 / 平均
- 字符串字段：最小 / 最大（字典序）
- 所有字段：Top-N 高频值

**选项**：

| 参数         | 说明                                           |
| ------------ | ---------------------------------------------- |
| `-f/--field` | 仅统计指定字段（可多次指定），默认统计全部顶层字段 |
| `-top N`     | 每个字段展示的高频值数量，默认 5               |
| `-deep`      | 递归深入嵌套对象和数组                         |
| `--json`     | 以 JSON 格式输出（便于下游程序消费）           |

**示例**：

```bash
# 概览全表
json-handle-cli stats -i users.json

# 深入统计嵌套字段并输出 JSON
json-handle-cli stats -i orders.json -deep --json > report.json

# 仅关注核心指标
json-handle-cli stats -i sales.json -f amount -f region -top 10
```

---

## 🔗 ETL 流水线示例

多个操作可以通过文件链或管道组合使用。以下是一个典型的日批处理脚本：

```bash
#!/usr/bin/env bash
set -euo pipefail

BIN=./json-handle-cli
RAW=/data/raw_dump.jsonl
CLEAN=/data/stage_clean.json
RESULT=/data/result

# 1) 统计原始数据用于日报
$BIN stats -i $RAW -json -top 3 > /reports/raw_stats.json

# 2) 过滤：只保留正式地区（排除测试区）
$BIN filter -i $RAW -f lines -o /tmp/step1.json -e "region!=TEST"

# 3) 转换：脱敏 + 字段规整
$BIN transform -i /tmp/step1.json -o /tmp/step2.json \
    -r "del:phone" \
    -r "del:id_card" \
    -r "set:ingested_at=$(date +%Y-%m-%dT%H:%M:%SZ)" \
    -r "type:user_id=int"

# 4) 合并维度表
$BIN merge -i /tmp/step2.json -i /data/dim_user_tags.json \
    -s deduplicate -k user_id -o $CLEAN

# 5) 结果统计
$BIN stats -i $CLEAN > /reports/result_summary.txt
```

---

## 🧠 设计要点

1. **流式解析**：`encoding/json.Decoder.Token()` + 递归定界符匹配，数组文件一次只在内存中保留单个元素。
2. **格式自动探测**：通过文件首个非空字符加后续 token 区分数组 / 对象 / JSON Lines。
3. **嵌套路径一致性**：`getFieldValue` / `setField` / `getParentAndKey` 统一实现 `a.b.c` 路径语义。
4. **数值比较回退策略**：过滤操作优先按数值比较，任一侧无法转数值时自动回退为字符串字典序。
5. **流式写入器**：`StreamArrayWriter` 处理逗号和方括号，让下游操作无需关心边界。

---

## 📂 项目结构

```
json-handle-cli/
├── cmd/cli/main.go                   # CLI 入口，子命令分发
├── internal/processor/
│   ├── stream.go                     # 流式读写核心（自动格式识别）
│   ├── filter.go                     # filter 子命令实现
│   ├── transform.go                  # transform 子命令实现
│   ├── merge.go                      # merge 子命令实现
│   └── stats.go                      # stats 子命令实现
├── go.mod
├── go.sum
└── README.md
```

---

## 🛡 License

MIT
