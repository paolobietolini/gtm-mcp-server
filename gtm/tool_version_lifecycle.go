package gtm

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VersionLifecycleInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	VersionID   string `json:"versionId" jsonschema:"description:The container version ID"`
}

type ConfirmVersionLifecycleInput struct {
	VersionLifecycleInput
	Confirm bool `json:"confirm" jsonschema:"description:Must be true to confirm the version state change"`
}

type UpdateVersionInput struct {
	VersionLifecycleInput
	Name        *string `json:"name,omitempty" jsonschema:"description:New version name; omit to preserve or pass empty to clear"`
	Description *string `json:"description,omitempty" jsonschema:"description:New version description; omit to preserve or pass empty to clear"`
}

type VersionLifecycleOutput struct {
	Success bool              `json:"success"`
	Version *VersionLifecycle `json:"version,omitempty"`
	Message string            `json:"message"`
}

func registerDeleteVersion(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "delete_version", Description: "Soft-delete a container version. Requires confirm: true and can be reversed with undelete_version."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input ConfirmVersionLifecycleInput) (*mcp.CallToolResult, VersionLifecycleOutput, error) {
			if !input.Confirm {
				return nil, VersionLifecycleOutput{Message: "Version deletion requires confirm: true"}, nil
			}
			cc, err := resolveVersionLifecycle(ctx, input.VersionLifecycleInput)
			if err != nil {
				return nil, VersionLifecycleOutput{}, err
			}
			err = cc.Client.DeleteVersion(ctx, cc.AccountID, cc.ContainerID, input.VersionID)
			return nil, VersionLifecycleOutput{Success: err == nil, Message: "Version deleted successfully"}, err
		})
}

func registerUndeleteVersion(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "undelete_version", Description: "Restore a soft-deleted container version. Requires confirm: true."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input ConfirmVersionLifecycleInput) (*mcp.CallToolResult, VersionLifecycleOutput, error) {
			if !input.Confirm {
				return nil, VersionLifecycleOutput{Message: "Version restoration requires confirm: true"}, nil
			}
			cc, err := resolveVersionLifecycle(ctx, input.VersionLifecycleInput)
			if err != nil {
				return nil, VersionLifecycleOutput{}, err
			}
			version, err := cc.Client.UndeleteVersion(ctx, cc.AccountID, cc.ContainerID, input.VersionID)
			return nil, VersionLifecycleOutput{Success: err == nil, Version: version, Message: "Version restored successfully"}, err
		})
}

func registerSetLatestVersion(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "set_latest_version", Description: "Set a container version as Latest without publishing it Live. Requires confirm: true."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input ConfirmVersionLifecycleInput) (*mcp.CallToolResult, VersionLifecycleOutput, error) {
			if !input.Confirm {
				return nil, VersionLifecycleOutput{Message: "Changing the Latest version requires confirm: true"}, nil
			}
			cc, err := resolveVersionLifecycle(ctx, input.VersionLifecycleInput)
			if err != nil {
				return nil, VersionLifecycleOutput{}, err
			}
			version, err := cc.Client.SetLatestVersion(ctx, cc.AccountID, cc.ContainerID, input.VersionID)
			return nil, VersionLifecycleOutput{Success: err == nil, Version: version, Message: "Latest version updated successfully"}, err
		})
}

func registerUpdateVersion(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "update_version", Description: "Update a saved version's name or description with fingerprint concurrency control."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input UpdateVersionInput) (*mcp.CallToolResult, VersionLifecycleOutput, error) {
			if input.Name == nil && input.Description == nil {
				return nil, VersionLifecycleOutput{}, fmt.Errorf("provide name or description to update")
			}
			cc, err := resolveVersionLifecycle(ctx, input.VersionLifecycleInput)
			if err != nil {
				return nil, VersionLifecycleOutput{}, err
			}
			version, err := cc.Client.UpdateVersion(ctx, cc.AccountID, cc.ContainerID, input.VersionID, input.Name, input.Description)
			return nil, VersionLifecycleOutput{Success: err == nil, Version: version, Message: "Version updated successfully"}, err
		})
}

func resolveVersionLifecycle(ctx context.Context, input VersionLifecycleInput) (*ContainerContext, error) {
	if strings.TrimSpace(input.VersionID) == "" {
		return nil, fmt.Errorf("version ID is required")
	}
	return resolveContainer(ctx, input.AccountID, input.ContainerID)
}
