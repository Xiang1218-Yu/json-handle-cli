package processor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, path string) any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestFilterTransformMergeAndStatsWorkflow(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "users.json")
	filtered := filepath.Join(dir, "adults.json")
	transformed := filepath.Join(dir, "clean.json")
	merged := filepath.Join(dir, "merged.json")
	writeJSON(t, input, []map[string]any{
		{"id": 1, "name": " alice ", "age": 17, "region": "east"},
		{"id": 2, "name": " bob ", "age": 21, "region": "west"},
		{"id": 3, "name": " cara ", "age": 28, "region": "east"},
	})

	total, kept, err := RunFilter(FilterOptions{Input: input, Output: filtered, Expr: "age>=18", OutputFmt: "array"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || kept != 2 {
		t.Fatalf("filter counts = %d,%d; want 3,2", total, kept)
	}

	rules := []TransformRule{{Action: "trim", Field: "name"}, {Action: "upper", Field: "name"}, {Action: "set", Field: "source", Value: "daily"}}
	if _, err := RunTransform(TransformOptions{Input: filtered, Output: transformed, Rules: rules, OutputFmt: "array"}); err != nil {
		t.Fatal(err)
	}

	writeJSON(t, filepath.Join(dir, "later.json"), []map[string]any{{"id": 2, "name": "BOB", "age": 21}, {"id": 4, "name": "DAN", "age": 42}})
	if _, kept, err := RunMerge(MergeOptions{Inputs: []string{transformed, filepath.Join(dir, "later.json")}, Output: merged, Strategy: "deduplicate", KeyField: "id", OutputFmt: "array"}); err != nil {
		t.Fatal(err)
	} else if kept != 3 {
		t.Fatalf("merge kept %d; want 3", kept)
	}

	result, err := RunStats(StatsOptions{Input: merged, Fields: []string{"age"}, TopN: 2})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalRecords != 3 {
		t.Fatalf("records = %d; want 3", result.TotalRecords)
	}
	if got := result.Fields["age"].NumberCount; got != 3 {
		t.Fatalf("age numbers = %d; want 3", got)
	}

	items := readJSON(t, merged).([]any)
	if got := items[0].(map[string]any)["name"]; got != "BOB" {
		t.Fatalf("first transformed name = %v", got)
	}
}

func TestTransformRuleParsing(t *testing.T) {
	rule, err := ParseTransformRule("rename:profile.city=location.city")
	if err != nil {
		t.Fatal(err)
	}
	if rule.Action != "rename" || rule.Field != "profile.city" || rule.Value != "location.city" {
		t.Fatalf("unexpected rule: %#v", rule)
	}
	if _, err := ParseTransformRule("rename:missing-value"); err == nil {
		t.Fatal("expected malformed rename rule to fail")
	}
}
