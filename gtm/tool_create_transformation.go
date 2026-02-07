package gtm

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateTransformationInput is the input for create_transformation tool.
type CreateTransformationInput struct {
	AccountID      string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID    string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID    string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Name           string `json:"name" jsonschema:"description:Transformation name"`
	Type           string `json:"type" jsonschema:"description:Transformation type. Valid values: tf_exclude_params (exclude parameters from tags), tf_allow_params (allow only specified parameters), tf_augment_event (add/modify event parameters)"`
	ParametersJSON string `json:"parametersJson,omitempty" jsonschema:"description:Transformation parameters as JSON array (optional). Each parameter: {type, key, value} or {type, key, list/map}"`
	Notes          string `json:"notes,omitempty" jsonschema:"description:Transformation notes (optional)"`
}

// CreateTransformationOutput is the output for create_transformation tool.
type CreateTransformationOutput struct {
	Success        bool                  `json:"success"`
	Transformation CreatedTransformation `json:"transformation"`
	Message        string                `json:"message"`
}

func registerCreateTransformation(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input CreateTransformationInput) (*mcp.CallToolResult, CreateTransformationOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, CreateTransformationOutput{}, err
		}

		if err := ValidateTransformationInput(input.Name, input.Type); err != nil {
			return nil, CreateTransformationOutput{}, err
		}

		var params []Parameter
		if input.ParametersJSON != "" {
			if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
				return nil, CreateTransformationOutput{}, err
			}
		}

		transformationInput := &TransformationInput{
			Name:      input.Name,
			Type:      input.Type,
			Parameter: params,
			Notes:     input.Notes,
		}

		t, err := wc.Client.CreateTransformation(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, transformationInput)
		if err != nil {
			return nil, CreateTransformationOutput{}, err
		}

		return nil, CreateTransformationOutput{
			Success:        true,
			Transformation: *t,
			Message:        "Transformation created successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name: "create_transformation",
		Description: `Create a new transformation in a GTM workspace (server-side containers only).

Type must be one of: tf_exclude_params, tf_allow_params, tf_augment_event.

Each type uses a different table key and column names in parametersJson:
- tf_allow_params: "allowedParamsTable" with column "allowedParams"
- tf_exclude_params: "excludedParamsTable" with column "excludedParams"
- tf_augment_event: "augmentEventTable" with columns "paramName" and "paramValue"

Common parameters shared by all types:
- matchingConditionsEnabled (boolean) — whether conditions must match
- allTagsExcept (boolean) — if true, apply to all tags except listed ones
- affectedTags (list of maps with tagReference) — specific tags to target
- affectedTagTypes (list of maps with tagType + tagTypeExceptions) — tag types to target
- matchingConditionsTable (list of maps with variableName, variableReference, expressionType, expressionValue)

Example for tf_exclude_params:
[{"key":"excludedParamsTable","type":"list","list":[{"type":"map","map":[{"key":"excludedParams","type":"template","value":"x-fb-ck-fbp"}]}]},{"key":"matchingConditionsEnabled","type":"boolean","value":"false"},{"key":"allTagsExcept","type":"boolean","value":"false"},{"key":"affectedTags","type":"list"},{"key":"affectedTagTypes","type":"list"}]`,
	}, handler)
}
