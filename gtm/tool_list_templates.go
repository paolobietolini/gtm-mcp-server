package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	Type             string                `json:"type"` // cvt_{containerId}_{templateId}
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
		// Validate required fields
		if input.AccountID == "" {
			return nil, ListTemplatesOutput{}, fmt.Errorf("accountId is required")
		}
		if input.ContainerID == "" {
			return nil, ListTemplatesOutput{}, fmt.Errorf("containerId is required")
		}
		if input.WorkspaceID == "" {
			return nil, ListTemplatesOutput{}, fmt.Errorf("workspaceId is required")
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, ListTemplatesOutput{}, err
		}

		parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", input.AccountID, input.ContainerID, input.WorkspaceID)
		resp, err := client.Service.Accounts.Containers.Workspaces.Templates.List(parent).Context(ctx).Do()
		if err != nil {
			return nil, ListTemplatesOutput{}, mapGoogleError(err)
		}

		templates := make([]TemplateInfo, 0)
		if resp.Template != nil {
			for _, t := range resp.Template {
				info := TemplateInfo{
					TemplateID:    t.TemplateId,
					Name:          t.Name,
					Type:          fmt.Sprintf("cvt_%s_%s", input.ContainerID, t.TemplateId),
					TagManagerUrl: t.TagManagerUrl,
				}
				if t.GalleryReference != nil {
					info.GalleryReference = &GalleryReferenceInfo{
						Owner:             t.GalleryReference.Owner,
						Repository:        t.GalleryReference.Repository,
						Version:           t.GalleryReference.Version,
						GalleryTemplateId: t.GalleryReference.GalleryTemplateId,
					}
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
		Description: "List all GTM Custom Templates in a workspace. Returns template IDs and their type strings (cvt_{containerId}_{templateId}) for use in variables/tags.",
	}, handler)
}
