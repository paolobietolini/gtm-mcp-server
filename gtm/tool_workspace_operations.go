package gtm

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type BulkUpdateWorkspaceInput struct {
	WorkspaceInput
	ChangesJSON string `json:"changesJson" jsonschema:"description:JSON object containing a non-empty changes array of GTM Entity objects"`
	Confirm     bool   `json:"confirm" jsonschema:"description:Must be true to apply all proposed workspace changes"`
}

type BulkUpdateWorkspaceOutput struct {
	Success bool   `json:"success"`
	Changes any    `json:"changes,omitempty"`
	Message string `json:"message"`
}

type ResolveWorkspaceConflictInput struct {
	WorkspaceInput
	Fingerprint string `json:"fingerprint" jsonschema:"description:Fingerprint of entityInWorkspace from the merge conflict"`
	EntityJSON  string `json:"entityJson" jsonschema:"description:The fully resolved GTM Entity as JSON"`
	Confirm     bool   `json:"confirm" jsonschema:"description:Must be true to replace the conflicting entity"`
}

type WorkspaceActionOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type SyncWorkspaceInput struct {
	WorkspaceInput
	Confirm bool `json:"confirm" jsonschema:"description:Must be true to synchronize the workspace"`
}

type SyncWorkspaceOutput struct {
	Success bool                `json:"success"`
	Result  WorkspaceSyncResult `json:"result"`
	Message string              `json:"message"`
}

func registerBulkUpdateWorkspace(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "bulk_update_workspace", Description: "Apply multiple GTM entity changes atomically. Requires confirm: true; new IDs must use the new_N format."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input BulkUpdateWorkspaceInput) (*mcp.CallToolResult, BulkUpdateWorkspaceOutput, error) {
			if !input.Confirm {
				return nil, BulkUpdateWorkspaceOutput{Message: "Workspace bulk update requires confirm: true"}, nil
			}
			if strings.TrimSpace(input.ChangesJSON) == "" {
				return nil, BulkUpdateWorkspaceOutput{}, fmt.Errorf("changesJson is required")
			}
			wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
			if err != nil {
				return nil, BulkUpdateWorkspaceOutput{}, err
			}
			changes, err := wc.Client.BulkUpdateWorkspace(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ChangesJSON)
			return nil, BulkUpdateWorkspaceOutput{Success: err == nil, Changes: changes, Message: "Workspace bulk update completed"}, err
		})
}

func registerResolveWorkspaceConflict(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "resolve_workspace_conflict", Description: "Replace a conflicting workspace entity with a fully resolved entity. Requires confirm: true and the conflict fingerprint."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input ResolveWorkspaceConflictInput) (*mcp.CallToolResult, WorkspaceActionOutput, error) {
			if !input.Confirm {
				return nil, WorkspaceActionOutput{Message: "Conflict resolution requires confirm: true"}, nil
			}
			if strings.TrimSpace(input.Fingerprint) == "" || strings.TrimSpace(input.EntityJSON) == "" {
				return nil, WorkspaceActionOutput{}, fmt.Errorf("fingerprint and entityJson are required")
			}
			wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
			if err != nil {
				return nil, WorkspaceActionOutput{}, err
			}
			err = wc.Client.ResolveWorkspaceConflict(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.Fingerprint, input.EntityJSON)
			return nil, WorkspaceActionOutput{Success: err == nil, Message: "Workspace conflict resolved"}, err
		})
}

func registerSyncWorkspace(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "sync_workspace", Description: "Synchronize a workspace to the latest container version and return any merge conflicts. Requires confirm: true."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input SyncWorkspaceInput) (*mcp.CallToolResult, SyncWorkspaceOutput, error) {
			if !input.Confirm {
				return nil, SyncWorkspaceOutput{Message: "Workspace synchronization requires confirm: true"}, nil
			}
			wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
			if err != nil {
				return nil, SyncWorkspaceOutput{}, err
			}
			result, err := wc.Client.SyncWorkspace(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
			if err != nil {
				return nil, SyncWorkspaceOutput{}, err
			}
			return nil, SyncWorkspaceOutput{Success: !result.SyncError, Result: *result, Message: "Workspace synchronized"}, nil
		})
}
