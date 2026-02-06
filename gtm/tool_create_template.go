package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// CreateTemplateInput is the input for create_template tool.
type CreateTemplateInput struct {
	AccountID    string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID  string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID  string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Name         string `json:"name" jsonschema:"description:Template display name"`
	TemplateData string `json:"templateData" jsonschema:"description:The template code in .tpl format (the full template file content)"`
}

// CreateTemplateOutput is the output for create_template tool.
type CreateTemplateOutput struct {
	Success       bool   `json:"success"`
	TemplateID    string `json:"templateId"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Path          string `json:"path"`
	TagManagerUrl string `json:"tagManagerUrl,omitempty"`
	Message       string `json:"message"`
}

func registerCreateTemplate(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input CreateTemplateInput) (*mcp.CallToolResult, CreateTemplateOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, CreateTemplateOutput{}, err
		}

		if input.Name == "" {
			return nil, CreateTemplateOutput{}, fmt.Errorf("name is required")
		}
		if input.TemplateData == "" {
			return nil, CreateTemplateOutput{}, fmt.Errorf("templateData is required")
		}

		parent := wc.WorkspacePath()

		template := &tagmanager.CustomTemplate{
			Name:         input.Name,
			TemplateData: input.TemplateData,
		}

		created, err := wc.Client.Service.Accounts.Containers.Workspaces.Templates.Create(parent, template).Context(ctx).Do()
		if err != nil {
			return nil, CreateTemplateOutput{}, mapGoogleError(err)
		}

		return nil, CreateTemplateOutput{
			Success:       true,
			TemplateID:    created.TemplateId,
			Name:          created.Name,
			Type:          fmt.Sprintf("cvt_%s_%s", wc.ContainerID, created.TemplateId),
			Path:          created.Path,
			TagManagerUrl: created.TagManagerUrl,
			Message:       fmt.Sprintf("Template '%s' created successfully. Use type 'cvt_%s_%s' when creating tags.", created.Name, wc.ContainerID, created.TemplateId),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_template",
		Description: "Create a new custom template in a GTM workspace. Requires the full template code in .tpl format. For gallery templates, use import_gallery_template instead.",
	}, handler)
}
