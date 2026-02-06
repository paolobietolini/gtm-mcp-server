package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetVariableInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	VariableID  string `json:"variableId" jsonschema:"description:The variable ID to retrieve"`
}
type GetVariableOutput struct {
	Variable Variable `json:"variable"`
}

func registerGetVariable(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input GetVariableInput) (*mcp.CallToolResult, GetVariableOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, GetVariableOutput{}, err
		}

		if input.VariableID == "" {
			return nil, GetVariableOutput{}, fmt.Errorf("variableId is required")
		}

		variable, err := wc.Client.GetVariable(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)
		if err != nil {
			return nil, GetVariableOutput{}, err
		}

		return nil, GetVariableOutput{Variable: *variable}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_variable",
		Description: "Get a specific variable by ID",
	}, handler)
}
