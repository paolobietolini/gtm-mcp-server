package gtm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/googleapi"
)

func TestPermission403IsNotRetried(t *testing.T) {
	calls := 0
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(403)
		fmt.Fprint(w, `{"error":{"code":403,"message":"Insufficient Permission","errors":[{"reason":"insufficientPermissions","message":"Insufficient Permission"}]}}`)
	})
	_, err := client.ListAccounts(context.Background())
	if !errors.Is(err, ErrPermission) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestRateLimit403MapsToRateLimit(t *testing.T) {
	for _, apiErr := range []*googleapi.Error{
		{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "rateLimitExceeded"}}},
		{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "userRateLimitExceeded"}}},
		{Code: 403, Message: "Quota exceeded for quota metric 'Queries'"},
	} {
		if err := mapGoogleError(apiErr); !errors.Is(err, ErrRateLimit) {
			t.Errorf("%+v mapped to %v", apiErr, err)
		}
	}
}

func TestCoreListsFollowEveryPage(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		field string
		list  func(*Client) (int, error)
	}{
		{"account", func(c *Client) (int, error) { v, err := c.ListAccounts(ctx); return len(v), err }},
		{"container", func(c *Client) (int, error) { v, err := c.ListContainers(ctx, "1"); return len(v), err }},
		{"tag", func(c *Client) (int, error) { v, err := c.ListTags(ctx, "1", "2", "3"); return len(v), err }},
		{"trigger", func(c *Client) (int, error) { v, err := c.ListTriggers(ctx, "1", "2", "3"); return len(v), err }},
		{"variable", func(c *Client) (int, error) { v, err := c.ListVariables(ctx, "1", "2", "3"); return len(v), err }},
	} {
		t.Run(tc.field, func(t *testing.T) {
			client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				next := `,"nextPageToken":"p2"`
				if r.URL.Query().Get("pageToken") == "p2" {
					next = ""
				}
				fmt.Fprintf(w, `{"%s":[{"name":"x"}]%s}`, tc.field, next)
			})
			n, err := tc.list(client)
			if err != nil || n != 2 {
				t.Fatalf("got %d items, err=%v", n, err)
			}
		})
	}
}
