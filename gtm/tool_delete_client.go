package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteClientInput is the input for delete_client tool.
type DeleteClientInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	ClientID    string `json:"clientId" jsonschema:"description:The client ID to delete"`
	Confirm     bool   `json:"confirm" jsonschema:"description:Must be true to confirm deletion. This is a safety guard."`
}

// DeleteClientOutput is the output for delete_client tool.
type DeleteClientOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerDeleteClient(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input DeleteClientInput) (*mcp.CallToolResult, DeleteClientOutput, error) {
		if !input.Confirm {
			return nil, DeleteClientOutput{
				Success: false,
				Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
			}, nil
		}

		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, DeleteClientOutput{}, err
		}

		if input.ClientID == "" {
			return nil, DeleteClientOutput{}, fmt.Errorf("client ID is required")
		}

		path := BuildClientPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ClientID)

		if err := wc.Client.DeleteClient(ctx, path); err != nil {
			return nil, DeleteClientOutput{}, err
		}

		return nil, DeleteClientOutput{
			Success: true,
			Message: fmt.Sprintf("Client %s deleted successfully", input.ClientID),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_client",
		Description: "Delete a client from a workspace. Requires confirm: true as a safety guard. Server-side containers only.",
	}, handler)
}
