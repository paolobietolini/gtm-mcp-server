package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// CreateContainerInput is the input for create_container tool.
type CreateContainerInput struct {
	AccountID         string   `json:"accountId" jsonschema:"description:The GTM account ID"`
	Name              string   `json:"name" jsonschema:"description:Container display name"`
	UsageContext      []string `json:"usageContext" jsonschema:"description:Usage context for the container. Valid values: web, android, ios, amp, server"`
	Notes             string   `json:"notes,omitempty" jsonschema:"description:Container notes (optional)"`
	DomainName        []string `json:"domainName,omitempty" jsonschema:"description:List of domain names associated with the container (optional)"`
	TaggingServerUrls []string `json:"taggingServerUrls,omitempty" jsonschema:"description:Server-side container URLs (for server containers only)"`
}

// CreateContainerOutput is the output for create_container tool.
type CreateContainerOutput struct {
	Success   bool             `json:"success"`
	Container CreatedContainer `json:"container"`
	Message   string           `json:"message"`
}

// CreatedContainer is a simplified container response.
type CreatedContainer struct {
	ContainerID   string   `json:"containerId"`
	Name          string   `json:"name"`
	PublicID      string   `json:"publicId"`
	UsageContext  []string `json:"usageContext"`
	Path          string   `json:"path"`
	TagManagerUrl string   `json:"tagManagerUrl,omitempty"`
}

func registerCreateContainer(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input CreateContainerInput) (*mcp.CallToolResult, CreateContainerOutput, error) {
		client, err := resolveAccount(ctx, input.AccountID)
		if err != nil {
			return nil, CreateContainerOutput{}, err
		}

		// Validate name
		if input.Name == "" {
			return nil, CreateContainerOutput{}, fmt.Errorf("name is required")
		}

		// Validate usage context
		if len(input.UsageContext) == 0 {
			return nil, CreateContainerOutput{}, fmt.Errorf("usageContext is required (valid values: web, android, ios, amp, server)")
		}
		validContexts := map[string]bool{"web": true, "android": true, "ios": true, "androidSdk5": true, "iosSdk5": true, "amp": true, "server": true}
		for _, uc := range input.UsageContext {
			if !validContexts[uc] {
				return nil, CreateContainerOutput{}, fmt.Errorf("invalid usageContext '%s' (valid values: web, android, ios, amp, server)", uc)
			}
		}

		parent := fmt.Sprintf("accounts/%s", input.AccountID)
		container := &tagmanager.Container{
			Name:         input.Name,
			UsageContext: input.UsageContext,
			Notes:        input.Notes,
			DomainName:   input.DomainName,
		}

		if len(input.TaggingServerUrls) > 0 {
			container.TaggingServerUrls = input.TaggingServerUrls
		}

		created, err := client.Service.Accounts.Containers.Create(parent, container).Context(ctx).Do()
		if err != nil {
			return nil, CreateContainerOutput{}, mapGoogleError(err)
		}

		return nil, CreateContainerOutput{
			Success: true,
			Container: CreatedContainer{
				ContainerID:   created.ContainerId,
				Name:          created.Name,
				PublicID:      created.PublicId,
				UsageContext:  created.UsageContext,
				Path:          created.Path,
				TagManagerUrl: created.TagManagerUrl,
			},
			Message: "Container created successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_container",
		Description: "Create a new container in a GTM account. UsageContext specifies the container type (web, android, ios, amp, server).",
	}, handler)
}
