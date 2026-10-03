package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GoogleTagConfigInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	ConfigID    string `json:"gtagConfigId" jsonschema:"description:The Google tag config ID"`
}

type ListGoogleTagConfigsOutput struct {
	Configs []GoogleTagConfig `json:"gtagConfigs"`
}

type GoogleTagConfigOutput struct {
	Success bool            `json:"success,omitempty"`
	Config  GoogleTagConfig `json:"gtagConfig"`
	Message string          `json:"message,omitempty"`
}

type CreateGoogleTagConfigInput struct {
	AccountID      string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID    string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID    string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Type           string `json:"type" jsonschema:"description:The Google tag configuration type"`
	ParametersJSON string `json:"parametersJson,omitempty" jsonschema:"description:Google tag parameters as a JSON array; each parameter has type, key, value, list, or map fields"`
}

type UpdateGoogleTagConfigInput struct {
	GoogleTagConfigInput
	Type           *string `json:"type,omitempty" jsonschema:"description:New config type; omit to preserve"`
	ParametersJSON *string `json:"parametersJson,omitempty" jsonschema:"description:New parameters as a JSON array; omit to preserve or pass [] to clear"`
}

type DeleteGoogleTagConfigInput struct {
	GoogleTagConfigInput
	Confirm bool `json:"confirm" jsonschema:"description:Must be true to confirm deletion"`
}

type DeleteGoogleTagConfigOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerListGoogleTagConfigs(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "list_google_tag_configs", Description: "List Google tag configurations in a workspace across every result page."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input WorkspaceInput) (*mcp.CallToolResult, ListGoogleTagConfigsOutput, error) {
			wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
			if err != nil {
				return nil, ListGoogleTagConfigsOutput{}, err
			}
			configs, err := wc.Client.ListGoogleTagConfigs(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
			return nil, ListGoogleTagConfigsOutput{Configs: configs}, err
		})
}

func registerGetGoogleTagConfig(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "get_google_tag_config", Description: "Get a Google tag configuration by ID."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input GoogleTagConfigInput) (*mcp.CallToolResult, GoogleTagConfigOutput, error) {
			wc, err := resolveGoogleTagConfig(ctx, input)
			if err != nil {
				return nil, GoogleTagConfigOutput{}, err
			}
			config, err := wc.Client.GetGoogleTagConfig(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ConfigID)
			if err != nil {
				return nil, GoogleTagConfigOutput{}, err
			}
			return nil, GoogleTagConfigOutput{Config: *config}, nil
		})
}

func registerCreateGoogleTagConfig(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "create_google_tag_config", Description: "Create a Google tag configuration in a workspace."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input CreateGoogleTagConfigInput) (*mcp.CallToolResult, GoogleTagConfigOutput, error) {
			if strings.TrimSpace(input.Type) == "" {
				return nil, GoogleTagConfigOutput{}, fmt.Errorf("type is required")
			}
			var parameters []Parameter
			if strings.TrimSpace(input.ParametersJSON) != "" {
				if err := json.Unmarshal([]byte(input.ParametersJSON), &parameters); err != nil {
					return nil, GoogleTagConfigOutput{}, fmt.Errorf("invalid parametersJson: %w", err)
				}
			}
			wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
			if err != nil {
				return nil, GoogleTagConfigOutput{}, err
			}
			config, err := wc.Client.CreateGoogleTagConfig(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, GoogleTagConfigCreateConfig{Type: input.Type, Parameter: parameters})
			if err != nil {
				return nil, GoogleTagConfigOutput{}, err
			}
			return nil, GoogleTagConfigOutput{Success: true, Config: *config, Message: "Google tag config created successfully"}, nil
		})
}

func registerUpdateGoogleTagConfig(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "update_google_tag_config", Description: "Update selected Google tag configuration fields while preserving omitted values and checking its fingerprint."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input UpdateGoogleTagConfigInput) (*mcp.CallToolResult, GoogleTagConfigOutput, error) {
			if input.Type == nil && input.ParametersJSON == nil {
				return nil, GoogleTagConfigOutput{}, fmt.Errorf("provide type or parameter to update")
			}
			if input.Type != nil && strings.TrimSpace(*input.Type) == "" {
				return nil, GoogleTagConfigOutput{}, fmt.Errorf("type cannot be empty")
			}
			var parameters *[]Parameter
			if input.ParametersJSON != nil {
				var parsed []Parameter
				if err := json.Unmarshal([]byte(*input.ParametersJSON), &parsed); err != nil {
					return nil, GoogleTagConfigOutput{}, fmt.Errorf("invalid parametersJson: %w", err)
				}
				parameters = &parsed
			}
			wc, err := resolveGoogleTagConfig(ctx, input.GoogleTagConfigInput)
			if err != nil {
				return nil, GoogleTagConfigOutput{}, err
			}
			config, err := wc.Client.UpdateGoogleTagConfig(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ConfigID, GoogleTagConfigUpdateConfig{Type: input.Type, Parameter: parameters})
			if err != nil {
				return nil, GoogleTagConfigOutput{}, err
			}
			return nil, GoogleTagConfigOutput{Success: true, Config: *config, Message: "Google tag config updated successfully"}, nil
		})
}

func registerDeleteGoogleTagConfig(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "delete_google_tag_config", Description: "Delete a Google tag configuration. Requires confirm: true."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input DeleteGoogleTagConfigInput) (*mcp.CallToolResult, DeleteGoogleTagConfigOutput, error) {
			if !input.Confirm {
				return nil, DeleteGoogleTagConfigOutput{Message: "Google tag config deletion requires confirm: true"}, nil
			}
			wc, err := resolveGoogleTagConfig(ctx, input.GoogleTagConfigInput)
			if err != nil {
				return nil, DeleteGoogleTagConfigOutput{}, err
			}
			if err := wc.Client.DeleteGoogleTagConfig(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ConfigID); err != nil {
				return nil, DeleteGoogleTagConfigOutput{}, err
			}
			return nil, DeleteGoogleTagConfigOutput{Success: true, Message: "Google tag config deleted successfully"}, nil
		})
}

func resolveGoogleTagConfig(ctx context.Context, input GoogleTagConfigInput) (*WorkspaceContext, error) {
	if strings.TrimSpace(input.ConfigID) == "" {
		return nil, fmt.Errorf("google tag config ID is required")
	}
	return resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
}
