package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// mockTokenStore implements TokenStore for testing middleware.
type mockTokenStore struct {
	tokens map[string]*TokenInfo
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{tokens: make(map[string]*TokenInfo)}
}

func (m *mockTokenStore) StoreToken(info *TokenInfo) error {
	m.tokens[info.AccessToken] = info
	return nil
}

func (m *mockTokenStore) GetTokenByAccess(accessToken string) (*TokenInfo, error) {
	info, ok := m.tokens[accessToken]
	if !ok {
		return nil, ErrTokenNotFound
	}
	if time.Now().After(info.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	return info, nil
}

func (m *mockTokenStore) GetTokenByAccessIncludeExpired(accessToken string) (*TokenInfo, error) {
	info, ok := m.tokens[accessToken]
	if !ok {
		return nil, ErrTokenNotFound
	}
	return info, nil
}

func (m *mockTokenStore) GetTokenByRefresh(refreshToken string) (*TokenInfo, error) {
	return nil, ErrTokenNotFound
}

func (m *mockTokenStore) DeleteToken(accessToken string) error {
	delete(m.tokens, accessToken)
	return nil
}

func (m *mockTokenStore) UpdateGoogleToken(accessToken string, googleToken *oauth2.Token) error {
	info, ok := m.tokens[accessToken]
	if !ok {
		return ErrTokenNotFound
	}
	info.GoogleToken = googleToken
	return nil
}

func (m *mockTokenStore) ExtendTokenExpiry(accessToken string, newExpiry time.Time) error {
	info, ok := m.tokens[accessToken]
	if !ok {
		return ErrTokenNotFound
	}
	info.ExpiresAt = newExpiry
	return nil
}

func (m *mockTokenStore) StoreState(state *AuthState) error              { return nil }
func (m *mockTokenStore) GetState(stateValue string) (*AuthState, error) { return nil, ErrInvalidState }
func (m *mockTokenStore) ConsumeState(stateValue string) (*AuthState, error) {
	return nil, ErrInvalidState
}
func (m *mockTokenStore) DeleteState(stateValue string) error  { return nil }
func (m *mockTokenStore) StoreClient(client *ClientInfo) error { return nil }
func (m *mockTokenStore) GetClient(clientID string) (*ClientInfo, error) {
	return nil, ErrClientNotFound
}
func (m *mockTokenStore) DeleteClient(clientID string) error { return nil }

// mockGoogleProvider wraps GoogleProvider for testing. Since GoogleProvider
// is a concrete struct, we test the middleware with a real GoogleProvider
// that's configured to hit a test server.
func newTestGoogleProvider(tokenServerURL string) *GoogleProvider {
	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServerURL + "/token",
			},
		},
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// dummyHandler is a simple handler that returns 200 OK.
var dummyHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	tokenInfo := GetTokenInfo(r.Context())
	if tokenInfo != nil {
		w.Header().Set("X-Client-ID", tokenInfo.ClientID)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
})

func TestMiddleware_ValidToken(t *testing.T) {
	store := newMockTokenStore()
	logger := testLogger()

	token := &TokenInfo{
		AccessToken: "valid-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		GoogleToken: &oauth2.Token{
			AccessToken: "google-token",
			Expiry:      time.Now().Add(1 * time.Hour),
		},
		ClientID:  "test-client",
		CreatedAt: time.Now(),
	}
	store.StoreToken(token)

	mw := Middleware(store, nil, logger, "http://localhost:8080", 1*time.Hour, nil, nil, "")
	handler := mw(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("X-Client-ID") != "test-client" {
		t.Errorf("expected X-Client-ID 'test-client', got %q", w.Header().Get("X-Client-ID"))
	}
}

func TestMiddleware_MissingAuthHeader(t *testing.T) {
	store := newMockTokenStore()
	logger := testLogger()

	mw := Middleware(store, nil, logger, "http://localhost:8080", 1*time.Hour, nil, nil, "")
	handler := mw(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got %q", resp["error"])
	}
	if resp["authorization_endpoint"] != "http://localhost:8080/authorize" {
		t.Errorf("expected authorization_endpoint, got %q", resp["authorization_endpoint"])
	}
	if resp["token_endpoint"] != "http://localhost:8080/token" {
		t.Errorf("expected token_endpoint, got %q", resp["token_endpoint"])
	}
}

