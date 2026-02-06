package gtm

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateVariableInput is the input for create_variable tool.
type CreateVariableInput struct {
	AccountID      string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID    string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID    string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Name           string `json:"name" jsonschema:"description:Variable name"`
	Type           string `json:"type" jsonschema:"description:Variable type (e.g. c for Constant, v for Data Layer, k for Cookie, jsm for Custom JavaScript)"`
	ParametersJSON string `json:"parametersJson,omitempty" jsonschema:"description:Variable parameters as JSON array (required for most types)"`
	Notes          string `json:"notes,omitempty" jsonschema:"description:Variable notes (optional)"`
}

// CreateVariableOutput is the output for create_variable tool.
type CreateVariableOutput struct {
	Success  bool            `json:"success"`
	Variable CreatedVariable `json:"variable"`
	Message  string          `json:"message"`
}

func registerCreateVariable(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input CreateVariableInput) (*mcp.CallToolResult, CreateVariableOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, CreateVariableOutput{}, err
		}

		// Validate variable input
		if err := ValidateVariableInput(input.Name, input.Type); err != nil {
			return nil, CreateVariableOutput{}, err
		}

		// Parse parameters JSON if provided
		var params []Parameter
		if input.ParametersJSON != "" {
			if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
				return nil, CreateVariableOutput{}, err
			}
		}

		variableInput := &VariableInput{
			Name:      input.Name,
			Type:      input.Type,
			Parameter: params,
			Notes:     input.Notes,
		}

		variable, err := wc.Client.CreateVariable(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, variableInput)
		if err != nil {
			return nil, CreateVariableOutput{}, err
		}

		return nil, CreateVariableOutput{
			Success:  true,
			Variable: *variable,
			Message:  "Variable created successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_variable",
		Description: "Create a new variable in a GTM workspace. Common types: c (Constant), v (Data Layer), k (Cookie), jsm (Custom JavaScript), u (URL).",
	}, handler)
}
