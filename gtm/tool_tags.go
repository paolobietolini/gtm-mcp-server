package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListTagsInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
}
type ListTagsOutput struct {
	Tags []Tag `json:"tags"`
}

type GetTagInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TagID       string `json:"tagId" jsonschema:"description:The tag ID to retrieve"`
}
type GetTagOutput struct {
	Tag Tag `json:"tag"`
}

func registerListTags(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListTagsInput) (*mcp.CallToolResult, ListTagsOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, ListTagsOutput{}, err
		}

		tags, err := wc.Client.ListTags(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
		if err != nil {
			return nil, ListTagsOutput{}, err
		}

		return nil, ListTagsOutput{Tags: tags}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tags",
		Description: "List all tags in a GTM workspace",
	}, handler)
}

func registerGetTag(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input GetTagInput) (*mcp.CallToolResult, GetTagOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, GetTagOutput{}, err
		}

		tag, err := wc.Client.GetTag(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)
		if err != nil {
			return nil, GetTagOutput{}, err
		}

		return nil, GetTagOutput{Tag: *tag}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_tag",
		Description: "Get a specific tag by ID",
	}, handler)
}
