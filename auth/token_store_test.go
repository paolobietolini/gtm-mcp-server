package auth

import (
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestMemoryTokenStore_StoreAndGetTokenByAccess(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	tokenInfo := &TokenInfo{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		GoogleToken: &oauth2.Token{
			AccessToken: "google-token",
			Expiry:      time.Now().Add(1 * time.Hour),
		},
		ClientID:  "test-client",
		CreatedAt: time.Now(),
	}

	err := store.StoreToken(tokenInfo)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	retrieved, err := store.GetTokenByAccess("test-access-token")
	if err != nil {
		t.Fatalf("GetTokenByAccess failed: %v", err)
	}

	if retrieved.AccessToken != tokenInfo.AccessToken {
		t.Errorf("expected access token %q, got %q", tokenInfo.AccessToken, retrieved.AccessToken)
	}
	if retrieved.RefreshToken != tokenInfo.RefreshToken {
		t.Errorf("expected refresh token %q, got %q", tokenInfo.RefreshToken, retrieved.RefreshToken)
	}
	if retrieved.ClientID != tokenInfo.ClientID {
		t.Errorf("expected client ID %q, got %q", tokenInfo.ClientID, retrieved.ClientID)
	}
}

func TestMemoryTokenStore_GetTokenByAccess_NotFound(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	_, err := store.GetTokenByAccess("nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestMemoryTokenStore_GetTokenByAccess_Expired(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	tokenInfo := &TokenInfo{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		CreatedAt:   time.Now(),
	}

	err := store.StoreToken(tokenInfo)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	_, err = store.GetTokenByAccess("expired-token")
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestMemoryTokenStore_GetTokenByRefresh(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	tokenInfo := &TokenInfo{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		CreatedAt:    time.Now(),
	}

	err := store.StoreToken(tokenInfo)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	retrieved, err := store.GetTokenByRefresh("test-refresh")
	if err != nil {
		t.Fatalf("GetTokenByRefresh failed: %v", err)
	}

	if retrieved.AccessToken != "test-access" {
		t.Errorf("expected access token %q, got %q", "test-access", retrieved.AccessToken)
	}
	if retrieved.RefreshToken != "test-refresh" {
		t.Errorf("expected refresh token %q, got %q", "test-refresh", retrieved.RefreshToken)
	}
}

func TestMemoryTokenStore_GetTokenByRefresh_NotFound(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	_, err := store.GetTokenByRefresh("nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestMemoryTokenStore_UpdateGoogleToken(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	tokenInfo := &TokenInfo{
		AccessToken: "test-access",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		GoogleToken: &oauth2.Token{
			AccessToken: "old-google-token",
		},
		CreatedAt: time.Now(),
	}

	err := store.StoreToken(tokenInfo)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	newGoogleToken := &oauth2.Token{
		AccessToken: "new-google-token",
		Expiry:      time.Now().Add(2 * time.Hour),
	}

	err = store.UpdateGoogleToken("test-access", newGoogleToken)
	if err != nil {
		t.Fatalf("UpdateGoogleToken failed: %v", err)
	}

	retrieved, err := store.GetTokenByAccess("test-access")
	if err != nil {
		t.Fatalf("GetTokenByAccess failed: %v", err)
	}

	if retrieved.GoogleToken.AccessToken != "new-google-token" {
		t.Errorf("expected google token %q, got %q", "new-google-token", retrieved.GoogleToken.AccessToken)
	}
}

func TestMemoryTokenStore_UpdateGoogleToken_NilToken(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	err := store.UpdateGoogleToken("test-access", nil)
	if err == nil {
		t.Error("expected error when passing nil googleToken, got nil")
	}
	if err.Error() != "googleToken cannot be nil" {
		t.Errorf("expected error 'googleToken cannot be nil', got %v", err)
	}
}

func TestMemoryTokenStore_UpdateGoogleToken_NotFound(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	err := store.UpdateGoogleToken("nonexistent", &oauth2.Token{})
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestMemoryTokenStore_DeleteToken(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	tokenInfo := &TokenInfo{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		CreatedAt:    time.Now(),
	}

	err := store.StoreToken(tokenInfo)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	err = store.DeleteToken("test-access")
	if err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}

	// Verify token is deleted
	_, err = store.GetTokenByAccess("test-access")
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound after deletion, got %v", err)
	}

	// Verify refresh index is also cleaned up
	_, err = store.GetTokenByRefresh("test-refresh")
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound for refresh token after deletion, got %v", err)
	}
}

