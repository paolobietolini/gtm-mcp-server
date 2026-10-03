package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

type Environment struct {
	AccountID              string `json:"accountId"`
	ContainerID            string `json:"containerId"`
	EnvironmentID          string `json:"environmentId"`
	Name                   string `json:"name"`
	Description            string `json:"description,omitempty"`
	Type                   string `json:"type"`
	URL                    string `json:"url,omitempty"`
	EnableDebug            bool   `json:"enableDebug"`
	AuthorizationCode      string `json:"authorizationCode,omitempty"`
	AuthorizationTimestamp string `json:"authorizationTimestamp,omitempty"`
	ContainerVersionID     string `json:"containerVersionId,omitempty"`
	WorkspaceID            string `json:"workspaceId,omitempty"`
	Fingerprint            string `json:"fingerprint"`
	Path                   string `json:"path"`
	TagManagerURL          string `json:"tagManagerUrl,omitempty"`
}

type EnvironmentCreateConfig struct {
	Name, Description, URL string
	EnableDebug            bool
}

type EnvironmentUpdateConfig struct {
	Name, Description, URL          *string
	EnableDebug                     *bool
	ContainerVersionID, WorkspaceID *string
}

func (c *Client) ListEnvironments(ctx context.Context, accountID, containerID string) ([]Environment, error) {
	parent := BuildContainerPath(accountID, containerID)
	result := make([]Environment, 0)
	pageToken := ""
	for {
		response, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListEnvironmentsResponse, error) {
			return c.Service.Accounts.Containers.Environments.List(parent).PageToken(pageToken).Context(ctx).Do()
		})
		if err != nil {
			return nil, mapGoogleError(err)
		}
		for _, environment := range response.Environment {
			result = append(result, toEnvironment(environment))
		}
		if response.NextPageToken == "" {
			return result, nil
		}
		pageToken = response.NextPageToken
	}
}

func (c *Client) GetEnvironment(ctx context.Context, accountID, containerID, environmentID string) (*Environment, error) {
	path := BuildEnvironmentPath(accountID, containerID, environmentID)
	environment, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Environment, error) {
		return c.Service.Accounts.Containers.Environments.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toEnvironment(environment)
	return &result, nil
}

func (c *Client) CreateEnvironment(ctx context.Context, accountID, containerID string, config EnvironmentCreateConfig) (*Environment, error) {
	body := &tagmanager.Environment{
		Name: config.Name, Description: config.Description, Url: config.URL,
		EnableDebug: config.EnableDebug, Type: "user",
	}
	if !config.EnableDebug {
		body.ForceSendFields = append(body.ForceSendFields, "EnableDebug")
	}
	environment, err := c.Service.Accounts.Containers.Environments.Create(BuildContainerPath(accountID, containerID), body).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toEnvironment(environment)
	return &result, nil
}

func (c *Client) UpdateEnvironment(ctx context.Context, accountID, containerID, environmentID string, config EnvironmentUpdateConfig) (*Environment, error) {
	path := BuildEnvironmentPath(accountID, containerID, environmentID)
	current, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Environment, error) {
		return c.Service.Accounts.Containers.Environments.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	body := &tagmanager.Environment{
		Name: current.Name, Description: current.Description, Url: current.Url,
		EnableDebug: current.EnableDebug, ContainerVersionId: current.ContainerVersionId,
		WorkspaceId: current.WorkspaceId, Type: current.Type,
	}
	applyEnvironmentUpdate(body, config)
	environment, err := c.Service.Accounts.Containers.Environments.Update(path, body).Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toEnvironment(environment)
	return &result, nil
}

func (c *Client) ReauthorizeEnvironment(ctx context.Context, accountID, containerID, environmentID string) (*Environment, error) {
	path := BuildEnvironmentPath(accountID, containerID, environmentID)
	environment, err := c.Service.Accounts.Containers.Environments.Reauthorize(path, &tagmanager.Environment{}).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toEnvironment(environment)
	return &result, nil
}

func (c *Client) DeleteEnvironment(ctx context.Context, accountID, containerID, environmentID string) error {
	return mapGoogleError(c.Service.Accounts.Containers.Environments.Delete(BuildEnvironmentPath(accountID, containerID, environmentID)).Context(ctx).Do())
}

func BuildEnvironmentPath(accountID, containerID, environmentID string) string {
	return fmt.Sprintf("%s/environments/%s", BuildContainerPath(accountID, containerID), environmentID)
}

func applyEnvironmentUpdate(environment *tagmanager.Environment, config EnvironmentUpdateConfig) {
	if config.Name != nil {
		environment.Name = *config.Name
	}
	if config.Description != nil {
		environment.Description = *config.Description
		if *config.Description == "" {
			environment.ForceSendFields = append(environment.ForceSendFields, "Description")
		}
	}
	if config.URL != nil {
		environment.Url = *config.URL
		if *config.URL == "" {
			environment.ForceSendFields = append(environment.ForceSendFields, "Url")
		}
	}
	if config.EnableDebug != nil {
		environment.EnableDebug = *config.EnableDebug
		if !*config.EnableDebug {
			environment.ForceSendFields = append(environment.ForceSendFields, "EnableDebug")
		}
	}
	if config.ContainerVersionID != nil {
		environment.ContainerVersionId = *config.ContainerVersionID
		if *config.ContainerVersionID == "" {
			environment.ForceSendFields = append(environment.ForceSendFields, "ContainerVersionId")
		}
	}
	if config.WorkspaceID != nil {
		environment.WorkspaceId = *config.WorkspaceID
		if *config.WorkspaceID == "" {
			environment.ForceSendFields = append(environment.ForceSendFields, "WorkspaceId")
		}
	}
}

func toEnvironment(environment *tagmanager.Environment) Environment {
	return Environment{
		AccountID: environment.AccountId, ContainerID: environment.ContainerId,
		EnvironmentID: environment.EnvironmentId, Name: environment.Name,
		Description: environment.Description, Type: environment.Type, URL: environment.Url,
		EnableDebug: environment.EnableDebug, AuthorizationCode: environment.AuthorizationCode,
		AuthorizationTimestamp: environment.AuthorizationTimestamp,
		ContainerVersionID:     environment.ContainerVersionId, WorkspaceID: environment.WorkspaceId,
		Fingerprint: environment.Fingerprint, Path: environment.Path, TagManagerURL: environment.TagManagerUrl,
	}
}
