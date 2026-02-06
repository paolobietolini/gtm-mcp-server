package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// ListTemplatesInput is the input for list_templates tool.
type ListTemplatesInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
}

// ListTemplatesOutput is the output for list_templates tool.
type ListTemplatesOutput struct {
	Templates []TemplateInfo `json:"templates"`
}

// TemplateInfo is a simplified template response.
type TemplateInfo struct {
	TemplateID       string                `json:"templateId"`
	Name             string                `json:"name"`
	Type             string                `json:"type"` // cvt_{galleryTemplateId} for gallery templates
	GalleryReference *GalleryReferenceInfo `json:"galleryReference,omitempty"`
	TagManagerUrl    string                `json:"tagManagerUrl,omitempty"`
}

// GalleryReferenceInfo contains gallery template info.
type GalleryReferenceInfo struct {
	Owner             string `json:"owner"`
	Repository        string `json:"repository"`
	Version           string `json:"version,omitempty"`
	GalleryTemplateId string `json:"galleryTemplateId,omitempty"`
}

func registerListTemplates(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListTemplatesInput) (*mcp.CallToolResult, ListTemplatesOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, ListTemplatesOutput{}, err
		}

		parent := wc.WorkspacePath()
		resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListTemplatesResponse, error) {
			return wc.Client.Service.Accounts.Containers.Workspaces.Templates.List(parent).Context(ctx).Do()
		})
		if err != nil {
			return nil, ListTemplatesOutput{}, mapGoogleError(err)
		}

		templates := make([]TemplateInfo, 0)
		if resp.Template != nil {
			for _, t := range resp.Template {
				info := TemplateInfo{
					TemplateID:    t.TemplateId,
					Name:          t.Name,
					TagManagerUrl: t.TagManagerUrl,
				}
				// For gallery templates, use cvt_{galleryTemplateId}
				// For custom templates, use cvt_{containerId}_{templateId}
				if t.GalleryReference != nil && t.GalleryReference.GalleryTemplateId != "" {
					info.Type = fmt.Sprintf("cvt_%s", t.GalleryReference.GalleryTemplateId)
					info.GalleryReference = &GalleryReferenceInfo{
						Owner:             t.GalleryReference.Owner,
						Repository:        t.GalleryReference.Repository,
						Version:           t.GalleryReference.Version,
						GalleryTemplateId: t.GalleryReference.GalleryTemplateId,
					}
				} else {
					info.Type = fmt.Sprintf("cvt_%s_%s", wc.ContainerID, t.TemplateId)
				}
				templates = append(templates, info)
			}
		}

		return nil, ListTemplatesOutput{
			Templates: templates,
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_templates",
		Description: "List all GTM Custom Templates in a workspace. Returns template IDs and their type strings (cvt_{galleryTemplateId} for gallery templates) for use when creating tags.",
	}, handler)
}
