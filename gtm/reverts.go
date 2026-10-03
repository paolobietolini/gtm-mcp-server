package gtm

import (
	"context"
	"fmt"
)

type RevertResult struct {
	ResourceType      string `json:"resourceType"`
	ResourceID        string `json:"resourceId"`
	ExistsAfterRevert bool   `json:"existsAfterRevert"`
	Resource          any    `json:"resource,omitempty"`
}

func (c *Client) RevertWorkspaceEntity(ctx context.Context, accountID, containerID, workspaceID, resourceType, resourceID string) (*RevertResult, error) {
	workspacePath := BuildWorkspacePath(accountID, containerID, workspaceID)
	result := &RevertResult{ResourceType: resourceType, ResourceID: resourceID}

	switch resourceType {
	case "builtInVariable":
		response, err := c.Service.Accounts.Containers.Workspaces.BuiltInVariables.Revert(workspacePath).Type(resourceID).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		result.ExistsAfterRevert = response.Enabled
		result.Resource = map[string]any{"type": resourceID, "enabled": response.Enabled}
	case "client":
		path := BuildClientPath(accountID, containerID, workspaceID, resourceID)
		current, err := c.Service.Accounts.Containers.Workspaces.Clients.Get(path).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		response, err := c.Service.Accounts.Containers.Workspaces.Clients.Revert(path).Fingerprint(current.Fingerprint).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		result.ExistsAfterRevert, result.Resource = response.Client != nil, response.Client
	case "tag":
		path := BuildTagPath(accountID, containerID, workspaceID, resourceID)
		current, err := c.Service.Accounts.Containers.Workspaces.Tags.Get(path).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		response, err := c.Service.Accounts.Containers.Workspaces.Tags.Revert(path).Fingerprint(current.Fingerprint).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		result.ExistsAfterRevert, result.Resource = response.Tag != nil, response.Tag
	case "template":
		path := fmt.Sprintf("%s/templates/%s", workspacePath, resourceID)
		current, err := c.Service.Accounts.Containers.Workspaces.Templates.Get(path).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		response, err := c.Service.Accounts.Containers.Workspaces.Templates.Revert(path).Fingerprint(current.Fingerprint).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		result.ExistsAfterRevert, result.Resource = response.Template != nil, response.Template
	case "transformation":
		path := BuildTransformationPath(accountID, containerID, workspaceID, resourceID)
		current, err := c.Service.Accounts.Containers.Workspaces.Transformations.Get(path).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		response, err := c.Service.Accounts.Containers.Workspaces.Transformations.Revert(path).Fingerprint(current.Fingerprint).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		result.ExistsAfterRevert, result.Resource = response.Transformation != nil, response.Transformation
	case "trigger":
		path := BuildTriggerPath(accountID, containerID, workspaceID, resourceID)
		current, err := c.Service.Accounts.Containers.Workspaces.Triggers.Get(path).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		response, err := c.Service.Accounts.Containers.Workspaces.Triggers.Revert(path).Fingerprint(current.Fingerprint).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		result.ExistsAfterRevert, result.Resource = response.Trigger != nil, response.Trigger
	case "variable":
		path := BuildVariablePath(accountID, containerID, workspaceID, resourceID)
		current, err := c.Service.Accounts.Containers.Workspaces.Variables.Get(path).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		response, err := c.Service.Accounts.Containers.Workspaces.Variables.Revert(path).Fingerprint(current.Fingerprint).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		result.ExistsAfterRevert, result.Resource = response.Variable != nil, response.Variable
	case "zone":
		path := BuildZonePath(accountID, containerID, workspaceID, resourceID)
		current, err := c.Service.Accounts.Containers.Workspaces.Zones.Get(path).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		response, err := c.Service.Accounts.Containers.Workspaces.Zones.Revert(path).Fingerprint(current.Fingerprint).Context(ctx).Do()
		if err != nil {
			return nil, mapGoogleError(err)
		}
		result.ExistsAfterRevert, result.Resource = response.Zone != nil, response.Zone
	default:
		return nil, fmt.Errorf("unsupported resource type %q", resourceType)
	}
	return result, nil
}
