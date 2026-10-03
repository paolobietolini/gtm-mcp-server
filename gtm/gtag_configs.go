package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

type GoogleTagConfig struct {
	AccountID   string `json:"accountId"`
	ContainerID string `json:"containerId"`
	WorkspaceID string `json:"workspaceId"`
	ConfigID    string `json:"gtagConfigId"`
	Type        string `json:"type"`
	// Use any because GTM parameters recursively contain list and map parameters.
	Parameter     any    `json:"parameter,omitempty"`
	Fingerprint   string `json:"fingerprint"`
	Path          string `json:"path"`
	TagManagerURL string `json:"tagManagerUrl,omitempty"`
}

type GoogleTagConfigCreateConfig struct {
	Type      string
	Parameter []Parameter
}

type GoogleTagConfigUpdateConfig struct {
	Type      *string
	Parameter *[]Parameter
}

func (c *Client) ListGoogleTagConfigs(ctx context.Context, accountID, containerID, workspaceID string) ([]GoogleTagConfig, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)
	result := make([]GoogleTagConfig, 0)
	pageToken := ""
	for {
		response, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListGtagConfigResponse, error) {
			return c.Service.Accounts.Containers.Workspaces.GtagConfig.List(parent).PageToken(pageToken).Context(ctx).Do()
		})
		if err != nil {
			return nil, mapGoogleError(err)
		}
		for _, config := range response.GtagConfig {
			result = append(result, toGoogleTagConfig(config))
		}
		if response.NextPageToken == "" {
			return result, nil
		}
		pageToken = response.NextPageToken
	}
}

func (c *Client) GetGoogleTagConfig(ctx context.Context, accountID, containerID, workspaceID, configID string) (*GoogleTagConfig, error) {
	config, err := retryWithBackoff(ctx, 3, func() (*tagmanager.GtagConfig, error) {
		return c.Service.Accounts.Containers.Workspaces.GtagConfig.Get(BuildGoogleTagConfigPath(accountID, containerID, workspaceID, configID)).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toGoogleTagConfig(config)
	return &result, nil
}

func (c *Client) CreateGoogleTagConfig(ctx context.Context, accountID, containerID, workspaceID string, input GoogleTagConfigCreateConfig) (*GoogleTagConfig, error) {
	body := &tagmanager.GtagConfig{Type: input.Type, Parameter: toAPIParams(input.Parameter)}
	config, err := c.Service.Accounts.Containers.Workspaces.GtagConfig.Create(BuildWorkspacePath(accountID, containerID, workspaceID), body).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toGoogleTagConfig(config)
	return &result, nil
}

func (c *Client) UpdateGoogleTagConfig(ctx context.Context, accountID, containerID, workspaceID, configID string, input GoogleTagConfigUpdateConfig) (*GoogleTagConfig, error) {
	path := BuildGoogleTagConfigPath(accountID, containerID, workspaceID, configID)
	current, err := retryWithBackoff(ctx, 3, func() (*tagmanager.GtagConfig, error) {
		return c.Service.Accounts.Containers.Workspaces.GtagConfig.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	body := &tagmanager.GtagConfig{Type: current.Type, Parameter: current.Parameter}
	if input.Type != nil {
		body.Type = *input.Type
	}
	if input.Parameter != nil {
		body.Parameter = toAPIParams(*input.Parameter)
		if len(*input.Parameter) == 0 {
			body.Parameter = []*tagmanager.Parameter{}
			body.ForceSendFields = append(body.ForceSendFields, "Parameter")
		}
	}
	config, err := c.Service.Accounts.Containers.Workspaces.GtagConfig.Update(path, body).Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	result := toGoogleTagConfig(config)
	return &result, nil
}

func (c *Client) DeleteGoogleTagConfig(ctx context.Context, accountID, containerID, workspaceID, configID string) error {
	return mapGoogleError(c.Service.Accounts.Containers.Workspaces.GtagConfig.Delete(BuildGoogleTagConfigPath(accountID, containerID, workspaceID, configID)).Context(ctx).Do())
}

func BuildGoogleTagConfigPath(accountID, containerID, workspaceID, configID string) string {
	return fmt.Sprintf("%s/gtag_config/%s", BuildWorkspacePath(accountID, containerID, workspaceID), configID)
}

func toGoogleTagConfig(config *tagmanager.GtagConfig) GoogleTagConfig {
	return GoogleTagConfig{
		AccountID: config.AccountId, ContainerID: config.ContainerId, WorkspaceID: config.WorkspaceId,
		ConfigID: config.GtagConfigId, Type: config.Type, Parameter: config.Parameter,
		Fingerprint: config.Fingerprint, Path: config.Path, TagManagerURL: config.TagManagerUrl,
	}
}
