// cmd/features.go
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aveni/pb-cli/api/entities"
	"github.com/aveni/pb-cli/output"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const (
	defaultComponentID = "060e336f-2149-4c06-8e42-0c87b7d987b8" // Agent Assure
	defaultProductID   = "c48c1312-9da5-4b75-b6b1-824ce6837894" // Assurance
)

// AddFeaturesCmd registers the `features` sub-command tree on parent.
// serverURL is the base API URL; tokenFn is called at runtime to get the bearer token.
func AddFeaturesCmd(parent *cobra.Command, serverURL string, tokenFn func() string) {
	var componentID string
	var productID string
	var statusFilter string

	featuresCmd := &cobra.Command{
		Use:   "features",
		Short: "Manage Productboard features",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List features",
		RunE: func(cmd *cobra.Command, args []string) error {
			tok := tokenFn()
			httpClient := AuthedHTTPClient(tok)
			client, err := entities.NewClientWithResponses(serverURL,
				entities.WithHTTPClient(httpClient),
			)
			if err != nil {
				return fmt.Errorf("build client: %w", err)
			}
			return runFeaturesList(cmd.OutOrStdout(), client, componentID, productID, statusFilter)
		},
	}

	listCmd.Flags().StringVar(&componentID, "component", defaultComponentID, "Filter by component ID (parent)")
	listCmd.Flags().StringVar(&productID, "product", defaultProductID, "Filter by product ID (parent, used when --component is empty)")
	listCmd.Flags().StringVar(&statusFilter, "status", "", "Filter by status name")

	featuresCmd.AddCommand(listCmd)
	parent.AddCommand(featuresCmd)
}

func runFeaturesList(w io.Writer, client *entities.ClientWithResponses, componentID, productID, statusFilter string) error {
	featureType := entities.Feature
	types := []entities.EntityType{featureType}

	params := &entities.ListEntitiesParams{
		Type: &types,
	}

	// Determine parent filter: prefer component, fall back to product
	parentIDStr := componentID
	if parentIDStr == "" {
		parentIDStr = productID
	}
	if parentIDStr != "" {
		uid, err := uuid.Parse(parentIDStr)
		if err != nil {
			return fmt.Errorf("invalid parent ID %q: %w", parentIDStr, err)
		}
		params.ParentId = &uid
	}

	if statusFilter != "" {
		params.StatusName = &statusFilter
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

	// JSON output mode
	if jsonOutput {
		return output.JSON(w, resp.JSON200)
	}

	// ID-only mode
	if idOnly {
		for _, e := range *data {
			fmt.Fprintln(w, e.Id)
		}
		return nil
	}

	// Table output
	headers := []string{"ID", "NAME", "STATUS", "OWNER"}
	rows := make([][]string, 0, len(*data))
	for _, e := range *data {
		id := e.Id.String()
		name := extractName(e)
		status := extractStatus(e)
		owner := extractOwner(e)
		rows = append(rows, []string{id, name, status, owner})
	}
	output.Table(w, headers, rows)
	return nil
}

// extractName pulls the "name" field from entity fields.
func extractName(e entities.Entity) string {
	if e.Fields == nil {
		return ""
	}
	fields := *e.Fields
	val, ok := fields["name"]
	if !ok {
		return ""
	}
	// EntityFieldValue is a union; for name it's a plain string (NameFieldValue = string)
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

// extractStatus pulls the "status" field name from entity fields.
func extractStatus(e entities.Entity) string {
	if e.Fields == nil {
		return ""
	}
	fields := *e.Fields
	val, ok := fields["status"]
	if !ok {
		return ""
	}
	sv, err := val.AsStatusFieldValue()
	if err != nil {
		return ""
	}
	return sv.Name
}

// extractOwner pulls the "owner" (member) field email from entity fields.
func extractOwner(e entities.Entity) string {
	if e.Fields == nil {
		return ""
	}
	fields := *e.Fields
	val, ok := fields["owner"]
	if !ok {
		return ""
	}
	mv, err := val.AsMemberFieldValue()
	if err != nil {
		return ""
	}
	return mv.Email
}
