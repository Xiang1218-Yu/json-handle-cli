package main

import (
	"flag"
	"fmt"
	"json-handle-cli/internal/processor"
	"os"
	"strings"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printGlobalUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "filter":
		err = runFilter(args)
	case "transform", "tf":
		err = runTransform(args)
	case "merge", "mg":
		err = runMerge(args)
	case "stats", "st":
		err = runStats(args)
	case "help", "--help", "-h", "-?":
		printGlobalUsage()
	case "version", "--version", "-v":
		fmt.Printf("json-handle-cli v%s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n\n", cmd)
		printGlobalUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(2)
	}
}

func printGlobalUsage() {
	fmt.Printf(`json-handle-cli v%s - 大型JSON文件处理命令行工具
用于数据预处理和ETL任务，支持流式处理避免内存溢出。

用法:
  json-handle-cli <子命令> [选项]

子命令:
  filter      按条件过滤JSON元素（支持嵌套字段路径）
  transform   批量转换/清洗字段（增删改/类型转换/大小写）
  merge       合并多个JSON文件（拼接/去重/对象合并）
  stats       统计字段分布、类型、极值、Top值等
  help        显示此帮助
  version     显示版本号

查看子命令详情:
  json-handle-cli filter --help
  json-handle-cli transform --help
  json-handle-cli merge --help
  json-handle-cli stats --help
`, version)
}

// ========================= filter =========================
func runFilter(args []string) error {
	fs := flag.NewFlagSet("filter", flag.ExitOnError)
	var opts processor.FilterOptions
	var help bool
	fs.StringVar(&opts.Input, "i", "", "输入JSON文件路径（必填）")
	fs.StringVar(&opts.Input, "input", "", "输入JSON文件路径（必填）")
	fs.StringVar(&opts.Output, "o", "", "输出JSON文件路径（必填）")
	fs.StringVar(&opts.Output, "output", "", "输出JSON文件路径（必填）")
	fs.StringVar(&opts.Expr, "e", "", "过滤表达式，如: age>=18, name=John, status!=active, tags~go")
	fs.StringVar(&opts.Expr, "expr", "", "过滤表达式")
	fs.BoolVar(&opts.Invert, "v", false, "反转匹配，保留不匹配的记录")
	fs.BoolVar(&opts.Invert, "invert", false, "反转匹配")
	fs.StringVar(&opts.OutputFmt, "f", "array", "输出格式: array(数组) 或 lines(JSON Lines)")
	fs.StringVar(&opts.OutputFmt, "format", "array", "输出格式")
	fs.BoolVar(&help, "h", false, "显示帮助")
	fs.BoolVar(&help, "help", false, "显示帮助")
	_ = fs.Parse(args)

	if help {
		fmt.Println(`filter - 按条件过滤JSON元素

用法:
  json-handle-cli filter -i input.json -o output.json -e "age>=18"

支持的比较操作符:
  =  ==  等于（字符串或数值）
  !=     不等于
  >  >=  大于 / 大于等于
  <  <=  小于 / 小于等于
  ~      包含（字符串包含匹配）

字段路径支持嵌套:
  address.city=Beijing   匹配嵌套对象字段
  user.profile.age>30    多层嵌套

示例:
  json-handle-cli filter -i users.json -o adults.json -e "age>=18"
  json-handle-cli filter -i data.json -o china.json -e "country=CN"
  json-handle-cli filter -i logs.json -o err.logs -e "level~error" -f lines
  json-handle-cli filter -i items.json -o filtered.json -e "price<=99.9" -v`)
		return nil
	}
	if opts.Input == "" || opts.Output == "" || opts.Expr == "" {
		return fmt.Errorf("缺少必要参数，请使用 --help 查看用法")
	}

	total, kept, err := processor.RunFilter(opts)
	if err != nil {
		return err
	}
	fmt.Printf("过滤完成: 共 %d 条，保留 %d 条，剔除 %d 条，输出至: %s\n",
		total, kept, total-kept, opts.Output)
	return nil
}

