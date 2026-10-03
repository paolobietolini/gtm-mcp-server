package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateTagInput is the input for update_tag tool.
type UpdateTagInput struct {
	AccountID          string   `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID        string   `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID        string   `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TagID              string   `json:"tagId" jsonschema:"description:The tag ID to update"`
	Name               string   `json:"name,omitempty" jsonschema:"description:Tag name. If omitted\\, existing name is preserved."`
	Type               string   `json:"type,omitempty" jsonschema:"description:Tag type. If omitted\\, existing type is preserved."`
	FiringTriggerIDs   []string `json:"firingTriggerIds,omitempty" jsonschema:"description:Array of trigger IDs that fire this tag. If omitted\\, existing triggers are preserved."`
	BlockingTriggerIDs []string `json:"blockingTriggerIds,omitempty" jsonschema:"description:Array of trigger IDs that block this tag. If omitted\\, existing blocking triggers are preserved."`
	ParametersJSON     string   `json:"parametersJson,omitempty" jsonschema:"description:JSON parameters; omit to preserve; see gtm://best-practices/tool-input-formats"`
	SetupTagJSON       string   `json:"setupTagJson,omitempty" jsonschema:"description:JSON setup sequence; omit to preserve\\, [] to clear; see gtm://best-practices/tool-input-formats"`
	TeardownTagJSON    string   `json:"teardownTagJson,omitempty" jsonschema:"description:JSON teardown sequence; omit to preserve\\, [] to clear; see gtm://best-practices/tool-input-formats"`
	ConsentStatus      string   `json:"consentStatus,omitempty" jsonschema:"description:Consent status: notSet (default/clear)\\, notNeeded (no consent required)\\, needed (requires consent types to be granted before firing). If omitted\\, existing consent settings are preserved."`
	ConsentTypes       string   `json:"consentTypes,omitempty" jsonschema:"description:Comma-separated consent types when consentStatus is needed (e.g. ad_storage\\,analytics_storage\\,ad_user_data\\,ad_personalization). Ignored when consentStatus is notSet or notNeeded."`
	Notes              string   `json:"notes,omitempty" jsonschema:"description:Tag notes. If omitted\\, existing notes are preserved."`
	Paused             *bool    `json:"paused,omitempty" jsonschema:"description:Whether tag is paused. If omitted\\, existing paused state is preserved."`
}

// UpdateTagOutput is the output for update_tag tool.
type UpdateTagOutput struct {
	Success bool       `json:"success"`
	Tag     CreatedTag `json:"tag"`
	Message string     `json:"message"`
}

func registerUpdateTag(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input UpdateTagInput) (*mcp.CallToolResult, UpdateTagOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, UpdateTagOutput{}, err
		}

		// Validate tag ID
		if input.TagID == "" {
			return nil, UpdateTagOutput{}, fmt.Errorf("tag ID is required")
		}

		path := BuildTagPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)

		// Parse parameters JSON if provided
		var params []Parameter
		var hasParams bool
		if input.ParametersJSON != "" {
			hasParams = true
			if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
				return nil, UpdateTagOutput{}, err
			}
		}

		// Parse setup tag JSON if provided
		var setupTags []SetupTagInput
		var hasSetupTag bool
		var clearSetup bool
		if input.SetupTagJSON != "" {
			hasSetupTag = true
			if err := json.Unmarshal([]byte(input.SetupTagJSON), &setupTags); err != nil {
				return nil, UpdateTagOutput{}, fmt.Errorf("invalid setupTagJson: %w", err)
			}
			if len(setupTags) == 0 {
				clearSetup = true
			}
		}

		// Parse teardown tag JSON if provided
		var teardownTags []TeardownTagInput
		var hasTeardownTag bool
		var clearTeardown bool
		if input.TeardownTagJSON != "" {
			hasTeardownTag = true
			if err := json.Unmarshal([]byte(input.TeardownTagJSON), &teardownTags); err != nil {
				return nil, UpdateTagOutput{}, fmt.Errorf("invalid teardownTagJson: %w", err)
			}
			if len(teardownTags) == 0 {
				clearTeardown = true
			}
		}

		// Parse consent types if provided
		var consentTypes []string
		if input.ConsentTypes != "" {
			for _, t := range strings.Split(input.ConsentTypes, ",") {
				if trimmed := strings.TrimSpace(t); trimmed != "" {
					consentTypes = append(consentTypes, trimmed)
				}
			}
		}

		tagInput := &TagInput{
			Name:               input.Name,
			Type:               input.Type,
			FiringTriggerId:    input.FiringTriggerIDs,
			BlockingTriggerId:  input.BlockingTriggerIDs,
			Parameter:          params,
			HasParameter:       hasParams,
			Notes:              input.Notes,
			Paused:             input.Paused != nil && *input.Paused,
			HasPaused:          input.Paused != nil,
			SetupTag:           setupTags,
			TeardownTag:        teardownTags,
			HasSetupTag:        hasSetupTag,
			HasTeardownTag:     hasTeardownTag,
			ClearSetupTag:      clearSetup,
			ClearTeardownTag:   clearTeardown,
			ConsentStatus:      input.ConsentStatus,
			ConsentTypes:       consentTypes,
			HasConsentSettings: input.ConsentStatus != "",
		}

		tag, err := wc.Client.UpdateTag(ctx, path, tagInput)
		if err != nil {
			return nil, UpdateTagOutput{}, err
		}

		return nil, UpdateTagOutput{
			Success: true,
			Tag:     *tag,
			Message: "Tag updated successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_tag",
		Description: "Update an existing tag. Only provided fields are changed — all other fields (parameters, triggers, consent, etc.) are preserved from the existing tag. Automatically handles fingerprint for concurrency control.",
	}, handler)
}
