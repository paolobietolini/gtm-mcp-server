package gtm

import (
	"context"
	"fmt"

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
	items, err := listAll(ctx, func(token string) (*tagmanager.ListAccountsResponse, error) {
		return c.Service.Accounts.List().PageToken(token).Context(ctx).Do()
	}, func(r *tagmanager.ListAccountsResponse) ([]*tagmanager.Account, string) {
		return r.Account, r.NextPageToken
	})
	if err != nil {
		return nil, err
	}

	return toAccounts(items), nil
}

// UpdateAccount renames a GTM account. It fetches the current account first to get the fingerprint.
func (c *Client) UpdateAccount(ctx context.Context, accountID, name string) (*Account, error) {
	path := fmt.Sprintf("accounts/%s", accountID)

	current, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Account, error) {
		return c.Service.Accounts.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	current.Name = name

	updated, err := c.Service.Accounts.Update(path, current).Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &Account{
		AccountID: updated.AccountId,
		Name:      updated.Name,
		Path:      updated.Path,
	}, nil
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
