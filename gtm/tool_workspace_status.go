package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetWorkspaceStatusInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
}
type GetWorkspaceStatusOutput struct {
	Status WorkspaceStatus `json:"status"`
}

func registerGetWorkspaceStatus(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input GetWorkspaceStatusInput) (*mcp.CallToolResult, GetWorkspaceStatusOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, GetWorkspaceStatusOutput{}, err
		}

		status, err := wc.Client.GetWorkspaceStatus(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
		if err != nil {
			return nil, GetWorkspaceStatusOutput{}, err
		}

		return nil, GetWorkspaceStatusOutput{Status: *status}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_workspace_status",
		Description: "Check if a workspace has pending changes or merge conflicts before versioning.",
	}, handler)
}
