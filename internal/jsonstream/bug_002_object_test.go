package jsonstream

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNullDocumentRemainsNullRecord(t *testing.T) {
	p := filepath.Join(t.TempDir(), "null.json")
	if err := os.WriteFile(p, []byte("null"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := DetectStreamMode(p)
	if err != nil {
		t.Fatal(err)
	}
	var record any
	if err := StreamProcess(p, m, func(v any) bool { record = v; return true }); err != nil {
		t.Fatal(err)
	}
	if record != nil {
		t.Fatalf("record=%#v want nil", record)
	}
}
