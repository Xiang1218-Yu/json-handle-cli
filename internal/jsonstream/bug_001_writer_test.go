package jsonstream

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestArrayWriterKeepsSingleRecordOutputValid(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStreamArrayWriter(&buf)
	if err := writer.Write(map[string]any{"id": 1}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &records); err != nil {
		t.Fatalf("writer output must be valid JSON: %v", err)
	}
}
