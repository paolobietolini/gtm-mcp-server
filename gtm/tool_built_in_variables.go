package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// -- List Built-In Variables --

type ListBuiltInVariablesInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
}

type ListBuiltInVariablesOutput struct {
	BuiltInVariables []BuiltInVariable `json:"builtInVariables"`
}

func registerListBuiltInVariables(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListBuiltInVariablesInput) (*mcp.CallToolResult, ListBuiltInVariablesOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, ListBuiltInVariablesOutput{}, err
		}

		vars, err := wc.Client.ListBuiltInVariables(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
		if err != nil {
			return nil, ListBuiltInVariablesOutput{}, err
		}

		return nil, ListBuiltInVariablesOutput{BuiltInVariables: vars}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_built_in_variables",
		Description: "List all enabled built-in variables in a GTM workspace",
	}, handler)
}

// -- Enable Built-In Variables --

type EnableBuiltInVariablesInput struct {
	AccountID   string   `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string   `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string   `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Types       []string `json:"types" jsonschema:"description:Array of built-in variable types to enable (e.g. eventName, clientName, requestPath, pageUrl, event)"`
}

type EnableBuiltInVariablesOutput struct {
	Success          bool              `json:"success"`
	BuiltInVariables []BuiltInVariable `json:"builtInVariables"`
	Message          string            `json:"message"`
}

func registerEnableBuiltInVariables(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input EnableBuiltInVariablesInput) (*mcp.CallToolResult, EnableBuiltInVariablesOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, EnableBuiltInVariablesOutput{}, err
		}

		if len(input.Types) == 0 {
			return nil, EnableBuiltInVariablesOutput{}, fmt.Errorf("at least one built-in variable type is required")
		}

		vars, err := wc.Client.EnableBuiltInVariables(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.Types)
		if err != nil {
			return nil, EnableBuiltInVariablesOutput{}, err
		}

		return nil, EnableBuiltInVariablesOutput{
			Success:          true,
			BuiltInVariables: vars,
			Message:          "Built-in variables enabled successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "enable_built_in_variables",
		Description: "Enable one or more built-in variable types in a GTM workspace",
	}, handler)
}

// -- Disable Built-In Variables --

type DisableBuiltInVariablesInput struct {
	AccountID   string   `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string   `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string   `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Types       []string `json:"types" jsonschema:"description:Array of built-in variable types to disable"`
	Confirm     bool     `json:"confirm" jsonschema:"description:Must be true to confirm disabling. This is a safety guard."`
}

type DisableBuiltInVariablesOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerDisableBuiltInVariables(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input DisableBuiltInVariablesInput) (*mcp.CallToolResult, DisableBuiltInVariablesOutput, error) {
		if !input.Confirm {
			return nil, DisableBuiltInVariablesOutput{
				Success: false,
				Message: "Disabling requires confirm: true. This is a safety guard to prevent accidental changes.",
			}, nil
		}

		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, DisableBuiltInVariablesOutput{}, err
		}

		if len(input.Types) == 0 {
			return nil, DisableBuiltInVariablesOutput{}, fmt.Errorf("at least one built-in variable type is required")
		}

		if err := wc.Client.DisableBuiltInVariables(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.Types); err != nil {
			return nil, DisableBuiltInVariablesOutput{}, err
		}

		return nil, DisableBuiltInVariablesOutput{
			Success: true,
			Message: "Built-in variables disabled successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "disable_built_in_variables",
		Description: "Disable one or more built-in variable types. Requires confirm: true as a safety guard.",
	}, handler)
}
