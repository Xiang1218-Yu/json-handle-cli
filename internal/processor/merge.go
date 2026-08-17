package processor

import (
	"fmt"
	"json-handle-cli/internal/jsonstream"
	"os"
)

// MergeOptions 合并操作配置
type MergeOptions struct {
	Inputs    []string // 输入文件列表
	Output    string   // 输出文件
	KeyField  string   // 去重/合并的主键字段（空表示不做去重，单纯拼接）
	Strategy  string   // 合并策略: concat(拼接), deduplicate(去重), union(对象按键合并)
	OutputFmt string   // 输出格式
}

// RunMerge 执行合并操作
func RunMerge(opts MergeOptions) (int, int, error) {
	// 准备输出
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

	totalRead := 0
	totalWritten := 0

	switch opts.Strategy {
	case "deduplicate", "dedup":
		seen := map[string]bool{}
		for _, input := range opts.Inputs {
			mode, err := jsonstream.DetectStreamMode(input)
			if err != nil {
				return totalRead, totalWritten, fmt.Errorf("检测输入格式失败 %s: %v", input, err)
			}
			err = jsonstream.StreamProcess(input, mode, func(item interface{}) bool {
				totalRead++
				key, ok := buildKey(item, opts.KeyField)
				if !ok {
					// 没有key字段时，默认写入
					if wErr := writer.Write(item); wErr != nil {
						err = fmt.Errorf("写入输出失败: %v", wErr)
						return false
					}
					totalWritten++
					return true
				}
				if !seen[key] {
					seen[key] = true
					if wErr := writer.Write(item); wErr != nil {
						err = fmt.Errorf("写入输出失败: %v", wErr)
						return false
					}
					totalWritten++
				}
				return true
			})
			if err != nil {
				return totalRead, totalWritten, err
			}
		}

	case "union", "merge_objects":
		// 将所有对象按键合并（适合单对象文件），以KeyField为分组键合并数组元素
		merged := map[string]interface{}{}
		var mergedList []interface{}
		if opts.KeyField == "" {
			mergedList = []interface{}{}
		}
		for _, input := range opts.Inputs {
			mode, err := jsonstream.DetectStreamMode(input)
			if err != nil {
				return totalRead, totalWritten, fmt.Errorf("检测输入格式失败 %s: %v", input, err)
			}
			err = jsonstream.StreamProcess(input, mode, func(item interface{}) bool {
				totalRead++
				if opts.KeyField == "" {
					// 合并所有对象的顶层key
					if obj, ok := item.(map[string]interface{}); ok {
						for k, v := range obj {
							merged[k] = v
						}
					} else {
						mergedList = append(mergedList, item)
					}
				} else {
					// 按键分组合并
					key, ok := buildKey(item, opts.KeyField)
					if !ok {
						mergedList = append(mergedList, item)
					} else {
						if existing, exists := merged[key]; exists {
							// 尝试深度合并对象
							if eObj, ok1 := existing.(map[string]interface{}); ok1 {
								if nObj, ok2 := item.(map[string]interface{}); ok2 {
									for k, v := range nObj {
										eObj[k] = v
									}
									merged[key] = eObj
									return true
								}
							}
						}
						merged[key] = item
					}
				}
				return true
			})
			if err != nil {
				return totalRead, totalWritten, err
			}
		}
		// 输出
		if opts.KeyField == "" && len(merged) > 0 {
			if wErr := writer.Write(merged); wErr != nil {
				return totalRead, totalWritten, fmt.Errorf("写入输出失败: %v", wErr)
			}
			totalWritten++
		}
		// 输出去重后/合并后的值
		for _, v := range merged {
			if opts.KeyField != "" {
				if wErr := writer.Write(v); wErr != nil {
					return totalRead, totalWritten, fmt.Errorf("写入输出失败: %v", wErr)
				}
				totalWritten++
			}
		}
		for _, v := range mergedList {
			if wErr := writer.Write(v); wErr != nil {
				return totalRead, totalWritten, fmt.Errorf("写入输出失败: %v", wErr)
			}
			totalWritten++
		}

	default:
		// concat: 简单拼接
		for _, input := range opts.Inputs {
			mode, err := jsonstream.DetectStreamMode(input)
			if err != nil {
				return totalRead, totalWritten, fmt.Errorf("检测输入格式失败 %s: %v", input, err)
			}
			err = jsonstream.StreamProcess(input, mode, func(item interface{}) bool {
				totalRead++
				if wErr := writer.Write(item); wErr != nil {
					err = fmt.Errorf("写入输出失败: %v", wErr)
					return false
				}
				totalWritten++
				return true
			})
			if err != nil {
				return totalRead, totalWritten, err
			}
		}
	}

	return totalRead, totalWritten, nil
}

func buildKey(item interface{}, keyField string) (string, bool) {
	if keyField == "" {
		return fmt.Sprintf("%#v", item), true
	}
	v, ok := getFieldValue(item, keyField)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%v", v), true
}
