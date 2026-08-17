package processor

import (
	"path/filepath"
	"testing"
)

func TestMergeFinalizesJSONArray(t *testing.T) {
	dir := t.TempDir()
	in, out := filepath.Join(dir, "in.json"), filepath.Join(dir, "out.json")
	writeJSON(t, in, []map[string]any{{"id": 1}})
	if _, _, err := RunMerge(MergeOptions{Inputs: []string{in}, Output: out, Strategy: "concat", OutputFmt: "array"}); err != nil {
		t.Fatal(err)
	}
	records := readJSON(t, out).([]any)
	if len(records) != 1 {
		t.Fatalf("records=%d want 1", len(records))
	}
}
