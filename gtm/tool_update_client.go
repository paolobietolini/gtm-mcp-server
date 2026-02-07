package gtm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateClientInput is the input for update_client tool.
type UpdateClientInput struct {
	AccountID      string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID    string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID    string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	ClientID       string `json:"clientId" jsonschema:"description:The client ID to update"`
	Name           string `json:"name" jsonschema:"description:Client name"`
	Type           string `json:"type" jsonschema:"description:Client type"`
	Priority       int64  `json:"priority,omitempty" jsonschema:"description:Client priority (optional, higher runs first)"`
	ParametersJSON string `json:"parametersJson,omitempty" jsonschema:"description:Client parameters as JSON array (optional)"`
	Notes          string `json:"notes,omitempty" jsonschema:"description:Client notes (optional)"`
}

// UpdateClientOutput is the output for update_client tool.
type UpdateClientOutput struct {
	Success bool          `json:"success"`
	Client  CreatedClient `json:"client"`
	Message string        `json:"message"`
}

func registerUpdateClient(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input UpdateClientInput) (*mcp.CallToolResult, UpdateClientOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, UpdateClientOutput{}, err
		}

		if input.ClientID == "" {
			return nil, UpdateClientOutput{}, fmt.Errorf("client ID is required")
		}

		if err := ValidateClientInput(input.Name, input.Type); err != nil {
			return nil, UpdateClientOutput{}, err
		}

		path := BuildClientPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ClientID)

		var params []Parameter
		if input.ParametersJSON != "" {
			if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
				return nil, UpdateClientOutput{}, err
			}
		}

		clientInput := &ClientInput{
			Name:      input.Name,
			Type:      input.Type,
			Priority:  input.Priority,
			Parameter: params,
			Notes:     input.Notes,
		}

		cl, err := wc.Client.UpdateClient(ctx, path, clientInput)
		if err != nil {
			return nil, UpdateClientOutput{}, err
		}

		return nil, UpdateClientOutput{
			Success: true,
			Client:  *cl,
			Message: "Client updated successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_client",
		Description: "Update an existing client. Automatically handles fingerprint for concurrency control. Server-side containers only.",
	}, handler)
}
