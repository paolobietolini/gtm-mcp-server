package gtm

import "context"

// listAll follows nextPageToken until every page of a list call has been read,
// retrying each page on rate limits.
func listAll[R, T any](ctx context.Context, fetch func(pageToken string) (R, error), page func(R) (items []T, next string)) ([]T, error) {
	var all []T
	token := ""
	for {
		resp, err := retryWithBackoff(ctx, 3, func() (R, error) { return fetch(token) })
		if err != nil {
			return nil, mapGoogleError(err)
		}
		items, next := page(resp)
		all = append(all, items...)
		if next == "" {
			return all, nil
		}
		token = next
	}
}
