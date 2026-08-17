package jsonstream

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAndProcessSupportedFormats(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"array.json":  `[{"id":1},{"id":2}]`,
		"object.json": `{"id":1}`,
		"lines.jsonl": "{\"id\":1}\n{\"id\":2}\n",
	}
	for name, content := range cases {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		mode, err := DetectStreamMode(path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		count := 0
		if err := StreamProcess(path, mode, func(any) bool { count++; return true }); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := 1
		if name != "object.json" {
			want = 2
		}
		if count != want {
			t.Fatalf("%s: count=%d want=%d", name, count, want)
		}
	}
}
