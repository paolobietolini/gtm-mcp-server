package gtm

import "context"

func (c *Client) CombineContainers(ctx context.Context, accountID, containerID, sourceContainerID, settingSource string) (*Container, error) {
	result, err := c.Service.Accounts.Containers.Combine(BuildContainerPath(accountID, containerID)).
		ContainerId(sourceContainerID).
		SettingSource(settingSource).
		Context(ctx).
		Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	container := toContainer(result)
	return &container, nil
}

func (c *Client) MoveTagID(ctx context.Context, accountID, containerID, tagID, tagName string, copySettings bool) (*Container, error) {
	result, err := c.Service.Accounts.Containers.MoveTagId(BuildContainerPath(accountID, containerID)).
		TagId(tagID).
		TagName(tagName).
		CopySettings(copySettings).
		CopyTermsOfService(true).
		CopyUsers(false).
		Context(ctx).
		Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}
	container := toContainer(result)
	return &container, nil
}
