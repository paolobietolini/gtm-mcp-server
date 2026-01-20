package gtm

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateTagInput is the input for create_tag tool.
type CreateTagInput struct {
	AccountID          string   `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID        string   `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID        string   `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Name               string   `json:"name" jsonschema:"description:Tag name"`
	Type               string   `json:"type" jsonschema:"description:Tag type (e.g. gaawe for GA4, html for Custom HTML)"`
	FiringTriggerIDs   []string `json:"firingTriggerIds" jsonschema:"description:Array of trigger IDs that fire this tag"`
	BlockingTriggerIDs []string `json:"blockingTriggerIds,omitempty" jsonschema:"description:Array of trigger IDs that block this tag (optional)"`
	ParametersJSON     string   `json:"parametersJson,omitempty" jsonschema:"description:Tag parameters as JSON array (optional). Each parameter: {type, key, value} or {type, key, list/map}"`
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
		// Validate workspace path
		if err := ValidateWorkspacePath(input.AccountID, input.ContainerID, input.WorkspaceID); err != nil {
			return nil, CreateTagOutput{}, err
		}

		// Validate tag input
		if err := ValidateTagInput(input.Name, input.Type, input.FiringTriggerIDs); err != nil {
			return nil, CreateTagOutput{}, err
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, CreateTagOutput{}, err
		}

		// Parse parameters JSON if provided
		var params []Parameter
		if input.ParametersJSON != "" {
			if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
				return nil, CreateTagOutput{}, err
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
		}

		tag, err := client.CreateTag(ctx, input.AccountID, input.ContainerID, input.WorkspaceID, tagInput)
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
		Description: "Create a new tag in a GTM workspace. Requires at least one firing trigger ID.",
	}, handler)
}
