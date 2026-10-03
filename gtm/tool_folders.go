package gtm

import (
	"context"
	"fmt"
	"strings"

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

type FolderInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	FolderID    string `json:"folderId" jsonschema:"description:The folder ID"`
}

type FolderOutput struct {
	Success bool    `json:"success,omitempty"`
	Folder  *Folder `json:"folder,omitempty"`
	Message string  `json:"message,omitempty"`
}

type CreateFolderInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Name        string `json:"name" jsonschema:"description:Folder display name"`
	Notes       string `json:"notes,omitempty" jsonschema:"description:Optional folder notes"`
}

type UpdateFolderInput struct {
	FolderInput
	Name  *string `json:"name,omitempty" jsonschema:"description:New name; omit to preserve"`
	Notes *string `json:"notes,omitempty" jsonschema:"description:New notes; omit to preserve or pass empty to clear"`
}

type ConfirmFolderInput struct {
	FolderInput
	Confirm bool `json:"confirm" jsonschema:"description:Must be true to confirm the operation"`
}

type MoveEntitiesToFolderInput struct {
	FolderInput
	TagIDs      []string `json:"tagIds,omitempty" jsonschema:"description:Tag IDs to move"`
	TriggerIDs  []string `json:"triggerIds,omitempty" jsonschema:"description:Trigger IDs to move"`
	VariableIDs []string `json:"variableIds,omitempty" jsonschema:"description:Variable IDs to move"`
	Confirm     bool     `json:"confirm" jsonschema:"description:Must be true to confirm moving the entities"`
}

type FolderActionOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
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

func registerGetFolder(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "get_folder", Description: "Get a GTM folder by ID."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input FolderInput) (*mcp.CallToolResult, FolderOutput, error) {
			wc, err := resolveFolder(ctx, input)
			if err != nil {
				return nil, FolderOutput{}, err
			}
			folder, err := wc.Client.GetFolder(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.FolderID)
			return nil, FolderOutput{Folder: folder}, err
		})
}

func registerCreateFolder(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "create_folder", Description: "Create a folder in a GTM workspace."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input CreateFolderInput) (*mcp.CallToolResult, FolderOutput, error) {
			if strings.TrimSpace(input.Name) == "" {
				return nil, FolderOutput{}, fmt.Errorf("name is required")
			}
			wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
			if err != nil {
				return nil, FolderOutput{}, err
			}
			folder, err := wc.Client.CreateFolder(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.Name, input.Notes)
			return nil, FolderOutput{Success: err == nil, Folder: folder, Message: "Folder created successfully"}, err
		})
}

func registerUpdateFolder(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "update_folder", Description: "Update selected folder fields while preserving omitted values and checking its fingerprint."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input UpdateFolderInput) (*mcp.CallToolResult, FolderOutput, error) {
			if input.Name == nil && input.Notes == nil {
				return nil, FolderOutput{}, fmt.Errorf("provide name or notes to update")
			}
			if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
				return nil, FolderOutput{}, fmt.Errorf("name cannot be empty")
			}
			wc, err := resolveFolder(ctx, input.FolderInput)
			if err != nil {
				return nil, FolderOutput{}, err
			}
			folder, err := wc.Client.UpdateFolder(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.FolderID, input.Name, input.Notes)
			return nil, FolderOutput{Success: err == nil, Folder: folder, Message: "Folder updated successfully"}, err
		})
}

func registerDeleteFolder(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "delete_folder", Description: "Delete a GTM folder. Requires confirm: true."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input ConfirmFolderInput) (*mcp.CallToolResult, FolderActionOutput, error) {
			if !input.Confirm {
				return nil, FolderActionOutput{Message: "Folder deletion requires confirm: true"}, nil
			}
			wc, err := resolveFolder(ctx, input.FolderInput)
			if err != nil {
				return nil, FolderActionOutput{}, err
			}
			err = wc.Client.DeleteFolder(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.FolderID)
			return nil, FolderActionOutput{Success: err == nil, Message: "Folder deleted successfully"}, err
		})
}

func registerMoveEntitiesToFolder(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "move_entities_to_folder", Description: "Move tags, triggers, and variables into a folder. Requires confirm: true."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input MoveEntitiesToFolderInput) (*mcp.CallToolResult, FolderActionOutput, error) {
			if !input.Confirm {
				return nil, FolderActionOutput{Message: "Moving entities requires confirm: true"}, nil
			}
			if len(input.TagIDs)+len(input.TriggerIDs)+len(input.VariableIDs) == 0 {
				return nil, FolderActionOutput{}, fmt.Errorf("provide at least one tag, trigger, or variable ID")
			}
			wc, err := resolveFolder(ctx, input.FolderInput)
			if err != nil {
				return nil, FolderActionOutput{}, err
			}
			err = wc.Client.MoveEntitiesToFolder(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.FolderID, input.TagIDs, input.TriggerIDs, input.VariableIDs)
			return nil, FolderActionOutput{Success: err == nil, Message: "Entities moved successfully"}, err
		})
}

func registerRevertFolder(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "revert_folder", Description: "Discard workspace changes to a folder and restore its latest-version state. Requires confirm: true."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input ConfirmFolderInput) (*mcp.CallToolResult, FolderOutput, error) {
			if !input.Confirm {
				return nil, FolderOutput{Message: "Folder revert requires confirm: true"}, nil
			}
			wc, err := resolveFolder(ctx, input.FolderInput)
			if err != nil {
				return nil, FolderOutput{}, err
			}
			folder, err := wc.Client.RevertFolder(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.FolderID)
			return nil, FolderOutput{Success: err == nil, Folder: folder, Message: "Folder reverted successfully"}, err
		})
}

func resolveFolder(ctx context.Context, input FolderInput) (*WorkspaceContext, error) {
	if strings.TrimSpace(input.FolderID) == "" {
		return nil, fmt.Errorf("folder ID is required")
	}
	return resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
}
