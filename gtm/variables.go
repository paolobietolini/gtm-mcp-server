package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Variable is a simplified representation of a GTM variable.
type Variable struct {
	VariableID string `json:"variableId"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Path       string `json:"path"`
}

// ListVariables returns all variables in a workspace.
func (c *Client) ListVariables(ctx context.Context, accountID, containerID, workspaceID string) ([]Variable, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListVariablesResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.Variables.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return toVariables(resp.Variable), nil
}

func toVariables(variables []*tagmanager.Variable) []Variable {
	result := make([]Variable, 0, len(variables))
	for _, v := range variables {
		result = append(result, Variable{
			VariableID: v.VariableId,
			Name:       v.Name,
			Type:       v.Type,
			Path:       v.Path,
		})
	}
	return result
}
