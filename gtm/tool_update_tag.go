package gtm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateTagInput is the input for update_tag tool.
type UpdateTagInput struct {
	AccountID          string   `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID        string   `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID        string   `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TagID              string   `json:"tagId" jsonschema:"description:The tag ID to update"`
	Name               string   `json:"name" jsonschema:"description:Tag name"`
	Type               string   `json:"type" jsonschema:"description:Tag type"`
	FiringTriggerIDs   []string `json:"firingTriggerIds" jsonschema:"description:Array of trigger IDs that fire this tag"`
	BlockingTriggerIDs []string `json:"blockingTriggerIds,omitempty" jsonschema:"description:Array of trigger IDs that block this tag (optional)"`
	ParametersJSON     string   `json:"parametersJson,omitempty" jsonschema:"description:Tag parameters as JSON array (optional)"`
	Notes              string   `json:"notes,omitempty" jsonschema:"description:Tag notes (optional)"`
	Paused             bool     `json:"paused,omitempty" jsonschema:"description:Whether tag is paused (optional)"`
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

		// Validate tag input
		if err := ValidateTagInput(input.Name, input.Type, input.FiringTriggerIDs); err != nil {
			return nil, UpdateTagOutput{}, err
		}

		path := BuildTagPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)

		// Parse parameters JSON if provided
		var params []Parameter
		if input.ParametersJSON != "" {
			if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
				return nil, UpdateTagOutput{}, err
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
		Description: "Update an existing tag. Automatically handles fingerprint for concurrency control.",
	}, handler)
}
