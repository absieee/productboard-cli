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

func TestFeaturesGetSuccess(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"id":   "11111111-1111-1111-1111-111111111111",
			"type": "feature",
			"fields": map[string]any{
				"name":   "AI Call Summarisation",
				"status": map[string]any{"name": "in_progress"},
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

	root.SetArgs([]string{"features", "get", "11111111-1111-1111-1111-111111111111"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "11111111-1111-1111-1111-111111111111") {
		t.Errorf("expected entity ID in output:\n%s", buf.String())
	}
}

func TestFeaturesCreateSuccess(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"id": "22222222-2222-2222-2222-222222222222",
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	root := &cobra.Command{Use: "pb", PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil }}
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	cmd.AddFeaturesCmd(root, srv.URL, func() string { return "test-token" })

	root.SetArgs([]string{"features", "create", "--name", "New Feature"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "22222222-2222-2222-2222-222222222222") {
		t.Errorf("expected created ID in output:\n%s", buf.String())
	}
}

func TestFeaturesDeleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	root := &cobra.Command{Use: "pb", PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil }}
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	cmd.AddFeaturesCmd(root, srv.URL, func() string { return "test-token" })

	root.SetArgs([]string{"features", "delete", "11111111-1111-1111-1111-111111111111"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "deleted") {
		t.Errorf("expected 'deleted' in output:\n%s", buf.String())
	}
}

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
