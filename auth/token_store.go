package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
	ErrInvalidState  = errors.New("invalid state")
)

// TokenInfo holds information about an issued token and the associated Google tokens.
type TokenInfo struct {
	// Our token (issued to Claude)
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time

	// Google tokens (for calling GTM API)
	GoogleToken *oauth2.Token

	// Metadata
	ClientID  string
	CreatedAt time.Time
}

// AuthState holds temporary state during OAuth flow.
type AuthState struct {
	State        string
	CodeVerifier string
	RedirectURI  string
	ClientID     string
	CreatedAt    time.Time
}

// TokenStore defines the interface for token storage.
type TokenStore interface {
	// Token operations
	StoreToken(info *TokenInfo) error
	GetTokenByAccess(accessToken string) (*TokenInfo, error)
	GetTokenByRefresh(refreshToken string) (*TokenInfo, error)
	DeleteToken(accessToken string) error
	UpdateGoogleToken(accessToken string, googleToken *oauth2.Token) error

	// State operations (for OAuth flow)
	StoreState(state *AuthState) error
	GetState(stateValue string) (*AuthState, error)
	DeleteState(stateValue string) error
}

// MemoryTokenStore is an in-memory implementation of TokenStore.
type MemoryTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*TokenInfo  // keyed by access token
	states map[string]*AuthState  // keyed by state value

	// Secondary index for refresh token lookup
	refreshIndex map[string]string // refresh token -> access token
}

// NewMemoryTokenStore creates a new in-memory token store.
func NewMemoryTokenStore() *MemoryTokenStore {
	store := &MemoryTokenStore{
		tokens:       make(map[string]*TokenInfo),
		states:       make(map[string]*AuthState),
		refreshIndex: make(map[string]string),
	}

	// Start cleanup goroutine
	go store.cleanup()

	return store
}

func (s *MemoryTokenStore) StoreToken(info *TokenInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[info.AccessToken] = info
	if info.RefreshToken != "" {
		s.refreshIndex[info.RefreshToken] = info.AccessToken
	}

	return nil
}

func (s *MemoryTokenStore) GetTokenByAccess(accessToken string) (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.tokens[accessToken]
	if !ok {
		return nil, ErrTokenNotFound
	}

	if time.Now().After(info.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return info, nil
}

func (s *MemoryTokenStore) GetTokenByRefresh(refreshToken string) (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accessToken, ok := s.refreshIndex[refreshToken]
	if !ok {
		return nil, ErrTokenNotFound
	}

	info, ok := s.tokens[accessToken]
	if !ok {
		return nil, ErrTokenNotFound
	}

	return info, nil
}

func (s *MemoryTokenStore) DeleteToken(accessToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if info, ok := s.tokens[accessToken]; ok {
		delete(s.refreshIndex, info.RefreshToken)
	}
	delete(s.tokens, accessToken)

	return nil
}

func (s *MemoryTokenStore) UpdateGoogleToken(accessToken string, googleToken *oauth2.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, ok := s.tokens[accessToken]
	if !ok {
		return ErrTokenNotFound
	}

	info.GoogleToken = googleToken
	return nil
}

func (s *MemoryTokenStore) StoreState(state *AuthState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state.State] = state
	return nil
}

func (s *MemoryTokenStore) GetState(stateValue string) (*AuthState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.states[stateValue]
	if !ok {
		return nil, ErrInvalidState
	}

	// States expire after 10 minutes
	if time.Since(state.CreatedAt) > 10*time.Minute {
		return nil, ErrInvalidState
	}

	return state, nil
}

func (s *MemoryTokenStore) DeleteState(stateValue string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, stateValue)
	return nil
}

// cleanup periodically removes expired tokens and states.
func (s *MemoryTokenStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()

		now := time.Now()

		// Clean expired tokens (keep if refresh token might still work)
		for accessToken, info := range s.tokens {
			// Remove if both access and refresh are expired (24h grace for refresh)
			if now.After(info.ExpiresAt.Add(24 * time.Hour)) {
				delete(s.refreshIndex, info.RefreshToken)
				delete(s.tokens, accessToken)
			}
		}

		// Clean expired states
		for stateValue, state := range s.states {
			if now.Sub(state.CreatedAt) > 10*time.Minute {
				delete(s.states, stateValue)
			}
		}

		s.mu.Unlock()
	}
}

// GenerateToken creates a cryptographically secure random token.
func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
