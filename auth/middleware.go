package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

// ContextKey is the type for context keys.
type ContextKey string

const (
	// TokenInfoKey is the context key for TokenInfo.
	TokenInfoKey ContextKey = "token_info"
	// GoogleTokenKey is the context key for the Google OAuth token.
	GoogleTokenKey ContextKey = "google_token"
)

// Middleware creates HTTP middleware that validates bearer tokens.
func Middleware(store TokenStore, logger *slog.Logger, baseURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract bearer token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				unauthorized(w, baseURL, "Missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				unauthorized(w, baseURL, "Invalid authorization header format")
				return
			}

			accessToken := parts[1]

			// Look up the token
			tokenInfo, err := store.GetTokenByAccess(accessToken)
			if err != nil {
				if err == ErrTokenExpired {
					unauthorized(w, baseURL, "Token expired")
				} else {
					unauthorized(w, baseURL, "Invalid token")
				}
				return
			}

			// Add token info to context
			ctx := context.WithValue(r.Context(), TokenInfoKey, tokenInfo)
			ctx = context.WithValue(ctx, GoogleTokenKey, tokenInfo.GoogleToken)

			logger.Debug("authenticated request",
				"client_id", tokenInfo.ClientID,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalMiddleware allows unauthenticated requests but adds token info if present.
func OptionalMiddleware(store TokenStore, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				next.ServeHTTP(w, r)
				return
			}

			accessToken := parts[1]
			tokenInfo, err := store.GetTokenByAccess(accessToken)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), TokenInfoKey, tokenInfo)
			ctx = context.WithValue(ctx, GoogleTokenKey, tokenInfo.GoogleToken)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTokenInfo retrieves TokenInfo from context.
func GetTokenInfo(ctx context.Context) *TokenInfo {
	if info, ok := ctx.Value(TokenInfoKey).(*TokenInfo); ok {
		return info
	}
	return nil
}

// GetGoogleToken retrieves the Google OAuth token from context.
func GetGoogleToken(ctx context.Context) *oauth2.Token {
	if token, ok := ctx.Value(GoogleTokenKey).(*oauth2.Token); ok {
		return token
	}
	return nil
}

// unauthorized sends a 401 response with WWW-Authenticate header.
func unauthorized(w http.ResponseWriter, baseURL, message string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="`+baseURL+`"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"unauthorized","error_description":"` + message + `"}`))
}
