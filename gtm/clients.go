package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// ListClients returns all clients in a workspace (server-side containers only).
func (c *Client) ListClients(ctx context.Context, accountID, containerID, workspaceID string) ([]ClientInfo, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListClientsResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.Clients.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	if resp == nil {
		return []ClientInfo{}, nil
	}

	return toClients(resp.Client), nil
}

// GetClient returns a specific client by ID.
func (c *Client) GetClient(ctx context.Context, accountID, containerID, workspaceID, clientID string) (*ClientInfo, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/clients/%s",
		accountID, containerID, workspaceID, clientID)

	cl, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Client, error) {
		return c.Service.Accounts.Containers.Workspaces.Clients.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	result := toClient(cl)
	return &result, nil
}

// CreateClient creates a new client in the workspace.
func (c *Client) CreateClient(ctx context.Context, accountID, containerID, workspaceID string, input *ClientInput) (*CreatedClient, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)

	cl := &tagmanager.Client{
		Name:      input.Name,
		Type:      input.Type,
		Priority:  input.Priority,
		Parameter: toAPIParams(input.Parameter),
		Notes:     input.Notes,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Clients.Create(parent, cl).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedClient{
		ClientID:    result.ClientId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

// UpdateClient updates an existing client. It fetches the current client first to get the fingerprint.
func (c *Client) UpdateClient(ctx context.Context, path string, input *ClientInput) (*CreatedClient, error) {
	current, err := c.Service.Accounts.Containers.Workspaces.Clients.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	cl := &tagmanager.Client{
		Name:      input.Name,
		Type:      input.Type,
		Priority:  input.Priority,
		Parameter: toAPIParams(input.Parameter),
		Notes:     input.Notes,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Clients.Update(path, cl).Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedClient{
		ClientID:    result.ClientId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

// DeleteClient deletes a client from the workspace.
func (c *Client) DeleteClient(ctx context.Context, path string) error {
	err := c.Service.Accounts.Containers.Workspaces.Clients.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

func toClients(clients []*tagmanager.Client) []ClientInfo {
	result := make([]ClientInfo, 0, len(clients))
	for _, cl := range clients {
		result = append(result, toClient(cl))
	}
	return result
}

func toClient(cl *tagmanager.Client) ClientInfo {
	info := ClientInfo{
		ClientID:       cl.ClientId,
		Name:           cl.Name,
		Type:           cl.Type,
		Priority:       cl.Priority,
		Notes:          cl.Notes,
		ParentFolderID: cl.ParentFolderId,
		Path:           cl.Path,
		Fingerprint:    cl.Fingerprint,
	}
	if len(cl.Parameter) > 0 {
		info.Parameter = cl.Parameter
	}
	return info
}
