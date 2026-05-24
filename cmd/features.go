// cmd/features.go
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aveni/pb-cli/api/entities"
	"github.com/aveni/pb-cli/output"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/spf13/cobra"
)

const (
	defaultComponentID = "060e336f-2149-4c06-8e42-0c87b7d987b8" // Agent Assure
	defaultProductID   = "c48c1312-9da5-4b75-b6b1-824ce6837894" // Assurance
	defaultAATag       = "Agent Assure"                         // tag applied by `features new`
)

// parseTagList splits a comma-separated tag list and trims whitespace.
// Empty entries are skipped.
func parseTagList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// buildTimeframe builds a TimeframeFieldValue from ISO date strings and an optional granularity.
// If both startISO and endISO are empty, returns ok=false.
func buildTimeframe(startISO, endISO, granularity string) (entities.EntityCreateOrUpdateFieldValue, bool, error) {
	var v entities.EntityCreateOrUpdateFieldValue
	if startISO == "" && endISO == "" {
		return v, false, nil
	}
	tf := entities.TimeframeFieldValue{}
	if startISO != "" {
		t, err := time.Parse("2006-01-02", startISO)
		if err != nil {
			return v, false, fmt.Errorf("invalid --start %q: %w", startISO, err)
		}
		d := openapi_types.Date{Time: t}
		tf.StartDate = &d
	}
	if endISO != "" {
		t, err := time.Parse("2006-01-02", endISO)
		if err != nil {
			return v, false, fmt.Errorf("invalid --end %q: %w", endISO, err)
		}
		d := openapi_types.Date{Time: t}
		tf.EndDate = &d
	}
	if granularity != "" {
		g := entities.GranularityFieldValue(granularity)
		if !g.Valid() {
			return v, false, fmt.Errorf("invalid --granularity %q: must be year|quarter|month|day", granularity)
		}
		tf.Granularity = &g
	}
	if err := v.FromTimeframeFieldValue(tf); err != nil {
		return v, false, fmt.Errorf("build timeframe field: %w", err)
	}
	return v, true, nil
}

// buildTagsAssign builds a MultiSelectFieldAssign field value referencing tags by name.
func buildTagsAssign(tagNames []string) (entities.EntityCreateOrUpdateFieldValue, error) {
	var v entities.EntityCreateOrUpdateFieldValue
	items := make(entities.MultiSelectFieldAssign, 0, len(tagNames))
	for _, name := range tagNames {
		var item entities.MultiSelectFieldAssign_Item
		if err := item.FromSingleSelectFieldAssignByName(entities.SingleSelectFieldAssignByName{Name: name}); err != nil {
			return v, fmt.Errorf("build tag %q: %w", name, err)
		}
		items = append(items, item)
	}
	if err := v.FromMultiSelectFieldAssign(items); err != nil {
		return v, fmt.Errorf("build tags field: %w", err)
	}
	return v, nil
}

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
	var createTags string
	var createStart, createEnd, createGranularity string

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
			return runFeaturesCreate(cmd.OutOrStdout(), client, createName, createComponent, createDescription, parseTagList(createTags), createStart, createEnd, createGranularity)
		},
	}
	createCmd.Flags().StringVar(&createName, "name", "", "Feature name (required)")
	createCmd.Flags().StringVar(&createComponent, "component", defaultComponentID, "Parent component ID")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Feature description (HTML)")
	createCmd.Flags().StringVar(&createTags, "tags", "", "Comma-separated tag names to apply on create")
	createCmd.Flags().StringVar(&createStart, "start", "", "Timeframe start date (YYYY-MM-DD)")
	createCmd.Flags().StringVar(&createEnd, "end", "", "Timeframe end date (YYYY-MM-DD)")
	createCmd.Flags().StringVar(&createGranularity, "granularity", "", "Timeframe granularity: year|quarter|month|day")
	featuresCmd.AddCommand(createCmd)

	// --- new (templated: Assurance / Agent Assure component, "Agent Assure" tag) ---
	var newName string
	var newDescription string
	var newExtraTags string
	var newStart, newEnd, newGranularity string
	newCmd := &cobra.Command{
		Use:   "new",
		Short: "Create a feature under Assurance/Agent Assure with the \"Agent Assure\" tag applied",
		Long: "Templated create. Defaults the parent component to Agent Assure (under Assurance)\n" +
			"and always applies the \"Agent Assure\" tag. Add extra tags with --extra-tags.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if newName == "" {
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
			tags := append([]string{defaultAATag}, parseTagList(newExtraTags)...)
			return runFeaturesCreate(cmd.OutOrStdout(), client, newName, defaultComponentID, newDescription, tags, newStart, newEnd, newGranularity)
		},
	}
	newCmd.Flags().StringVar(&newName, "name", "", "Feature name (required)")
	newCmd.Flags().StringVar(&newDescription, "description", "", "Feature description (HTML)")
	newCmd.Flags().StringVar(&newExtraTags, "extra-tags", "", "Extra comma-separated tag names (Agent Assure is always added)")
	newCmd.Flags().StringVar(&newStart, "start", "", "Timeframe start date (YYYY-MM-DD)")
	newCmd.Flags().StringVar(&newEnd, "end", "", "Timeframe end date (YYYY-MM-DD)")
	newCmd.Flags().StringVar(&newGranularity, "granularity", "month", "Timeframe granularity: year|quarter|month|day")
	featuresCmd.AddCommand(newCmd)

	// --- update ---
	var updateName string
	var updateStatus string
	var updateDescription string
	var updateHealth string
	var updateHealthComment string
	var updateTags string
	var updateAddTags string
	var updateRemoveTags string

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
			return runFeaturesUpdate(cmd.OutOrStdout(), client, args[0],
				updateName, updateStatus, updateDescription, updateHealth, updateHealthComment,
				updateTags, parseTagList(updateAddTags), parseTagList(updateRemoveTags))
		},
	}
	updateCmd.Flags().StringVar(&updateName, "name", "", "New feature name")
	updateCmd.Flags().StringVar(&updateStatus, "status", "", "New status name")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "New description (HTML)")
	updateCmd.Flags().StringVar(&updateHealth, "health", "", "Health status: onTrack, atRisk, offTrack, notSet")
	updateCmd.Flags().StringVar(&updateHealthComment, "health-comment", "", "Comment for the health update (HTML)")
	updateCmd.Flags().StringVar(&updateTags, "tags", "", "Replace tags with this comma-separated list (use empty list to clear)")
	updateCmd.Flags().StringVar(&updateAddTags, "add-tags", "", "Comma-separated tag names to add (additive, leaves existing tags)")
	updateCmd.Flags().StringVar(&updateRemoveTags, "remove-tags", "", "Comma-separated tag names to remove")
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

