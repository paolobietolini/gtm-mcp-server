package gtm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateTransformationInput is the input for update_transformation tool.
type UpdateTransformationInput struct {
	AccountID        string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID      string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID      string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TransformationID string `json:"transformationId" jsonschema:"description:The transformation ID to update"`
	Name             string `json:"name" jsonschema:"description:Transformation name"`
	Type             string `json:"type,omitempty" jsonschema:"description:Transformation type (optional). Valid values: tf_exclude_params, tf_allow_params, tf_augment_event"`
	ParametersJSON   string `json:"parametersJson,omitempty" jsonschema:"description:Transformation parameters as JSON array (optional)"`
	Notes            string `json:"notes,omitempty" jsonschema:"description:Transformation notes (optional)"`
}

// UpdateTransformationOutput is the output for update_transformation tool.
type UpdateTransformationOutput struct {
	Success        bool                  `json:"success"`
	Transformation CreatedTransformation `json:"transformation"`
	Message        string                `json:"message"`
}

func registerUpdateTransformation(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input UpdateTransformationInput) (*mcp.CallToolResult, UpdateTransformationOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, UpdateTransformationOutput{}, err
		}

		if input.TransformationID == "" {
			return nil, UpdateTransformationOutput{}, fmt.Errorf("transformation ID is required")
		}

		if err := ValidateTransformationInput(input.Name, input.Type); err != nil {
			return nil, UpdateTransformationOutput{}, err
		}

		path := BuildTransformationPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TransformationID)

		var params []Parameter
		if input.ParametersJSON != "" {
			if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
				return nil, UpdateTransformationOutput{}, err
			}
		}

		transformationInput := &TransformationInput{
			Name:      input.Name,
			Type:      input.Type,
			Parameter: params,
			Notes:     input.Notes,
		}

		t, err := wc.Client.UpdateTransformation(ctx, path, transformationInput)
		if err != nil {
			return nil, UpdateTransformationOutput{}, err
		}

		return nil, UpdateTransformationOutput{
			Success:        true,
			Transformation: *t,
			Message:        "Transformation updated successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "update_transformation",
		Description: `Update an existing transformation. Automatically handles fingerprint for concurrency control. Server-side containers only.

Type must be one of: tf_exclude_params, tf_allow_params, tf_augment_event. Table key and columns per type:
- tf_allow_params: "allowedParamsTable" with column "allowedParams"
- tf_exclude_params: "excludedParamsTable" with column "excludedParams"
- tf_augment_event: "augmentEventTable" with columns "paramName" and "paramValue"`,
	}, handler)
}
