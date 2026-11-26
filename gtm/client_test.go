package gtm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

type quotaSource struct{ oauth2.TokenSource }

func (quotaSource) QuotaProjectID() string { return "my-project" }

func TestClientsSendCredentialsAndQuotaProject(t *testing.T) {
	var auth, quota string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, quota = r.Header.Get("Authorization"), r.Header.Get("X-Goog-User-Project")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	ts := quotaSource{oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "tok"})}
	endpoint := option.WithEndpoint(server.URL + "/")
	wrapped := 0

	constructors := map[string]func() (*Client, error){
		"NewClient": func() (*Client, error) { return NewClient(context.Background(), ts, endpoint) },
		"NewClientWithTransport": func() (*Client, error) {
			return NewClientWithTransport(context.Background(), ts, func(next http.RoundTripper) http.RoundTripper {
				return roundTripFunc(func(r *http.Request) (*http.Response, error) {
					wrapped++
					return next.RoundTrip(r)
				})
			}, endpoint)
		},
	}
	for name, build := range constructors {
		t.Run(name, func(t *testing.T) {
			auth, quota = "", ""
			client, err := build()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.ListAccounts(context.Background()); err != nil {
				t.Fatal(err)
			}
			if auth != "Bearer tok" || quota != "my-project" {
				t.Fatalf("Authorization=%q quota=%q", auth, quota)
			}
		})
	}
	if wrapped != 1 {
		t.Fatalf("wrapper saw %d requests", wrapped)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
