package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListContainersInput struct {
	AccountID string `json:"accountId" jsonschema:"description:The GTM account ID"`
}
type ListContainersOutput struct {
	Containers []Container `json:"containers"`
}

func registerListContainers(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListContainersInput) (*mcp.CallToolResult, ListContainersOutput, error) {
		client, err := resolveAccount(ctx, input.AccountID)
		if err != nil {
			return nil, ListContainersOutput{}, err
		}

		containers, err := client.ListContainers(ctx, input.AccountID)
		if err != nil {
			return nil, ListContainersOutput{}, err
		}

		return nil, ListContainersOutput{Containers: containers}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_containers",
		Description: "List all containers in a GTM account",
	}, handler)
}
