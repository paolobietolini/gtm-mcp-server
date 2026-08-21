package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateContainerInput is the input for update_container tool.
type UpdateContainerInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:GTM container ID"`
	Name        string `json:"name" jsonschema:"description:New container display name"`
}

// UpdateContainerOutput is the output for update_container tool.
type UpdateContainerOutput struct {
	Success   bool      `json:"success"`
	Container Container `json:"container"`
	Message   string    `json:"message"`
}

func registerUpdateContainer(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input UpdateContainerInput) (*mcp.CallToolResult, UpdateContainerOutput, error) {
		cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
		if err != nil {
			return nil, UpdateContainerOutput{}, err
		}

		if input.Name == "" {
			return nil, UpdateContainerOutput{}, fmt.Errorf("name is required")
		}

		container, err := cc.Client.UpdateContainer(ctx, input.AccountID, input.ContainerID, input.Name)
		if err != nil {
			return nil, UpdateContainerOutput{}, err
		}

		return nil, UpdateContainerOutput{
			Success:   true,
			Container: *container,
			Message:   "Container updated successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_container",
		Description: "Rename a GTM container. Preserves all existing settings (usage context, domain, notes). Automatically handles fingerprint for concurrency control.",
	}, handler)
}
