package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListVariablesInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
}
type ListVariablesOutput struct {
	Variables []Variable `json:"variables"`
}

func registerListVariables(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListVariablesInput) (*mcp.CallToolResult, ListVariablesOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, ListVariablesOutput{}, err
		}

		variables, err := wc.Client.ListVariables(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
		if err != nil {
			return nil, ListVariablesOutput{}, err
		}

		return nil, ListVariablesOutput{Variables: variables}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_variables",
		Description: "List all variables in a GTM workspace",
	}, handler)
}
