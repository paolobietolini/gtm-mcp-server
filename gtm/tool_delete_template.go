package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteTemplateInput is the input for delete_template tool.
type DeleteTemplateInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TemplateID  string `json:"templateId" jsonschema:"description:The template ID to delete"`
	Confirm     bool   `json:"confirm" jsonschema:"description:Must be true to confirm deletion. This is a safety guard."`
}

// DeleteTemplateOutput is the output for delete_template tool.
type DeleteTemplateOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerDeleteTemplate(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input DeleteTemplateInput) (*mcp.CallToolResult, DeleteTemplateOutput, error) {
		// Validate required fields
		if input.AccountID == "" {
			return nil, DeleteTemplateOutput{}, fmt.Errorf("accountId is required")
		}
		if input.ContainerID == "" {
			return nil, DeleteTemplateOutput{}, fmt.Errorf("containerId is required")
		}
		if input.WorkspaceID == "" {
			return nil, DeleteTemplateOutput{}, fmt.Errorf("workspaceId is required")
		}
		if input.TemplateID == "" {
			return nil, DeleteTemplateOutput{}, fmt.Errorf("templateId is required")
		}
		if !input.Confirm {
			return nil, DeleteTemplateOutput{}, fmt.Errorf("confirm must be true to delete a template")
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, DeleteTemplateOutput{}, err
		}

		path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/templates/%s",
			input.AccountID, input.ContainerID, input.WorkspaceID, input.TemplateID)

		err = client.Service.Accounts.Containers.Workspaces.Templates.Delete(path).Context(ctx).Do()
		if err != nil {
			return nil, DeleteTemplateOutput{}, mapGoogleError(err)
		}

		return nil, DeleteTemplateOutput{
			Success: true,
			Message: fmt.Sprintf("Template %s deleted successfully", input.TemplateID),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_template",
		Description: "Delete a custom template from a workspace. Requires confirm: true as a safety guard. Note: Templates that are in use by tags cannot be deleted.",
	}, handler)
}
