package gtm

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type WorkspaceInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
}

type WorkspaceOutput struct {
	Workspace Workspace `json:"workspace"`
}

type UpdateWorkspaceInput struct {
	AccountID   string  `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string  `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string  `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Name        *string `json:"name,omitempty" jsonschema:"description:New workspace name; omit to preserve it"`
	Description *string `json:"description,omitempty" jsonschema:"description:New description; omit to preserve or pass an empty string to clear"`
}

type UpdateWorkspaceOutput struct {
	Success   bool      `json:"success"`
	Workspace Workspace `json:"workspace"`
	Message   string    `json:"message"`
}

type DeleteWorkspaceInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Confirm     bool   `json:"confirm" jsonschema:"description:Must be true to confirm workspace deletion"`
}

type DeleteWorkspaceOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type QuickPreviewWorkspaceOutput struct {
	Preview WorkspacePreview `json:"preview"`
}

func registerGetWorkspace(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_workspace",
		Description: "Get workspace metadata including its current fingerprint.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input WorkspaceInput) (*mcp.CallToolResult, WorkspaceOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, WorkspaceOutput{}, err
		}
		workspace, err := wc.Client.GetWorkspace(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
		if err != nil {
			return nil, WorkspaceOutput{}, err
		}
		return nil, WorkspaceOutput{Workspace: *workspace}, nil
	})
}

func registerUpdateWorkspace(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_workspace",
		Description: "Update a workspace name or description while preserving omitted fields and checking its fingerprint.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateWorkspaceInput) (*mcp.CallToolResult, UpdateWorkspaceOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, UpdateWorkspaceOutput{}, err
		}
		if input.Name == nil && input.Description == nil {
			return nil, UpdateWorkspaceOutput{}, fmt.Errorf("provide at least one of name or description")
		}
		if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
			return nil, UpdateWorkspaceOutput{}, fmt.Errorf("name cannot be empty")
		}
		workspace, err := wc.Client.UpdateWorkspace(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.Name, input.Description)
		if err != nil {
			return nil, UpdateWorkspaceOutput{}, err
		}
		return nil, UpdateWorkspaceOutput{
			Success:   true,
			Workspace: *workspace,
			Message:   "Workspace updated successfully",
		}, nil
	})
}

func registerDeleteWorkspace(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_workspace",
		Description: "Delete a workspace and its pending changes. Requires confirm: true.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteWorkspaceInput) (*mcp.CallToolResult, DeleteWorkspaceOutput, error) {
		if !input.Confirm {
			return nil, DeleteWorkspaceOutput{Message: "Workspace deletion requires confirm: true"}, nil
		}
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, DeleteWorkspaceOutput{}, err
		}
		if err := wc.Client.DeleteWorkspace(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID); err != nil {
			return nil, DeleteWorkspaceOutput{}, err
		}
		return nil, DeleteWorkspaceOutput{Success: true, Message: "Workspace deleted successfully"}, nil
	})
}

func registerQuickPreviewWorkspace(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "quick_preview_workspace",
		Description: "Compile an ephemeral preview version of a workspace without saving or publishing it.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input WorkspaceInput) (*mcp.CallToolResult, QuickPreviewWorkspaceOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, QuickPreviewWorkspaceOutput{}, err
		}
		preview, err := wc.Client.QuickPreviewWorkspace(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
		if err != nil {
			return nil, QuickPreviewWorkspaceOutput{}, err
		}
		return nil, QuickPreviewWorkspaceOutput{Preview: *preview}, nil
	})
}
