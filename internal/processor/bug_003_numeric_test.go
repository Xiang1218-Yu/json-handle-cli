package processor

import (
	"path/filepath"
	"testing"
)

func TestInvalidNumericValueDoesNotMatchThreshold(t *testing.T) {
	d := t.TempDir()
	in, out := filepath.Join(d, "in.json"), filepath.Join(d, "out.json")
	writeJSON(t, in, []map[string]any{{"amount": true}, {"amount": 4}})
	_, n, e := RunFilter(FilterOptions{Input: in, Output: out, Expr: "amount<10", OutputFmt: "array"})
	if e != nil {
		t.Fatal(e)
	}
	if n != 1 {
		t.Fatalf("kept=%d", n)
	}
}
