package auth

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider handles OAuth2 flow with Google.
type GoogleProvider struct {
	config *oauth2.Config
}

// GoogleScopes defines the scopes needed for GTM API access.
var GoogleScopes = []string{
	"https://www.googleapis.com/auth/tagmanager.delete.containers",
	"https://www.googleapis.com/auth/tagmanager.edit.containers",
	"https://www.googleapis.com/auth/tagmanager.edit.containerversions",
	"https://www.googleapis.com/auth/tagmanager.publish",
}

// NewGoogleProvider creates a new Google OAuth provider.
func NewGoogleProvider(clientID, clientSecret, redirectURI string) *GoogleProvider {
	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Scopes:       GoogleScopes,
			Endpoint:     google.Endpoint,
		},
	}
}

// AuthCodeURL generates the URL to redirect users to Google's OAuth consent page.
func (p *GoogleProvider) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	// Always request offline access for refresh tokens
	opts = append(opts, oauth2.AccessTypeOffline)
	// Force consent to always get refresh token
	opts = append(opts, oauth2.ApprovalForce)

	return p.config.AuthCodeURL(state, opts...)
}

// Exchange converts an authorization code into tokens.
func (p *GoogleProvider) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	token, err := p.config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	return token, nil
}

// RefreshToken uses a refresh token to get a new access token.
func (p *GoogleProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	tokenSource := p.config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return token, nil
}

// Client returns an HTTP client that automatically handles token refresh.
func (p *GoogleProvider) Client(ctx context.Context, token *oauth2.Token) *oauth2.Config {
	return p.config
}

// Config returns the underlying oauth2.Config.
func (p *GoogleProvider) Config() *oauth2.Config {
	return p.config
}

// AutoRefreshTokenSource wraps oauth2.Token with automatic refresh and store updates.
type AutoRefreshTokenSource struct {
	mu          sync.Mutex
	store       TokenStore
	accessToken string // Our token (to identify the record in store)
	config      *oauth2.Config
	current     *oauth2.Token
}

// NewAutoRefreshTokenSource creates a token source that auto-refreshes and updates the store.
func NewAutoRefreshTokenSource(store TokenStore, accessToken string, config *oauth2.Config, token *oauth2.Token) *AutoRefreshTokenSource {
	return &AutoRefreshTokenSource{
		store:       store,
		accessToken: accessToken,
		config:      config,
		current:     token,
	}
}

// Token returns a valid token, refreshing if necessary.
func (s *AutoRefreshTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If token is still valid, return it
	if s.current.Valid() {
		return s.current, nil
	}

	slog.Info("Google token expired, refreshing...")

	// Token expired or about to expire, refresh it
	tokenSource := s.config.TokenSource(context.Background(), s.current)
	newToken, err := tokenSource.Token()
	if err != nil {
		slog.Error("Failed to refresh Google token", "error", err)
		return nil, fmt.Errorf("failed to refresh Google token: %w", err)
	}

	slog.Info("Google token refreshed successfully", "new_expiry", newToken.Expiry)

	// Update our current token
	s.current = newToken

	// Update in store (best effort - don't fail if store update fails)
	if s.store != nil && s.accessToken != "" {
		if err := s.store.UpdateGoogleToken(s.accessToken, newToken); err != nil {
			slog.Warn("failed to update Google token in store", "error", err)
		}
	}

	return newToken, nil
}
