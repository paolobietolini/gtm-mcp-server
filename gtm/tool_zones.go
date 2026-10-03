package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ZoneInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	ZoneID      string `json:"zoneId" jsonschema:"description:The GTM zone ID"`
}

type ListZonesOutput struct {
	Zones []Zone `json:"zones"`
}

type ZoneOutput struct {
	Zone Zone `json:"zone"`
}

type ZoneMutationOutput struct {
	Success bool   `json:"success"`
	Zone    Zone   `json:"zone"`
	Message string `json:"message"`
}

type CreateZoneInput struct {
	AccountID                  string                    `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID                string                    `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID                string                    `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Name                       string                    `json:"name" jsonschema:"description:Zone display name"`
	Notes                      string                    `json:"notes,omitempty" jsonschema:"description:Optional zone notes"`
	BoundaryConditionsJSON     string                    `json:"boundaryConditionsJson,omitempty" jsonschema:"description:JSON array of boundary conditions; see the trigger condition format"`
	CustomEvaluationTriggerIDs []string                  `json:"customEvaluationTriggerIds,omitempty" jsonschema:"description:Trigger IDs that cause boundary evaluation"`
	ChildContainers            []ZoneChildContainerInput `json:"childContainers,omitempty" jsonschema:"description:Containers governed as children of this zone"`
	TypeRestriction            *ZoneTypeRestrictionInput `json:"typeRestriction,omitempty" jsonschema:"description:Optional tag type restriction settings"`
}

type UpdateZoneInput struct {
	AccountID                  string                     `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID                string                     `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID                string                     `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	ZoneID                     string                     `json:"zoneId" jsonschema:"description:The GTM zone ID"`
	Name                       *string                    `json:"name,omitempty" jsonschema:"description:New name; omit to preserve"`
	Notes                      *string                    `json:"notes,omitempty" jsonschema:"description:New notes; omit to preserve or pass empty to clear"`
	BoundaryConditionsJSON     *string                    `json:"boundaryConditionsJson,omitempty" jsonschema:"description:JSON boundary conditions; omit to preserve or pass [] to clear"`
	CustomEvaluationTriggerIDs *[]string                  `json:"customEvaluationTriggerIds,omitempty" jsonschema:"description:Trigger IDs; omit to preserve or pass [] to clear"`
	ChildContainers            *[]ZoneChildContainerInput `json:"childContainers,omitempty" jsonschema:"description:Child containers; omit to preserve or pass [] to clear"`
	TypeRestriction            *ZoneTypeRestrictionInput  `json:"typeRestriction,omitempty" jsonschema:"description:New tag type restriction settings; omit to preserve"`
}

type DeleteZoneInput struct {
	ZoneInput
	Confirm bool `json:"confirm" jsonschema:"description:Must be true to confirm zone deletion"`
}

type DeleteZoneOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func registerListZones(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "list_zones", Description: "List zones in a GTM workspace."},
		func(ctx context.Context, req *mcp.CallToolRequest, input WorkspaceInput) (*mcp.CallToolResult, ListZonesOutput, error) {
			wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
			if err != nil {
				return nil, ListZonesOutput{}, err
			}
			zones, err := wc.Client.ListZones(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
			return nil, ListZonesOutput{Zones: zones}, err
		})
}

func registerGetZone(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "get_zone", Description: "Get a zone and its boundary, child containers, and type restrictions."},
		func(ctx context.Context, req *mcp.CallToolRequest, input ZoneInput) (*mcp.CallToolResult, ZoneOutput, error) {
			wc, err := resolveZone(ctx, input)
			if err != nil {
				return nil, ZoneOutput{}, err
			}
			zone, err := wc.Client.GetZone(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ZoneID)
			if err != nil {
				return nil, ZoneOutput{}, err
			}
			return nil, ZoneOutput{Zone: *zone}, nil
		})
}

