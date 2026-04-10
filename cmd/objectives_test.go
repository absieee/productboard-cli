// cmd/objectives_test.go
package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aveni/pb-cli/cmd"
	"github.com/spf13/cobra"
)

func TestObjectivesListRendersTable(t *testing.T) {
	resp := map[string]any{
		"data": []map[string]any{
			{
				"id":   "44444444-4444-4444-4444-444444444444",
				"type": "objective",
				"fields": map[string]any{
					"name": "Improve customer retention",
				},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	root := &cobra.Command{Use: "pb", PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil }}
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	cmd.AddObjectivesCmd(root, srv.URL, func() string { return "test-token" })

	root.SetArgs([]string{"objectives", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, buf.String())
	}

	if !strings.Contains(buf.String(), "Improve customer retention") {
		t.Errorf("expected objective name in output:\n%s", buf.String())
	}
}