func runFeaturesCreate(w io.Writer, client *entities.ClientWithResponses, name, componentID, description string, tags []string, startISO, endISO, granularity string) error {
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

	if len(tags) > 0 {
		tagsField, err := buildTagsAssign(tags)
		if err != nil {
			return err
		}
		fields["tags"] = tagsField
	}

	if tfField, ok, err := buildTimeframe(startISO, endISO, granularity); err != nil {
		return err
	} else if ok {
		fields["timeframe"] = tfField
	}

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

func runFeaturesUpdate(w io.Writer, client *entities.ClientWithResponses, id, name, statusName, description, health, healthComment, tagsReplace string, addTags, removeTags []string) error {
	// A non-empty --tags string means "replace with this list". Empty string means flag not set;
	// to clear all tags use --remove-tags with the current tag names.
	tagsToReplace := parseTagList(tagsReplace)
	tagsReplaceSet := len(tagsToReplace) > 0

	if name == "" && statusName == "" && description == "" && health == "" && healthComment == "" &&
		!tagsReplaceSet && len(addTags) == 0 && len(removeTags) == 0 {
		return fmt.Errorf("at least one of --name, --status, --description, --health, --health-comment, --tags, --add-tags, or --remove-tags must be provided")
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

	if health != "" || healthComment != "" {
		mode := entities.HealthModeEnumManual
		hv := entities.HealthUpdateFieldValue{Mode: &mode}
		if health != "" {
			hs := entities.HealthStatusEnum(health)
			if !hs.Valid() {
				return fmt.Errorf("invalid --health value %q: must be one of onTrack, atRisk, offTrack, notSet", health)
			}
			hv.Status = &hs
		}
		if healthComment != "" {
			comment := entities.RichTextFieldValue(healthComment)
			hv.Comment = &comment
		}
		var healthField entities.EntityCreateOrUpdateFieldValue
		if err := healthField.FromHealthUpdateFieldValue(hv); err != nil {
			return fmt.Errorf("build health field: %w", err)
		}
		fields["health"] = healthField
	}

	if tagsReplaceSet {
		tagsField, err := buildTagsAssign(tagsToReplace)
		if err != nil {
			return err
		}
		fields["tags"] = tagsField
	}

	// Patch operations cannot be combined with `fields` on the same field;
	// we already enforce that by routing replace through `fields` and add/remove through `patch`.
	var patch entities.EntityPatch
	if len(addTags) > 0 {
		v, err := buildTagsAssign(addTags)
		if err != nil {
			return err
		}
		var item entities.EntityPatch_Item
		if err := item.FromEntityPatchOperation(entities.EntityPatchOperation{
			Op:    entities.AddItems,
			Path:  "tags",
			Value: v,
		}); err != nil {
			return fmt.Errorf("build add-tags patch: %w", err)
		}
		patch = append(patch, item)
	}
	if len(removeTags) > 0 {
		v, err := buildTagsAssign(removeTags)
		if err != nil {
			return err
		}
		var item entities.EntityPatch_Item
		if err := item.FromEntityPatchOperation(entities.EntityPatchOperation{
			Op:    entities.RemoveItems,
			Path:  "tags",
			Value: v,
		}); err != nil {
			return fmt.Errorf("build remove-tags patch: %w", err)
		}
		patch = append(patch, item)
	}

	bodyData := &struct {
		Fields   *entities.EntityCreateOrUpdateFields `json:"fields,omitempty"`
		Metadata *entities.EntityMetadata             `json:"metadata,omitempty"`
		Patch    *entities.EntityPatch                `json:"patch,omitempty"`
	}{}
	if len(fields) > 0 {
		bodyData.Fields = &fields
	}
	if len(patch) > 0 {
		bodyData.Patch = &patch
	}
	body := entities.UpdateEntityJSONRequestBody{Data: bodyData}

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
