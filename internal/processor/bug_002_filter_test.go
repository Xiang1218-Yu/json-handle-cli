package processor

import (
	"path/filepath"
	"testing"
)

func TestNestedFilterSkipsNullProfile(t *testing.T) {
	d := t.TempDir()
	in, out := filepath.Join(d, "in.json"), filepath.Join(d, "out.json")
	writeJSON(t, in, []map[string]any{{"profile": nil}, {"profile": map[string]any{"city": "Tokyo"}}})
	_, kept, err := RunFilter(FilterOptions{Input: in, Output: out, Expr: "profile.city=Tokyo", OutputFmt: "array"})
	if err != nil {
		t.Fatal(err)
	}
	if kept != 1 {
		t.Fatalf("kept=%d", kept)
	}
}