// ========================= transform =========================
func runTransform(args []string) error {
	fs := flag.NewFlagSet("transform", flag.ExitOnError)
	var opts processor.TransformOptions
	var rulesFlag []string
	var help bool
	fs.StringVar(&opts.Input, "i", "", "输入JSON文件路径（必填）")
	fs.StringVar(&opts.Input, "input", "", "输入JSON文件路径（必填）")
	fs.StringVar(&opts.Output, "o", "", "输出JSON文件路径（必填）")
	fs.StringVar(&opts.Output, "output", "", "输出JSON文件路径（必填）")
	fs.Var(stringSlice(&rulesFlag), "r", "转换规则，可重复指定")
	fs.Var(stringSlice(&rulesFlag), "rule", "转换规则，可重复指定")
	fs.StringVar(&opts.OutputFmt, "f", "array", "输出格式: array 或 lines")
	fs.StringVar(&opts.OutputFmt, "format", "array", "输出格式")
	fs.BoolVar(&help, "h", false, "显示帮助")
	fs.BoolVar(&help, "help", false, "显示帮助")
	_ = fs.Parse(args)

	if help {
		fmt.Println(`transform - 批量转换/清洗字段

用法:
  json-handle-cli transform -i input.json -o output.json \
      -r "delete:password" \
      -r "rename:oldName=newName" \
      -r "set:source=imported" \
      -r "type:age=int" \
      -r "upper:status"

规则格式: action:field[=value]

支持的 action:
  delete|del|remove:field          删除字段
  rename|mv:field=newName          重命名字段
  set|add:field=value              设置/新增字段值
  type|cast:field=int|float|string|bool
                                   转换字段类型
  upper|uppercase:field            字符串转大写
  lower|lowercase:field            字符串转小写
  trim:field                       去除首尾空白

示例:
  json-handle-cli transform -i raw.json -o clean.json \
      -r "del:password" \
      -r "type:id=int" \
      -r "set:processed=true"`)
		return nil
	}
	if opts.Input == "" || opts.Output == "" || len(rulesFlag) == 0 {
		return fmt.Errorf("缺少必要参数，请使用 --help 查看用法")
	}

	for _, rStr := range rulesFlag {
		r, err := processor.ParseTransformRule(rStr)
		if err != nil {
			return err
		}
		opts.Rules = append(opts.Rules, r)
	}

	count, err := processor.RunTransform(opts)
	if err != nil {
		return err
	}
	fmt.Printf("转换完成: 处理 %d 条记录，输出至: %s\n", count, opts.Output)
	return nil
}

