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
		if !input.Confirm {
			return nil, DeleteTemplateOutput{
				Success: false,
				Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions. Templates in use by tags cannot be deleted.",
			}, nil
		}

		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, DeleteTemplateOutput{}, err
		}

		if input.TemplateID == "" {
			return nil, DeleteTemplateOutput{}, fmt.Errorf("templateId is required")
		}

		path := fmt.Sprintf("%s/templates/%s", wc.WorkspacePath(), input.TemplateID)

		err = wc.Client.Service.Accounts.Containers.Workspaces.Templates.Delete(path).Context(ctx).Do()
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
