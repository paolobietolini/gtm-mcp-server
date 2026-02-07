package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// -- List Clients --

type ListClientsInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
}

type ListClientsOutput struct {
	Clients []ClientInfo `json:"clients"`
}

func registerListClients(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListClientsInput) (*mcp.CallToolResult, ListClientsOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, ListClientsOutput{}, err
		}

		clients, err := wc.Client.ListClients(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
		if err != nil {
			return nil, ListClientsOutput{}, err
		}

		return nil, ListClientsOutput{Clients: clients}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_clients",
		Description: "List all clients in a GTM workspace (server-side containers only)",
	}, handler)
}

// -- Get Client --

type GetClientInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	ClientID    string `json:"clientId" jsonschema:"description:The client ID to retrieve"`
}

type GetClientOutput struct {
	Client ClientInfo `json:"client"`
}

func registerGetClient(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input GetClientInput) (*mcp.CallToolResult, GetClientOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, GetClientOutput{}, err
		}

		cl, err := wc.Client.GetClient(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ClientID)
		if err != nil {
			return nil, GetClientOutput{}, err
		}

		return nil, GetClientOutput{Client: *cl}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_client",
		Description: "Get a specific client by ID (server-side containers only)",
	}, handler)
}
