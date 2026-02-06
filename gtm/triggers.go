package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Trigger is a simplified representation of a GTM trigger.
type Trigger struct {
	TriggerID         string `json:"triggerId"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	Path              string `json:"path"`
	ParentFolderID    string `json:"parentFolderId,omitempty"`
	Notes             string `json:"notes,omitempty"`
	Filter            any    `json:"filter,omitempty"`            // For pageview triggers
	AutoEventFilter   any    `json:"autoEventFilter,omitempty"`   // For click/form triggers
	CustomEventFilter any    `json:"customEventFilter,omitempty"` // For customEvent triggers
	// Parameter contains trigger configuration. For triggerGroup type, includes member trigger IDs.
	// Using any to avoid recursive type cycle in schema generation.
	Parameter any `json:"parameter,omitempty"`
}

// ListTriggers returns all triggers in a workspace.
func (c *Client) ListTriggers(ctx context.Context, accountID, containerID, workspaceID string) ([]Trigger, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListTriggersResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.Triggers.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return toTriggers(resp.Trigger), nil
}

func toTriggers(triggers []*tagmanager.Trigger) []Trigger {
	result := make([]Trigger, 0, len(triggers))
	for _, t := range triggers {
		trigger := Trigger{
			TriggerID:      t.TriggerId,
			Name:           t.Name,
			Type:           t.Type,
			Path:           t.Path,
			ParentFolderID: t.ParentFolderId,
			Notes:          t.Notes,
		}
		// Include filter fields when present
		if len(t.Filter) > 0 {
			trigger.Filter = t.Filter
		}
		if len(t.AutoEventFilter) > 0 {
			trigger.AutoEventFilter = t.AutoEventFilter
		}
		if len(t.CustomEventFilter) > 0 {
			trigger.CustomEventFilter = t.CustomEventFilter
		}
		// Include parameters for triggerGroup type or when parameters exist
		if len(t.Parameter) > 0 {
			trigger.Parameter = t.Parameter
		}
		result = append(result, trigger)
	}
	return result
}
