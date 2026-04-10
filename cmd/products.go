// cmd/products.go
package cmd

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/aveni/pb-cli/api/entities"
	"github.com/aveni/pb-cli/output"
	"github.com/spf13/cobra"
)

// AddProductsCmd registers the `products` sub-command tree on parent.
// serverURL is the base API URL; tokenFn is called at runtime to get the bearer token.
func AddProductsCmd(parent *cobra.Command, serverURL string, tokenFn func() string) {
	productsCmd := &cobra.Command{
		Use:   "products",
		Short: "Manage Productboard products",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all products",
		RunE: func(cmd *cobra.Command, args []string) error {
			tok := tokenFn()
			httpClient := AuthedHTTPClient(tok)
			client, err := entities.NewClientWithResponses(serverURL,
				entities.WithHTTPClient(httpClient),
			)
			if err != nil {
				return fmt.Errorf("build client: %w", err)
			}
			return runProductsList(cmd.OutOrStdout(), client)
		},
	}

	hierarchyCmd := &cobra.Command{
		Use:   "hierarchy",
		Short: "Show product/component hierarchy",
		RunE: func(cmd *cobra.Command, args []string) error {
			tok := tokenFn()
			httpClient := AuthedHTTPClient(tok)
			client, err := entities.NewClientWithResponses(serverURL,
				entities.WithHTTPClient(httpClient),
			)
			if err != nil {
				return fmt.Errorf("build client: %w", err)
			}
			return runProductsHierarchy(cmd.OutOrStdout(), client)
		},
	}

	productsCmd.AddCommand(listCmd)
	productsCmd.AddCommand(hierarchyCmd)
	parent.AddCommand(productsCmd)
}

func runProductsList(w io.Writer, client *entities.ClientWithResponses) error {
	productType := entities.Product
	types := []entities.EntityType{productType}

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

// hierarchyNode represents a node in the product/component tree.
type hierarchyNode struct {
	entity   entities.Entity
	children []*hierarchyNode
}

func runProductsHierarchy(w io.Writer, client *entities.ClientWithResponses) error {
	// Fetch products
	productType := entities.Product
	componentType := entities.Component
	types := []entities.EntityType{productType, componentType}

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

	// JSON output mode
	if jsonOutput {
		return output.JSON(w, resp.JSON200)
	}

	// Build ID → node map
	nodeMap := make(map[string]*hierarchyNode, len(*data))
	for i := range *data {
		e := (*data)[i]
		id := e.Id.String()
		nodeMap[id] = &hierarchyNode{entity: e}
	}

	// Build parent → children map using relationships
	var roots []*hierarchyNode
	for i := range *data {
		e := (*data)[i]
		nodeID := e.Id.String()
		node := nodeMap[nodeID]

		parentID := findParentID(e)
		if parentID == "" {
			roots = append(roots, node)
		} else if parentNode, ok := nodeMap[parentID]; ok {
			parentNode.children = append(parentNode.children, node)
		} else {
			// Parent not in result set — treat as root
			roots = append(roots, node)
		}
	}

	// Sort roots and children by name for stable output
	sortNodes(roots)

	// Print tree
	for _, root := range roots {
		printHierarchyNode(w, root, 0)
	}
	return nil
}

// findParentID extracts the parent entity ID from an entity's relationships.
func findParentID(e entities.Entity) string {
	if e.Relationships == nil || e.Relationships.Data == nil {
		return ""
	}
	for _, rel := range *e.Relationships.Data {
		if rel.Type == entities.Parent {
			return rel.Target.Id.String()
		}
	}
	return ""
}

// sortNodes sorts a slice of hierarchy nodes by entity name alphabetically.
func sortNodes(nodes []*hierarchyNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return extractName(nodes[i].entity) < extractName(nodes[j].entity)
	})
	for _, n := range nodes {
		sortNodes(n.children)
	}
}

// printHierarchyNode prints a node and its children with indentation.
func printHierarchyNode(w io.Writer, node *hierarchyNode, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	name := extractName(node.entity)
	id := node.entity.Id.String()
	fmt.Fprintf(w, "%s• %s (%s)\n", indent, name, id)
	for _, child := range node.children {
		printHierarchyNode(w, child, depth+1)
	}
}
