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
	openapi_types "github.com/oapi-codegen/runtime/types"
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

	// --- get ---
	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a feature by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tok := tokenFn()
			httpClient := AuthedHTTPClient(tok)
			client, err := entities.NewClientWithResponses(serverURL,
				entities.WithHTTPClient(httpClient),
			)
			if err != nil {
				return fmt.Errorf("build client: %w", err)
			}
			return runFeaturesGet(cmd.OutOrStdout(), client, args[0])
		},
	}
	featuresCmd.AddCommand(getCmd)

	// --- create ---
	var createName string
	var createComponent string
	var createDescription string

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new feature",
		RunE: func(cmd *cobra.Command, args []string) error {
			if createName == "" {
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
			return runFeaturesCreate(cmd.OutOrStdout(), client, createName, createComponent, createDescription)
		},
	}
	createCmd.Flags().StringVar(&createName, "name", "", "Feature name (required)")
	createCmd.Flags().StringVar(&createComponent, "component", defaultComponentID, "Parent component ID")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Feature description (HTML)")
	featuresCmd.AddCommand(createCmd)

	// --- update ---
	var updateName string
	var updateStatus string
	var updateDescription string

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a feature by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tok := tokenFn()
			httpClient := AuthedHTTPClient(tok)
			client, err := entities.NewClientWithResponses(serverURL,
				entities.WithHTTPClient(httpClient),
			)
			if err != nil {
				return fmt.Errorf("build client: %w", err)
			}
			return runFeaturesUpdate(cmd.OutOrStdout(), client, args[0], updateName, updateStatus, updateDescription)
		},
	}
	updateCmd.Flags().StringVar(&updateName, "name", "", "New feature name")
	updateCmd.Flags().StringVar(&updateStatus, "status", "", "New status name")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "New description (HTML)")
	featuresCmd.AddCommand(updateCmd)

	// --- delete ---
	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a feature by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tok := tokenFn()
			httpClient := AuthedHTTPClient(tok)
			client, err := entities.NewClientWithResponses(serverURL,
				entities.WithHTTPClient(httpClient),
			)
			if err != nil {
				return fmt.Errorf("build client: %w", err)
			}
			return runFeaturesDelete(cmd.OutOrStdout(), client, args[0])
		},
	}
	featuresCmd.AddCommand(deleteCmd)

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

func runFeaturesGet(w io.Writer, client *entities.ClientWithResponses, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID %q: %w", id, err)
	}

	params := &entities.GetEntityParams{}
	resp, err := client.GetEntityWithResponse(context.Background(), uid, params)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected API response %d: %s", resp.StatusCode(), string(resp.Body))
	}

	return output.JSON(w, resp.JSON200)
}

