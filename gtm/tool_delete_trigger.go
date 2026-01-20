package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteTriggerInput is the input for delete_trigger tool.
type DeleteTriggerInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TriggerID   string `json:"triggerId" jsonschema:"description:The trigger ID to delete"`
	Confirm     bool   `json:"confirm" jsonschema:"description:Must be true to confirm deletion. This is a safety guard."`
}

// DeleteTriggerOutput is the output for delete_trigger tool.
type DeleteTriggerOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerDeleteTrigger(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input DeleteTriggerInput) (*mcp.CallToolResult, DeleteTriggerOutput, error) {
		// Safety guard: require explicit confirmation
		if !input.Confirm {
			return nil, DeleteTriggerOutput{
				Success: false,
				Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
			}, nil
		}

		// Validate workspace path
		if err := ValidateWorkspacePath(input.AccountID, input.ContainerID, input.WorkspaceID); err != nil {
			return nil, DeleteTriggerOutput{}, err
		}

		// Validate trigger ID
		if input.TriggerID == "" {
			return nil, DeleteTriggerOutput{}, fmt.Errorf("trigger ID is required")
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, DeleteTriggerOutput{}, err
		}

		path := BuildTriggerPath(input.AccountID, input.ContainerID, input.WorkspaceID, input.TriggerID)

		if err := client.DeleteTrigger(ctx, path); err != nil {
			return nil, DeleteTriggerOutput{}, err
		}

		return nil, DeleteTriggerOutput{
			Success: true,
			Message: fmt.Sprintf("Trigger %s deleted successfully", input.TriggerID),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_trigger",
		Description: "Delete a trigger from a workspace. Requires confirm: true as a safety guard. Note: Triggers that are members of a trigger group cannot be deleted until the trigger group is deleted first.",
	}, handler)
}
