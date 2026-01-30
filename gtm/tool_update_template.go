package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// UpdateTemplateInput is the input for update_template tool.
type UpdateTemplateInput struct {
	AccountID    string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID  string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID  string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TemplateID   string `json:"templateId" jsonschema:"description:The template ID to update"`
	Name         string `json:"name,omitempty" jsonschema:"description:New template display name (optional)"`
	TemplateData string `json:"templateData,omitempty" jsonschema:"description:New template code in .tpl format (optional)"`
}

// UpdateTemplateOutput is the output for update_template tool.
type UpdateTemplateOutput struct {
	Success       bool   `json:"success"`
	TemplateID    string `json:"templateId"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Path          string `json:"path"`
	Fingerprint   string `json:"fingerprint"`
	TagManagerUrl string `json:"tagManagerUrl,omitempty"`
	Message       string `json:"message"`
}

func registerUpdateTemplate(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input UpdateTemplateInput) (*mcp.CallToolResult, UpdateTemplateOutput, error) {
		// Validate required fields
		if input.AccountID == "" {
			return nil, UpdateTemplateOutput{}, fmt.Errorf("accountId is required")
		}
		if input.ContainerID == "" {
			return nil, UpdateTemplateOutput{}, fmt.Errorf("containerId is required")
		}
		if input.WorkspaceID == "" {
			return nil, UpdateTemplateOutput{}, fmt.Errorf("workspaceId is required")
		}
		if input.TemplateID == "" {
			return nil, UpdateTemplateOutput{}, fmt.Errorf("templateId is required")
		}
		if input.Name == "" && input.TemplateData == "" {
			return nil, UpdateTemplateOutput{}, fmt.Errorf("at least one of name or templateData must be provided")
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, UpdateTemplateOutput{}, err
		}

		path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/templates/%s",
			input.AccountID, input.ContainerID, input.WorkspaceID, input.TemplateID)

		// Get current template to get fingerprint and current values
		current, err := client.Service.Accounts.Containers.Workspaces.Templates.Get(path).Context(ctx).Do()
		if err != nil {
			return nil, UpdateTemplateOutput{}, mapGoogleError(err)
		}

		// Build update with current values as defaults
		template := &tagmanager.CustomTemplate{
			Name:         current.Name,
			TemplateData: current.TemplateData,
			Fingerprint:  current.Fingerprint,
		}

		// Override with provided values
		if input.Name != "" {
			template.Name = input.Name
		}
		if input.TemplateData != "" {
			template.TemplateData = input.TemplateData
		}

		updated, err := client.Service.Accounts.Containers.Workspaces.Templates.Update(path, template).Context(ctx).Do()
		if err != nil {
			return nil, UpdateTemplateOutput{}, mapGoogleError(err)
		}

		// Determine type
		templateType := fmt.Sprintf("cvt_%s_%s", input.ContainerID, updated.TemplateId)
		if updated.GalleryReference != nil && updated.GalleryReference.GalleryTemplateId != "" {
			templateType = fmt.Sprintf("cvt_%s", updated.GalleryReference.GalleryTemplateId)
		}

		return nil, UpdateTemplateOutput{
			Success:       true,
			TemplateID:    updated.TemplateId,
			Name:          updated.Name,
			Type:          templateType,
			Path:          updated.Path,
			Fingerprint:   updated.Fingerprint,
			TagManagerUrl: updated.TagManagerUrl,
			Message:       fmt.Sprintf("Template '%s' updated successfully", updated.Name),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_template",
		Description: "Update an existing custom template. Automatically handles fingerprint for concurrency control. Note: Updating gallery templates may break the link to the gallery.",
	}, handler)
}
