package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteContainerInput is the input for delete_container tool.
type DeleteContainerInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	Confirm     bool   `json:"confirm" jsonschema:"description:Must be true to confirm deletion. This is a safety guard."`
}

// DeleteContainerOutput is the output for delete_container tool.
type DeleteContainerOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerDeleteContainer(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input DeleteContainerInput) (*mcp.CallToolResult, DeleteContainerOutput, error) {
		// Safety guard: require explicit confirmation
		if !input.Confirm {
			return nil, DeleteContainerOutput{
				Success: false,
				Message: "Deletion requires confirm: true. WARNING: This will permanently delete the container and all its contents (tags, triggers, variables, versions).",
			}, nil
		}

		// Validate container path
		if err := ValidateContainerPath(input.AccountID, input.ContainerID); err != nil {
			return nil, DeleteContainerOutput{}, err
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, DeleteContainerOutput{}, err
		}

		path := BuildContainerPath(input.AccountID, input.ContainerID)

		if err := client.DeleteContainer(ctx, path); err != nil {
			return nil, DeleteContainerOutput{}, err
		}

		return nil, DeleteContainerOutput{
			Success: true,
			Message: fmt.Sprintf("Container %s deleted successfully", input.ContainerID),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_container",
		Description: "Delete a GTM container. Requires confirm: true as a safety guard. WARNING: This permanently deletes the container and ALL its contents including tags, triggers, variables, and versions.",
	}, handler)
}
