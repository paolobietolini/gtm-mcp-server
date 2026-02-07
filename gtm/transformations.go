package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// ListTransformations returns all transformations in a workspace (server-side containers only).
func (c *Client) ListTransformations(ctx context.Context, accountID, containerID, workspaceID string) ([]TransformationInfo, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListTransformationsResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.Transformations.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	if resp == nil {
		return []TransformationInfo{}, nil
	}

	return toTransformations(resp.Transformation), nil
}

// GetTransformation returns a specific transformation by ID.
func (c *Client) GetTransformation(ctx context.Context, accountID, containerID, workspaceID, transformationID string) (*TransformationInfo, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/transformations/%s",
		accountID, containerID, workspaceID, transformationID)

	t, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Transformation, error) {
		return c.Service.Accounts.Containers.Workspaces.Transformations.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	result := toTransformation(t)
	return &result, nil
}

// CreateTransformation creates a new transformation in the workspace.
func (c *Client) CreateTransformation(ctx context.Context, accountID, containerID, workspaceID string, input *TransformationInput) (*CreatedTransformation, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)

	t := &tagmanager.Transformation{
		Name:      input.Name,
		Type:      input.Type,
		Parameter: toAPIParams(input.Parameter),
		Notes:     input.Notes,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Transformations.Create(parent, t).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedTransformation{
		TransformationID: result.TransformationId,
		Name:             result.Name,
		Type:             result.Type,
		Path:             result.Path,
		Fingerprint:      result.Fingerprint,
	}, nil
}

// UpdateTransformation updates an existing transformation. It fetches the current transformation first to get the fingerprint.
func (c *Client) UpdateTransformation(ctx context.Context, path string, input *TransformationInput) (*CreatedTransformation, error) {
	current, err := c.Service.Accounts.Containers.Workspaces.Transformations.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	t := &tagmanager.Transformation{
		Name:      input.Name,
		Type:      input.Type,
		Parameter: toAPIParams(input.Parameter),
		Notes:     input.Notes,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Transformations.Update(path, t).Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedTransformation{
		TransformationID: result.TransformationId,
		Name:             result.Name,
		Type:             result.Type,
		Path:             result.Path,
		Fingerprint:      result.Fingerprint,
	}, nil
}

// DeleteTransformation deletes a transformation from the workspace.
func (c *Client) DeleteTransformation(ctx context.Context, path string) error {
	err := c.Service.Accounts.Containers.Workspaces.Transformations.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

func toTransformations(transformations []*tagmanager.Transformation) []TransformationInfo {
	result := make([]TransformationInfo, 0, len(transformations))
	for _, t := range transformations {
		result = append(result, toTransformation(t))
	}
	return result
}

func toTransformation(t *tagmanager.Transformation) TransformationInfo {
	info := TransformationInfo{
		TransformationID: t.TransformationId,
		Name:             t.Name,
		Type:             t.Type,
		Notes:            t.Notes,
		ParentFolderID:   t.ParentFolderId,
		Path:             t.Path,
		Fingerprint:      t.Fingerprint,
	}
	if len(t.Parameter) > 0 {
		info.Parameter = t.Parameter
	}
	return info
}
