// cmd/root.go
package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	apiToken   string
	jsonOutput bool
	idOnly     bool
)

// tokenTransport injects Bearer token into every request.
type tokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Content-Type", "application/json")
	return t.base.RoundTrip(req)
}

// AuthedHTTPClient returns an http.Client that injects the bearer token.
func AuthedHTTPClient(token string) *http.Client {
	return &http.Client{
		Transport: &tokenTransport{token: token, base: http.DefaultTransport},
	}
}

func loadToken() (string, error) {
	if apiToken != "" {
		return apiToken, nil
	}
	if v := os.Getenv("PRODUCTBOARD_API_TOKEN"); v != "" {
		return v, nil
	}
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(home + "/.config/pb/token")
	if err == nil && len(strings.TrimSpace(string(data))) > 0 {
		return strings.TrimSpace(string(data)), nil
	}
	return "", fmt.Errorf("no API token: set PRODUCTBOARD_API_TOKEN or write token to ~/.config/pb/token")
}

// LoadToken is the exported version for main.go.
func LoadToken() (string, error) {
	return loadToken()
}

// NewRootCmd constructs the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "pb",
		Short: "Productboard CLI",
	}
	root.PersistentFlags().StringVar(&apiToken, "token", "", "API token (overrides env/config)")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	root.PersistentFlags().BoolVar(&idOnly, "id-only", false, "Output IDs only, one per line")
	return root
}

// Execute is kept for backward compat but main.go uses NewRootCmd directly.
func Execute() error {
	return NewRootCmd().Execute()
}
