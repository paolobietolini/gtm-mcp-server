package gtm

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateClientInput is the input for create_client tool.
type CreateClientInput struct {
	AccountID      string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID    string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID    string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Name           string `json:"name" jsonschema:"description:Client name"`
	Type           string `json:"type" jsonschema:"description:Client type (e.g. __ga4 for GA4, __googtag for Google tag)"`
	Priority       int64  `json:"priority,omitempty" jsonschema:"description:Client priority (optional, higher runs first)"`
	ParametersJSON string `json:"parametersJson,omitempty" jsonschema:"description:Client parameters as JSON array (optional). Each parameter: {type, key, value} or {type, key, list/map}"`
	Notes          string `json:"notes,omitempty" jsonschema:"description:Client notes (optional)"`
}

// CreateClientOutput is the output for create_client tool.
type CreateClientOutput struct {
	Success bool          `json:"success"`
	Client  CreatedClient `json:"client"`
	Message string        `json:"message"`
}

func registerCreateClient(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input CreateClientInput) (*mcp.CallToolResult, CreateClientOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, CreateClientOutput{}, err
		}

		if err := ValidateClientInput(input.Name, input.Type); err != nil {
			return nil, CreateClientOutput{}, err
		}

		var params []Parameter
		if input.ParametersJSON != "" {
			if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
				return nil, CreateClientOutput{}, err
			}
		}

		clientInput := &ClientInput{
			Name:      input.Name,
			Type:      input.Type,
			Priority:  input.Priority,
			Parameter: params,
			Notes:     input.Notes,
		}

		cl, err := wc.Client.CreateClient(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, clientInput)
		if err != nil {
			return nil, CreateClientOutput{}, err
		}

		return nil, CreateClientOutput{
			Success: true,
			Client:  *cl,
			Message: "Client created successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_client",
		Description: "Create a new client in a GTM workspace (server-side containers only)",
	}, handler)
}
