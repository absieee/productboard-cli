// cmd/releases_test.go
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

func TestReleasesListRendersTable(t *testing.T) {
	resp := map[string]any{
		"data": []map[string]any{
			{
				"id":   "11111111-1111-1111-1111-111111111111",
				"type": "release",
				"fields": map[string]any{
					"name": "v1.0",
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
	cmd.AddReleasesCmd(root, srv.URL, func() string { return "test-token" })

	root.SetArgs([]string{"releases", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, buf.String())
	}

	if !strings.Contains(buf.String(), "v1.0") {
		t.Errorf("expected release name in output:\n%s", buf.String())
	}
}

func TestReleasesCreateSuccess(t *testing.T) {
	createResp := map[string]any{
		"data": map[string]any{
			"id":   "22222222-2222-2222-2222-222222222222",
			"type": "release",
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createResp)
	}))
	defer srv.Close()

	root := &cobra.Command{Use: "pb", PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil }}
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	cmd.AddReleasesCmd(root, srv.URL, func() string { return "test-token" })

	root.SetArgs([]string{"releases", "create", "--name", "Release 1.0"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, buf.String())
	}

	if !strings.Contains(buf.String(), "22222222-2222-2222-2222-222222222222") {
		t.Errorf("expected release ID in output:\n%s", buf.String())
	}
}
