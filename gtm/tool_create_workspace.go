package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// CreateWorkspaceInput is the input for create_workspace tool.
type CreateWorkspaceInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	Name        string `json:"name" jsonschema:"description:Workspace display name"`
	Description string `json:"description,omitempty" jsonschema:"description:Workspace description (optional)"`
}

// CreateWorkspaceOutput is the output for create_workspace tool.
type CreateWorkspaceOutput struct {
	Success   bool             `json:"success"`
	Workspace CreatedWorkspace `json:"workspace"`
	Message   string           `json:"message"`
}

// CreatedWorkspace is a simplified workspace response.
type CreatedWorkspace struct {
	WorkspaceID   string `json:"workspaceId"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Path          string `json:"path"`
	TagManagerUrl string `json:"tagManagerUrl,omitempty"`
}

func registerCreateWorkspace(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input CreateWorkspaceInput) (*mcp.CallToolResult, CreateWorkspaceOutput, error) {
		// Validate required fields
		if input.AccountID == "" {
			return nil, CreateWorkspaceOutput{}, fmt.Errorf("accountId is required")
		}
		if input.ContainerID == "" {
			return nil, CreateWorkspaceOutput{}, fmt.Errorf("containerId is required")
		}
		if input.Name == "" {
			return nil, CreateWorkspaceOutput{}, fmt.Errorf("name is required")
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, CreateWorkspaceOutput{}, err
		}

		parent := fmt.Sprintf("accounts/%s/containers/%s", input.AccountID, input.ContainerID)
		workspace := &tagmanager.Workspace{
			Name:        input.Name,
			Description: input.Description,
		}

		created, err := client.Service.Accounts.Containers.Workspaces.Create(parent, workspace).Context(ctx).Do()
		if err != nil {
			return nil, CreateWorkspaceOutput{}, mapGoogleError(err)
		}

		return nil, CreateWorkspaceOutput{
			Success: true,
			Workspace: CreatedWorkspace{
				WorkspaceID:   created.WorkspaceId,
				Name:          created.Name,
				Description:   created.Description,
				Path:          created.Path,
				TagManagerUrl: created.TagManagerUrl,
			},
			Message: "Workspace created successfully",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_workspace",
		Description: "Create a new workspace in a GTM container. Workspaces are used to make changes that can later be versioned and published.",
	}, handler)
}
