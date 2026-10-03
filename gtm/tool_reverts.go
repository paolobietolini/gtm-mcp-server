package gtm

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RevertWorkspaceEntityInput struct {
	WorkspaceInput
	ResourceType string `json:"resourceType" jsonschema:"description:Resource family: builtInVariable, client, tag, template, transformation, trigger, variable, or zone"`
	ResourceID   string `json:"resourceId" jsonschema:"description:Entity ID, or the built-in variable type for builtInVariable"`
	Confirm      bool   `json:"confirm" jsonschema:"description:Must be true to discard the entity's workspace changes"`
}

type RevertWorkspaceEntityOutput struct {
	Success bool          `json:"success"`
	Result  *RevertResult `json:"result,omitempty"`
	Message string        `json:"message"`
}

func registerRevertWorkspaceEntity(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "revert_workspace_entity", Description: "Discard workspace changes to a built-in variable, client, tag, template, transformation, trigger, variable, or zone. Requires confirm: true and uses the current fingerprint."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input RevertWorkspaceEntityInput) (*mcp.CallToolResult, RevertWorkspaceEntityOutput, error) {
			if !input.Confirm {
				return nil, RevertWorkspaceEntityOutput{Message: "Entity revert requires confirm: true"}, nil
			}
			if strings.TrimSpace(input.ResourceType) == "" || strings.TrimSpace(input.ResourceID) == "" {
				return nil, RevertWorkspaceEntityOutput{}, fmt.Errorf("resourceType and resourceId are required")
			}
			wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
			if err != nil {
				return nil, RevertWorkspaceEntityOutput{}, err
			}
			result, err := wc.Client.RevertWorkspaceEntity(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ResourceType, input.ResourceID)
			return nil, RevertWorkspaceEntityOutput{Success: err == nil, Result: result, Message: "Workspace entity reverted successfully"}, err
		})
}
