package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListWorkspacesInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
}
type ListWorkspacesOutput struct {
	Workspaces []Workspace `json:"workspaces"`
}

func registerListWorkspaces(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListWorkspacesInput) (*mcp.CallToolResult, ListWorkspacesOutput, error) {
		cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
		if err != nil {
			return nil, ListWorkspacesOutput{}, err
		}

		workspaces, err := cc.Client.ListWorkspaces(ctx, cc.AccountID, cc.ContainerID)
		if err != nil {
			return nil, ListWorkspacesOutput{}, err
		}

		return nil, ListWorkspacesOutput{Workspaces: workspaces}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_workspaces",
		Description: "List all workspaces in a GTM container",
	}, handler)
}
