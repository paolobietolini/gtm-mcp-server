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

	// For click/form triggers with autoEventFilter, set required companion fields
	if len(input.AutoEventFilter) > 0 && (input.Type == "linkClick" || input.Type == "formSubmission" || input.Type == "click") {
		trigger.WaitForTags = &tagmanager.Parameter{Type: "boolean", Value: "false"}
		trigger.WaitForTagsTimeout = &tagmanager.Parameter{Type: "integer", Value: "2000"}
		trigger.CheckValidation = &tagmanager.Parameter{Type: "boolean", Value: "false"}
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
// Fields not provided in input are preserved from the current trigger.
func (c *Client) UpdateTrigger(ctx context.Context, path string, input *TriggerInput) (*CreatedTrigger, error) {
	// Get current trigger for fingerprint and to preserve unset fields
	current, err := c.Service.Accounts.Containers.Workspaces.Triggers.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	// Preserve existing fields when not provided in input
	filter := toAPIConditions(input.Filter)
	if filter == nil {
		filter = current.Filter
	}
	autoEventFilter := toAPIConditions(input.AutoEventFilter)
	if autoEventFilter == nil {
		autoEventFilter = current.AutoEventFilter
	}
	customEventFilter := toAPIConditions(input.CustomEventFilter)
	if customEventFilter == nil {
		customEventFilter = current.CustomEventFilter
	}
	params := toAPIParams(input.Parameter)
	if params == nil {
		params = current.Parameter
	}

	trigger := &tagmanager.Trigger{
		Name:              input.Name,
		Type:              input.Type,
		Filter:            filter,
		AutoEventFilter:   autoEventFilter,
		CustomEventFilter: customEventFilter,
		Parameter:         params,
		Notes:             input.Notes,
		// Preserve trigger-specific fields from current trigger (exclude auto-generated ones)
		CheckValidation:                current.CheckValidation,
		WaitForTags:                    current.WaitForTags,
		WaitForTagsTimeout:             current.WaitForTagsTimeout,
		ContinuousTimeMinMilliseconds:  current.ContinuousTimeMinMilliseconds,
		HorizontalScrollPercentageList: current.HorizontalScrollPercentageList,
		Interval:                       current.Interval,
		IntervalSeconds:                current.IntervalSeconds,
		Limit:                          current.Limit,
		MaxTimerLengthSeconds:          current.MaxTimerLengthSeconds,
		Selector:                       current.Selector,
		TotalTimeMinMilliseconds:       current.TotalTimeMinMilliseconds,
		VerticalScrollPercentageList:   current.VerticalScrollPercentageList,
		VisibilitySelector:             current.VisibilitySelector,
		VisiblePercentageMax:           current.VisiblePercentageMax,
		VisiblePercentageMin:           current.VisiblePercentageMin,
	}
	// NOTE: Do NOT include UniqueTriggerId - it's auto-generated during output generation
	// NOTE: Fingerprint is passed as URL parameter, not in body

	if input.EventName != nil {
		trigger.EventName = toAPIParam(input.EventName)
	} else {
		trigger.EventName = current.EventName
	}

	// For click/form triggers with autoEventFilter, ensure companion fields have proper boolean values
	// (not empty template params which indicate "All Clicks" mode)
	if len(autoEventFilter) > 0 && (input.Type == "linkClick" || input.Type == "formSubmission" || input.Type == "click") {
		if trigger.WaitForTags == nil || trigger.WaitForTags.Value == "" {
			trigger.WaitForTags = &tagmanager.Parameter{Type: "boolean", Value: "false"}
		}
		if trigger.WaitForTagsTimeout == nil || trigger.WaitForTagsTimeout.Value == "" {
			trigger.WaitForTagsTimeout = &tagmanager.Parameter{Type: "integer", Value: "2000"}
		}
		if trigger.CheckValidation == nil || trigger.CheckValidation.Value == "" {
			trigger.CheckValidation = &tagmanager.Parameter{Type: "boolean", Value: "false"}
		}
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Triggers.Update(path, trigger).Fingerprint(current.Fingerprint).Context(ctx).Do()
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

// UpdateVariable updates an existing variable. It fetches the current variable first to get the fingerprint.
func (c *Client) UpdateVariable(ctx context.Context, path string, input *VariableInput) (*CreatedVariable, error) {
	// Get current variable for fingerprint
	current, err := c.Service.Accounts.Containers.Workspaces.Variables.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	variable := &tagmanager.Variable{
		Name:        input.Name,
		Type:        input.Type,
		Parameter:   toAPIParams(input.Parameter),
		Notes:       input.Notes,
		Fingerprint: current.Fingerprint,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Variables.Update(path, variable).Context(ctx).Do()
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
		Type:            p.Type,
		Key:             p.Key,
		Value:           p.Value,
		ForceSendFields: []string{"Type", "Key", "Value"},
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
		params := toAPIParams(c.Parameter)
		if c.Negate {
			params = append(params, &tagmanager.Parameter{
				Type:  "boolean",
				Key:   "negate",
				Value: "true",
			})
		}
		result[i] = &tagmanager.Condition{
			Type:            c.Type,
			Parameter:       params,
			ForceSendFields: []string{"Type", "Parameter"},
		}
	}
	return result
}

// triggerForceSendFields returns the list of fields that must be force-sent
// to the Google API to prevent omitempty from dropping them.
func triggerForceSendFields(input *TriggerInput) []string {
	var fields []string
	if len(input.Filter) > 0 {
		fields = append(fields, "Filter")
	}
	if len(input.AutoEventFilter) > 0 {
		fields = append(fields, "AutoEventFilter")
	}
	if len(input.CustomEventFilter) > 0 {
		fields = append(fields, "CustomEventFilter")
	}
	if len(input.Parameter) > 0 {
		fields = append(fields, "Parameter")
	}
	if input.EventName != nil {
		fields = append(fields, "EventName")
	}
	return fields
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
