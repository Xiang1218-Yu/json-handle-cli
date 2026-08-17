package jsonstream

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMalformedJSONLineReturnsError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "events.jsonl")
	os.WriteFile(p, []byte("{\"id\":1}\n{bad}\n"), 0o600)
	m, _ := DetectStreamMode(p)
	if err := StreamProcess(p, m, func(any) bool { return true }); err == nil {
		t.Fatal("expected malformed line error")
	}
}
