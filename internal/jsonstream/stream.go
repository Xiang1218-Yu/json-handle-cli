package jsonstream

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// StreamMode 定义JSON流式处理模式
type StreamMode int

const (
	// StreamArray 处理顶层为JSON数组的文件，逐个元素处理
	StreamArray StreamMode = iota
	// StreamObject 处理顶层为JSON对象的文件（整体作为一个元素）
	StreamObject
	// StreamLines 处理JSON Lines格式（每行一个JSON对象）
	StreamLines
)

// DetectStreamMode 自动检测JSON文件的流式处理模式
func DetectStreamMode(filename string) (StreamMode, error) {
	f, err := os.Open(filename)
	if err != nil {
		return StreamArray, err
	}
	defer f.Close()

	// 读取第一个非空白字符
	buf := make([]byte, 1)
	for {
		_, err := f.Read(buf)
		if err != nil {
			return StreamArray, fmt.Errorf("无法检测文件格式: %v", err)
		}
		c := buf[0]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		if c == '[' {
			return StreamArray, nil
		}
		if c == '{' {
			// 可能是对象或JSON Lines，读取更多判断
			return detectObjectOrLines(f)
		}
		if c == 'n' {
			// 顶层 null 文档：按单条记录处理，值保留为 nil
			return StreamObject, nil
		}
		return StreamArray, fmt.Errorf("不支持的JSON格式，首字符: %c", c)
	}
}

func detectObjectOrLines(f *os.File) (StreamMode, error) {
	// 简单策略：检查接下来的内容是否包含结束大括号后跟换行和左大括号（JSON Lines特征）
	// 更可靠的是尝试用Decoder解析
	dec := json.NewDecoder(f)
	_, err := dec.Token() // 消耗已读的 '{'
	if err != nil {
		return StreamObject, nil
	}

	// 读取整个对象
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err == io.EOF {
			return StreamObject, nil
		}
		if err != nil {
			return StreamLines, nil // 解析失败，尝试Lines模式
		}
		switch tok {
		case json.Delim('{'):
			depth++
		case json.Delim('}'):
			depth--
		}
	}

	// 对象读取完成后，查看是否还有内容（除空白外）
	_, err = dec.Token()
	if err == io.EOF {
		return StreamObject, nil
	}
	// 还有内容，可能是JSON Lines
	return StreamLines, nil
}

// StreamProcess 流式处理JSON文件，对每个元素调用handler
// handler返回false时停止处理
func StreamProcess(filename string, mode StreamMode, handler func(interface{}) bool) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	switch mode {
	case StreamArray:
		return processArray(f, handler)
	case StreamObject:
		return processObject(f, handler)
	case StreamLines:
		return processLines(f, handler)
	default:
		return fmt.Errorf("未知的流模式: %d", mode)
	}
}

func processArray(f *os.File, handler func(interface{}) bool) error {
	dec := json.NewDecoder(f)

	// 读取开始的 '['
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("读取数组开头失败: %v", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("期望数组开头 '['，实际得到 %v", tok)
	}

	for dec.More() {
		var item interface{}
		if err := dec.Decode(&item); err != nil {
			return fmt.Errorf("解析数组元素失败: %v", err)
		}
		if !handler(item) {
			return nil
		}
	}

	// 读取结束的 ']'
	_, err = dec.Token()
	if err != nil && err != io.EOF {
		return fmt.Errorf("读取数组结尾失败: %v", err)
	}
	return nil
}

func processObject(f *os.File, handler func(interface{}) bool) error {
	dec := json.NewDecoder(f)
	var obj interface{}
	if err := dec.Decode(&obj); err != nil {
		return fmt.Errorf("解析JSON对象失败: %v", err)
	}
	// 顶层 null 文档或任何顶层标量都作为单条记录原样交付，
	// 不用空对象替换 nil，以免掩盖"空记录"这一事实。
	handler(obj)
	return nil
}

func processLines(f *os.File, handler func(interface{}) bool) error {
	dec := json.NewDecoder(f)
	for {
		var item interface{}
		if err := dec.Decode(&item); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("解析JSON行失败: %v", err)
		}
		if !handler(item) {
			return nil
		}
	}
}

// StreamWriteArray 流式写入JSON数组到文件（逐个写入，无需全量内存）
type StreamArrayWriter struct {
	w       io.Writer
	enc     *json.Encoder
	started bool
	count   int
}

func NewStreamArrayWriter(w io.Writer) *StreamArrayWriter {
	return &StreamArrayWriter{w: w, enc: json.NewEncoder(w)}
}

func (sw *StreamArrayWriter) Write(v interface{}) error {
	if !sw.started {
		if _, err := sw.w.Write([]byte("[\n")); err != nil {
			return err
		}
		sw.enc.SetIndent("  ", "  ")
		sw.started = true
	}
	if sw.count > 0 {
		if _, err := sw.w.Write([]byte(",\n")); err != nil {
			return err
		}
	}
	sw.count++
	return sw.enc.Encode(v)
}

func (sw *StreamArrayWriter) Close() error {
	if !sw.started {
		_, err := sw.w.Write([]byte("[]"))
		return err
	}
	_, err := sw.w.Write([]byte("\n]\n"))
	return err
}

// WriteJSONLines 写入JSON Lines格式
type StreamLinesWriter struct {
	w   io.Writer
	enc *json.Encoder
}

func NewStreamLinesWriter(w io.Writer) *StreamLinesWriter {
	return &StreamLinesWriter{w: w, enc: json.NewEncoder(w)}
}

func (lw *StreamLinesWriter) Write(v interface{}) error {
	return lw.enc.Encode(v)
}

func (lw *StreamLinesWriter) Close() error {
	return nil
}