func TestMemoryTokenStore_DeleteToken_NotFound(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	// Should not error when deleting non-existent token
	err := store.DeleteToken("nonexistent")
	if err != nil {
		t.Errorf("expected no error when deleting nonexistent token, got %v", err)
	}
}

func TestMemoryTokenStore_StoreAndGetState(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	authState := &AuthState{
		State:        "test-state",
		CodeVerifier: "test-verifier",
		RedirectURI:  "https://example.com/callback",
		ClientID:     "test-client",
		Resource:     "test-resource",
		CreatedAt:    time.Now(),
	}

	err := store.StoreState(authState)
	if err != nil {
		t.Fatalf("StoreState failed: %v", err)
	}

	retrieved, err := store.GetState("test-state")
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}

	if retrieved.State != authState.State {
		t.Errorf("expected state %q, got %q", authState.State, retrieved.State)
	}
	if retrieved.CodeVerifier != authState.CodeVerifier {
		t.Errorf("expected code verifier %q, got %q", authState.CodeVerifier, retrieved.CodeVerifier)
	}
	if retrieved.RedirectURI != authState.RedirectURI {
		t.Errorf("expected redirect URI %q, got %q", authState.RedirectURI, retrieved.RedirectURI)
	}
}

func TestMemoryTokenStore_GetState_NotFound(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	_, err := store.GetState("nonexistent")
	if err != ErrInvalidState {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestMemoryTokenStore_GetState_Expired(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	authState := &AuthState{
		State:     "expired-state",
		CreatedAt: time.Now().Add(-11 * time.Minute), // Expired (>10 min)
	}

	err := store.StoreState(authState)
	if err != nil {
		t.Fatalf("StoreState failed: %v", err)
	}

	_, err = store.GetState("expired-state")
	if err != ErrInvalidState {
		t.Errorf("expected ErrInvalidState for expired state, got %v", err)
	}
}

func TestMemoryTokenStore_DeleteState(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	authState := &AuthState{
		State:     "test-state",
		CreatedAt: time.Now(),
	}

	err := store.StoreState(authState)
	if err != nil {
		t.Fatalf("StoreState failed: %v", err)
	}

	err = store.DeleteState("test-state")
	if err != nil {
		t.Fatalf("DeleteState failed: %v", err)
	}

	_, err = store.GetState("test-state")
	if err != ErrInvalidState {
		t.Errorf("expected ErrInvalidState after deletion, got %v", err)
	}
}

func TestMemoryTokenStore_StoreAndGetClient(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	clientInfo := &ClientInfo{
		ClientID:     "test-client-id",
		RedirectURIs: []string{"https://example.com/callback"},
		ClientName:   "Test Client",
		GrantTypes:   []string{"authorization_code", "refresh_token"},
		ResponseTypes: []string{"code"},
		TokenEndpointAuthMethod: "none",
		CreatedAt:    time.Now(),
	}

	err := store.StoreClient(clientInfo)
	if err != nil {
		t.Fatalf("StoreClient failed: %v", err)
	}

	retrieved, err := store.GetClient("test-client-id")
	if err != nil {
		t.Fatalf("GetClient failed: %v", err)
	}

	if retrieved.ClientID != clientInfo.ClientID {
		t.Errorf("expected client ID %q, got %q", clientInfo.ClientID, retrieved.ClientID)
	}
	if retrieved.ClientName != clientInfo.ClientName {
		t.Errorf("expected client name %q, got %q", clientInfo.ClientName, retrieved.ClientName)
	}
	if len(retrieved.RedirectURIs) != 1 || retrieved.RedirectURIs[0] != "https://example.com/callback" {
		t.Errorf("expected redirect URIs %v, got %v", clientInfo.RedirectURIs, retrieved.RedirectURIs)
	}
}

func TestMemoryTokenStore_GetClient_NotFound(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	_, err := store.GetClient("nonexistent")
	if err != ErrClientNotFound {
		t.Errorf("expected ErrClientNotFound, got %v", err)
	}
}

func TestMemoryTokenStore_DeleteClient(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	clientInfo := &ClientInfo{
		ClientID:  "test-client-id",
		CreatedAt: time.Now(),
	}

	err := store.StoreClient(clientInfo)
	if err != nil {
		t.Fatalf("StoreClient failed: %v", err)
	}

	err = store.DeleteClient("test-client-id")
	if err != nil {
		t.Fatalf("DeleteClient failed: %v", err)
	}

	_, err = store.GetClient("test-client-id")
	if err != ErrClientNotFound {
		t.Errorf("expected ErrClientNotFound after deletion, got %v", err)
	}
}

func TestMemoryTokenStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	const numGoroutines = 100
	const numOperations = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent token operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				accessToken := "token-" + string(rune(id)) + "-" + string(rune(j))
				refreshToken := "refresh-" + string(rune(id)) + "-" + string(rune(j))

				// Store token
				tokenInfo := &TokenInfo{
					AccessToken:  accessToken,
					RefreshToken: refreshToken,
					ExpiresAt:    time.Now().Add(1 * time.Hour),
					CreatedAt:    time.Now(),
				}
				store.StoreToken(tokenInfo)

				// Get by access token
				_, _ = store.GetTokenByAccess(accessToken)

				// Get by refresh token
				_, _ = store.GetTokenByRefresh(refreshToken)

				// Update Google token
				_ = store.UpdateGoogleToken(accessToken, &oauth2.Token{
					AccessToken: "google-" + accessToken,
				})

				// Delete token
				_ = store.DeleteToken(accessToken)
			}
		}(i)
	}

	wg.Wait()
}

