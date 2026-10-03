package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

type Destination struct {
	AccountID         string `json:"accountId"`
	ContainerID       string `json:"containerId"`
	DestinationID     string `json:"destinationId"`
	DestinationLinkID string `json:"destinationLinkId,omitempty"`
	Name              string `json:"name"`
	Fingerprint       string `json:"fingerprint,omitempty"`
	Path              string `json:"path"`
	TagManagerURL     string `json:"tagManagerUrl,omitempty"`
}

func (c *Client) ListDestinations(ctx context.Context, accountID, containerID string) ([]Destination, error) {
	response, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListDestinationsResponse, error) {
		return c.Service.Accounts.Containers.Destinations.List(BuildContainerPath(accountID, containerID)).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := make([]Destination, 0, len(response.Destination))
	for _, destination := range response.Destination {
		result = append(result, toDestination(destination))
	}
	return result, nil
}

func (c *Client) GetDestination(ctx context.Context, accountID, containerID, destinationLinkID string) (*Destination, error) {
	destination, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Destination, error) {
		return c.Service.Accounts.Containers.Destinations.Get(BuildDestinationPath(accountID, containerID, destinationLinkID)).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toDestination(destination)
	return &result, nil
}

func (c *Client) LinkDestination(ctx context.Context, accountID, containerID, destinationID string) (*Destination, error) {
	destination, err := c.Service.Accounts.Containers.Destinations.Link(BuildContainerPath(accountID, containerID)).DestinationId(destinationID).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toDestination(destination)
	return &result, nil
}

func BuildDestinationPath(accountID, containerID, destinationLinkID string) string {
	return fmt.Sprintf("%s/destinations/%s", BuildContainerPath(accountID, containerID), destinationLinkID)
}

func toDestination(destination *tagmanager.Destination) Destination {
	return Destination{
		AccountID: destination.AccountId, ContainerID: destination.ContainerId,
		DestinationID: destination.DestinationId, DestinationLinkID: destination.DestinationLinkId,
		Name: destination.Name, Fingerprint: destination.Fingerprint, Path: destination.Path,
		TagManagerURL: destination.TagManagerUrl,
	}
}
