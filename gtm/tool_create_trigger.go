package gtm

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateTriggerInput is the input for create_trigger tool.
type CreateTriggerInput struct {
	AccountID             string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID           string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID           string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Name                  string `json:"name" jsonschema:"description:Trigger name"`
	Type                  string `json:"type" jsonschema:"description:Trigger type (e.g. pageview, customEvent, linkClick, formSubmission, timer)"`
	FilterJSON            string `json:"filterJson,omitempty" jsonschema:"description:Filter conditions as JSON array for pageview triggers (optional)"`
	AutoEventFilterJSON   string `json:"autoEventFilterJson,omitempty" jsonschema:"description:Auto-event filter as JSON array for click/form triggers (optional)"`
	CustomEventFilterJSON string `json:"customEventFilterJson,omitempty" jsonschema:"description:Custom event filter as JSON array for customEvent triggers. REQUIRED for customEvent type. Must contain exactly one condition matching the event name."`
	EventNameJSON         string `json:"eventNameJson,omitempty" jsonschema:"description:Event name as JSON object {type, value} for timer triggers (optional)"`
	Notes                 string `json:"notes,omitempty" jsonschema:"description:Trigger notes (optional)"`
}

// CreateTriggerOutput is the output for create_trigger tool.
type CreateTriggerOutput struct {
	Success bool           `json:"success"`
	Trigger CreatedTrigger `json:"trigger"`
	Message string         `json:"message"`
}

func registerCreateTrigger(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input CreateTriggerInput) (*mcp.CallToolResult, CreateTriggerOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, CreateTriggerOutput{}, err
		}

		// Validate trigger input
		if err := ValidateTriggerInput(input.Name, input.Type); err != nil {
			return nil, CreateTriggerOutput{}, err
		}

		// Parse filter JSON if provided
		var filter []Condition
		if input.FilterJSON != "" {
			if err := json.Unmarshal([]byte(input.FilterJSON), &filter); err != nil {
				return nil, CreateTriggerOutput{}, err
			}
		}

		// Parse auto-event filter JSON if provided
		var autoEventFilter []Condition
		if input.AutoEventFilterJSON != "" {
			if err := json.Unmarshal([]byte(input.AutoEventFilterJSON), &autoEventFilter); err != nil {
				return nil, CreateTriggerOutput{}, err
			}
		}

		// Parse custom event filter JSON if provided (required for customEvent type)
		var customEventFilter []Condition
		if input.CustomEventFilterJSON != "" {
			if err := json.Unmarshal([]byte(input.CustomEventFilterJSON), &customEventFilter); err != nil {
				return nil, CreateTriggerOutput{}, err
			}
		}

		// Parse event name JSON if provided
		var eventName *Parameter
		if input.EventNameJSON != "" {
			eventName = &Parameter{}
			if err := json.Unmarshal([]byte(input.EventNameJSON), eventName); err != nil {
				return nil, CreateTriggerOutput{}, err
			}
		}

		triggerInput := &TriggerInput{
			Name:              input.Name,
			Type:              input.Type,
			Filter:            filter,
			AutoEventFilter:   autoEventFilter,
			CustomEventFilter: customEventFilter,
			EventName:         eventName,
			Notes:             input.Notes,
		}

		trigger, err := wc.Client.CreateTrigger(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, triggerInput)
		if err != nil {
			return nil, CreateTriggerOutput{}, err
		}

		return nil, CreateTriggerOutput{
			Success: true,
			Trigger: *trigger,
			Message: "Trigger created successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_trigger",
		Description: "Create a new trigger in a GTM workspace. Common types: pageview, customEvent, linkClick, formSubmission, timer, scrollDepth.",
	}, handler)
}
