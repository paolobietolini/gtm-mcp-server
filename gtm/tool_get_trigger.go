package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetTriggerInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TriggerID   string `json:"triggerId" jsonschema:"description:The trigger ID to retrieve"`
}
type GetTriggerOutput struct {
	Trigger Trigger `json:"trigger"`
}

func registerGetTrigger(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input GetTriggerInput) (*mcp.CallToolResult, GetTriggerOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, GetTriggerOutput{}, err
		}

		if input.TriggerID == "" {
			return nil, GetTriggerOutput{}, fmt.Errorf("triggerId is required")
		}

		trigger, err := wc.Client.GetTrigger(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TriggerID)
		if err != nil {
			return nil, GetTriggerOutput{}, err
		}

		return nil, GetTriggerOutput{Trigger: *trigger}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_trigger",
		Description: "Get a specific trigger by ID",
	}, handler)
}
