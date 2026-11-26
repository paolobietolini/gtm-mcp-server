// Package gtm provides a client for the Google Tag Manager API.
package gtm

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	tagmanager "google.golang.org/api/tagmanager/v2"
	htransport "google.golang.org/api/transport/http"
)

var authHeaderRe = regexp.MustCompile(`(?i)(Authorization:\s*)Bearer\s+\S+`)

// loggingTransport wraps an http.RoundTripper and logs request/response bodies
// with sensitive headers redacted.
type loggingTransport struct {
	wrapped http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dump, _ := httputil.DumpRequestOut(req, true)
	redacted := authHeaderRe.ReplaceAllString(string(dump), "${1}Bearer [REDACTED]")
	log.Printf("[HTTP REQUEST] %s", redacted)

	resp, err := t.wrapped.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	respDump, _ := httputil.DumpResponse(resp, true)
	log.Printf("[HTTP RESPONSE] %s", string(respDump))

	return resp, nil
}

// Client wraps the Google Tag Manager API service.
type Client struct {
	Service *tagmanager.Service
}

// NewClient creates a GTM client from an OAuth2 token source.
// The token source should handle automatic refresh. If it also has a
// QuotaProjectID method, that project is billed for requests.
func NewClient(ctx context.Context, tokenSource oauth2.TokenSource, opts ...option.ClientOption) (*Client, error) {
	if tokenSource == nil {
		return nil, fmt.Errorf("token source is required")
	}

	// Enable HTTP request/response logging when GTM_DEBUG is set
	if os.Getenv("GTM_DEBUG") != "" {
		baseURL := os.Getenv("BASE_URL")
		if baseURL != "" && !strings.Contains(baseURL, "localhost") && !strings.Contains(baseURL, "127.0.0.1") {
			log.Printf("WARNING: GTM_DEBUG ignored in production (BASE_URL=%s)", baseURL)
		} else {
			log.Printf("WARNING: GTM_DEBUG is enabled — HTTP bodies will be logged (headers redacted)")
			return NewClientWithTransport(ctx, tokenSource, func(next http.RoundTripper) http.RoundTripper {
				return &loggingTransport{wrapped: next}
			}, opts...)
		}
	}

	return newClient(ctx, append(credentialOptions(tokenSource), opts...)...)
}

// NewClientWithTransport is NewClient with a wrapper around the authenticated
// HTTP transport, so a caller can observe or intercept requests.
func NewClientWithTransport(ctx context.Context, tokenSource oauth2.TokenSource, wrap func(http.RoundTripper) http.RoundTripper, opts ...option.ClientOption) (*Client, error) {
	if tokenSource == nil {
		return nil, fmt.Errorf("token source is required")
	}
	transport, err := htransport.NewTransport(ctx, http.DefaultTransport, credentialOptions(tokenSource)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticated transport: %w", err)
	}
	httpClient := &http.Client{Transport: wrap(transport)}
	return newClient(ctx, append([]option.ClientOption{option.WithHTTPClient(httpClient)}, opts...)...)
}

func credentialOptions(tokenSource oauth2.TokenSource) []option.ClientOption {
	opts := []option.ClientOption{option.WithTokenSource(tokenSource)}
	if quota, ok := tokenSource.(interface{ QuotaProjectID() string }); ok && quota.QuotaProjectID() != "" {
		opts = append(opts, option.WithQuotaProject(quota.QuotaProjectID()))
	}
	return opts
}

func newClient(ctx context.Context, opts ...option.ClientOption) (*Client, error) {
	service, err := tagmanager.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create tagmanager service: %w", err)
	}

	return &Client{Service: service}, nil
}
