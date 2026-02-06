package gtm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// ListVersionsInput is the input for list_versions tool.
type ListVersionsInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
}

// ListVersionsOutput is the output for list_versions tool.
type ListVersionsOutput struct {
	Versions []VersionInfo `json:"versions"`
}

// VersionInfo is a simplified version header response.
type VersionInfo struct {
	VersionID          string `json:"versionId"`
	Name               string `json:"name,omitempty"`
	Deleted            bool   `json:"deleted,omitempty"`
	NumTags            string `json:"numTags,omitempty"`
	NumTriggers        string `json:"numTriggers,omitempty"`
	NumVariables       string `json:"numVariables,omitempty"`
	NumCustomTemplates string `json:"numCustomTemplates,omitempty"`
	Path               string `json:"path"`
}

func registerListVersions(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListVersionsInput) (*mcp.CallToolResult, ListVersionsOutput, error) {
		// Validate required fields
		if input.AccountID == "" {
			return nil, ListVersionsOutput{}, fmt.Errorf("accountId is required")
		}
		if input.ContainerID == "" {
			return nil, ListVersionsOutput{}, fmt.Errorf("containerId is required")
		}

		client, err := getClient(ctx)
		if err != nil {
			return nil, ListVersionsOutput{}, err
		}

		parent := fmt.Sprintf("accounts/%s/containers/%s", input.AccountID, input.ContainerID)

		resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListContainerVersionsResponse, error) {
			return client.Service.Accounts.Containers.VersionHeaders.List(parent).Context(ctx).Do()
		})
		if err != nil {
			return nil, ListVersionsOutput{}, mapGoogleError(err)
		}

		versions := make([]VersionInfo, 0)
		if resp.ContainerVersionHeader != nil {
			for _, v := range resp.ContainerVersionHeader {
				versions = append(versions, VersionInfo{
					VersionID:          v.ContainerVersionId,
					Name:               v.Name,
					Deleted:            v.Deleted,
					NumTags:            v.NumTags,
					NumTriggers:        v.NumTriggers,
					NumVariables:       v.NumVariables,
					NumCustomTemplates: v.NumCustomTemplates,
					Path:               v.Path,
				})
			}
		}

		return nil, ListVersionsOutput{
			Versions: versions,
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_versions",
		Description: "List all container versions. Returns version headers with counts of tags, triggers, variables, and templates.",
	}, handler)
}