func TestMiddleware_InvalidFormat(t *testing.T) {
	store := newMockTokenStore()
	logger := testLogger()

	mw := Middleware(store, nil, logger, "http://localhost:8080", 1*time.Hour, nil, nil, "")
	handler := mw(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestMiddleware_TokenNotFound(t *testing.T) {
	store := newMockTokenStore()
	logger := testLogger()

	mw := Middleware(store, nil, logger, "http://localhost:8080", 1*time.Hour, nil, nil, "")
	handler := mw(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer nonexistent-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestMiddleware_ExpiredToken_AutoRefreshSuccess(t *testing.T) {
	store := newMockTokenStore()
	logger := testLogger()

	// Set up a fake Google token endpoint
	googleTokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "new-google-access",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "new-google-refresh",
		})
	}))
	defer googleTokenServer.Close()

	google := newTestGoogleProvider(googleTokenServer.URL)

	// Store an expired token with a valid refresh token
	token := &TokenInfo{
		AccessToken:      "expired-token",
		RefreshToken:     "our-refresh-token",
		ExpiresAt:        time.Now().Add(-1 * time.Hour),      // Expired
		RefreshExpiresAt: time.Now().Add(30 * 24 * time.Hour), // Valid
		GoogleToken: &oauth2.Token{
			AccessToken:  "old-google-access",
			RefreshToken: "google-refresh-token",
			Expiry:       time.Now().Add(-1 * time.Hour),
		},
		ClientID:  "test-client",
		CreatedAt: time.Now(),
	}
	store.StoreToken(token)

	mw := Middleware(store, google, logger, "http://localhost:8080", 1*time.Hour, nil, nil, "")
	handler := mw(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 after auto-refresh, got %d", w.Code)
	}

	// The same access token should still be valid (extended in-place)
	refreshed, err := store.GetTokenByAccessIncludeExpired("expired-token")
	if err != nil {
		t.Fatal("expected token to still exist after in-place refresh")
	}

	// Google token should be updated
	if refreshed.GoogleToken.AccessToken != "new-google-access" {
		t.Errorf("expected Google token to be refreshed, got %s", refreshed.GoogleToken.AccessToken)
	}

	// Expiry should be extended
	if refreshed.ExpiresAt.Before(time.Now()) {
		t.Error("expected token expiry to be extended into the future")
	}
}

func TestMiddleware_ExpiredToken_NoRefreshToken(t *testing.T) {
	store := newMockTokenStore()
	logger := testLogger()

	// Expired token without refresh token
	token := &TokenInfo{
		AccessToken:  "expired-no-refresh",
		RefreshToken: "", // No refresh token
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
		GoogleToken: &oauth2.Token{
			AccessToken: "old-google",
			// No RefreshToken
		},
		ClientID:  "test-client",
		CreatedAt: time.Now(),
	}
	store.StoreToken(token)

	mw := Middleware(store, nil, logger, "http://localhost:8080", 1*time.Hour, nil, nil, "")
	handler := mw(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expired-no-refresh")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	// Should have Retry-After header since token is expired
	if w.Header().Get("Retry-After") != "0" {
		t.Errorf("expected Retry-After header for expired token")
	}
}

func TestMiddleware_ExpiredToken_ExpiredRefreshToken(t *testing.T) {
	store := newMockTokenStore()
	logger := testLogger()

	token := &TokenInfo{
		AccessToken:      "expired-both",
		RefreshToken:     "expired-refresh",
		ExpiresAt:        time.Now().Add(-1 * time.Hour),
		RefreshExpiresAt: time.Now().Add(-1 * time.Hour), // Refresh also expired
		GoogleToken: &oauth2.Token{
			AccessToken:  "old-google",
			RefreshToken: "google-refresh",
		},
		ClientID:  "test-client",
		CreatedAt: time.Now(),
	}
	store.StoreToken(token)

	mw := Middleware(store, nil, logger, "http://localhost:8080", 1*time.Hour, nil, nil, "")
	handler := mw(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expired-both")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestMiddleware_ExpiredToken_GoogleRefreshFails(t *testing.T) {
	store := newMockTokenStore()
	logger := testLogger()

	// Set up a Google token endpoint that returns an error
	googleTokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "Token has been revoked",
		})
	}))
	defer googleTokenServer.Close()

	google := newTestGoogleProvider(googleTokenServer.URL)

	token := &TokenInfo{
		AccessToken:      "expired-revoked",
		RefreshToken:     "our-refresh",
		ExpiresAt:        time.Now().Add(-1 * time.Hour),
		RefreshExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		GoogleToken: &oauth2.Token{
			AccessToken:  "old-google",
			RefreshToken: "revoked-google-refresh",
		},
		ClientID:  "test-client",
		CreatedAt: time.Now(),
	}
	store.StoreToken(token)

	mw := Middleware(store, google, logger, "http://localhost:8080", 1*time.Hour, nil, nil, "")
	handler := mw(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expired-revoked")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestUnauthorized_IncludesEndpoints(t *testing.T) {
	w := httptest.NewRecorder()
	unauthorized(w, "https://mcp.example.com", "Test error")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["authorization_endpoint"] != "https://mcp.example.com/authorize" {
		t.Errorf("expected authorization_endpoint, got %q", resp["authorization_endpoint"])
	}
	if resp["token_endpoint"] != "https://mcp.example.com/token" {
		t.Errorf("expected token_endpoint, got %q", resp["token_endpoint"])
	}
}

