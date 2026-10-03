package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Zone is a compact representation of a GTM zone. The three configuration
// objects retain Google's JSON shape without expanding the MCP output schema.
type Zone struct {
	AccountID       string `json:"accountId"`
	ContainerID     string `json:"containerId"`
	WorkspaceID     string `json:"workspaceId"`
	ZoneID          string `json:"zoneId"`
	Name            string `json:"name"`
	Notes           string `json:"notes,omitempty"`
	Fingerprint     string `json:"fingerprint"`
	Path            string `json:"path"`
	TagManagerURL   string `json:"tagManagerUrl,omitempty"`
	Boundary        any    `json:"boundary,omitempty"`
	ChildContainers any    `json:"childContainer,omitempty"`
	TypeRestriction any    `json:"typeRestriction,omitempty"`
}

type ZoneChildContainerInput struct {
	PublicID string `json:"publicId" jsonschema:"description:Child container public ID such as GTM-ABC123"`
	Nickname string `json:"nickname,omitempty" jsonschema:"description:Optional nickname inside the zone"`
}

type ZoneTypeRestrictionInput struct {
	Enabled            bool     `json:"enabled" jsonschema:"description:Whether tag type restrictions are enabled"`
	WhitelistedTypeIDs []string `json:"whitelistedTypeIds,omitempty" jsonschema:"description:Allowed tag template public IDs"`
}

type ZoneCreateConfig struct {
	Name                       string                    `json:"name"`
	Notes                      string                    `json:"notes,omitempty"`
	BoundaryConditions         []Condition               `json:"-"`
	CustomEvaluationTriggerIDs []string                  `json:"-"`
	ChildContainers            []ZoneChildContainerInput `json:"-"`
	TypeRestriction            *ZoneTypeRestrictionInput `json:"-"`
}

type ZoneUpdateConfig struct {
	Name                       *string
	Notes                      *string
	BoundaryConditions         *[]Condition
	CustomEvaluationTriggerIDs *[]string
	ChildContainers            *[]ZoneChildContainerInput
	TypeRestriction            *ZoneTypeRestrictionInput
}

func (c *Client) ListZones(ctx context.Context, accountID, containerID, workspaceID string) ([]Zone, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)
	zones := make([]Zone, 0)
	pageToken := ""
	for {
		response, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListZonesResponse, error) {
			return c.Service.Accounts.Containers.Workspaces.Zones.List(parent).PageToken(pageToken).Context(ctx).Do()
		})
		if err != nil {
			return nil, mapGoogleError(err)
		}
		for _, zone := range response.Zone {
			zones = append(zones, toZone(zone))
		}
		if response.NextPageToken == "" {
			return zones, nil
		}
		pageToken = response.NextPageToken
	}
}

func (c *Client) GetZone(ctx context.Context, accountID, containerID, workspaceID, zoneID string) (*Zone, error) {
	path := BuildZonePath(accountID, containerID, workspaceID, zoneID)
	zone, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Zone, error) {
		return c.Service.Accounts.Containers.Workspaces.Zones.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toZone(zone)
	return &result, nil
}

func (c *Client) CreateZone(ctx context.Context, accountID, containerID, workspaceID string, config ZoneCreateConfig) (*Zone, error) {
	created, err := c.Service.Accounts.Containers.Workspaces.Zones.Create(
		BuildWorkspacePath(accountID, containerID, workspaceID), zoneFromCreateConfig(config),
	).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toZone(created)
	return &result, nil
}

func (c *Client) UpdateZone(ctx context.Context, accountID, containerID, workspaceID, zoneID string, config ZoneUpdateConfig) (*Zone, error) {
	path := BuildZonePath(accountID, containerID, workspaceID, zoneID)
	current, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Zone, error) {
		return c.Service.Accounts.Containers.Workspaces.Zones.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	update := &tagmanager.Zone{
		Name:            current.Name,
		Notes:           current.Notes,
		Boundary:        current.Boundary,
		ChildContainer:  current.ChildContainer,
		TypeRestriction: current.TypeRestriction,
	}
	applyZoneUpdate(update, config)
	updated, err := c.Service.Accounts.Containers.Workspaces.Zones.Update(path, update).
		Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toZone(updated)
	return &result, nil
}

