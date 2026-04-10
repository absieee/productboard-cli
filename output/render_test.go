package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aveni/pb-cli/output"
)

func TestRenderTable(t *testing.T) {
	rows := [][]string{
		{"AI Call Summarisation", "in_progress", "abs@aveni.ai"},
		{"Compliance Audit Trail", "new", "abs@aveni.ai"},
	}
	headers := []string{"NAME", "STATUS", "OWNER"}

	var buf bytes.Buffer
	output.Table(&buf, headers, rows)

	got := buf.String()
	if !strings.Contains(got, "AI Call Summarisation") {
		t.Errorf("expected table to contain feature name, got:\n%s", got)
	}
	if !strings.Contains(got, "NAME") {
		t.Errorf("expected table to contain header NAME, got:\n%s", got)
	}
}

func TestRenderJSON(t *testing.T) {
	data := map[string]string{"id": "abc", "name": "test"}
	var buf bytes.Buffer
	if err := output.JSON(&buf, data); err != nil {
		t.Fatal(err)
	}
	var out map[string]string
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out["name"] != "test" {
		t.Errorf("expected name=test, got %s", out["name"])
	}
}
