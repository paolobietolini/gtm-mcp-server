package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Container is a simplified representation of a GTM container.
type Container struct {
	ContainerID  string   `json:"containerId"`
	Name         string   `json:"name"`
	PublicID     string   `json:"publicId"`
	UsageContext []string `json:"usageContext"`
	Path         string   `json:"path"`
}

// ListContainers returns all containers in an account.
func (c *Client) ListContainers(ctx context.Context, accountID string) ([]Container, error) {
	parent := fmt.Sprintf("accounts/%s", accountID)

	resp, err := c.Service.Accounts.Containers.List(parent).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return toContainers(resp.Container), nil
}

func toContainers(containers []*tagmanager.Container) []Container {
	result := make([]Container, 0, len(containers))
	for _, c := range containers {
		result = append(result, Container{
			ContainerID:  c.ContainerId,
			Name:         c.Name,
			PublicID:     c.PublicId,
			UsageContext: c.UsageContext,
			Path:         c.Path,
		})
	}
	return result
}