func registerCreateZone(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "create_zone", Description: "Create a zone in a GTM workspace."},
		func(ctx context.Context, req *mcp.CallToolRequest, input CreateZoneInput) (*mcp.CallToolResult, ZoneMutationOutput, error) {
			wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
			if err != nil {
				return nil, ZoneMutationOutput{}, err
			}
			if strings.TrimSpace(input.Name) == "" {
				return nil, ZoneMutationOutput{}, fmt.Errorf("name is required")
			}
			conditions, err := parseZoneConditions(input.BoundaryConditionsJSON)
			if err != nil {
				return nil, ZoneMutationOutput{}, err
			}
			if err := validateZoneChildren(input.ChildContainers); err != nil {
				return nil, ZoneMutationOutput{}, err
			}
			zone, err := wc.Client.CreateZone(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, ZoneCreateConfig{
				Name: input.Name, Notes: input.Notes, BoundaryConditions: conditions,
				CustomEvaluationTriggerIDs: input.CustomEvaluationTriggerIDs,
				ChildContainers:            input.ChildContainers, TypeRestriction: input.TypeRestriction,
			})
			if err != nil {
				return nil, ZoneMutationOutput{}, err
			}
			return nil, ZoneMutationOutput{Success: true, Zone: *zone, Message: "Zone created successfully"}, nil
		})
}

func registerUpdateZone(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "update_zone", Description: "Update selected zone fields while preserving omitted values and checking its fingerprint."},
		func(ctx context.Context, req *mcp.CallToolRequest, input UpdateZoneInput) (*mcp.CallToolResult, ZoneMutationOutput, error) {
			wc, err := resolveZone(ctx, ZoneInput{input.AccountID, input.ContainerID, input.WorkspaceID, input.ZoneID})
			if err != nil {
				return nil, ZoneMutationOutput{}, err
			}
			if input.Name == nil && input.Notes == nil && input.BoundaryConditionsJSON == nil &&
				input.CustomEvaluationTriggerIDs == nil && input.ChildContainers == nil && input.TypeRestriction == nil {
				return nil, ZoneMutationOutput{}, fmt.Errorf("provide at least one field to update")
			}
			if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
				return nil, ZoneMutationOutput{}, fmt.Errorf("name cannot be empty")
			}
			var conditions *[]Condition
			if input.BoundaryConditionsJSON != nil {
				parsed, err := parseZoneConditions(*input.BoundaryConditionsJSON)
				if err != nil {
					return nil, ZoneMutationOutput{}, err
				}
				conditions = &parsed
			}
			if input.ChildContainers != nil {
				if err := validateZoneChildren(*input.ChildContainers); err != nil {
					return nil, ZoneMutationOutput{}, err
				}
			}
			zone, err := wc.Client.UpdateZone(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ZoneID, ZoneUpdateConfig{
				Name: input.Name, Notes: input.Notes, BoundaryConditions: conditions,
				CustomEvaluationTriggerIDs: input.CustomEvaluationTriggerIDs,
				ChildContainers:            input.ChildContainers, TypeRestriction: input.TypeRestriction,
			})
			if err != nil {
				return nil, ZoneMutationOutput{}, err
			}
			return nil, ZoneMutationOutput{Success: true, Zone: *zone, Message: "Zone updated successfully"}, nil
		})
}

func registerDeleteZone(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "delete_zone", Description: "Delete a zone and require confirm: true."},
		func(ctx context.Context, req *mcp.CallToolRequest, input DeleteZoneInput) (*mcp.CallToolResult, DeleteZoneOutput, error) {
			if !input.Confirm {
				return nil, DeleteZoneOutput{Message: "Zone deletion requires confirm: true"}, nil
			}
			wc, err := resolveZone(ctx, input.ZoneInput)
			if err != nil {
				return nil, DeleteZoneOutput{}, err
			}
			if err := wc.Client.DeleteZone(ctx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ZoneID); err != nil {
				return nil, DeleteZoneOutput{}, err
			}
			return nil, DeleteZoneOutput{Success: true, Message: "Zone deleted successfully"}, nil
		})
}

func resolveZone(ctx context.Context, input ZoneInput) (*WorkspaceContext, error) {
	if strings.TrimSpace(input.ZoneID) == "" {
		return nil, fmt.Errorf("zone ID is required")
	}
	return resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
}

func parseZoneConditions(raw string) ([]Condition, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var conditions []Condition
	if err := json.Unmarshal([]byte(raw), &conditions); err != nil {
		return nil, fmt.Errorf("invalid boundaryConditionsJson: %w", err)
	}
	return conditions, nil
}

func validateZoneChildren(children []ZoneChildContainerInput) error {
	for _, child := range children {
		if strings.TrimSpace(child.PublicID) == "" {
			return fmt.Errorf("child container publicId cannot be empty")
		}
	}
	return nil
}