func runFeaturesCreate(w io.Writer, client *entities.ClientWithResponses, name, componentID, description string) error {
	featureType := entities.Feature

	fields := entities.EntityCreateOrUpdateFields{}

	var nameField entities.EntityCreateOrUpdateFieldValue
	if err := nameField.FromNameFieldValue(name); err != nil {
		return fmt.Errorf("build name field: %w", err)
	}
	fields["name"] = nameField

	if description != "" {
		var descField entities.EntityCreateOrUpdateFieldValue
		if err := descField.FromRichTextFieldValue(description); err != nil {
			return fmt.Errorf("build description field: %w", err)
		}
		fields["description"] = descField
	}

	// Always set owner to abhishek.sharma@aveni.ai
	var ma entities.MemberFieldAssign
	if err := ma.FromMemberAssignByEmail(entities.MemberAssignByEmail{
		Email: openapi_types.Email("abhishek.sharma@aveni.ai"),
	}); err != nil {
		return fmt.Errorf("build owner field: %w", err)
	}
	var ownerField entities.EntityCreateOrUpdateFieldValue
	if err := ownerField.FromMemberFieldAssign(ma); err != nil {
		return fmt.Errorf("build owner field value: %w", err)
	}
	fields["owner"] = ownerField

	// Build parent relationship
	var rels []entities.EntityRelationshipCreate
	if componentID != "" {
		componentUID, err := uuid.Parse(componentID)
		if err != nil {
			return fmt.Errorf("invalid component ID %q: %w", componentID, err)
		}
		parentType := entities.Parent
		rels = []entities.EntityRelationshipCreate{
			{
				Target: entities.ResourceReferenceAssign{Id: &componentUID},
				Type:   &parentType,
			},
		}
	}

	body := entities.CreateEntityJSONRequestBody{
		Data: &struct {
			Fields        *entities.EntityCreateOrUpdateFields `json:"fields,omitempty"`
			Metadata      *entities.EntityMetadata             `json:"metadata,omitempty"`
			Relationships *[]entities.EntityRelationshipCreate `json:"relationships,omitempty"`
			Type          *entities.EntityType                 `json:"type,omitempty"`
		}{
			Fields:        &fields,
			Type:          &featureType,
			Relationships: &rels,
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
		fmt.Fprintf(w, "Created feature: %s\n", resp.JSON201.Data.Id)
	} else {
		fmt.Fprintln(w, "Feature created successfully")
	}
	return nil
}

func runFeaturesUpdate(w io.Writer, client *entities.ClientWithResponses, id, name, statusName, description string) error {
	if name == "" && statusName == "" && description == "" {
		return fmt.Errorf("at least one of --name, --status, or --description must be provided")
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID %q: %w", id, err)
	}

	fields := entities.EntityCreateOrUpdateFields{}

	if name != "" {
		var nameField entities.EntityCreateOrUpdateFieldValue
		if err := nameField.FromNameFieldValue(name); err != nil {
			return fmt.Errorf("build name field: %w", err)
		}
		fields["name"] = nameField
	}

	if description != "" {
		var descField entities.EntityCreateOrUpdateFieldValue
		if err := descField.FromRichTextFieldValue(description); err != nil {
			return fmt.Errorf("build description field: %w", err)
		}
		fields["description"] = descField
	}

	if statusName != "" {
		var sa entities.StatusFieldAssign
		if err := sa.FromStatusFieldAssignByName(entities.StatusFieldAssignByName{Name: statusName}); err != nil {
			return fmt.Errorf("build status field: %w", err)
		}
		var statusField entities.EntityCreateOrUpdateFieldValue
		if err := statusField.FromStatusFieldAssign(sa); err != nil {
			return fmt.Errorf("build status field value: %w", err)
		}
		fields["status"] = statusField
	}

	body := entities.UpdateEntityJSONRequestBody{
		Data: &struct {
			Fields   *entities.EntityCreateOrUpdateFields `json:"fields,omitempty"`
			Metadata *entities.EntityMetadata             `json:"metadata,omitempty"`
			Patch    *entities.EntityPatch                `json:"patch,omitempty"`
		}{
			Fields: &fields,
		},
	}

	resp, err := client.UpdateEntityWithResponse(context.Background(), uid, body)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected API response %d: %s", resp.StatusCode(), string(resp.Body))
	}

	if jsonOutput {
		return output.JSON(w, resp.JSON200)
	}

	if resp.JSON200.Data != nil {
		fmt.Fprintf(w, "Updated feature: %s\n", resp.JSON200.Data.Id)
	} else {
		fmt.Fprintln(w, "Feature updated successfully")
	}
	return nil
}

func runFeaturesDelete(w io.Writer, client *entities.ClientWithResponses, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID %q: %w", id, err)
	}

	resp, err := client.DeleteEntityWithResponse(context.Background(), uid)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	if resp.StatusCode() != 204 {
		return fmt.Errorf("unexpected API response %d: %s", resp.StatusCode(), string(resp.Body))
	}

	fmt.Fprintf(w, "deleted %s\n", id)
	return nil
}
