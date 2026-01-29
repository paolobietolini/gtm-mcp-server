package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ImportGalleryTemplateInput is the input for import_gallery_template tool.
type ImportGalleryTemplateInput struct {
	AccountID     string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID   string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID   string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	GalleryOwner  string `json:"galleryOwner" jsonschema:"description:Owner of the Gallery template (e.g. 'iubenda' or 'GoogleAnalytics')"`
	GalleryRepo   string `json:"galleryRepository" jsonschema:"description:Repository of the Gallery template (e.g. 'gtm-cookie-solution')"`
	GallerySha    string `json:"gallerySha,omitempty" jsonschema:"description:SHA version of the Gallery template. Defaults to latest if not provided"`
}

// ImportGalleryTemplateOutput is the output for import_gallery_template tool.
type ImportGalleryTemplateOutput struct {
	Success  bool         `json:"success"`
	Template TemplateInfo `json:"template"`
	Message  string       `json:"message"`
}

func registerImportGalleryTemplate(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ImportGalleryTemplateInput) (*mcp.CallToolResult, ImportGalleryTemplateOutput, error) {
		// Validate required fields
		if input.AccountID == "" {
			return nil, ImportGalleryTemplateOutput{}, fmt.Errorf("accountId is required")
		}
		if input.ContainerID == "" {
			return nil, ImportGalleryTemplateOutput{}, fmt.Errorf("containerId is required")
		}
		if input.WorkspaceID == "" {
			return nil, ImportGalleryTemplateOutput{}, fmt.Errorf("workspaceId is required")
		}
		if input.GalleryOwner == "" {
			return nil, ImportGalleryTemplateOutput{}, fmt.Errorf("galleryOwner is required")
		}
		if input.GalleryRepo == "" {
			return nil, ImportGalleryTemplateOutput{}, fmt.Errorf("galleryRepository is required")
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, ImportGalleryTemplateOutput{}, err
		}

		parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", input.AccountID, input.ContainerID, input.WorkspaceID)

		call := client.Service.Accounts.Containers.Workspaces.Templates.ImportFromGallery(parent).
			GalleryOwner(input.GalleryOwner).
			GalleryRepository(input.GalleryRepo).
			AcknowledgePermissions(true)

		if input.GallerySha != "" {
			call = call.GallerySha(input.GallerySha)
		}

		template, err := call.Context(ctx).Do()
		if err != nil {
			return nil, ImportGalleryTemplateOutput{}, mapGoogleError(err)
		}

		result := TemplateInfo{
			TemplateID:    template.TemplateId,
			Name:          template.Name,
			Type:          fmt.Sprintf("cvt_%s_%s", input.ContainerID, template.TemplateId),
			TagManagerUrl: template.TagManagerUrl,
		}

		if template.GalleryReference != nil {
			result.GalleryReference = &GalleryReferenceInfo{
				Owner:             template.GalleryReference.Owner,
				Repository:        template.GalleryReference.Repository,
				Version:           template.GalleryReference.Version,
				GalleryTemplateId: template.GalleryReference.GalleryTemplateId,
			}
		}

		return nil, ImportGalleryTemplateOutput{
			Success:  true,
			Template: result,
			Message:  fmt.Sprintf("Template '%s' imported successfully. Use type '%s' when creating tags.", template.Name, result.Type),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "import_gallery_template",
		Description: "Import a GTM Custom Template from the Community Template Gallery into a workspace. Returns the template type string to use when creating tags. Example: import_gallery_template(galleryOwner='iubenda', galleryRepository='gtm-cookie-solution')",
	}, handler)
}
