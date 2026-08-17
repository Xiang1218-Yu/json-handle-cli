package jsonstream

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONLinesMode(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.jsonl")
	os.WriteFile(p, []byte("{\"id\":1}\n{\"id\":2}\n"), 0o600)
	m, _ := DetectStreamMode(p)
	if m != StreamLines {
		t.Fatalf("mode=%d", m)
	}
}
