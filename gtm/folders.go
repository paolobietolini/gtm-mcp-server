package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Folder is a simplified representation of a GTM folder.
type Folder struct {
	AccountID     string `json:"accountId,omitempty"`
	ContainerID   string `json:"containerId,omitempty"`
	WorkspaceID   string `json:"workspaceId,omitempty"`
	FolderID      string `json:"folderId"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	Notes         string `json:"notes,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	TagManagerURL string `json:"tagManagerUrl,omitempty"`
}

// FolderEntities contains the entities within a folder.
type FolderEntities struct {
	Tags      []string `json:"tags,omitempty"`
	Triggers  []string `json:"triggers,omitempty"`
	Variables []string `json:"variables,omitempty"`
}

// ListFolders returns all folders in a workspace.
func (c *Client) ListFolders(ctx context.Context, accountID, containerID, workspaceID string) ([]Folder, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	result := make([]Folder, 0)
	pageToken := ""
	for {
		resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListFoldersResponse, error) {
			return c.Service.Accounts.Containers.Workspaces.Folders.List(parent).PageToken(pageToken).Context(ctx).Do()
		})
		if err != nil {
			return nil, mapGoogleError(err)
		}
		result = append(result, toFolders(resp.Folder)...)
		if resp.NextPageToken == "" {
			return result, nil
		}
		pageToken = resp.NextPageToken
	}
}

func (c *Client) GetFolder(ctx context.Context, accountID, containerID, workspaceID, folderID string) (*Folder, error) {
	folder, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Folder, error) {
		return c.Service.Accounts.Containers.Workspaces.Folders.Get(BuildFolderPath(accountID, containerID, workspaceID, folderID)).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toFolder(folder)
	return &result, nil
}

func (c *Client) CreateFolder(ctx context.Context, accountID, containerID, workspaceID, name, notes string) (*Folder, error) {
	folder, err := c.Service.Accounts.Containers.Workspaces.Folders.Create(BuildWorkspacePath(accountID, containerID, workspaceID), &tagmanager.Folder{Name: name, Notes: notes}).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toFolder(folder)
	return &result, nil
}

func (c *Client) UpdateFolder(ctx context.Context, accountID, containerID, workspaceID, folderID string, name, notes *string) (*Folder, error) {
	path := BuildFolderPath(accountID, containerID, workspaceID, folderID)
	current, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Folder, error) {
		return c.Service.Accounts.Containers.Workspaces.Folders.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	body := &tagmanager.Folder{Name: current.Name, Notes: current.Notes}
	if name != nil {
		body.Name = *name
	}
	if notes != nil {
		body.Notes = *notes
		if *notes == "" {
			body.ForceSendFields = append(body.ForceSendFields, "Notes")
		}
	}
	folder, err := c.Service.Accounts.Containers.Workspaces.Folders.Update(path, body).Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toFolder(folder)
	return &result, nil
}

func (c *Client) DeleteFolder(ctx context.Context, accountID, containerID, workspaceID, folderID string) error {
	return mapGoogleError(c.Service.Accounts.Containers.Workspaces.Folders.Delete(BuildFolderPath(accountID, containerID, workspaceID, folderID)).Context(ctx).Do())
}

func (c *Client) MoveEntitiesToFolder(ctx context.Context, accountID, containerID, workspaceID, folderID string, tagIDs, triggerIDs, variableIDs []string) error {
	call := c.Service.Accounts.Containers.Workspaces.Folders.MoveEntitiesToFolder(BuildFolderPath(accountID, containerID, workspaceID, folderID), &tagmanager.Folder{})
	if len(tagIDs) > 0 {
		call.TagId(tagIDs...)
	}
	if len(triggerIDs) > 0 {
		call.TriggerId(triggerIDs...)
	}
	if len(variableIDs) > 0 {
		call.VariableId(variableIDs...)
	}
	return mapGoogleError(call.Context(ctx).Do())
}

func (c *Client) RevertFolder(ctx context.Context, accountID, containerID, workspaceID, folderID string) (*Folder, error) {
	path := BuildFolderPath(accountID, containerID, workspaceID, folderID)
	current, err := c.Service.Accounts.Containers.Workspaces.Folders.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	response, err := c.Service.Accounts.Containers.Workspaces.Folders.Revert(path).Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	if response.Folder == nil {
		return nil, nil
	}
	result := toFolder(response.Folder)
	return &result, nil
}

func BuildFolderPath(accountID, containerID, workspaceID, folderID string) string {
	return fmt.Sprintf("%s/folders/%s", BuildWorkspacePath(accountID, containerID, workspaceID), folderID)
}

// GetFolderEntities returns the entities (tags, triggers, variables) in a folder.
func (c *Client) GetFolderEntities(ctx context.Context, accountID, containerID, workspaceID, folderID string) (*FolderEntities, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/folders/%s",
		accountID, containerID, workspaceID, folderID)

	entities := &FolderEntities{}
	pageToken := ""
	for {
		resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.FolderEntities, error) {
			return c.Service.Accounts.Containers.Workspaces.Folders.Entities(path).PageToken(pageToken).Context(ctx).Do()
		})
		if err != nil {
			return nil, mapGoogleError(err)
		}
		for _, tag := range resp.Tag {
			entities.Tags = append(entities.Tags, tag.Name)
		}
		for _, trigger := range resp.Trigger {
			entities.Triggers = append(entities.Triggers, trigger.Name)
		}
		for _, variable := range resp.Variable {
			entities.Variables = append(entities.Variables, variable.Name)
		}
		if resp.NextPageToken == "" {
			return entities, nil
		}
		pageToken = resp.NextPageToken
	}
}

func toFolders(folders []*tagmanager.Folder) []Folder {
	result := make([]Folder, 0, len(folders))
	for _, f := range folders {
		result = append(result, toFolder(f))
	}
	return result
}

func toFolder(folder *tagmanager.Folder) Folder {
	return Folder{
		AccountID: folder.AccountId, ContainerID: folder.ContainerId, WorkspaceID: folder.WorkspaceId,
		FolderID: folder.FolderId, Name: folder.Name, Path: folder.Path, Notes: folder.Notes,
		Fingerprint: folder.Fingerprint, TagManagerURL: folder.TagManagerUrl,
	}
}
