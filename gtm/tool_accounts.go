package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListAccountsInput struct{}
type ListAccountsOutput struct {
	Accounts []Account `json:"accounts"`
}

func registerListAccounts(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListAccountsInput) (*mcp.CallToolResult, ListAccountsOutput, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, ListAccountsOutput{}, err
		}

		accounts, err := client.ListAccounts(ctx)
		if err != nil {
			return nil, ListAccountsOutput{}, err
		}

		return nil, ListAccountsOutput{Accounts: accounts}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_accounts",
		Description: "List all GTM accounts accessible to the authenticated user",
	}, handler)
}
