package gtm

import (
	"context"
	"fmt"
)

// WorkspaceContext holds a validated workspace path and an authenticated GTM client.
type WorkspaceContext struct {
	Client      *Client
	AccountID   string
	ContainerID string
	WorkspaceID string
}

// ContainerContext holds a validated container path and an authenticated GTM client.
type ContainerContext struct {
	Client      *Client
	AccountID   string
	ContainerID string
}

// resolveWorkspace validates the workspace path IDs and creates an authenticated GTM client.
// Use this in any tool handler that operates at the workspace level.
func resolveWorkspace(ctx context.Context, accountID, containerID, workspaceID string) (*WorkspaceContext, error) {
	if err := ValidateWorkspacePath(accountID, containerID, workspaceID); err != nil {
		return nil, err
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	return &WorkspaceContext{
		Client:      client,
		AccountID:   accountID,
		ContainerID: containerID,
		WorkspaceID: workspaceID,
	}, nil
}

// resolveContainer validates the container path IDs and creates an authenticated GTM client.
// Use this in any tool handler that operates at the container level.
func resolveContainer(ctx context.Context, accountID, containerID string) (*ContainerContext, error) {
	if err := ValidateContainerPath(accountID, containerID); err != nil {
		return nil, err
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	return &ContainerContext{
		Client:      client,
		AccountID:   accountID,
		ContainerID: containerID,
	}, nil
}

// resolveAccount validates the account ID and creates an authenticated GTM client.
// Use this in any tool handler that operates at the account level.
func resolveAccount(ctx context.Context, accountID string) (*Client, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	return getClient(ctx)
}

// WorkspacePath returns the formatted workspace path string.
func (wc *WorkspaceContext) WorkspacePath() string {
	return BuildWorkspacePath(wc.AccountID, wc.ContainerID, wc.WorkspaceID)
}

// ContainerPath returns the formatted container path string.
func (cc *ContainerContext) ContainerPath() string {
	return BuildContainerPath(cc.AccountID, cc.ContainerID)
}