func TestUnauthorized_RetryAfterOnExpired(t *testing.T) {
	w := httptest.NewRecorder()
	unauthorized(w, "https://mcp.example.com", "Token expired")

	if w.Header().Get("Retry-After") != "0" {
		t.Errorf("expected Retry-After: 0 for expired token message")
	}

	w2 := httptest.NewRecorder()
	unauthorized(w2, "https://mcp.example.com", "Invalid token")

	if w2.Header().Get("Retry-After") != "" {
		t.Errorf("expected no Retry-After for non-expired message")
	}
}

func TestTruncateToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abcdefghijklmnop", "abcdefgh..."},
		{"short", "short..."},
		{"12345678", "12345678..."},
		{"123456789", "12345678..."},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("len=%d", len(tt.input)), func(t *testing.T) {
			result := truncateToken(tt.input)
			if result != tt.expected {
				t.Errorf("truncateToken(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMiddleware_ErrorResponseFormat(t *testing.T) {
	store := newMockTokenStore()
	logger := testLogger()

	mw := Middleware(store, nil, logger, "https://mcp.gtmeditor.com", 1*time.Hour, nil, nil, "")
	handler := mw(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check WWW-Authenticate header
	wwwAuth := w.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("expected WWW-Authenticate header")
	}
	if !contains(wwwAuth, "resource_metadata") {
		t.Error("expected resource_metadata in WWW-Authenticate header")
	}

	// Check Content-Type
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", w.Header().Get("Content-Type"))
	}

	// Check JSON body
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp["error"] != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got %q", resp["error"])
	}
	if resp["authorization_endpoint"] == "" {
		t.Error("expected authorization_endpoint in response")
	}
	if resp["token_endpoint"] == "" {
		t.Error("expected token_endpoint in response")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestMiddleware_ContextValues(t *testing.T) {
	store := newMockTokenStore()
	logger := testLogger()
	google := newTestGoogleProvider("http://localhost:9999")

	token := &TokenInfo{
		AccessToken: "ctx-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		GoogleToken: &oauth2.Token{
			AccessToken: "google-ctx-token",
			Expiry:      time.Now().Add(1 * time.Hour),
		},
		ClientID:  "ctx-client",
		CreatedAt: time.Now(),
	}
	store.StoreToken(token)

	var capturedCtx context.Context
	captureHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	mw := Middleware(store, google, logger, "http://localhost:8080", 1*time.Hour, nil, nil, "")
	handler := mw(captureHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ctx-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify all context values are set
	tokenInfo := GetTokenInfo(capturedCtx)
	if tokenInfo == nil || tokenInfo.ClientID != "ctx-client" {
		t.Error("expected TokenInfo in context")
	}

	googleToken := GetGoogleToken(capturedCtx)
	if googleToken == nil || googleToken.AccessToken != "google-ctx-token" {
		t.Error("expected GoogleToken in context")
	}

	tokenStore := GetTokenStore(capturedCtx)
	if tokenStore == nil {
		t.Error("expected TokenStore in context")
	}

	googleProvider := GetGoogleProvider(capturedCtx)
	if googleProvider == nil {
		t.Error("expected GoogleProvider in context")
	}
}

// saMiddlewareWithOAuth is a helper that builds Middleware with S2S configured.
func saMiddlewareWithOAuth(store TokenStore, apiKey string, saTS oauth2.TokenSource) func(http.Handler) http.Handler {
	return Middleware(store, nil, testLogger(), "http://localhost:8080", 1*time.Hour, nil, saTS, apiKey)
}

func TestMiddleware_SAMode_CorrectKey(t *testing.T) {
	store := newMockTokenStore()
	saTS := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "google-sa-token",
		Expiry:      time.Now().Add(time.Hour),
	})

	var capturedCtx context.Context
	captureHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	handler := saMiddlewareWithOAuth(store, "my-api-key", saTS)(captureHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer my-api-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	got := GetSATokenSource(capturedCtx)
	if got == nil {
		t.Fatal("expected SA token source in context")
	}
	tokenInfo := GetTokenInfo(capturedCtx)
	if tokenInfo == nil {
		t.Fatal("expected TokenInfo in context for auth_status compatibility")
	}
	if tokenInfo.ClientID != "service-account" {
		t.Errorf("expected ClientID 'service-account', got %q", tokenInfo.ClientID)
	}
}

func TestMiddleware_SAMode_WrongKey_NoOAuthToken(t *testing.T) {
	store := newMockTokenStore()
	saTS := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "google-sa-token",
		Expiry:      time.Now().Add(time.Hour),
	})

	handler := saMiddlewareWithOAuth(store, "my-api-key", saTS)(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_SAMode_WrongKey_ValidOAuthToken(t *testing.T) {
	store := newMockTokenStore()
	saTS := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "google-sa-token",
		Expiry:      time.Now().Add(time.Hour),
	})

	oauthToken := &TokenInfo{
		AccessToken: "valid-oauth-token",
		ExpiresAt:   time.Now().Add(time.Hour),
		GoogleToken: &oauth2.Token{
			AccessToken: "google-oauth-token",
			Expiry:      time.Now().Add(time.Hour),
		},
		ClientID:  "oauth-client",
		CreatedAt: time.Now(),
	}
	store.StoreToken(oauthToken)

	handler := saMiddlewareWithOAuth(store, "my-api-key", saTS)(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-oauth-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// OAuth path should succeed even when SA mode is active
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid OAuth token in SA mode, got %d", w.Code)
	}
	if w.Header().Get("X-Client-ID") != "oauth-client" {
		t.Errorf("expected OAuth client ID in header, got %q", w.Header().Get("X-Client-ID"))
	}
}

func TestMiddleware_OAuthUser_NoSATokenSource(t *testing.T) {
	store := newMockTokenStore()
	saTS := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "google-sa-token",
		Expiry:      time.Now().Add(time.Hour),
	})

	oauthToken := &TokenInfo{
		AccessToken: "valid-oauth-token",
		ExpiresAt:   time.Now().Add(time.Hour),
		GoogleToken: &oauth2.Token{
			AccessToken: "google-oauth-token",
			Expiry:      time.Now().Add(time.Hour),
		},
		ClientID:  "oauth-client",
		CreatedAt: time.Now(),
	}
	store.StoreToken(oauthToken)

	var capturedCtx context.Context
	captureHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	handler := saMiddlewareWithOAuth(store, "my-api-key", saTS)(captureHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-oauth-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if got := GetSATokenSource(capturedCtx); got != nil {
		t.Error("OAuth user must NOT receive SA token source — would grant access to SA-managed containers")
	}

	if got := GetGoogleToken(capturedCtx); got == nil {
		t.Error("OAuth user must have their own Google token in context")
	} else if got.AccessToken != "google-oauth-token" {
		t.Errorf("expected user's Google token, got %q", got.AccessToken)
	}
}
