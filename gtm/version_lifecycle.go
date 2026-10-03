package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

type VersionLifecycle struct {
	AccountID     string `json:"accountId,omitempty"`
	ContainerID   string `json:"containerId,omitempty"`
	VersionID     string `json:"containerVersionId"`
	Name          string `json:"name,omitempty"`
	Description   string `json:"description,omitempty"`
	Deleted       bool   `json:"deleted,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	Path          string `json:"path"`
	TagManagerURL string `json:"tagManagerUrl,omitempty"`
}

func (c *Client) DeleteVersion(ctx context.Context, accountID, containerID, versionID string) error {
	return mapGoogleError(c.Service.Accounts.Containers.Versions.Delete(BuildVersionPath(accountID, containerID, versionID)).Context(ctx).Do())
}

func (c *Client) UndeleteVersion(ctx context.Context, accountID, containerID, versionID string) (*VersionLifecycle, error) {
	version, err := c.Service.Accounts.Containers.Versions.Undelete(BuildVersionPath(accountID, containerID, versionID)).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toVersionLifecycle(version)
	return &result, nil
}

func (c *Client) SetLatestVersion(ctx context.Context, accountID, containerID, versionID string) (*VersionLifecycle, error) {
	version, err := c.Service.Accounts.Containers.Versions.SetLatest(BuildVersionPath(accountID, containerID, versionID)).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toVersionLifecycle(version)
	return &result, nil
}

func (c *Client) UpdateVersion(ctx context.Context, accountID, containerID, versionID string, name, description *string) (*VersionLifecycle, error) {
	path := BuildVersionPath(accountID, containerID, versionID)
	current, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ContainerVersion, error) {
		return c.Service.Accounts.Containers.Versions.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	body := &tagmanager.ContainerVersion{Name: current.Name, Description: current.Description}
	if name != nil {
		body.Name = *name
		if *name == "" {
			body.ForceSendFields = append(body.ForceSendFields, "Name")
		}
	}
	if description != nil {
		body.Description = *description
		if *description == "" {
			body.ForceSendFields = append(body.ForceSendFields, "Description")
		}
	}
	version, err := c.Service.Accounts.Containers.Versions.Update(path, body).Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toVersionLifecycle(version)
	return &result, nil
}

func BuildVersionPath(accountID, containerID, versionID string) string {
	return fmt.Sprintf("%s/versions/%s", BuildContainerPath(accountID, containerID), versionID)
}

func toVersionLifecycle(version *tagmanager.ContainerVersion) VersionLifecycle {
	return VersionLifecycle{
		AccountID: version.AccountId, ContainerID: version.ContainerId,
		VersionID: version.ContainerVersionId, Name: version.Name, Description: version.Description,
		Deleted: version.Deleted, Fingerprint: version.Fingerprint, Path: version.Path,
		TagManagerURL: version.TagManagerUrl,
	}
}
