// cmd/features_test.go
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

func TestFeaturesListRendersTable(t *testing.T) {
	// Minimal v2 entities list response
	resp := map[string]any{
		"data": []map[string]any{
			{
				"id":   "11111111-1111-1111-1111-111111111111",
				"type": "feature",
				"fields": map[string]any{
					"name":   "AI Call Summarisation",
					"status": map[string]any{"name": "in_progress"},
					"owner":  map[string]any{"email": "abs@aveni.ai"},
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
	cmd.AddFeaturesCmd(root, srv.URL, func() string { return "test-token" })

	root.SetArgs([]string{"features", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, buf.String())
	}

	if !strings.Contains(buf.String(), "AI Call Summarisation") {
		t.Errorf("expected feature name in output:\n%s", buf.String())
	}
}
