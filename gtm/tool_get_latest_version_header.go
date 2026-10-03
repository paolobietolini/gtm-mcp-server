package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

type LatestVersionHeaderOutput struct {
	Version VersionInfo `json:"version"`
}

func registerGetLatestVersionHeader(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_latest_version_header",
		Description: "Get the latest container version header. The latest version may differ from the published live version.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListVersionsInput) (*mcp.CallToolResult, LatestVersionHeaderOutput, error) {
		if input.AccountID == "" || input.ContainerID == "" {
			return nil, LatestVersionHeaderOutput{}, fmt.Errorf("accountId and containerId are required")
		}
		client, err := getClient(ctx)
		if err != nil {
			return nil, LatestVersionHeaderOutput{}, err
		}
		version, err := client.LatestVersionHeader(ctx, BuildContainerPath(input.AccountID, input.ContainerID))
		if err != nil {
			return nil, LatestVersionHeaderOutput{}, err
		}
		return nil, LatestVersionHeaderOutput{Version: version}, nil
	})
}

func versionHeaderInfo(v *tagmanager.ContainerVersionHeader) VersionInfo {
	return VersionInfo{
		VersionID: v.ContainerVersionId, Name: v.Name, Deleted: v.Deleted,
		NumTags: v.NumTags, NumTriggers: v.NumTriggers, NumVariables: v.NumVariables,
		NumCustomTemplates: v.NumCustomTemplates, Path: v.Path,
	}
}

func (c *Client) LatestVersionHeader(ctx context.Context, parent string) (VersionInfo, error) {
	header, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ContainerVersionHeader, error) {
		return c.Service.Accounts.Containers.VersionHeaders.Latest(parent).Context(ctx).Do()
	})
	if err != nil {
		return VersionInfo{}, mapGoogleError(err)
	}
	return versionHeaderInfo(header), nil
}

func (c *Client) ListVersionHeaders(ctx context.Context, parent string) ([]VersionInfo, error) {
	versions := make([]VersionInfo, 0)
	pageToken := ""
	for {
		response, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListContainerVersionsResponse, error) {
			return c.Service.Accounts.Containers.VersionHeaders.List(parent).PageToken(pageToken).Context(ctx).Do()
		})
		if err != nil {
			return nil, mapGoogleError(err)
		}
		for _, header := range response.ContainerVersionHeader {
			versions = append(versions, versionHeaderInfo(header))
		}
		pageToken = response.NextPageToken
		if pageToken == "" {
			return versions, nil
		}
	}
}
