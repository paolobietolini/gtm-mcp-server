package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// ListBuiltInVariables returns all enabled built-in variables in a workspace.
func (c *Client) ListBuiltInVariables(ctx context.Context, accountID, containerID, workspaceID string) ([]BuiltInVariable, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListEnabledBuiltInVariablesResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.BuiltInVariables.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	if resp == nil {
		return []BuiltInVariable{}, nil
	}

	return toBuiltInVariables(resp.BuiltInVariable), nil
}

// EnableBuiltInVariables enables one or more built-in variable types in a workspace.
func (c *Client) EnableBuiltInVariables(ctx context.Context, accountID, containerID, workspaceID string, types []string) ([]BuiltInVariable, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.CreateBuiltInVariableResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.BuiltInVariables.Create(parent).Type(types...).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return toBuiltInVariables(resp.BuiltInVariable), nil
}

// DisableBuiltInVariables disables one or more built-in variable types in a workspace.
func (c *Client) DisableBuiltInVariables(ctx context.Context, accountID, containerID, workspaceID string, types []string) error {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/built_in_variables", accountID, containerID, workspaceID)

	err := c.Service.Accounts.Containers.Workspaces.BuiltInVariables.Delete(path).Type(types...).Context(ctx).Do()
	return mapGoogleError(err)
}

func toBuiltInVariables(vars []*tagmanager.BuiltInVariable) []BuiltInVariable {
	result := make([]BuiltInVariable, 0, len(vars))
	for _, v := range vars {
		result = append(result, BuiltInVariable{
			Name: v.Name,
			Type: v.Type,
			Path: v.Path,
		})
	}
	return result
}
