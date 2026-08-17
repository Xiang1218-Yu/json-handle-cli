package processor

import (
	"path/filepath"
	"testing"
)

func TestTransformFinalizesOutput(t *testing.T) {
	d := t.TempDir()
	in, out := filepath.Join(d, "in.json"), filepath.Join(d, "out.json")
	writeJSON(t, in, []map[string]any{{"id": 1}})
	RunTransform(TransformOptions{Input: in, Output: out, OutputFmt: "array"})
	readJSON(t, out)
}
