package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteTagInput is the input for delete_tag tool.
type DeleteTagInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TagID       string `json:"tagId" jsonschema:"description:The tag ID to delete"`
	Confirm     bool   `json:"confirm" jsonschema:"description:Must be true to confirm deletion. This is a safety guard."`
}

// DeleteTagOutput is the output for delete_tag tool.
type DeleteTagOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerDeleteTag(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input DeleteTagInput) (*mcp.CallToolResult, DeleteTagOutput, error) {
		// Safety guard: require explicit confirmation
		if !input.Confirm {
			return nil, DeleteTagOutput{
				Success: false,
				Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
			}, nil
		}

		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, DeleteTagOutput{}, err
		}

		// Validate tag ID
		if input.TagID == "" {
			return nil, DeleteTagOutput{}, fmt.Errorf("tag ID is required")
		}

		path := BuildTagPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)

		if err := wc.Client.DeleteTag(ctx, path); err != nil {
			return nil, DeleteTagOutput{}, err
		}

		return nil, DeleteTagOutput{
			Success: true,
			Message: fmt.Sprintf("Tag %s deleted successfully", input.TagID),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_tag",
		Description: "Delete a tag from a workspace. Requires confirm: true as a safety guard.",
	}, handler)
}
