package gtm

import (
	"context"
	"encoding/json"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

type WorkspaceSyncResult struct {
	MergeConflict bool `json:"mergeConflict"`
	SyncError     bool `json:"syncError"`
	Conflicts     any  `json:"conflicts,omitempty"`
}

func (c *Client) BulkUpdateWorkspace(ctx context.Context, accountID, containerID, workspaceID, changesJSON string) (any, error) {
	var proposed tagmanager.ProposedChange
	if err := json.Unmarshal([]byte(changesJSON), &proposed); err != nil {
		return nil, fmt.Errorf("invalid changesJson: %w", err)
	}
	if len(proposed.Changes) == 0 {
		return nil, fmt.Errorf("changesJson must contain a non-empty changes array")
	}
	response, err := c.Service.Accounts.Containers.Workspaces.BulkUpdate(BuildWorkspacePath(accountID, containerID, workspaceID), &proposed).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	return response.Changes, nil
}

func (c *Client) ResolveWorkspaceConflict(ctx context.Context, accountID, containerID, workspaceID, fingerprint, entityJSON string) error {
	var entity tagmanager.Entity
	if err := json.Unmarshal([]byte(entityJSON), &entity); err != nil {
		return fmt.Errorf("invalid entityJson: %w", err)
	}
	return mapGoogleError(c.Service.Accounts.Containers.Workspaces.ResolveConflict(BuildWorkspacePath(accountID, containerID, workspaceID), &entity).Fingerprint(fingerprint).Context(ctx).Do())
}

func (c *Client) SyncWorkspace(ctx context.Context, accountID, containerID, workspaceID string) (*WorkspaceSyncResult, error) {
	response, err := c.Service.Accounts.Containers.Workspaces.Sync(BuildWorkspacePath(accountID, containerID, workspaceID)).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := &WorkspaceSyncResult{Conflicts: response.MergeConflict}
	if response.SyncStatus != nil {
		result.MergeConflict = response.SyncStatus.MergeConflict
		result.SyncError = response.SyncStatus.SyncError
	}
	return result, nil
}