func TestMemoryTokenStore_ConcurrentStateAccess(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	const numGoroutines = 50
	const numOperations = 30

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				state := "state-" + string(rune(id)) + "-" + string(rune(j))

				authState := &AuthState{
					State:     state,
					CreatedAt: time.Now(),
				}
				store.StoreState(authState)

				_, _ = store.GetState(state)

				store.DeleteState(state)
			}
		}(i)
	}

	wg.Wait()
}

func TestMemoryTokenStore_ConcurrentClientAccess(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	const numGoroutines = 50
	const numOperations = 30

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				clientID := "client-" + string(rune(id)) + "-" + string(rune(j))

				clientInfo := &ClientInfo{
					ClientID:  clientID,
					CreatedAt: time.Now(),
				}
				store.StoreClient(clientInfo)

				_, _ = store.GetClient(clientID)

				store.DeleteClient(clientID)
			}
		}(i)
	}

	wg.Wait()
}

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"short", 16},
		{"medium", 32},
		{"long", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token1, err := GenerateToken(tt.length)
			if err != nil {
				t.Fatalf("GenerateToken failed: %v", err)
			}

			token2, err := GenerateToken(tt.length)
			if err != nil {
				t.Fatalf("GenerateToken failed: %v", err)
			}

			// Tokens should be different
			if token1 == token2 {
				t.Error("expected different tokens, got identical")
			}

			// Tokens should not be empty
			if token1 == "" || token2 == "" {
				t.Error("expected non-empty tokens")
			}
		})
	}
}

func TestMemoryTokenStore_TokenWithoutRefreshToken(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	// Test storing a token without a refresh token
	tokenInfo := &TokenInfo{
		AccessToken: "access-only-token",
		RefreshToken: "", // No refresh token
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		CreatedAt:   time.Now(),
	}

	err := store.StoreToken(tokenInfo)
	if err != nil {
		t.Fatalf("StoreToken failed: %v", err)
	}

	// Should be able to get by access token
	retrieved, err := store.GetTokenByAccess("access-only-token")
	if err != nil {
		t.Fatalf("GetTokenByAccess failed: %v", err)
	}

	if retrieved.AccessToken != "access-only-token" {
		t.Errorf("expected access token %q, got %q", "access-only-token", retrieved.AccessToken)
	}

	// Deleting should not cause issues
	err = store.DeleteToken("access-only-token")
	if err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}
}
