package gtm

import (
	"context"
	"fmt"
	"strings"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// ContainerVersionDetails contains version metadata and the complete entity
// collections returned by Google. Entity collections use their native JSON
// representation without expanding those large recursive schemas in tools/list.
type ContainerVersionDetails struct {
	AccountID        string     `json:"accountId"`
	ContainerID      string     `json:"containerId"`
	VersionID        string     `json:"containerVersionId"`
	Name             string     `json:"name,omitempty"`
	Description      string     `json:"description,omitempty"`
	Deleted          bool       `json:"deleted,omitempty"`
	Fingerprint      string     `json:"fingerprint,omitempty"`
	Path             string     `json:"path"`
	TagManagerURL    string     `json:"tagManagerUrl,omitempty"`
	Container        *Container `json:"container,omitempty"`
	BuiltInVariables any        `json:"builtInVariable,omitempty"`
	Clients          any        `json:"client,omitempty"`
	CustomTemplates  any        `json:"customTemplate,omitempty"`
	Folders          any        `json:"folder,omitempty"`
	GoogleTagConfigs any        `json:"gtagConfig,omitempty"`
	Tags             any        `json:"tag,omitempty"`
	Transformations  any        `json:"transformation,omitempty"`
	Triggers         any        `json:"trigger,omitempty"`
	Variables        any        `json:"variable,omitempty"`
	Zones            any        `json:"zone,omitempty"`
}

// ContainerSnippet contains the install snippet for web containers or the
// provisioning configuration for server containers.
type ContainerSnippet struct {
	Snippet         string `json:"snippet,omitempty"`
	ContainerConfig string `json:"containerConfig,omitempty"`
}

func (c *Client) GetContainerVersion(ctx context.Context, accountID, containerID, versionID string) (*ContainerVersionDetails, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/versions/%s", accountID, containerID, versionID)
	version, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ContainerVersion, error) {
		return c.Service.Accounts.Containers.Versions.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toContainerVersionDetails(version)
	return &result, nil
}

func (c *Client) GetLiveContainerVersion(ctx context.Context, accountID, containerID string) (*ContainerVersionDetails, error) {
	parent := BuildContainerPath(accountID, containerID)
	version, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ContainerVersion, error) {
		return c.Service.Accounts.Containers.Versions.Live(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toContainerVersionDetails(version)
	return &result, nil
}

func (c *Client) LookupContainer(ctx context.Context, destinationID, tagID string) (*Container, error) {
	destinationID = strings.TrimSpace(destinationID)
	tagID = strings.TrimSpace(tagID)
	if (destinationID == "") == (tagID == "") {
		return nil, fmt.Errorf("provide exactly one of destination ID or tag ID")
	}

	call := c.Service.Accounts.Containers.Lookup()
	if destinationID != "" {
		call = call.DestinationId(destinationID)
	} else {
		call = call.TagId(tagID)
	}
	container, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Container, error) {
		return call.Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toContainer(container)
	return &result, nil
}

func (c *Client) GetContainerSnippet(ctx context.Context, accountID, containerID string) (*ContainerSnippet, error) {
	path := BuildContainerPath(accountID, containerID)
	response, err := retryWithBackoff(ctx, 3, func() (*tagmanager.GetContainerSnippetResponse, error) {
		return c.Service.Accounts.Containers.Snippet(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	return &ContainerSnippet{
		Snippet:         response.Snippet,
		ContainerConfig: response.ContainerConfig,
	}, nil
}

func toContainerVersionDetails(version *tagmanager.ContainerVersion) ContainerVersionDetails {
	result := ContainerVersionDetails{
		AccountID:     version.AccountId,
		ContainerID:   version.ContainerId,
		VersionID:     version.ContainerVersionId,
		Name:          version.Name,
		Description:   version.Description,
		Deleted:       version.Deleted,
		Fingerprint:   version.Fingerprint,
		Path:          version.Path,
		TagManagerURL: version.TagManagerUrl,
	}
	if version.Container != nil {
		container := toContainer(version.Container)
		result.Container = &container
	}
	if len(version.BuiltInVariable) > 0 {
		result.BuiltInVariables = version.BuiltInVariable
	}
	if len(version.Client) > 0 {
		result.Clients = version.Client
	}
	if len(version.CustomTemplate) > 0 {
		result.CustomTemplates = version.CustomTemplate
	}
	if len(version.Folder) > 0 {
		result.Folders = version.Folder
	}
	if len(version.GtagConfig) > 0 {
		result.GoogleTagConfigs = version.GtagConfig
	}
	if len(version.Tag) > 0 {
		result.Tags = version.Tag
	}
	if len(version.Transformation) > 0 {
		result.Transformations = version.Transformation
	}
	if len(version.Trigger) > 0 {
		result.Triggers = version.Trigger
	}
	if len(version.Variable) > 0 {
		result.Variables = version.Variable
	}
	if len(version.Zone) > 0 {
		result.Zones = version.Zone
	}
	return result
}
