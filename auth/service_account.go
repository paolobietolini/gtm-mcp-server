package auth

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

const SATokenSourceKey ContextKey = "sa_token_source"

// NewServiceAccountTokenSource creates an oauth2.TokenSource scoped to GTM API access.
// If keyJSON is non-empty, it is parsed as a service account JSON key.
// Otherwise Application Default Credentials are used (Workload Identity on GCP).
func NewServiceAccountTokenSource(ctx context.Context, keyJSON string) (oauth2.TokenSource, error) {
	creds, err := ResolveCredentials(ctx, keyJSON)
	if err != nil {
		return nil, err
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