// ========================= merge =========================
func runMerge(args []string) error {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	var opts processor.MergeOptions
	var inputsFlag []string
	var help bool
	fs.Var(stringSlice(&inputsFlag), "i", "输入文件，可多次指定（至少2个）")
	fs.Var(stringSlice(&inputsFlag), "input", "输入文件，可多次指定")
	fs.StringVar(&opts.Output, "o", "", "输出JSON文件路径（必填）")
	fs.StringVar(&opts.Output, "output", "", "输出JSON文件路径（必填）")
	fs.StringVar(&opts.Strategy, "s", "concat", "合并策略: concat(拼接) | deduplicate(去重) | union(合并对象)")
	fs.StringVar(&opts.Strategy, "strategy", "concat", "合并策略")
	fs.StringVar(&opts.KeyField, "k", "", "主键字段路径（用于去重和分组合并）")
	fs.StringVar(&opts.KeyField, "key", "", "主键字段路径")
	fs.StringVar(&opts.OutputFmt, "f", "array", "输出格式: array 或 lines")
	fs.StringVar(&opts.OutputFmt, "format", "array", "输出格式")
	fs.BoolVar(&help, "h", false, "显示帮助")
	fs.BoolVar(&help, "help", false, "显示帮助")
	_ = fs.Parse(args)

	if help {
		fmt.Println(`merge - 合并多个JSON文件

用法:
  json-handle-cli merge -i a.json -i b.json -o merged.json

策略 (-s/--strategy):
  concat        简单拼接所有输入元素（默认）
  deduplicate   按指定主键字段去重（需搭配 -k 指定key）
  union         顶层对象按键合并，或按主键分组合并数组元素

示例:
  # 简单拼接两个JSON数组
  json-handle-cli merge -i jan.json -i feb.json -o h1.json

  # 按用户ID去重
  json-handle-cli merge -i batch1.json -i batch2.json -o users.json \
      -s deduplicate -k user_id

  # 合并两个配置对象（后者覆盖前者同名key）
  json-handle-cli merge -i base.json -i override.json -o config.json -s union`)
		return nil
	}
	if opts.Output == "" || len(inputsFlag) < 2 {
		return fmt.Errorf("缺少必要参数，至少需要2个输入文件，请使用 --help 查看用法")
	}
	opts.Inputs = inputsFlag

	totalRead, totalWritten, err := processor.RunMerge(opts)
	if err != nil {
		return err
	}
	fmt.Printf("合并完成: 读取 %d 条，输出 %d 条，输出至: %s\n",
		totalRead, totalWritten, opts.Output)
	return nil
}

// ========================= stats =========================
func runStats(args []string) error {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	var opts processor.StatsOptions
	var fieldsFlag []string
	var asJSON bool
	var help bool
	fs.StringVar(&opts.Input, "i", "", "输入JSON文件路径（必填）")
	fs.StringVar(&opts.Input, "input", "", "输入JSON文件路径（必填）")
	fs.Var(stringSlice(&fieldsFlag), "f", "指定统计字段，可多次指定（默认统计所有顶层字段）")
	fs.Var(stringSlice(&fieldsFlag), "field", "指定统计字段")
	fs.IntVar(&opts.TopN, "top", 5, "每个字段展示的高频值Top数量")
	fs.BoolVar(&opts.Deep, "deep", false, "递归深入嵌套对象/数组统计")
	fs.BoolVar(&asJSON, "json", false, "以JSON格式输出统计结果")
	fs.BoolVar(&help, "h", false, "显示帮助")
	fs.BoolVar(&help, "help", false, "显示帮助")
	_ = fs.Parse(args)

	if help {
		fmt.Println(`stats - 统计字段分布与数值特征

用法:
  json-handle-cli stats -i data.json
  json-handle-cli stats -i data.json -f age -f city --json
  json-handle-cli stats -i big.json -deep -top 10

输出内容:
  * 记录总数
  * 字段级: 出现次数、空值数、类型分布、唯一值数
  * 数值字段: 最小/最大/和/平均
  * 字符串字段: 最小/最大（字典序）
  * 高频值 TOP N

示例:
  # 概览全部顶层字段
  json-handle-cli stats -i users.json

  # 仅统计指定字段并输出JSON，便于后续程序处理
  json-handle-cli stats -i sales.json -f amount -f region --json > report.json`)
		return nil
	}
	if opts.Input == "" {
		return fmt.Errorf("缺少必要参数，请使用 --help 查看用法")
	}
	opts.Fields = fieldsFlag

	result, err := processor.RunStats(opts)
	if err != nil {
		return err
	}
	if asJSON {
		s, err := result.ToJSON()
		if err != nil {
			return err
		}
		fmt.Println(s)
	} else {
		fmt.Print(result.PrettyPrint())
	}
	return nil
}

// ========================= helper: string slice flag =========================
type stringSliceValue []string

func (s *stringSliceValue) String() string {
	return strings.Join(*s, ", ")
}
func (s *stringSliceValue) Set(v string) error {
	*s = append(*s, v)
	return nil
}
func stringSlice(p *[]string) flag.Value {
	return (*stringSliceValue)(p)
}
