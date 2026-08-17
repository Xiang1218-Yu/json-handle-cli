package processor

import (
	"fmt"
	"json-handle-cli/internal/jsonstream"
	"os"
	"strconv"
	"strings"
)

// TransformRule 单个字段转换规则
type TransformRule struct {
	Action string // 操作: rename, delete, set, type, uppercase, lowercase, trim
	Field  string // 目标字段路径（支持 a.b.c）
	Value  string // 参数值（新名称/设定值/目标类型等）
}

// TransformOptions 转换操作配置
type TransformOptions struct {
	Input     string          // 输入文件
	Output    string          // 输出文件
	Rules     []TransformRule // 转换规则列表
	OutputFmt string          // 输出格式
}

// RunTransform 执行转换操作
func RunTransform(opts TransformOptions) (int, error) {
	mode, err := jsonstream.DetectStreamMode(opts.Input)
	if err != nil {
		return 0, fmt.Errorf("检测输入格式失败: %v", err)
	}

	out, err := os.Create(opts.Output)
	if err != nil {
		return 0, fmt.Errorf("创建输出文件失败: %v", err)
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
	defer func() {
		if cerr := writer.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	count := 0
	err = jsonstream.StreamProcess(opts.Input, mode, func(item interface{}) bool {
		transformed, ok := item.(map[string]interface{})
		if !ok {
			// 非对象元素尝试写入原数据
			if wErr := writer.Write(item); wErr != nil {
				err = fmt.Errorf("写入输出失败: %v", wErr)
				return false
			}
			count++
			return true
		}
		for _, r := range opts.Rules {
			if err2 := applyRule(transformed, r); err2 != nil {
				err = err2
				return false
			}
		}
		if wErr := writer.Write(transformed); wErr != nil {
			err = fmt.Errorf("写入输出失败: %v", wErr)
			return false
		}
		count++
		return true
	})
	return count, err
}

func applyRule(obj map[string]interface{}, rule TransformRule) error {
	switch rule.Action {
	case "delete", "del", "remove":
		return deleteField(obj, rule.Field)
	case "rename", "mv":
		return renameField(obj, rule.Field, rule.Value)
	case "set", "add":
		return setField(obj, rule.Field, parseValue(rule.Value))
	case "type", "cast":
		return castField(obj, rule.Field, rule.Value)
	case "upper", "uppercase":
		return strMapField(obj, rule.Field, strings.ToUpper)
	case "lower", "lowercase":
		return strMapField(obj, rule.Field, strings.ToLower)
	case "trim":
		return strMapField(obj, rule.Field, strings.TrimSpace)
	default:
		return fmt.Errorf("未知的转换操作: %s", rule.Action)
	}
}

func parseValue(s string) interface{} {
	// 尝试解析布尔
	if strings.EqualFold(s, "true") {
		return true
	}
	if strings.EqualFold(s, "false") {
		return false
	}
	if s == "null" {
		return nil
	}
	// 尝试数值
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		// 但如果带引号则视为字符串，这里简化：若看起来像整数返回int
		if f == float64(int64(f)) && !strings.Contains(s, ".") {
			return int64(f)
		}
		return f
	}
	// 去除首尾引号视为字符串
	return strings.Trim(s, "\"'")
}

func getParentAndKey(obj map[string]interface{}, path string) (map[string]interface{}, string, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, "", fmt.Errorf("空路径")
	}
	current := obj
	for i := 0; i < len(parts)-1; i++ {
		next, ok := current[parts[i]].(map[string]interface{})
		if !ok {
			return nil, "", fmt.Errorf("路径不存在或不是对象: %s", strings.Join(parts[:i+1], "."))
		}
		current = next
	}
	return current, parts[len(parts)-1], nil
}

func deleteField(obj map[string]interface{}, path string) error {
	parent, key, err := getParentAndKey(obj, path)
	if err != nil {
		return err
	}
	delete(parent, key)
	return nil
}

