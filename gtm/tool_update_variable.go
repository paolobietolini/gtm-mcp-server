package gtm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateVariableInput is the input for update_variable tool.
type UpdateVariableInput struct {
	AccountID      string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID    string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID    string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	VariableID     string `json:"variableId" jsonschema:"description:The variable ID to update"`
	Name           string `json:"name" jsonschema:"description:Variable name"`
	Type           string `json:"type" jsonschema:"description:Variable type (e.g. c for Constant, v for Data Layer, k for Cookie, jsm for Custom JavaScript)"`
	ParametersJSON string `json:"parametersJson,omitempty" jsonschema:"description:Variable parameters as JSON array (required for most types)"`
	Notes          string `json:"notes,omitempty" jsonschema:"description:Variable notes (optional)"`
}

// UpdateVariableOutput is the output for update_variable tool.
type UpdateVariableOutput struct {
	Success  bool            `json:"success"`
	Variable CreatedVariable `json:"variable"`
	Message  string          `json:"message"`
}

func registerUpdateVariable(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input UpdateVariableInput) (*mcp.CallToolResult, UpdateVariableOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, UpdateVariableOutput{}, err
		}

		if input.VariableID == "" {
			return nil, UpdateVariableOutput{}, fmt.Errorf("variableId is required")
		}
		if input.Name == "" {
			return nil, UpdateVariableOutput{}, fmt.Errorf("name is required")
		}
		if input.Type == "" {
			return nil, UpdateVariableOutput{}, fmt.Errorf("type is required")
		}

		path := BuildVariablePath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)

		var params []Parameter
		if input.ParametersJSON != "" {
			if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
				return nil, UpdateVariableOutput{}, fmt.Errorf("invalid parametersJson: %w", err)
			}
		}

		variableInput := &VariableInput{
			Name:      input.Name,
			Type:      input.Type,
			Parameter: params,
			Notes:     input.Notes,
		}

		variable, err := wc.Client.UpdateVariable(ctx, path, variableInput)
		if err != nil {
			return nil, UpdateVariableOutput{}, err
		}

		return nil, UpdateVariableOutput{
			Success:  true,
			Variable: *variable,
			Message:  "Variable updated successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_variable",
		Description: "Update an existing variable. Automatically handles fingerprint for concurrency control.",
	}, handler)
}
