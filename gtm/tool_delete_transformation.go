package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteTransformationInput is the input for delete_transformation tool.
type DeleteTransformationInput struct {
	AccountID        string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID      string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID      string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TransformationID string `json:"transformationId" jsonschema:"description:The transformation ID to delete"`
	Confirm          bool   `json:"confirm" jsonschema:"description:Must be true to confirm deletion. This is a safety guard."`
}

// DeleteTransformationOutput is the output for delete_transformation tool.
type DeleteTransformationOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerDeleteTransformation(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input DeleteTransformationInput) (*mcp.CallToolResult, DeleteTransformationOutput, error) {
		if !input.Confirm {
			return nil, DeleteTransformationOutput{
				Success: false,
				Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
			}, nil
		}

		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, DeleteTransformationOutput{}, err
		}

		if input.TransformationID == "" {
			return nil, DeleteTransformationOutput{}, fmt.Errorf("transformation ID is required")
		}

		path := BuildTransformationPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TransformationID)

		if err := wc.Client.DeleteTransformation(ctx, path); err != nil {
			return nil, DeleteTransformationOutput{}, err
		}

		return nil, DeleteTransformationOutput{
			Success: true,
			Message: fmt.Sprintf("Transformation %s deleted successfully", input.TransformationID),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_transformation",
		Description: "Delete a transformation from a workspace. Requires confirm: true as a safety guard. Server-side containers only.",
	}, handler)
}
