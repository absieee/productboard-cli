// cmd/notes.go
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aveni/pb-cli/api/notes"
	"github.com/aveni/pb-cli/output"
	"github.com/spf13/cobra"
)

// AddNotesCmd registers the `notes` sub-command tree on parent.
func AddNotesCmd(parent *cobra.Command, serverURL string, tokenFn func() string) {
	notesCmd := &cobra.Command{
		Use:   "notes",
		Short: "Manage Productboard notes",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			tok := tokenFn()
			httpClient := AuthedHTTPClient(tok)
			client, err := notes.NewClientWithResponses(serverURL,
				notes.WithHTTPClient(httpClient),
			)
			if err != nil {
				return fmt.Errorf("build client: %w", err)
			}
			return runNotesList(cmd.OutOrStdout(), client)
		},
	}

	notesCmd.AddCommand(listCmd)
	parent.AddCommand(notesCmd)
}

func runNotesList(w io.Writer, client *notes.ClientWithResponses) error {
	resp, err := client.ListNotesWithResponse(context.Background(), &notes.ListNotesParams{})
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected API response %d: %s", resp.StatusCode(), string(resp.Body))
	}

	data := resp.JSON200.Data
	if data == nil {
		data = &[]notes.Note{}
	}

	if jsonOutput {
		return output.JSON(w, resp.JSON200)
	}

	if idOnly {
		for _, n := range *data {
			fmt.Fprintln(w, n.Id)
		}
		return nil
	}

	headers := []string{"ID", "TITLE"}
	rows := make([][]string, 0, len(*data))
	for _, n := range *data {
		id := n.Id.String()
		title := extractNoteTitle(n)
		rows = append(rows, []string{id, title})
	}
	output.Table(w, headers, rows)
	return nil
}

// extractNoteTitle pulls the "name" field from note fields.
func extractNoteTitle(n notes.Note) string {
	val, ok := n.Fields["name"]
	if !ok {
		return ""
	}
	name, err := val.AsNameFieldValue()
	if err == nil {
		return name
	}
	// Fallback: try raw JSON unmarshal as string
	raw, merr := json.Marshal(val)
	if merr == nil {
		var s string
		if jerr := json.Unmarshal(raw, &s); jerr == nil {
			return s
		}
	}
	return ""
}
