package processor

import (
	"fmt"
	"json-handle-cli/internal/jsonstream"
	"os"
	"strconv"
	"strings"
)

// FilterOptions 过滤操作配置
type FilterOptions struct {
	Input     string // 输入文件路径
	Output    string // 输出文件路径
	Expr      string // 过滤表达式，格式: field=value 或 field>value 等
	Invert    bool   // 反转匹配（保留不匹配的）
	OutputFmt string // 输出格式: array 或 lines
}

// 支持的比较操作符
var filterOps = []string{">=", "<=", "!=", "==", ">", "<", "=", "~"}

// RunFilter 执行过滤操作
func RunFilter(opts FilterOptions) (int, int, error) {
	mode, err := jsonstream.DetectStreamMode(opts.Input)
	if err != nil {
		return 0, 0, fmt.Errorf("检测输入格式失败: %v", err)
	}

	field, op, value, err := parseFilterExpr(opts.Expr)
	if err != nil {
		return 0, 0, err
	}

	out, err := os.Create(opts.Output)
	if err != nil {
		return 0, 0, fmt.Errorf("创建输出文件失败: %v", err)
	}
	defer out.Close()

	var writer interface {
		Write(interface{}) error
		Close() error
	}
	if opts.OutputFmt == "lines" {
		writer = jsonstream.NewStreamLinesWriter(out)
	} else {
		writer = jsonstream.NewStreamArrayWriter(out)
	}
	defer writer.Close()

	total := 0
	kept := 0
	err = jsonstream.StreamProcess(opts.Input, mode, func(item interface{}) bool {
		total++
		match := matchCondition(item, field, op, value)
		if opts.Invert {
			match = !match
		}
		if match {
			kept++
			if wErr := writer.Write(item); wErr != nil {
				err = fmt.Errorf("写入输出失败: %v", wErr)
				return false
			}
		}
		return true
	})
	if err != nil {
		return total, kept, err
	}
	return total, kept, nil
}

func parseFilterExpr(expr string) (field, op, value string, err error) {
	for _, o := range filterOps {
		idx := strings.Index(expr, o)
		if idx > 0 {
			field = strings.TrimSpace(expr[:idx])
			op = o
			value = strings.TrimSpace(expr[idx+len(o):])
			// 处理带引号的值
			value = strings.Trim(value, "\"'")
			return field, op, value, nil
		}
	}
	return "", "", "", fmt.Errorf("无效的过滤表达式: %s，格式示例: age>=18, name=John, status!=active", expr)
}

// getFieldValue 通过路径获取嵌套字段值，支持 a.b.c 格式
func getFieldValue(item interface{}, fieldPath string) (interface{}, bool) {
	if fieldPath == "" || fieldPath == "." {
		return item, true
	}
	parts := strings.Split(fieldPath, ".")
	current := item
	for _, p := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		v, exists := m[p]
		if !exists {
			return nil, false
		}
		current = v
	}
	return current, true
}

func matchCondition(item interface{}, field, op, expected string) bool {
	val, ok := getFieldValue(item, field)
	if !ok {
		// 字段不存在时的匹配策略
		return op == "!=" // 只有 != 时认为匹配（不存在不等于期望值）
	}

	// 转为字符串比较或数值比较
	switch op {
	case "=", "==":
		return fmt.Sprintf("%v", val) == expected
	case "!=":
		return fmt.Sprintf("%v", val) != expected
	case "~":
		// 包含匹配（字符串包含）
		return strings.Contains(fmt.Sprintf("%v", val), expected)
	}

	// 数值比较
	fv, errNum := toFloat64(val)
	fe, errExp := strconv.ParseFloat(expected, 64)
	if errNum != nil || errExp != nil {
		// 回退到字符串比较
		s := fmt.Sprintf("%v", val)
		switch op {
		case ">":
			return s > expected
		case "<":
			return s < expected
		case ">=":
			return s >= expected
		case "<=":
			return s <= expected
		}
		return false
	}
	switch op {
	case ">":
		return fv > fe
	case "<":
		return fv < fe
	case ">=":
		return fv >= fe
	case "<=":
		return fv <= fe
	}
	return false
}

func toFloat64(v interface{}) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case string:
		return strconv.ParseFloat(x, 64)
	default:
		return 0, fmt.Errorf("无法转为数值: %v", v)
	}
}
