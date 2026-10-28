package auth

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const SATokenSourceKey ContextKey = "sa_token_source"

// NewServiceAccountTokenSource creates an oauth2.TokenSource scoped to GTM API access.
// If keyJSON is non-empty, it is parsed as a service account JSON key.
// Otherwise Application Default Credentials are used (Workload Identity on GCP).
func NewServiceAccountTokenSource(ctx context.Context, keyJSON string) (oauth2.TokenSource, error) {
	if keyJSON != "" {
		config, err := google.JWTConfigFromJSON([]byte(keyJSON), GoogleScopes...)
		if err != nil {
			return nil, fmt.Errorf("invalid GOOGLE_SERVICE_ACCOUNT_KEY_JSON: %w", err)
		}
		return config.TokenSource(ctx), nil
	}

	creds, err := google.FindDefaultCredentials(ctx, GoogleScopes...)
	if err != nil {
		return nil, fmt.Errorf("no service account credentials found (set GOOGLE_SERVICE_ACCOUNT_KEY_JSON or GOOGLE_APPLICATION_CREDENTIALS, or use Workload Identity): %w", err)
	}
	return creds.TokenSource, nil
}

// ServiceAccountTokenInfo returns the TokenInfo attached to requests served with
// the shared service account, so auth_status reports them as authenticated.
func ServiceAccountTokenInfo() *TokenInfo {
	now := time.Now()
	return &TokenInfo{
		ClientID:  "service-account",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour * 365 * 10),
	}
}

// GetSATokenSource retrieves the service account token source from context.
// Returns nil when not in S2S mode.
func GetSATokenSource(ctx context.Context) oauth2.TokenSource {
	if ts, ok := ctx.Value(SATokenSourceKey).(oauth2.TokenSource); ok {
		return ts
	}
	return nil
}
