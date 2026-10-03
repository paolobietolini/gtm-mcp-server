package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// WorkspacePreview is the ephemeral version produced by Google's quick
// preview endpoint together with synchronization and compiler status.
type WorkspacePreview struct {
	CompilerError bool                     `json:"compilerError"`
	SyncStatus    *WorkspacePreviewStatus  `json:"syncStatus,omitempty"`
	Version       *ContainerVersionDetails `json:"containerVersion,omitempty"`
}

type WorkspacePreviewStatus struct {
	MergeConflict bool `json:"mergeConflict"`
	SyncError     bool `json:"syncError"`
}

func (c *Client) GetWorkspace(ctx context.Context, accountID, containerID, workspaceID string) (*Workspace, error) {
	path := BuildWorkspacePath(accountID, containerID, workspaceID)
	workspace, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Workspace, error) {
		return c.Service.Accounts.Containers.Workspaces.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toWorkspace(workspace)
	return &result, nil
}

func (c *Client) UpdateWorkspace(ctx context.Context, accountID, containerID, workspaceID string, name, description *string) (*Workspace, error) {
	if name == nil && description == nil {
		return nil, fmt.Errorf("provide at least one of name or description")
	}
	path := BuildWorkspacePath(accountID, containerID, workspaceID)
	current, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Workspace, error) {
		return c.Service.Accounts.Containers.Workspaces.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	update := &tagmanager.Workspace{Name: current.Name, Description: current.Description}
	if name != nil {
		update.Name = *name
	}
	if description != nil {
		update.Description = *description
		if *description == "" {
			update.ForceSendFields = append(update.ForceSendFields, "Description")
		}
	}
	updated, err := c.Service.Accounts.Containers.Workspaces.Update(path, update).
		Fingerprint(current.Fingerprint).
		Context(ctx).
		Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toWorkspace(updated)
	return &result, nil
}

func (c *Client) DeleteWorkspace(ctx context.Context, accountID, containerID, workspaceID string) error {
	path := BuildWorkspacePath(accountID, containerID, workspaceID)
	err := c.Service.Accounts.Containers.Workspaces.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

func (c *Client) QuickPreviewWorkspace(ctx context.Context, accountID, containerID, workspaceID string) (*WorkspacePreview, error) {
	path := BuildWorkspacePath(accountID, containerID, workspaceID)
	preview, err := retryWithBackoff(ctx, 3, func() (*tagmanager.QuickPreviewResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.QuickPreview(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := &WorkspacePreview{CompilerError: preview.CompilerError}
	if preview.SyncStatus != nil {
		result.SyncStatus = &WorkspacePreviewStatus{
			MergeConflict: preview.SyncStatus.MergeConflict,
			SyncError:     preview.SyncStatus.SyncError,
		}
	}
	if preview.ContainerVersion != nil {
		version := toContainerVersionDetails(preview.ContainerVersion)
		result.Version = &version
	}
	return result, nil
}
