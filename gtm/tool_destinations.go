package gtm

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DestinationInput struct {
	AccountID         string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID       string `json:"containerId" jsonschema:"description:The GTM container ID"`
	DestinationLinkID string `json:"destinationLinkId" jsonschema:"description:The container-specific destination link ID returned by list_destinations"`
}

type ListDestinationsOutput struct {
	Destinations []Destination `json:"destinations"`
}

type DestinationOutput struct {
	Destination Destination `json:"destination"`
}

type LinkDestinationInput struct {
	AccountID     string `json:"accountId" jsonschema:"description:The receiving GTM account ID"`
	ContainerID   string `json:"containerId" jsonschema:"description:The receiving GTM container ID"`
	DestinationID string `json:"destinationId" jsonschema:"description:The destination ID to move to this container"`
	Confirm       bool   `json:"confirm" jsonschema:"description:Must be true because linking moves the destination from its current container"`
}

func registerListDestinations(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "list_destinations", Description: "List Google tag destinations linked to a GTM container."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input EnvironmentContainerInput) (*mcp.CallToolResult, ListDestinationsOutput, error) {
			cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
			if err != nil {
				return nil, ListDestinationsOutput{}, err
			}
			destinations, err := cc.Client.ListDestinations(ctx, cc.AccountID, cc.ContainerID)
			return nil, ListDestinationsOutput{Destinations: destinations}, err
		})
}

func registerGetDestination(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "get_destination", Description: "Get a Google tag destination by its container-specific link ID."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input DestinationInput) (*mcp.CallToolResult, DestinationOutput, error) {
			if strings.TrimSpace(input.DestinationLinkID) == "" {
				return nil, DestinationOutput{}, fmt.Errorf("destination link ID is required")
			}
			cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
			if err != nil {
				return nil, DestinationOutput{}, err
			}
			destination, err := cc.Client.GetDestination(ctx, cc.AccountID, cc.ContainerID, input.DestinationLinkID)
			if err != nil {
				return nil, DestinationOutput{}, err
			}
			return nil, DestinationOutput{Destination: *destination}, nil
		})
}

func registerLinkDestination(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "link_destination", Description: "Move a Google tag destination to this container. Requires confirm: true; user permissions are not copied or changed."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input LinkDestinationInput) (*mcp.CallToolResult, DestinationOutput, error) {
			if !input.Confirm {
				return nil, DestinationOutput{}, fmt.Errorf("destination linking requires confirm: true")
			}
			if strings.TrimSpace(input.DestinationID) == "" {
				return nil, DestinationOutput{}, fmt.Errorf("destination ID is required")
			}
			cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
			if err != nil {
				return nil, DestinationOutput{}, err
			}
			destination, err := cc.Client.LinkDestination(ctx, cc.AccountID, cc.ContainerID, input.DestinationID)
			if err != nil {
				return nil, DestinationOutput{}, err
			}
			return nil, DestinationOutput{Destination: *destination}, nil
		})
}
