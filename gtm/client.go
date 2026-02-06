// Package gtm provides a client for the Google Tag Manager API.
package gtm

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// loggingTransport wraps an http.RoundTripper and logs request/response bodies.
type loggingTransport struct {
	wrapped http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dump, _ := httputil.DumpRequestOut(req, true)
	log.Printf("[HTTP REQUEST] %s", string(dump))

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
// The token source should handle automatic refresh.
func NewClient(ctx context.Context, tokenSource oauth2.TokenSource) (*Client, error) {
	if tokenSource == nil {
		return nil, fmt.Errorf("token source is required")
	}

	opts := []option.ClientOption{option.WithTokenSource(tokenSource)}

	// Enable HTTP request/response logging when GTM_DEBUG is set
	if os.Getenv("GTM_DEBUG") != "" {
		httpClient := oauth2.NewClient(ctx, tokenSource)
		httpClient.Transport = &loggingTransport{wrapped: httpClient.Transport}
		opts = []option.ClientOption{option.WithHTTPClient(httpClient)}
	}

	service, err := tagmanager.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create tagmanager service: %w", err)
	}

	return &Client{Service: service}, nil
}