func (c *Client) DeleteZone(ctx context.Context, accountID, containerID, workspaceID, zoneID string) error {
	err := c.Service.Accounts.Containers.Workspaces.Zones.Delete(
		BuildZonePath(accountID, containerID, workspaceID, zoneID),
	).Context(ctx).Do()
	return mapGoogleError(err)
}

func BuildZonePath(accountID, containerID, workspaceID, zoneID string) string {
	return fmt.Sprintf("%s/zones/%s", BuildWorkspacePath(accountID, containerID, workspaceID), zoneID)
}

func zoneFromCreateConfig(config ZoneCreateConfig) *tagmanager.Zone {
	zone := &tagmanager.Zone{Name: config.Name, Notes: config.Notes}
	if config.BoundaryConditions != nil || config.CustomEvaluationTriggerIDs != nil {
		zone.Boundary = &tagmanager.ZoneBoundary{
			Condition:                 toAPIConditions(config.BoundaryConditions),
			CustomEvaluationTriggerId: config.CustomEvaluationTriggerIDs,
		}
	}
	if config.ChildContainers != nil {
		zone.ChildContainer = toAPIChildContainers(config.ChildContainers)
	}
	if config.TypeRestriction != nil {
		zone.TypeRestriction = toAPITypeRestriction(config.TypeRestriction)
	}
	return zone
}

func applyZoneUpdate(zone *tagmanager.Zone, config ZoneUpdateConfig) {
	if config.Name != nil {
		zone.Name = *config.Name
	}
	if config.Notes != nil {
		zone.Notes = *config.Notes
		if *config.Notes == "" {
			zone.ForceSendFields = append(zone.ForceSendFields, "Notes")
		}
	}
	if config.BoundaryConditions != nil || config.CustomEvaluationTriggerIDs != nil {
		if zone.Boundary == nil {
			zone.Boundary = &tagmanager.ZoneBoundary{}
		}
		if config.BoundaryConditions != nil {
			zone.Boundary.Condition = toAPIConditions(*config.BoundaryConditions)
			if len(*config.BoundaryConditions) == 0 {
				zone.Boundary.ForceSendFields = append(zone.Boundary.ForceSendFields, "Condition")
			}
		}
		if config.CustomEvaluationTriggerIDs != nil {
			zone.Boundary.CustomEvaluationTriggerId = *config.CustomEvaluationTriggerIDs
			if len(*config.CustomEvaluationTriggerIDs) == 0 {
				zone.Boundary.ForceSendFields = append(zone.Boundary.ForceSendFields, "CustomEvaluationTriggerId")
			}
		}
	}
	if config.ChildContainers != nil {
		zone.ChildContainer = toAPIChildContainers(*config.ChildContainers)
		if len(*config.ChildContainers) == 0 {
			zone.ForceSendFields = append(zone.ForceSendFields, "ChildContainer")
		}
	}
	if config.TypeRestriction != nil {
		zone.TypeRestriction = toAPITypeRestriction(config.TypeRestriction)
	}
}

func toAPIChildContainers(children []ZoneChildContainerInput) []*tagmanager.ZoneChildContainer {
	result := make([]*tagmanager.ZoneChildContainer, len(children))
	for i, child := range children {
		result[i] = &tagmanager.ZoneChildContainer{PublicId: child.PublicID, Nickname: child.Nickname}
	}
	return result
}

func toAPITypeRestriction(restriction *ZoneTypeRestrictionInput) *tagmanager.ZoneTypeRestriction {
	result := &tagmanager.ZoneTypeRestriction{
		Enable:            restriction.Enabled,
		WhitelistedTypeId: restriction.WhitelistedTypeIDs,
		ForceSendFields:   []string{"Enable"},
	}
	if len(restriction.WhitelistedTypeIDs) == 0 {
		result.ForceSendFields = append(result.ForceSendFields, "WhitelistedTypeId")
	}
	return result
}

func toZone(zone *tagmanager.Zone) Zone {
	return Zone{
		AccountID:       zone.AccountId,
		ContainerID:     zone.ContainerId,
		WorkspaceID:     zone.WorkspaceId,
		ZoneID:          zone.ZoneId,
		Name:            zone.Name,
		Notes:           zone.Notes,
		Fingerprint:     zone.Fingerprint,
		Path:            zone.Path,
		TagManagerURL:   zone.TagManagerUrl,
		Boundary:        zone.Boundary,
		ChildContainers: zone.ChildContainer,
		TypeRestriction: zone.TypeRestriction,
	}
}
