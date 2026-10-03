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
	Type           string `json:"type" jsonschema:"description:One of tf_exclude_params, tf_allow_params, or tf_augment_event"`
	ParametersJSON string `json:"parametersJson,omitempty" jsonschema:"description:JSON parameter array; see gtm://best-practices/tool-input-formats"`
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
		Name:        "create_transformation",
		Description: "Create a transformation in a server-side workspace. Read gtm://best-practices/tool-input-formats for parameter table shapes.",
	}, handler)
}
