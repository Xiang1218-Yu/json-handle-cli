package processor

import (
	"encoding/json"
	"fmt"
	"json-handle-cli/internal/jsonstream"
	"math"
	"sort"
)

// FieldStats 单个字段的统计信息
type FieldStats struct {
	Name        string      `json:"name"`
	Count       int         `json:"count"`      // 出现次数
	NullCount   int         `json:"null_count"` // null/缺失次数
	StringCount int         `json:"string_count"`
	NumberCount int         `json:"number_count"`
	BoolCount   int         `json:"bool_count"`
	ObjectCount int         `json:"object_count"`
	ArrayCount  int         `json:"array_count"`
	UniqueCount int         `json:"unique_count"` // 唯一值估算（小集合精确统计）
	Min         interface{} `json:"min,omitempty"`
	Max         interface{} `json:"max,omitempty"`
	Sum         *float64    `json:"sum,omitempty"`
	Avg         *float64    `json:"avg,omitempty"`
	TopValues   []ValueFreq `json:"top_values,omitempty"` // 前N个高频值
}

type ValueFreq struct {
	Value interface{} `json:"value"`
	Freq  int         `json:"freq"`
}

// StatsOptions 统计配置
type StatsOptions struct {
	Input  string
	Fields []string // 指定统计的字段，空则统计全部顶层字段
	TopN   int      // Top值数量，默认5
	Deep   bool     // 是否递归统计嵌套对象字段
}

// StatsResult 统计结果
type StatsResult struct {
	TotalRecords int                   `json:"total_records"`
	FileSizeKB   int64                 `json:"file_size_kb"`
	Fields       map[string]FieldStats `json:"fields"`
}

// collector 单个字段统计的中间收集器（包级私有）
type collector struct {
	stats  FieldStats
	values map[string]int
	numMin float64
	numMax float64
	numSum float64
	hasNum bool
	strMin string
	strMax string
	hasStr bool
}

