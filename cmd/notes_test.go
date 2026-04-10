// cmd/notes_test.go
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

func TestNotesListRendersTable(t *testing.T) {
	resp := map[string]any{
		"data": []map[string]any{
			{
				"id":        "33333333-3333-3333-3333-333333333333",
				"type":      "textNote",
				"createdAt": "2024-01-01T00:00:00Z",
				"updatedAt": "2024-01-01T00:00:00Z",
				"fields": map[string]any{
					"name": "Customer feedback on AI feature",
				},
				"relationships": map[string]any{},
				"links":         map[string]any{},
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
	cmd.AddNotesCmd(root, srv.URL, func() string { return "test-token" })

	root.SetArgs([]string{"notes", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, buf.String())
	}

	if !strings.Contains(buf.String(), "Customer feedback on AI feature") {
		t.Errorf("expected note title in output:\n%s", buf.String())
	}
}
