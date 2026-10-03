package gtm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateTriggerInput is the input for update_trigger tool.
type UpdateTriggerInput struct {
	AccountID             string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID           string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID           string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TriggerID             string `json:"triggerId" jsonschema:"description:The trigger ID to update"`
	Name                  string `json:"name" jsonschema:"description:Trigger name"`
	Type                  string `json:"type" jsonschema:"description:Trigger type (e.g. pageview, customEvent, linkClick, triggerGroup)"`
	FilterJSON            string `json:"filterJson,omitempty" jsonschema:"description:JSON conditions; omit to preserve\\, [] to clear; see gtm://best-practices/tool-input-formats"`
	AutoEventFilterJSON   string `json:"autoEventFilterJson,omitempty" jsonschema:"description:JSON auto-event conditions; omit to preserve\\, [] to clear; see gtm://best-practices/tool-input-formats"`
	CustomEventFilterJSON string `json:"customEventFilterJson,omitempty" jsonschema:"description:JSON custom-event conditions; omit to preserve\\, [] to clear; see gtm://best-practices/tool-input-formats"`
	ParameterJSON         string `json:"parameterJson,omitempty" jsonschema:"description:JSON parameters; omit to preserve\\, [] to clear; see gtm://best-practices/tool-input-formats"`
	Notes                 string `json:"notes,omitempty" jsonschema:"description:Trigger notes (optional)"`
}

// UpdateTriggerOutput is the output for update_trigger tool.
type UpdateTriggerOutput struct {
	Success bool           `json:"success"`
	Trigger CreatedTrigger `json:"trigger"`
	Message string         `json:"message"`
}

func registerUpdateTrigger(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input UpdateTriggerInput) (*mcp.CallToolResult, UpdateTriggerOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, UpdateTriggerOutput{}, err
		}

		// Validate trigger ID
		if input.TriggerID == "" {
			return nil, UpdateTriggerOutput{}, fmt.Errorf("trigger ID is required")
		}

		// Validate trigger input
		if err := ValidateTriggerInput(input.Name, input.Type); err != nil {
			return nil, UpdateTriggerOutput{}, err
		}

		path := BuildTriggerPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TriggerID)

		// Parse filter JSON if provided
		var filter []Condition
		if input.FilterJSON != "" {
			if err := json.Unmarshal([]byte(input.FilterJSON), &filter); err != nil {
				return nil, UpdateTriggerOutput{}, fmt.Errorf("invalid filterJson: %w", err)
			}
		}

		// Parse auto-event filter JSON if provided
		var autoEventFilter []Condition
		if input.AutoEventFilterJSON != "" {
			if err := json.Unmarshal([]byte(input.AutoEventFilterJSON), &autoEventFilter); err != nil {
				return nil, UpdateTriggerOutput{}, fmt.Errorf("invalid autoEventFilterJson: %w", err)
			}
		}

		// Parse custom event filter JSON if provided
		var customEventFilter []Condition
		if input.CustomEventFilterJSON != "" {
			if err := json.Unmarshal([]byte(input.CustomEventFilterJSON), &customEventFilter); err != nil {
				return nil, UpdateTriggerOutput{}, fmt.Errorf("invalid customEventFilterJson: %w", err)
			}
		}

		// Parse parameter JSON if provided (for trigger groups)
		var params []Parameter
		if input.ParameterJSON != "" {
			if err := json.Unmarshal([]byte(input.ParameterJSON), &params); err != nil {
				return nil, UpdateTriggerOutput{}, fmt.Errorf("invalid parameterJson: %w", err)
			}
		}

		// The GTM API silently drops autoEventFilter for linkClick, click, and
		// formSubmission triggers. Remap to filter so the conditions are persisted.
		// See: https://github.com/paolobietolini/gtm-mcp-server/issues/39
		var autoEventFilterWarning string
		if len(autoEventFilter) > 0 {
			switch input.Type {
			case "linkClick", "click", "formSubmission":
				filter = append(filter, autoEventFilter...)
				autoEventFilter = nil
				autoEventFilterWarning = "Warning: the GTM API silently ignores autoEventFilter for " +
					input.Type + " triggers (issue #39). Conditions were automatically remapped to filter."
			}
		}

		triggerInput := &TriggerInput{
			Name:              input.Name,
			Type:              input.Type,
			Filter:            filter,
			AutoEventFilter:   autoEventFilter,
			CustomEventFilter: customEventFilter,
			Parameter:         params,
			Notes:             input.Notes,
		}

		trigger, err := wc.Client.UpdateTrigger(ctx, path, triggerInput)
		if err != nil {
			return nil, UpdateTriggerOutput{}, err
		}

		message := "Trigger updated successfully"
		if autoEventFilterWarning != "" {
			message += ". " + autoEventFilterWarning
		}

		return nil, UpdateTriggerOutput{
			Success: true,
			Trigger: *trigger,
			Message: message,
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_trigger",
		Description: "Update a trigger with automatic fingerprint handling. Read gtm://best-practices/tool-input-formats for filters and trigger groups.",
	}, handler)
}
