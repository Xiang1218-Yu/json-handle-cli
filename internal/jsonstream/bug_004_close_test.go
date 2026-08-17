package jsonstream

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCloseCompletesArray(t *testing.T) {
	var b bytes.Buffer
	w := NewStreamArrayWriter(&b)
	w.Write(map[string]any{"id": 1})
	w.Close()
	var v any
	if err := json.Unmarshal(b.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
}
