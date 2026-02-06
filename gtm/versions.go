package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// CreateVersion creates a new container version from a workspace.
func (c *Client) CreateVersion(ctx context.Context, accountID, containerID, workspaceID string, input *VersionInput) (*CreatedVersion, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)

	req := &tagmanager.CreateContainerVersionRequestVersionOptions{
		Name:  input.Name,
		Notes: input.Notes,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.CreateVersion(parent, req).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	if result == nil || result.ContainerVersion == nil {
		return nil, fmt.Errorf("no version created - workspace may have no changes")
	}

	return &CreatedVersion{
		VersionID: result.ContainerVersion.ContainerVersionId,
		Name:      result.ContainerVersion.Name,
		Path:      result.ContainerVersion.Path,
	}, nil
}

// PublishVersion publishes a container version to make it live.
func (c *Client) PublishVersion(ctx context.Context, accountID, containerID, versionID string) (*PublishedVersion, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/versions/%s",
		accountID, containerID, versionID)

	result, err := c.Service.Accounts.Containers.Versions.Publish(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &PublishedVersion{
		VersionID: result.ContainerVersion.ContainerVersionId,
		Name:      result.ContainerVersion.Name,
		Path:      result.ContainerVersion.Path,
	}, nil
}

// GetWorkspaceStatus checks if a workspace has changes to publish.
func (c *Client) GetWorkspaceStatus(ctx context.Context, accountID, containerID, workspaceID string) (*WorkspaceStatus, error) {
	path := BuildWorkspacePath(accountID, containerID, workspaceID)

	status, err := c.Service.Accounts.Containers.Workspaces.GetStatus(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &WorkspaceStatus{
		HasChanges:    len(status.WorkspaceChange) > 0,
		HasConflicts:  len(status.MergeConflict) > 0,
		ChangeCount:   len(status.WorkspaceChange),
		ConflictCount: len(status.MergeConflict),
	}, nil
}

// PublishedVersion represents the result of publishing a version.
type PublishedVersion struct {
	VersionID string `json:"containerVersionId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}

// WorkspaceStatus represents the status of a workspace.
type WorkspaceStatus struct {
	HasChanges    bool `json:"hasChanges"`
	HasConflicts  bool `json:"hasConflicts"`
	ChangeCount   int  `json:"changeCount"`
	ConflictCount int  `json:"conflictCount"`
}
