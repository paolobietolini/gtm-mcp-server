package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateTagInput is the input for create_tag tool.
type CreateTagInput struct {
	AccountID          string   `json:"accountId" jsonschema:"description:GTM account ID"`
	ContainerID        string   `json:"containerId" jsonschema:"description:GTM container ID"`
	WorkspaceID        string   `json:"workspaceId" jsonschema:"description:GTM workspace ID"`
	Name               string   `json:"name" jsonschema:"description:Tag name"`
	Type               string   `json:"type" jsonschema:"description:Tag type (e.g. gaawe for GA4, html for Custom HTML)"`
	FiringTriggerIDs   []string `json:"firingTriggerIds" jsonschema:"description:Array of trigger IDs that fire this tag"`
	BlockingTriggerIDs []string `json:"blockingTriggerIds,omitempty" jsonschema:"description:Array of trigger IDs that block this tag (optional)"`
	ParametersJSON     string   `json:"parametersJson,omitempty" jsonschema:"description:Tag parameters as JSON array (optional). Each parameter: {type, key, value} or {type, key, list/map}"`
	SetupTagJSON       string   `json:"setupTagJson,omitempty" jsonschema:"description:Setup tag sequencing as JSON array (optional). Each element: {tagName: string, stopOnSetupFailure: bool}. The setup tag fires before this tag."`
	TeardownTagJSON    string   `json:"teardownTagJson,omitempty" jsonschema:"description:Teardown tag sequencing as JSON array (optional). Each element: {tagName: string, stopTeardownOnFailure: bool}. The teardown tag fires after this tag."`
	ConsentStatus      string   `json:"consentStatus,omitempty" jsonschema:"description:Consent status: notSet (default)\\, notNeeded (no consent required)\\, needed (requires consent types to be granted before firing)."`
	ConsentTypes       string   `json:"consentTypes,omitempty" jsonschema:"description:Comma-separated consent types when consentStatus is needed (e.g. ad_storage\\,analytics_storage\\,ad_user_data\\,ad_personalization). Ignored when consentStatus is notSet or notNeeded."`
	Notes              string   `json:"notes,omitempty" jsonschema:"description:Tag notes (optional)"`
	Paused             bool     `json:"paused,omitempty" jsonschema:"description:Whether tag is paused (optional)"`
}

// CreateTagOutput is the output for create_tag tool.
type CreateTagOutput struct {
	Success bool       `json:"success"`
	Tag     CreatedTag `json:"tag"`
	Message string     `json:"message"`
}

func registerCreateTag(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input CreateTagInput) (*mcp.CallToolResult, CreateTagOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, CreateTagOutput{}, err
		}

		// Validate tag input
		if err := ValidateTagInput(input.Name, input.Type, input.FiringTriggerIDs); err != nil {
			return nil, CreateTagOutput{}, err
		}

		// Parse parameters JSON if provided
		var params []Parameter
		if input.ParametersJSON != "" {
			if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
				return nil, CreateTagOutput{}, err
			}
		}

		// Parse setup tag JSON if provided
		var setupTags []SetupTagInput
		if input.SetupTagJSON != "" {
			if err := json.Unmarshal([]byte(input.SetupTagJSON), &setupTags); err != nil {
				return nil, CreateTagOutput{}, fmt.Errorf("invalid setupTagJson: %w", err)
			}
		}

		// Parse teardown tag JSON if provided
		var teardownTags []TeardownTagInput
		if input.TeardownTagJSON != "" {
			if err := json.Unmarshal([]byte(input.TeardownTagJSON), &teardownTags); err != nil {
				return nil, CreateTagOutput{}, fmt.Errorf("invalid teardownTagJson: %w", err)
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
			Name:              input.Name,
			Type:              input.Type,
			FiringTriggerId:   input.FiringTriggerIDs,
			BlockingTriggerId: input.BlockingTriggerIDs,
			Parameter:         params,
			Notes:             input.Notes,
			Paused:            input.Paused,
			SetupTag:          setupTags,
			TeardownTag:       teardownTags,
			ConsentStatus:     input.ConsentStatus,
			ConsentTypes:      consentTypes,
		}

		tag, err := wc.Client.CreateTag(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, tagInput)
		if err != nil {
			return nil, CreateTagOutput{}, err
		}

		return nil, CreateTagOutput{
			Success: true,
			Tag:     *tag,
			Message: "Tag created successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_tag",
		Description: "Create a new tag in a GTM workspace. Requires at least one firing trigger ID. Always call get_tag_templates before creating GA4 tags",
	}, handler)
}