// RunStats 执行统计分析
func RunStats(opts StatsOptions) (*StatsResult, error) {
	mode, err := jsonstream.DetectStreamMode(opts.Input)
	if err != nil {
		return nil, fmt.Errorf("检测输入格式失败: %v", err)
	}

	if opts.TopN <= 0 {
		opts.TopN = 5
	}

	result := &StatsResult{
		Fields: map[string]FieldStats{},
	}

	collMap := map[string]*collector{}

	getCollector := func(fname string) *collector {
		c, ok := collMap[fname]
		if !ok {
			c = &collector{
				stats:  FieldStats{Name: fname},
				values: map[string]int{},
			}
			collMap[fname] = c
		}
		return c
	}

	err = jsonstream.StreamProcess(opts.Input, mode, func(item interface{}) bool {
		result.TotalRecords++
		obj, isObj := item.(map[string]interface{})
		if !isObj {
			// 非对象元素，作为一个顶层虚拟字段统计
			c := getCollector("<root>")
			c.stats.Count++
			recordValue(c, item)
			return true
		}

		// 确定要统计的字段
		var fieldsToCheck []string
		if len(opts.Fields) > 0 {
			fieldsToCheck = opts.Fields
			for _, f := range fieldsToCheck {
				c := getCollector(f)
				c.stats.Count++
				v, exists := getFieldValue(obj, f)
				if !exists {
					c.stats.NullCount++
					continue
				}
				recordValue(c, v)
			}
		} else {
			// 顶层或递归全部字段
			walkFields("", obj, opts.Deep, func(fname string, v interface{}) {
				c := getCollector(fname)
				c.stats.Count++
				if v == nil {
					c.stats.NullCount++
					c.values["<null>"]++
					return
				}
				recordValue(c, v)
			})
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	// 聚合collector到result
	for name, c := range collMap {
		// 计算数值统计
		if c.hasNum {
			c.stats.Min = c.numMin
			c.stats.Max = c.numMax
			s := c.numSum
			a := c.numSum / float64(c.stats.NumberCount)
			c.stats.Sum = &s
			c.stats.Avg = &a
		} else if c.hasStr {
			c.stats.Min = c.strMin
			c.stats.Max = c.strMax
		}

		// 唯一值
		c.stats.UniqueCount = len(c.values)

		// Top值
		if len(c.values) > 0 {
			type kv struct {
				k string
				v int
			}
			list := make([]kv, 0, len(c.values))
			for k, v := range c.values {
				list = append(list, kv{k, v})
			}
			sort.Slice(list, func(i, j int) bool {
				return list[i].v > list[j].v
			})
			limit := opts.TopN
			if limit > len(list) {
				limit = len(list)
			}
			for i := 0; i < limit; i++ {
				// 尝试还原值类型
				var orig interface{}
				orig = list[i].k
				// 对于数值，尽量转回float64（仅对精确匹配的值展示）
				c.stats.TopValues = append(c.stats.TopValues, ValueFreq{
					Value: orig,
					Freq:  list[i].v,
				})
			}
		}

		result.Fields[name] = c.stats
	}

	return result, nil
}

func walkFields(prefix string, v interface{}, deep bool, fn func(string, interface{})) {
	switch x := v.(type) {
	case map[string]interface{}:
		for k, vv := range x {
			name := k
			if prefix != "" {
				name = prefix + "." + k
			}
			fn(name, vv)
			if deep {
				switch vv.(type) {
				case map[string]interface{}, []interface{}:
					walkFields(name, vv, deep, fn)
				}
			}
		}
	case []interface{}:
		// 数组元素：若deep=true则遍历
		if deep {
			for i, elem := range x {
				name := fmt.Sprintf("%s[%d]", prefix, i)
				switch elem.(type) {
				case map[string]interface{}, []interface{}:
					walkFields(name, elem, deep, fn)
				}
			}
		}
	}
}

func recordValue(c *collector, v interface{}) {
	if v == nil {
		c.stats.NullCount++
		c.values["<null>"]++
		return
	}
	switch x := v.(type) {
	case string:
		c.stats.StringCount++
		key := x
		c.values[key]++
		if !c.hasStr || x < c.strMin {
			c.strMin = x
		}
		if !c.hasStr || x > c.strMax {
			c.strMax = x
		}
		c.hasStr = true
	case float64:
		c.stats.NumberCount++
		key := fmt.Sprintf("%v", x)
		c.values[key]++
		if !c.hasNum || x < c.numMin {
			c.numMin = x
		}
		if !c.hasNum || x > c.numMax {
			c.numMax = x
		}
		c.numSum += x
		c.hasNum = true
	case bool:
		c.stats.BoolCount++
		key := fmt.Sprintf("%v", x)
		c.values[key]++
	case map[string]interface{}:
		c.stats.ObjectCount++
		c.values["<object>"]++
	case []interface{}:
		c.stats.ArrayCount++
		key := fmt.Sprintf("<array[%d]>", len(x))
		c.values[key]++
	default:
		// 包括 json.Number 等
		if f, err := toFloat64(x); err == nil {
			c.stats.NumberCount++
			key := fmt.Sprintf("%v", f)
			c.values[key]++
			if !c.hasNum || f < c.numMin {
				c.numMin = f
			}
			if !c.hasNum || f > c.numMax {
				c.numMax = f
			}
			c.numSum += f
			c.hasNum = true
		} else {
			key := fmt.Sprintf("%v", x)
			c.values[key]++
		}
	}
}

// PrettyPrint 将统计结果格式化输出（便于阅读）
func (r *StatsResult) PrettyPrint() string {
	out := fmt.Sprintf("总记录数: %d\n", r.TotalRecords)
	out += fmt.Sprintf("字段数:   %d\n\n", len(r.Fields))

	// 字段排序输出
	names := make([]string, 0, len(r.Fields))
	for n := range r.Fields {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, n := range names {
		f := r.Fields[n]
		out += fmt.Sprintf("━━━ 字段: %s ━━━\n", n)
		out += fmt.Sprintf("  出现: %d  空值: %d\n", f.Count, f.NullCount)
		if f.Count-f.NullCount > 0 {
			parts := []string{}
			if f.StringCount > 0 {
				parts = append(parts, fmt.Sprintf("字符串=%d", f.StringCount))
			}
			if f.NumberCount > 0 {
				parts = append(parts, fmt.Sprintf("数值=%d", f.NumberCount))
			}
			if f.BoolCount > 0 {
				parts = append(parts, fmt.Sprintf("布尔=%d", f.BoolCount))
			}
			if f.ObjectCount > 0 {
				parts = append(parts, fmt.Sprintf("对象=%d", f.ObjectCount))
			}
			if f.ArrayCount > 0 {
				parts = append(parts, fmt.Sprintf("数组=%d", f.ArrayCount))
			}
			if len(parts) > 0 {
				out += fmt.Sprintf("  类型分布: %v\n", parts)
			}
			out += fmt.Sprintf("  唯一值数: %d\n", f.UniqueCount)
		}
		if f.Min != nil {
			out += fmt.Sprintf("  最小: %v    最大: %v\n", f.Min, f.Max)
		}
		if f.Sum != nil {
			sum := *f.Sum
			avg := *f.Avg
			if math.Abs(sum-float64(int64(sum))) < 1e-9 && math.Abs(avg-float64(int64(avg))) < 1e-9 {
				out += fmt.Sprintf("  数值和: %d    平均: %.4g\n", int64(sum), avg)
			} else {
				out += fmt.Sprintf("  数值和: %.4g    平均: %.4g\n", sum, avg)
			}
		}
		if len(f.TopValues) > 0 {
			out += fmt.Sprintf("  高频值 TOP %d:\n", len(f.TopValues))
			for _, tv := range f.TopValues {
				out += fmt.Sprintf("    %-6d  %v\n", tv.Freq, tv.Value)
			}
		}
		out += "\n"
	}
	return out
}

// ToJSON 将统计结果输出为JSON字符串
func (r *StatsResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
