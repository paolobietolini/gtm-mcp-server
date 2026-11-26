package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Variable is a simplified representation of a GTM variable.
type Variable struct {
	VariableID     string `json:"variableId"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Parameter      any    `json:"parameter,omitempty"`
	ParentFolderID string `json:"parentFolderId,omitempty"`
	Notes          string `json:"notes,omitempty"`
	Path           string `json:"path"`
}

// ListVariables returns all variables in a workspace.
func (c *Client) ListVariables(ctx context.Context, accountID, containerID, workspaceID string) ([]Variable, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	items, err := listAll(ctx, func(token string) (*tagmanager.ListVariablesResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.Variables.List(parent).PageToken(token).Context(ctx).Do()
	}, func(r *tagmanager.ListVariablesResponse) ([]*tagmanager.Variable, string) {
		return r.Variable, r.NextPageToken
	})
	if err != nil {
		return nil, err
	}

	return toVariables(items), nil
}

// GetVariable returns a specific variable by ID.
func (c *Client) GetVariable(ctx context.Context, accountID, containerID, workspaceID, variableID string) (*Variable, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/variables/%s",
		accountID, containerID, workspaceID, variableID)

	v, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Variable, error) {
		return c.Service.Accounts.Containers.Workspaces.Variables.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	result := toVariable(v)
	return &result, nil
}

func toVariables(variables []*tagmanager.Variable) []Variable {
	result := make([]Variable, 0, len(variables))
	for _, v := range variables {
		result = append(result, toVariable(v))
	}
	return result
}

func toVariable(v *tagmanager.Variable) Variable {
	variable := Variable{
		VariableID:     v.VariableId,
		Name:           v.Name,
		Type:           v.Type,
		ParentFolderID: v.ParentFolderId,
		Notes:          v.Notes,
		Path:           v.Path,
	}
	if len(v.Parameter) > 0 {
		variable.Parameter = v.Parameter
	}
	return variable
}
