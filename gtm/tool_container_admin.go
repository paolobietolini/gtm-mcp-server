package gtm

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ContainerAdminOutput struct {
	Success   bool      `json:"success"`
	Container Container `json:"container"`
	Message   string    `json:"message"`
}

type CombineContainersInput struct {
	AccountID         string `json:"accountId" jsonschema:"description:The GTM account ID shared by both containers"`
	ContainerID       string `json:"containerId" jsonschema:"description:The target container ID that remains after the combine"`
	SourceContainerID string `json:"sourceContainerId" jsonschema:"description:The container ID merged into the target"`
	SettingSource     string `json:"settingSource" jsonschema:"description:Which container settings to retain: current or other"`
	Confirm           bool   `json:"confirm" jsonschema:"description:Must be true to confirm the irreversible container combine"`
}

type MoveTagIDInput struct {
	AccountID    string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID  string `json:"containerId" jsonschema:"description:The container currently holding the tag ID"`
	TagID        string `json:"tagId" jsonschema:"description:The tag ID to remove from the current container"`
	TagName      string `json:"tagName" jsonschema:"description:The name for the newly created container"`
	CopySettings bool   `json:"copySettings,omitempty" jsonschema:"description:Copy tag settings to the newly created container"`
	AcceptTerms  bool   `json:"acceptTerms" jsonschema:"description:Must be true to accept the terms copied to the new container"`
	Confirm      bool   `json:"confirm" jsonschema:"description:Must be true to confirm moving the tag ID"`
}

func registerCombineContainers(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "combine_containers", Description: "Merge another container into this target container. Requires confirm: true and never changes the user-permissions feature."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input CombineContainersInput) (*mcp.CallToolResult, ContainerAdminOutput, error) {
			if !input.Confirm {
				return nil, ContainerAdminOutput{Message: "Container combine requires confirm: true"}, nil
			}
			if strings.TrimSpace(input.SourceContainerID) == "" || input.SourceContainerID == input.ContainerID {
				return nil, ContainerAdminOutput{}, fmt.Errorf("source container ID must identify a different container")
			}
			if input.SettingSource != "current" && input.SettingSource != "other" {
				return nil, ContainerAdminOutput{}, fmt.Errorf("settingSource must be current or other")
			}
			cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
			if err != nil {
				return nil, ContainerAdminOutput{}, err
			}
			container, err := cc.Client.CombineContainers(ctx, cc.AccountID, cc.ContainerID, input.SourceContainerID, input.SettingSource)
			if err != nil {
				return nil, ContainerAdminOutput{}, err
			}
			return nil, ContainerAdminOutput{Success: true, Container: *container, Message: "Containers combined successfully"}, nil
		})
}

func registerMoveTagID(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "move_tag_id", Description: "Move a tag ID out of a container into a new container. Requires confirm: true and acceptTerms: true; users are not copied."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input MoveTagIDInput) (*mcp.CallToolResult, ContainerAdminOutput, error) {
			if !input.Confirm {
				return nil, ContainerAdminOutput{Message: "Moving a tag ID requires confirm: true"}, nil
			}
			if !input.AcceptTerms {
				return nil, ContainerAdminOutput{}, fmt.Errorf("acceptTerms must be true")
			}
			if strings.TrimSpace(input.TagID) == "" || strings.TrimSpace(input.TagName) == "" {
				return nil, ContainerAdminOutput{}, fmt.Errorf("tagId and tagName are required")
			}
			cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
			if err != nil {
				return nil, ContainerAdminOutput{}, err
			}
			container, err := cc.Client.MoveTagID(ctx, cc.AccountID, cc.ContainerID, input.TagID, input.TagName, input.CopySettings)
			if err != nil {
				return nil, ContainerAdminOutput{}, err
			}
			return nil, ContainerAdminOutput{Success: true, Container: *container, Message: "Tag ID moved successfully"}, nil
		})
}
