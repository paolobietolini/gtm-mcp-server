package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteVariableInput is the input for delete_variable tool.
type DeleteVariableInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	VariableID  string `json:"variableId" jsonschema:"description:The variable ID to delete"`
	Confirm     bool   `json:"confirm" jsonschema:"description:Must be true to confirm deletion. This is a safety guard."`
}

// DeleteVariableOutput is the output for delete_variable tool.
type DeleteVariableOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerDeleteVariable(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input DeleteVariableInput) (*mcp.CallToolResult, DeleteVariableOutput, error) {
		// Safety guard: require explicit confirmation
		if !input.Confirm {
			return nil, DeleteVariableOutput{
				Success: false,
				Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
			}, nil
		}

		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, DeleteVariableOutput{}, err
		}

		// Validate variable ID
		if input.VariableID == "" {
			return nil, DeleteVariableOutput{}, fmt.Errorf("variable ID is required")
		}

		path := BuildVariablePath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)

		if err := wc.Client.DeleteVariable(ctx, path); err != nil {
			return nil, DeleteVariableOutput{}, err
		}

		return nil, DeleteVariableOutput{
			Success: true,
			Message: fmt.Sprintf("Variable %s deleted successfully", input.VariableID),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_variable",
		Description: "Delete a variable from a workspace. Requires confirm: true as a safety guard.",
	}, handler)
}
