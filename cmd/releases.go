// cmd/releases.go
package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/aveni/pb-cli/api/entities"
	"github.com/aveni/pb-cli/output"
	"github.com/spf13/cobra"
)

// AddReleasesCmd registers the `releases` sub-command tree on parent.
func AddReleasesCmd(parent *cobra.Command, serverURL string, tokenFn func() string) {
	releasesCmd := &cobra.Command{
		Use:   "releases",
		Short: "Manage Productboard releases",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			tok := tokenFn()
			httpClient := AuthedHTTPClient(tok)
			client, err := entities.NewClientWithResponses(serverURL,
				entities.WithHTTPClient(httpClient),
			)
			if err != nil {
				return fmt.Errorf("build client: %w", err)
			}
			return runReleasesList(cmd.OutOrStdout(), client)
		},
	}

	var releaseName string
	var releaseDescription string

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new release",
		RunE: func(cmd *cobra.Command, args []string) error {
			if releaseName == "" {
				return fmt.Errorf("--name is required")
			}
			tok := tokenFn()
			httpClient := AuthedHTTPClient(tok)
			client, err := entities.NewClientWithResponses(serverURL,
				entities.WithHTTPClient(httpClient),
			)
			if err != nil {
				return fmt.Errorf("build client: %w", err)
			}
			return runReleasesCreate(cmd.OutOrStdout(), client, releaseName, releaseDescription)
		},
	}
	createCmd.Flags().StringVar(&releaseName, "name", "", "Name of the release (required)")
	createCmd.Flags().StringVar(&releaseDescription, "description", "", "Description of the release")

	releasesCmd.AddCommand(listCmd)
	releasesCmd.AddCommand(createCmd)
	parent.AddCommand(releasesCmd)
}

func runReleasesList(w io.Writer, client *entities.ClientWithResponses) error {
	releaseType := entities.Release
	types := []entities.EntityType{releaseType}

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

func runReleasesCreate(w io.Writer, client *entities.ClientWithResponses, name, description string) error {
	releaseType := entities.Release

	var nameField entities.EntityCreateOrUpdateFieldValue
	if err := nameField.FromNameFieldValue(name); err != nil {
		return fmt.Errorf("build name field: %w", err)
	}

	fields := entities.EntityCreateOrUpdateFields{
		"name": nameField,
	}

	if description != "" {
		var descField entities.EntityCreateOrUpdateFieldValue
		if err := descField.FromRichTextFieldValue(description); err != nil {
			return fmt.Errorf("build description field: %w", err)
		}
		fields["description"] = descField
	}

	body := entities.CreateEntityJSONRequestBody{
		Data: &struct {
			Fields        *entities.EntityCreateOrUpdateFields  `json:"fields,omitempty"`
			Metadata      *entities.EntityMetadata              `json:"metadata,omitempty"`
			Relationships *[]entities.EntityRelationshipCreate  `json:"relationships,omitempty"`
			Type          *entities.EntityType                  `json:"type,omitempty"`
		}{
			Fields: &fields,
			Type:   &releaseType,
		},
	}

	resp, err := client.CreateEntityWithResponse(context.Background(), body)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if resp.JSON201 == nil {
		return fmt.Errorf("unexpected API response %d: %s", resp.StatusCode(), string(resp.Body))
	}

	if jsonOutput {
		return output.JSON(w, resp.JSON201)
	}

	if resp.JSON201.Data != nil {
		fmt.Fprintf(w, "Created release: %s\n", resp.JSON201.Data.Id)
	} else {
		fmt.Fprintln(w, "Release created successfully")
	}
	return nil
}
