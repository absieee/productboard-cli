// cmd/products_test.go
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

func TestProductsListRendersTable(t *testing.T) {
	resp := map[string]any{
		"data": []map[string]any{
			{
				"id":     "11111111-1111-1111-1111-111111111111",
				"type":   "product",
				"fields": map[string]any{"name": "Assurance"},
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
	cmd.AddProductsCmd(root, srv.URL, func() string { return "test-token" })

	root.SetArgs([]string{"products", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "Assurance") {
		t.Errorf("expected Assurance in output:\n%s", buf.String())
	}
}
