// cmd/root.go
package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const BaseURL = "https://api.productboard.com/v2"

var (
	apiToken      string
	jsonOutput    bool
	idOnly        bool
	resolvedToken string // set in PersistentPreRunE, read by subcommands
)

// tokenTransport injects Bearer token into every request.
type tokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return t.base.RoundTrip(req)
}

// AuthedHTTPClient returns an http.Client that injects the bearer token.
func AuthedHTTPClient(token string) *http.Client {
	return &http.Client{
		Transport: &tokenTransport{token: token, base: http.DefaultTransport},
	}
}

func resolveToken() (string, error) {
	if apiToken != "" {
		return apiToken, nil
	}
	if v := os.Getenv("PRODUCTBOARD_API_TOKEN"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "pb", "token"))
	if err == nil && len(strings.TrimSpace(string(data))) > 0 {
		return strings.TrimSpace(string(data)), nil
	}
	return "", fmt.Errorf("no API token: set PRODUCTBOARD_API_TOKEN or write token to ~/.config/pb/token")
}

// ResolvedToken returns the token after PersistentPreRunE has run.
// Subcommands use this to get the authenticated token.
func ResolvedToken() string {
	return resolvedToken
}

// NewRootCmd constructs the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "pb",
		Short: "Productboard CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			tok, err := resolveToken()
			if err != nil {
				return err
			}
			resolvedToken = tok
			return nil
		},
	}
	root.PersistentFlags().StringVar(&apiToken, "token", "", "API token (overrides env/config)")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	root.PersistentFlags().BoolVar(&idOnly, "id-only", false, "Output IDs only, one per line")
	return root
}
