package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateVersionInput is the input for create_version tool.
type CreateVersionInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Name        string `json:"name,omitempty" jsonschema:"description:Version name (optional)"`
	Notes       string `json:"notes,omitempty" jsonschema:"description:Version notes describing changes (optional)"`
}

// CreateVersionOutput is the output for create_version tool.
type CreateVersionOutput struct {
	Success bool           `json:"success"`
	Version CreatedVersion `json:"version"`
	Message string         `json:"message"`
}

// PublishVersionInput is the input for publish_version tool.
type PublishVersionInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	VersionID   string `json:"versionId" jsonschema:"description:The version ID to publish"`
	Confirm     bool   `json:"confirm" jsonschema:"description:Must be true to confirm publishing. This is a safety guard - publishing makes changes live."`
}

// PublishVersionOutput is the output for publish_version tool.
type PublishVersionOutput struct {
	Success bool             `json:"success"`
	Version PublishedVersion `json:"version"`
	Message string           `json:"message"`
}

func registerCreateVersion(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input CreateVersionInput) (*mcp.CallToolResult, CreateVersionOutput, error) {
		// Validate workspace path
		if err := ValidateWorkspacePath(input.AccountID, input.ContainerID, input.WorkspaceID); err != nil {
			return nil, CreateVersionOutput{}, err
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, CreateVersionOutput{}, err
		}

		// Check workspace status first
		status, err := client.GetWorkspaceStatus(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, CreateVersionOutput{}, err
		}

		if !status.HasChanges {
			return nil, CreateVersionOutput{
				Success: false,
				Message: "No changes in workspace to create version from",
			}, nil
		}

		if status.HasConflicts {
			return nil, CreateVersionOutput{
				Success: false,
				Message: fmt.Sprintf("Workspace has %d conflicts that must be resolved before creating a version", status.ConflictCount),
			}, nil
		}

		versionInput := &VersionInput{
			Name:  input.Name,
			Notes: input.Notes,
		}

		version, err := client.CreateVersion(ctx, input.AccountID, input.ContainerID, input.WorkspaceID, versionInput)
		if err != nil {
			return nil, CreateVersionOutput{}, err
		}

		return nil, CreateVersionOutput{
			Success: true,
			Version: *version,
			Message: "Version created successfully. Use publish_version to make it live.",
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_version",
		Description: "Create a new container version from workspace changes. This snapshots the current workspace state but does not publish it.",
	}, handler)
}

func registerPublishVersion(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input PublishVersionInput) (*mcp.CallToolResult, PublishVersionOutput, error) {
		// Safety guard: require explicit confirmation
		if !input.Confirm {
			return nil, PublishVersionOutput{
				Success: false,
				Message: "Publishing requires confirm: true. WARNING: This will make the version live on your website.",
			}, nil
		}

		// Validate input
		if input.AccountID == "" || input.ContainerID == "" || input.VersionID == "" {
			return nil, PublishVersionOutput{}, fmt.Errorf("accountId, containerId, and versionId are required")
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, PublishVersionOutput{}, err
		}

		version, err := client.PublishVersion(ctx, input.AccountID, input.ContainerID, input.VersionID)
		if err != nil {
			return nil, PublishVersionOutput{}, err
		}

		return nil, PublishVersionOutput{
			Success: true,
			Version: *version,
			Message: fmt.Sprintf("Version %s is now LIVE", version.VersionID),
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "publish_version",
		Description: "Publish a container version to make it live. Requires confirm: true as a safety guard. WARNING: This pushes changes to your live website.",
	}, handler)
}