func renameField(obj map[string]interface{}, oldPath, newName string) error {
	val, ok := getFieldValue(obj, oldPath)
	if !ok {
		return fmt.Errorf("字段不存在: %s", oldPath)
	}
	// 删除旧字段
	if err := deleteField(obj, oldPath); err != nil {
		return err
	}
	// 设置新字段（newName可以是完整路径或简单名称）
	// 若newName不包含点，就放在同一层级下
	var setPath string
	if strings.Contains(newName, ".") {
		setPath = newName
	} else {
		parentPath := ""
		if i := strings.LastIndex(oldPath, "."); i >= 0 {
			parentPath = oldPath[:i] + "."
		}
		setPath = parentPath + newName
	}
	return setField(obj, setPath, val)
}

func setField(obj map[string]interface{}, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return fmt.Errorf("空路径")
	}
	current := obj
	for i := 0; i < len(parts)-1; i++ {
		next, ok := current[parts[i]].(map[string]interface{})
		if !ok {
			// 创建中间对象
			next = map[string]interface{}{}
			current[parts[i]] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
	return nil
}

func castField(obj map[string]interface{}, path, targetType string) error {
	parent, key, err := getParentAndKey(obj, path)
	if err != nil {
		return err
	}
	val, exists := parent[key]
	if !exists {
		return fmt.Errorf("字段不存在: %s", path)
	}
	switch strings.ToLower(targetType) {
	case "string", "str":
		parent[key] = fmt.Sprintf("%v", val)
	case "int", "integer":
		f, err := toFloat64(val)
		if err != nil {
			return fmt.Errorf("字段 %s 无法转为int: %v", path, err)
		}
		parent[key] = int64(f)
	case "float", "number":
		f, err := toFloat64(val)
		if err != nil {
			return fmt.Errorf("字段 %s 无法转为float: %v", path, err)
		}
		parent[key] = f
	case "bool", "boolean":
		switch v := strings.ToLower(fmt.Sprintf("%v", val)); v {
		case "true", "1", "yes":
			parent[key] = true
		case "false", "0", "no":
			parent[key] = false
		default:
			return fmt.Errorf("字段 %s 无法转为bool: %v", path, val)
		}
	default:
		return fmt.Errorf("不支持的目标类型: %s (支持 string/int/float/bool)", targetType)
	}
	return nil
}

func strMapField(obj map[string]interface{}, path string, fn func(string) string) error {
	parent, key, err := getParentAndKey(obj, path)
	if err != nil {
		return err
	}
	val, exists := parent[key]
	if !exists {
		return fmt.Errorf("字段不存在: %s", path)
	}
	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("字段 %s 不是字符串类型", path)
	}
	parent[key] = fn(s)
	return nil
}

// ParseTransformRule 解析命令行传递的转换规则字符串
// 格式: action:field=value 或 action:field（不需要value的操作）
func ParseTransformRule(s string) (TransformRule, error) {
	colonIdx := strings.Index(s, ":")
	if colonIdx <= 0 {
		return TransformRule{}, fmt.Errorf("规则格式错误: %s，应为 action:field[=value]", s)
	}
	action := strings.ToLower(strings.TrimSpace(s[:colonIdx]))
	rest := s[colonIdx+1:]

	// 有些操作不需要 =value（delete/upper/lower/trim）
	switch action {
	case "delete", "del", "remove", "upper", "uppercase", "lower", "lowercase", "trim":
		return TransformRule{Action: action, Field: strings.TrimSpace(rest)}, nil
	}

	eqIdx := strings.Index(rest, "=")
	if eqIdx < 0 {
		return TransformRule{}, fmt.Errorf("规则 %s 需要 field=value 格式", s)
	}
	field := strings.TrimSpace(rest[:eqIdx])
	value := strings.TrimSpace(rest[eqIdx+1:])
	return TransformRule{Action: action, Field: field, Value: value}, nil
}
