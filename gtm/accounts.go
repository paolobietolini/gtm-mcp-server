package gtm

import (
	"context"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Account is a simplified representation of a GTM account.
type Account struct {
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}

// ListAccounts returns all GTM accounts accessible to the authenticated user.
func (c *Client) ListAccounts(ctx context.Context) ([]Account, error) {
	resp, err := c.Service.Accounts.List().Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return toAccounts(resp.Account), nil
}

func toAccounts(accounts []*tagmanager.Account) []Account {
	result := make([]Account, 0, len(accounts))
	for _, a := range accounts {
		result = append(result, Account{
			AccountID: a.AccountId,
			Name:      a.Name,
			Path:      a.Path,
		})
	}
	return result
}
