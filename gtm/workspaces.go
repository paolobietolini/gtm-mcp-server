package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Workspace is a simplified representation of a GTM workspace.
type Workspace struct {
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path"`
}

// ListWorkspaces returns all workspaces in a container.
func (c *Client) ListWorkspaces(ctx context.Context, accountID, containerID string) ([]Workspace, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s", accountID, containerID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListWorkspacesResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return toWorkspaces(resp.Workspace), nil
}

func toWorkspaces(workspaces []*tagmanager.Workspace) []Workspace {
	result := make([]Workspace, 0, len(workspaces))
	for _, w := range workspaces {
		result = append(result, Workspace{
			WorkspaceID: w.WorkspaceId,
			Name:        w.Name,
			Description: w.Description,
			Path:        w.Path,
		})
	}
	return result
}
