package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// CreateTag creates a new tag in the workspace.
func (c *Client) CreateTag(ctx context.Context, accountID, containerID, workspaceID string, input *TagInput) (*CreatedTag, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)

	tag := &tagmanager.Tag{
		Name:              input.Name,
		Type:              input.Type,
		FiringTriggerId:   input.FiringTriggerId,
		BlockingTriggerId: input.BlockingTriggerId,
		Parameter:         toAPIParams(input.Parameter),
		Notes:             input.Notes,
		Paused:            input.Paused,
		TagFiringOption:   input.TagFiringOption,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Tags.Create(parent, tag).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedTag{
		TagID:       result.TagId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

// UpdateTag updates an existing tag. It fetches the current tag first to get the fingerprint.
func (c *Client) UpdateTag(ctx context.Context, path string, input *TagInput) (*CreatedTag, error) {
	// Get current tag for fingerprint
	current, err := c.Service.Accounts.Containers.Workspaces.Tags.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	// Build updated tag with fingerprint
	tag := &tagmanager.Tag{
		Name:              input.Name,
		Type:              input.Type,
		FiringTriggerId:   input.FiringTriggerId,
		BlockingTriggerId: input.BlockingTriggerId,
		Parameter:         toAPIParams(input.Parameter),
		Notes:             input.Notes,
		Paused:            input.Paused,
		TagFiringOption:   input.TagFiringOption,
		Fingerprint:       current.Fingerprint,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Tags.Update(path, tag).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedTag{
		TagID:       result.TagId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

// DeleteTag deletes a tag from the workspace.
func (c *Client) DeleteTag(ctx context.Context, path string) error {
	err := c.Service.Accounts.Containers.Workspaces.Tags.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

// CreateTrigger creates a new trigger in the workspace.
func (c *Client) CreateTrigger(ctx context.Context, accountID, containerID, workspaceID string, input *TriggerInput) (*CreatedTrigger, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)

	trigger := &tagmanager.Trigger{
		Name:              input.Name,
		Type:              input.Type,
		Filter:            toAPIConditions(input.Filter),
		AutoEventFilter:   toAPIConditions(input.AutoEventFilter),
		CustomEventFilter: toAPIConditions(input.CustomEventFilter),
		Parameter:         toAPIParams(input.Parameter),
		Notes:             input.Notes,
	}

	if input.EventName != nil {
		trigger.EventName = toAPIParam(input.EventName)
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Triggers.Create(parent, trigger).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedTrigger{
		TriggerID:   result.TriggerId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

// DeleteTrigger deletes a trigger from the workspace.
func (c *Client) DeleteTrigger(ctx context.Context, path string) error {
	err := c.Service.Accounts.Containers.Workspaces.Triggers.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

// UpdateTrigger updates an existing trigger. It fetches the current trigger first to get the fingerprint.
func (c *Client) UpdateTrigger(ctx context.Context, path string, input *TriggerInput) (*CreatedTrigger, error) {
	// Get current trigger for fingerprint
	current, err := c.Service.Accounts.Containers.Workspaces.Triggers.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	// Build updated trigger with fingerprint
	trigger := &tagmanager.Trigger{
		Name:              input.Name,
		Type:              input.Type,
		Filter:            toAPIConditions(input.Filter),
		AutoEventFilter:   toAPIConditions(input.AutoEventFilter),
		CustomEventFilter: toAPIConditions(input.CustomEventFilter),
		Parameter:         toAPIParams(input.Parameter),
		Notes:             input.Notes,
		Fingerprint:       current.Fingerprint,
	}

	if input.EventName != nil {
		trigger.EventName = toAPIParam(input.EventName)
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Triggers.Update(path, trigger).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedTrigger{
		TriggerID:   result.TriggerId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

// CreateVariable creates a new variable in the workspace.
func (c *Client) CreateVariable(ctx context.Context, accountID, containerID, workspaceID string, input *VariableInput) (*CreatedVariable, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)

	variable := &tagmanager.Variable{
		Name:      input.Name,
		Type:      input.Type,
		Parameter: toAPIParams(input.Parameter),
		Notes:     input.Notes,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Variables.Create(parent, variable).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedVariable{
		VariableID:  result.VariableId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

// DeleteVariable deletes a variable from the workspace.
func (c *Client) DeleteVariable(ctx context.Context, path string) error {
	err := c.Service.Accounts.Containers.Workspaces.Variables.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

func toAPIParams(params []Parameter) []*tagmanager.Parameter {
	if len(params) == 0 {
		return nil
	}
	result := make([]*tagmanager.Parameter, len(params))
	for i, p := range params {
		result[i] = toAPIParam(&p)
	}
	return result
}

func toAPIParam(p *Parameter) *tagmanager.Parameter {
	if p == nil {
		return nil
	}
	param := &tagmanager.Parameter{
		Type:  p.Type,
		Key:   p.Key,
		Value: p.Value,
	}
	if len(p.List) > 0 {
		param.List = toAPIParams(p.List)
	}
	if len(p.Map) > 0 {
		param.Map = toAPIParams(p.Map)
	}
	return param
}

func toAPIConditions(conditions []Condition) []*tagmanager.Condition {
	if len(conditions) == 0 {
		return nil
	}
	result := make([]*tagmanager.Condition, len(conditions))
	for i, c := range conditions {
		result[i] = &tagmanager.Condition{
			Type:      c.Type,
			Parameter: toAPIParams(c.Parameter),
		}
	}
	return result
}

// BuildTagPath constructs a tag path from IDs.
func BuildTagPath(accountID, containerID, workspaceID, tagID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/tags/%s",
		accountID, containerID, workspaceID, tagID)
}

// BuildTriggerPath constructs a trigger path from IDs.
func BuildTriggerPath(accountID, containerID, workspaceID, triggerID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/triggers/%s",
		accountID, containerID, workspaceID, triggerID)
}

// BuildVariablePath constructs a variable path from IDs.
func BuildVariablePath(accountID, containerID, workspaceID, variableID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/variables/%s",
		accountID, containerID, workspaceID, variableID)
}
