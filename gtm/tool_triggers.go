package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListTriggersInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
}
type ListTriggersOutput struct {
	Triggers []Trigger `json:"triggers"`
}

func registerListTriggers(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListTriggersInput) (*mcp.CallToolResult, ListTriggersOutput, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, ListTriggersOutput{}, err
		}

		triggers, err := client.ListTriggers(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, ListTriggersOutput{}, err
		}

		return nil, ListTriggersOutput{Triggers: triggers}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_triggers",
		Description: "List all triggers in a GTM workspace",
	}, handler)
}
