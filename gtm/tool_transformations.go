package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// -- List Transformations --

type ListTransformationsInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
}

type ListTransformationsOutput struct {
	Transformations []TransformationInfo `json:"transformations"`
}

func registerListTransformations(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input ListTransformationsInput) (*mcp.CallToolResult, ListTransformationsOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, ListTransformationsOutput{}, err
		}

		transformations, err := wc.Client.ListTransformations(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
		if err != nil {
			return nil, ListTransformationsOutput{}, err
		}

		return nil, ListTransformationsOutput{Transformations: transformations}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_transformations",
		Description: "List all transformations in a GTM workspace (server-side containers only)",
	}, handler)
}

// -- Get Transformation --

type GetTransformationInput struct {
	AccountID        string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID      string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID      string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TransformationID string `json:"transformationId" jsonschema:"description:The transformation ID to retrieve"`
}

type GetTransformationOutput struct {
	Transformation TransformationInfo `json:"transformation"`
}

func registerGetTransformation(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input GetTransformationInput) (*mcp.CallToolResult, GetTransformationOutput, error) {
		wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
		if err != nil {
			return nil, GetTransformationOutput{}, err
		}

		t, err := wc.Client.GetTransformation(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TransformationID)
		if err != nil {
			return nil, GetTransformationOutput{}, err
		}

		return nil, GetTransformationOutput{Transformation: *t}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_transformation",
		Description: "Get a specific transformation by ID (server-side containers only)",
	}, handler)
}
