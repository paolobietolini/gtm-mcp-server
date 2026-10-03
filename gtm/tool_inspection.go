package gtm

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetVersionInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	VersionID   string `json:"versionId" jsonschema:"description:The saved container version ID"`
}

type GetVersionOutput struct {
	Version ContainerVersionDetails `json:"version"`
}

type LookupContainerInput struct {
	DestinationID string `json:"destinationId,omitempty" jsonschema:"description:A destination ID such as AW-123456; provide exactly one lookup field"`
	TagID         string `json:"tagId,omitempty" jsonschema:"description:A GTM public ID such as GTM-ABC123; provide exactly one lookup field"`
}

type LookupContainerOutput struct {
	Container Container `json:"container"`
}

type GetContainerSnippetOutput struct {
	Snippet ContainerSnippet `json:"snippet"`
}

func registerGetVersion(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_version",
		Description: "Get a saved container version with its tags, triggers, variables, templates, and other entities.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetVersionInput) (*mcp.CallToolResult, GetVersionOutput, error) {
		cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
		if err != nil {
			return nil, GetVersionOutput{}, err
		}
		versionID := strings.TrimSpace(input.VersionID)
		if versionID == "" {
			return nil, GetVersionOutput{}, fmt.Errorf("version ID is required")
		}
		version, err := cc.Client.GetContainerVersion(ctx, cc.AccountID, cc.ContainerID, versionID)
		if err != nil {
			return nil, GetVersionOutput{}, err
		}
		return nil, GetVersionOutput{Version: *version}, nil
	})
}

func registerGetLiveVersion(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_live_version",
		Description: "Get the published live container version with its complete entity collections.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListVersionsInput) (*mcp.CallToolResult, GetVersionOutput, error) {
		cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
		if err != nil {
			return nil, GetVersionOutput{}, err
		}
		version, err := cc.Client.GetLiveContainerVersion(ctx, cc.AccountID, cc.ContainerID)
		if err != nil {
			return nil, GetVersionOutput{}, err
		}
		return nil, GetVersionOutput{Version: *version}, nil
	})
}

func registerLookupContainer(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "lookup_container",
		Description: "Find a container by exactly one destination ID or GTM public tag ID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input LookupContainerInput) (*mcp.CallToolResult, LookupContainerOutput, error) {
		destinationID := strings.TrimSpace(input.DestinationID)
		tagID := strings.TrimSpace(input.TagID)
		if (destinationID == "") == (tagID == "") {
			return nil, LookupContainerOutput{}, fmt.Errorf("provide exactly one of destinationId or tagId")
		}
		client, err := getClient(ctx)
		if err != nil {
			return nil, LookupContainerOutput{}, err
		}
		container, err := client.LookupContainer(ctx, destinationID, tagID)
		if err != nil {
			return nil, LookupContainerOutput{}, err
		}
		return nil, LookupContainerOutput{Container: *container}, nil
	})
}

func registerGetContainerSnippet(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_container_snippet",
		Description: "Get the install snippet for a web container or provisioning configuration for a server container.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListVersionsInput) (*mcp.CallToolResult, GetContainerSnippetOutput, error) {
		cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
		if err != nil {
			return nil, GetContainerSnippetOutput{}, err
		}
		snippet, err := cc.Client.GetContainerSnippet(ctx, cc.AccountID, cc.ContainerID)
		if err != nil {
			return nil, GetContainerSnippetOutput{}, err
		}
		return nil, GetContainerSnippetOutput{Snippet: *snippet}, nil
	})
}
