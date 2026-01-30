package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetTemplateInput is the input for get_template tool.
type GetTemplateInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TemplateID  string `json:"templateId" jsonschema:"description:The template ID to retrieve"`
}

// GetTemplateOutput is the output for get_template tool.
type GetTemplateOutput struct {
	TemplateID       string                `json:"templateId"`
	Name             string                `json:"name"`
	Type             string                `json:"type"`
	TemplateData     string                `json:"templateData,omitempty"`
	GalleryReference *GalleryReferenceInfo `json:"galleryReference,omitempty"`
	Path             string                `json:"path"`
	Fingerprint      string                `json:"fingerprint"`
	TagManagerUrl    string                `json:"tagManagerUrl,omitempty"`
}

func registerGetTemplate(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input GetTemplateInput) (*mcp.CallToolResult, GetTemplateOutput, error) {
		// Validate required fields
		if input.AccountID == "" {
			return nil, GetTemplateOutput{}, fmt.Errorf("accountId is required")
		}
		if input.ContainerID == "" {
			return nil, GetTemplateOutput{}, fmt.Errorf("containerId is required")
		}
		if input.WorkspaceID == "" {
			return nil, GetTemplateOutput{}, fmt.Errorf("workspaceId is required")
		}
		if input.TemplateID == "" {
			return nil, GetTemplateOutput{}, fmt.Errorf("templateId is required")
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, GetTemplateOutput{}, err
		}

		path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/templates/%s",
			input.AccountID, input.ContainerID, input.WorkspaceID, input.TemplateID)

		template, err := client.Service.Accounts.Containers.Workspaces.Templates.Get(path).Context(ctx).Do()
		if err != nil {
			return nil, GetTemplateOutput{}, mapGoogleError(err)
		}

		output := GetTemplateOutput{
			TemplateID:    template.TemplateId,
			Name:          template.Name,
			TemplateData:  template.TemplateData,
			Path:          template.Path,
			Fingerprint:   template.Fingerprint,
			TagManagerUrl: template.TagManagerUrl,
		}

		// Set type based on gallery reference
		if template.GalleryReference != nil && template.GalleryReference.GalleryTemplateId != "" {
			output.Type = fmt.Sprintf("cvt_%s", template.GalleryReference.GalleryTemplateId)
			output.GalleryReference = &GalleryReferenceInfo{
				Owner:             template.GalleryReference.Owner,
				Repository:        template.GalleryReference.Repository,
				Version:           template.GalleryReference.Version,
				GalleryTemplateId: template.GalleryReference.GalleryTemplateId,
			}
		} else {
			output.Type = fmt.Sprintf("cvt_%s_%s", input.ContainerID, template.TemplateId)
		}

		return nil, output, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_template",
		Description: "Get a specific custom template by ID. Returns full template details including the template code.",
	}, handler)
}
