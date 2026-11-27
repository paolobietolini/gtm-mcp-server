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

	if result == nil || result.ContainerVersion == nil {
		if result != nil && result.CompilerError {
			return nil, fmt.Errorf("version %s was not published: compiler error", versionID)
		}
		return nil, fmt.Errorf("version %s was not published: the API returned no version", versionID)
	}

	return &PublishedVersion{
		VersionID: result.ContainerVersion.ContainerVersionId,
		Name:      result.ContainerVersion.Name,
		Path:      result.ContainerVersion.Path,
	}, nil
}

// GetWorkspaceStatus lists the entities a workspace changes and its merge
// conflicts, without the entities' contents.
func (c *Client) GetWorkspaceStatus(ctx context.Context, accountID, containerID, workspaceID string) (*WorkspaceStatus, error) {
	return c.workspaceStatus(ctx, accountID, containerID, workspaceID, false)
}

// GetWorkspaceStatusWithEntities is GetWorkspaceStatus with each changed or
// conflicting entity in full, as the API returns it, so it can be compared with
// the same entity in a version.
func (c *Client) GetWorkspaceStatusWithEntities(ctx context.Context, accountID, containerID, workspaceID string) (*WorkspaceStatus, error) {
	return c.workspaceStatus(ctx, accountID, containerID, workspaceID, true)
}

func (c *Client) workspaceStatus(ctx context.Context, accountID, containerID, workspaceID string, withEntities bool) (*WorkspaceStatus, error) {
	path := BuildWorkspacePath(accountID, containerID, workspaceID)

	status, err := retryWithBackoff(ctx, 3, func() (*tagmanager.GetWorkspaceStatusResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.GetStatus(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	result := &WorkspaceStatus{
		HasChanges:    len(status.WorkspaceChange) > 0,
		HasConflicts:  len(status.MergeConflict) > 0,
		ChangeCount:   len(status.WorkspaceChange),
		ConflictCount: len(status.MergeConflict),
		Changes:       make([]WorkspaceChange, 0, len(status.WorkspaceChange)),
		Conflicts:     make([]WorkspaceConflict, 0, len(status.MergeConflict)),
	}
	for _, e := range status.WorkspaceChange {
		ref, value := describeEntity(e)
		change := WorkspaceChange{EntityRef: ref}
		if e != nil {
			change.ChangeStatus = e.ChangeStatus
		}
		if withEntities {
			change.Entity = value
		}
		result.Changes = append(result.Changes, change)
	}
	for _, m := range status.MergeConflict {
		if m == nil {
			m = &tagmanager.MergeConflict{}
		}
		ref, inWorkspace := describeEntity(m.EntityInWorkspace)
		baseRef, inBase := describeEntity(m.EntityInBaseVersion)
		// An entity deleted on one side is only described by the other.
		if ref.Type == "" {
			ref = baseRef
		}
		conflict := WorkspaceConflict{EntityRef: ref}
		if withEntities {
			conflict.Entity, conflict.BaseEntity = inWorkspace, inBase
		}
		result.Conflicts = append(result.Conflicts, conflict)
	}
	return result, nil
}

// describeEntity names the one entity set in e and returns it as the API sent
// it. The type is the key of the same collection in a container version, and
// the ID is the entity's own ID field there (the type, for built-in variables).
func describeEntity(e *tagmanager.Entity) (EntityRef, any) {
	switch {
	case e == nil:
	case e.Tag != nil:
		return EntityRef{"tag", e.Tag.TagId, e.Tag.Name}, e.Tag
	case e.Trigger != nil:
		return EntityRef{"trigger", e.Trigger.TriggerId, e.Trigger.Name}, e.Trigger
	case e.Variable != nil:
		return EntityRef{"variable", e.Variable.VariableId, e.Variable.Name}, e.Variable
	case e.Folder != nil:
		return EntityRef{"folder", e.Folder.FolderId, e.Folder.Name}, e.Folder
	case e.BuiltInVariable != nil:
		return EntityRef{"builtInVariable", e.BuiltInVariable.Type, e.BuiltInVariable.Name}, e.BuiltInVariable
	case e.CustomTemplate != nil:
		return EntityRef{"customTemplate", e.CustomTemplate.TemplateId, e.CustomTemplate.Name}, e.CustomTemplate
	case e.Client != nil:
		return EntityRef{"client", e.Client.ClientId, e.Client.Name}, e.Client
	case e.Transformation != nil:
		return EntityRef{"transformation", e.Transformation.TransformationId, e.Transformation.Name}, e.Transformation
	case e.Zone != nil:
		return EntityRef{"zone", e.Zone.ZoneId, e.Zone.Name}, e.Zone
	case e.GtagConfig != nil:
		return EntityRef{"gtagConfig", e.GtagConfig.GtagConfigId, ""}, e.GtagConfig
	}
	return EntityRef{}, nil
}

// PublishedVersion represents the result of publishing a version.
type PublishedVersion struct {
	VersionID string `json:"containerVersionId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}

// WorkspaceStatus represents the status of a workspace.
type WorkspaceStatus struct {
	HasChanges    bool                `json:"hasChanges"`
	HasConflicts  bool                `json:"hasConflicts"`
	ChangeCount   int                 `json:"changeCount"`
	ConflictCount int                 `json:"conflictCount"`
	Changes       []WorkspaceChange   `json:"changes"`
	Conflicts     []WorkspaceConflict `json:"conflicts"`
}

// EntityRef identifies a workspace entity: Type is tag, trigger, variable,
// folder, builtInVariable, customTemplate, client, transformation, zone or
// gtagConfig.
type EntityRef struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// WorkspaceChange is an entity the workspace adds, updates or deletes compared
// with the version it is based on.
type WorkspaceChange struct {
	EntityRef
	ChangeStatus string `json:"changeStatus,omitempty"`
	Entity       any    `json:"entity,omitempty"`
}

// WorkspaceConflict is an entity changed both in the workspace and in a version
// created since the workspace was last updated.
type WorkspaceConflict struct {
	EntityRef
	Entity     any `json:"entity,omitempty"`
	BaseEntity any `json:"baseEntity,omitempty"`
}
