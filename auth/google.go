package auth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider handles OAuth2 flow with Google.
type GoogleProvider struct {
	config *oauth2.Config
}

// GoogleScopes defines the scopes needed for GTM API access.
var GoogleScopes = []string{
	"https://www.googleapis.com/auth/tagmanager.edit.containers",
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
