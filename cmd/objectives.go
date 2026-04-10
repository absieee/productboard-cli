// cmd/objectives.go
package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/aveni/pb-cli/api/entities"
	"github.com/aveni/pb-cli/output"
	"github.com/spf13/cobra"
)

// AddObjectivesCmd registers the `objectives` sub-command tree on parent.
func AddObjectivesCmd(parent *cobra.Command, serverURL string, tokenFn func() string) {
	objectivesCmd := &cobra.Command{
		Use:   "objectives",
		Short: "Manage Productboard objectives",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List objectives",
		RunE: func(cmd *cobra.Command, args []string) error {
			tok := tokenFn()
			httpClient := AuthedHTTPClient(tok)
			client, err := entities.NewClientWithResponses(serverURL,
				entities.WithHTTPClient(httpClient),
			)
			if err != nil {
				return fmt.Errorf("build client: %w", err)
			}
			return runObjectivesList(cmd.OutOrStdout(), client)
		},
	}

	objectivesCmd.AddCommand(listCmd)
	parent.AddCommand(objectivesCmd)
}

func runObjectivesList(w io.Writer, client *entities.ClientWithResponses) error {
	objectiveType := entities.Objective
	types := []entities.EntityType{objectiveType}

	params := &entities.ListEntitiesParams{
		Type: &types,
	}

	resp, err := client.ListEntitiesWithResponse(context.Background(), params)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected API response %d: %s", resp.StatusCode(), string(resp.Body))
	}

	data := resp.JSON200.Data
	if data == nil {
		data = &[]entities.Entity{}
	}

	if jsonOutput {
		return output.JSON(w, resp.JSON200)
	}

	if idOnly {
		for _, e := range *data {
			fmt.Fprintln(w, e.Id)
		}
		return nil
	}

	headers := []string{"ID", "NAME"}
	rows := make([][]string, 0, len(*data))
	for _, e := range *data {
		id := e.Id.String()
		name := extractName(e)
		rows = append(rows, []string{id, name})
	}
	output.Table(w, headers, rows)
	return nil
}
