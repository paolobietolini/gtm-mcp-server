package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListFoldersInput is the input for list_folders tool.
type ListFoldersInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
}

// ListFoldersOutput is the output for list_folders tool.
type ListFoldersOutput struct {
	Folders []Folder `json:"folders"`
}

// GetFolderEntitiesInput is the input for get_folder_entities tool.
type GetFolderEntitiesInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	FolderID    string `json:"folderId" jsonschema:"description:The folder ID"`
}

// GetFolderEntitiesOutput is the output for get_folder_entities tool.
type GetFolderEntitiesOutput struct {
	Entities FolderEntities `json:"entities"`
}

func registerListFolders(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListFoldersInput) (*mcp.CallToolResult, ListFoldersOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, ListFoldersOutput{}, err
		}

		folders, err := wc.Client.ListFolders(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
		if err != nil {
			return nil, ListFoldersOutput{}, err
		}

		return nil, ListFoldersOutput{Folders: folders}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_folders",
		Description: "List all folders (trigger groups) in a GTM workspace. Folders help organize tags, triggers, and variables.",
	}, handler)
}

func registerGetFolderEntities(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input GetFolderEntitiesInput) (*mcp.CallToolResult, GetFolderEntitiesOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, GetFolderEntitiesOutput{}, err
		}

		entities, err := wc.Client.GetFolderEntities(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.FolderID)
		if err != nil {
			return nil, GetFolderEntitiesOutput{}, err
		}

		return nil, GetFolderEntitiesOutput{Entities: *entities}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_folder_entities",
		Description: "Get the tags, triggers, and variables inside a specific folder.",
	}, handler)
}
