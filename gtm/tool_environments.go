package gtm

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type EnvironmentInput struct {
	AccountID     string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID   string `json:"containerId" jsonschema:"description:The GTM container ID"`
	EnvironmentID string `json:"environmentId" jsonschema:"description:The GTM environment ID"`
}

type EnvironmentContainerInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
}

type ListEnvironmentsOutput struct {
	Environments []Environment `json:"environments"`
}

type EnvironmentOutput struct {
	Environment Environment `json:"environment"`
}

type EnvironmentMutationOutput struct {
	Success     bool        `json:"success"`
	Environment Environment `json:"environment"`
	Message     string      `json:"message"`
}

type CreateEnvironmentInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	Name        string `json:"name" jsonschema:"description:Environment display name"`
	Description string `json:"description,omitempty" jsonschema:"description:Optional environment description"`
	URL         string `json:"url,omitempty" jsonschema:"description:Optional default preview URL"`
	EnableDebug bool   `json:"enableDebug,omitempty" jsonschema:"description:Enable debugging by default"`
}

type UpdateEnvironmentInput struct {
	EnvironmentInput
	Name               *string `json:"name,omitempty" jsonschema:"description:New name; omit to preserve"`
	Description        *string `json:"description,omitempty" jsonschema:"description:New description; omit to preserve or pass empty to clear"`
	URL                *string `json:"url,omitempty" jsonschema:"description:New preview URL; omit to preserve or pass empty to clear"`
	EnableDebug        *bool   `json:"enableDebug,omitempty" jsonschema:"description:New debug setting; omit to preserve"`
	ContainerVersionID *string `json:"containerVersionId,omitempty" jsonschema:"description:Container version exposed by this environment; omit to preserve or pass empty to clear"`
	WorkspaceID        *string `json:"workspaceId,omitempty" jsonschema:"description:Workspace exposed by this environment; omit to preserve or pass empty to clear"`
}

type ConfirmEnvironmentInput struct {
	EnvironmentInput
	Confirm bool `json:"confirm" jsonschema:"description:Must be true to confirm the operation"`
}

type DeleteEnvironmentOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerListEnvironments(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "list_environments", Description: "List all GTM environments in a container across every result page."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input EnvironmentContainerInput) (*mcp.CallToolResult, ListEnvironmentsOutput, error) {
			cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
			if err != nil {
				return nil, ListEnvironmentsOutput{}, err
			}
			environments, err := cc.Client.ListEnvironments(ctx, cc.AccountID, cc.ContainerID)
			return nil, ListEnvironmentsOutput{Environments: environments}, err
		})
}

func registerGetEnvironment(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "get_environment", Description: "Get a GTM environment, including its preview URL and authorization metadata."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input EnvironmentInput) (*mcp.CallToolResult, EnvironmentOutput, error) {
			cc, err := resolveEnvironment(ctx, input)
			if err != nil {
				return nil, EnvironmentOutput{}, err
			}
			environment, err := cc.Client.GetEnvironment(ctx, cc.AccountID, cc.ContainerID, input.EnvironmentID)
			if err != nil {
				return nil, EnvironmentOutput{}, err
			}
			return nil, EnvironmentOutput{Environment: *environment}, nil
		})
}

func registerCreateEnvironment(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "create_environment", Description: "Create a user GTM environment in a container."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input CreateEnvironmentInput) (*mcp.CallToolResult, EnvironmentMutationOutput, error) {
			if strings.TrimSpace(input.Name) == "" {
				return nil, EnvironmentMutationOutput{}, fmt.Errorf("name is required")
			}
			cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
			if err != nil {
				return nil, EnvironmentMutationOutput{}, err
			}
			environment, err := cc.Client.CreateEnvironment(ctx, cc.AccountID, cc.ContainerID, EnvironmentCreateConfig{
				Name: input.Name, Description: input.Description, URL: input.URL, EnableDebug: input.EnableDebug,
			})
			if err != nil {
				return nil, EnvironmentMutationOutput{}, err
			}
			return nil, EnvironmentMutationOutput{Success: true, Environment: *environment, Message: "Environment created successfully"}, nil
		})
}

func registerUpdateEnvironment(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "update_environment", Description: "Update selected GTM environment fields while preserving omitted values and checking its fingerprint."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input UpdateEnvironmentInput) (*mcp.CallToolResult, EnvironmentMutationOutput, error) {
			if input.Name == nil && input.Description == nil && input.URL == nil && input.EnableDebug == nil && input.ContainerVersionID == nil && input.WorkspaceID == nil {
				return nil, EnvironmentMutationOutput{}, fmt.Errorf("provide at least one field to update")
			}
			if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
				return nil, EnvironmentMutationOutput{}, fmt.Errorf("name cannot be empty")
			}
			cc, err := resolveEnvironment(ctx, input.EnvironmentInput)
			if err != nil {
				return nil, EnvironmentMutationOutput{}, err
			}
			environment, err := cc.Client.UpdateEnvironment(ctx, cc.AccountID, cc.ContainerID, input.EnvironmentID, EnvironmentUpdateConfig{
				Name: input.Name, Description: input.Description, URL: input.URL, EnableDebug: input.EnableDebug,
				ContainerVersionID: input.ContainerVersionID, WorkspaceID: input.WorkspaceID,
			})
			if err != nil {
				return nil, EnvironmentMutationOutput{}, err
			}
			return nil, EnvironmentMutationOutput{Success: true, Environment: *environment, Message: "Environment updated successfully"}, nil
		})
}

func registerReauthorizeEnvironment(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "reauthorize_environment", Description: "Generate a new authorization code for a GTM environment. Requires confirm: true because the previous code becomes invalid."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input ConfirmEnvironmentInput) (*mcp.CallToolResult, EnvironmentMutationOutput, error) {
			if !input.Confirm {
				return nil, EnvironmentMutationOutput{Message: "Environment reauthorization requires confirm: true"}, nil
			}
			cc, err := resolveEnvironment(ctx, input.EnvironmentInput)
			if err != nil {
				return nil, EnvironmentMutationOutput{}, err
			}
			environment, err := cc.Client.ReauthorizeEnvironment(ctx, cc.AccountID, cc.ContainerID, input.EnvironmentID)
			if err != nil {
				return nil, EnvironmentMutationOutput{}, err
			}
			return nil, EnvironmentMutationOutput{Success: true, Environment: *environment, Message: "Environment reauthorized successfully"}, nil
		})
}

func registerDeleteEnvironment(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "delete_environment", Description: "Delete a user GTM environment. Requires confirm: true."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input ConfirmEnvironmentInput) (*mcp.CallToolResult, DeleteEnvironmentOutput, error) {
			if !input.Confirm {
				return nil, DeleteEnvironmentOutput{Message: "Environment deletion requires confirm: true"}, nil
			}
			cc, err := resolveEnvironment(ctx, input.EnvironmentInput)
			if err != nil {
				return nil, DeleteEnvironmentOutput{}, err
			}
			if err := cc.Client.DeleteEnvironment(ctx, cc.AccountID, cc.ContainerID, input.EnvironmentID); err != nil {
				return nil, DeleteEnvironmentOutput{}, err
			}
			return nil, DeleteEnvironmentOutput{Success: true, Message: "Environment deleted successfully"}, nil
		})
}

func resolveEnvironment(ctx context.Context, input EnvironmentInput) (*ContainerContext, error) {
	if strings.TrimSpace(input.EnvironmentID) == "" {
		return nil, fmt.Errorf("environment ID is required")
	}
	return resolveContainer(ctx, input.AccountID, input.ContainerID)
}
