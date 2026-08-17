package processor

import (
	"path/filepath"
	"testing"
)

func TestStatsCountsAllRecords(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "x.json")
	writeJSON(t, p, []map[string]any{{"id": 1}, {"id": 2}})
	r, e := RunStats(StatsOptions{Input: p})
	if e != nil {
		t.Fatal(e)
	}
	if r.TotalRecords != 2 {
		t.Fatalf("count=%d", r.TotalRecords)
	}
}
