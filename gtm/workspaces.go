package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Workspace is a simplified representation of a GTM workspace.
type Workspace struct {
	AccountID     string `json:"accountId,omitempty"`
	ContainerID   string `json:"containerId,omitempty"`
	WorkspaceID   string `json:"workspaceId"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	Path          string `json:"path"`
	TagManagerURL string `json:"tagManagerUrl,omitempty"`
}

// ListWorkspaces returns all workspaces in a container.
func (c *Client) ListWorkspaces(ctx context.Context, accountID, containerID string) ([]Workspace, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s", accountID, containerID)

	result := make([]Workspace, 0)
	pageToken := ""
	for {
		resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListWorkspacesResponse, error) {
			return c.Service.Accounts.Containers.Workspaces.List(parent).PageToken(pageToken).Context(ctx).Do()
		})
		if err != nil {
			return nil, mapGoogleError(err)
		}
		result = append(result, toWorkspaces(resp.Workspace)...)
		pageToken = resp.NextPageToken
		if pageToken == "" {
			return result, nil
		}
	}
}

func toWorkspaces(workspaces []*tagmanager.Workspace) []Workspace {
	result := make([]Workspace, 0, len(workspaces))
	for _, w := range workspaces {
		result = append(result, toWorkspace(w))
	}
	return result
}

func toWorkspace(workspace *tagmanager.Workspace) Workspace {
	return Workspace{
		AccountID:     workspace.AccountId,
		ContainerID:   workspace.ContainerId,
		WorkspaceID:   workspace.WorkspaceId,
		Name:          workspace.Name,
		Description:   workspace.Description,
		Fingerprint:   workspace.Fingerprint,
		Path:          workspace.Path,
		TagManagerURL: workspace.TagManagerUrl,
	}
}
