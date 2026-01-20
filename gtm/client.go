// Package gtm provides a client for the Google Tag Manager API.
package gtm

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Client wraps the Google Tag Manager API service.
type Client struct {
	Service *tagmanager.Service
}

// NewClient creates a GTM client from an OAuth2 token.
func NewClient(ctx context.Context, token *oauth2.Token) (*Client, error) {
	if token == nil {
		return nil, fmt.Errorf("token is required")
	}

	tokenSource := oauth2.StaticTokenSource(token)
	service, err := tagmanager.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create tagmanager service: %w", err)
	}

	return &Client{Service: service}, nil
}
