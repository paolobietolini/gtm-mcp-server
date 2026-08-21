package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateAccountInput is the input for update_account tool.
type UpdateAccountInput struct {
	AccountID string `json:"accountId" jsonschema:"description:GTM account ID"`
	Name      string `json:"name" jsonschema:"description:New account display name"`
}

// UpdateAccountOutput is the output for update_account tool.
type UpdateAccountOutput struct {
	Success bool    `json:"success"`
	Account Account `json:"account"`
	Message string  `json:"message"`
}

func registerUpdateAccount(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input UpdateAccountInput) (*mcp.CallToolResult, UpdateAccountOutput, error) {
		client, err := resolveAccount(ctx, input.AccountID)
		if err != nil {
			return nil, UpdateAccountOutput{}, err
		}

		if input.Name == "" {
			return nil, UpdateAccountOutput{}, fmt.Errorf("name is required")
		}

		account, err := client.UpdateAccount(ctx, input.AccountID, input.Name)
		if err != nil {
			return nil, UpdateAccountOutput{}, err
		}

		return nil, UpdateAccountOutput{
			Success: true,
			Account: *account,
			Message: "Account updated successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_account",
		Description: "Rename a GTM account. Automatically handles fingerprint for concurrency control.",
	}, handler)
}
