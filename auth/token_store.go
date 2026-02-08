package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

var (
	ErrTokenNotFound  = errors.New("token not found")
	ErrTokenExpired   = errors.New("token expired")
	ErrInvalidState   = errors.New("invalid state")
	ErrClientNotFound = errors.New("client not found")
)

// TokenInfo holds information about an issued token and the associated Google tokens.
type TokenInfo struct {
	// Our token (issued to Claude)
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time

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
	Resource     string // RFC 9728: resource parameter for audience binding
	CreatedAt    time.Time
}

// ClientInfo holds information about a registered OAuth client (RFC 7591).
type ClientInfo struct {
	ClientID                string
	RedirectURIs            []string
	ClientName              string
	GrantTypes              []string
	ResponseTypes           []string
	TokenEndpointAuthMethod string
	CreatedAt               time.Time
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
	ConsumeState(stateValue string) (*AuthState, error)
	DeleteState(stateValue string) error

	// Client registration operations (RFC 7591)
	StoreClient(client *ClientInfo) error
	GetClient(clientID string) (*ClientInfo, error)
	DeleteClient(clientID string) error
}

// MemoryTokenStore is an in-memory implementation of TokenStore.
type MemoryTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*TokenInfo  // keyed by access token
	states map[string]*AuthState  // keyed by state value
	clients map[string]*ClientInfo // keyed by client_id

	// Secondary index for refresh token lookup
	refreshIndex map[string]string // refresh token -> access token

	// Cancellation for cleanup goroutine
	cancel context.CancelFunc
}

// NewMemoryTokenStore creates a new in-memory token store.
func NewMemoryTokenStore() *MemoryTokenStore {
	ctx, cancel := context.WithCancel(context.Background())
	store := &MemoryTokenStore{
		tokens:       make(map[string]*TokenInfo),
		states:       make(map[string]*AuthState),
		clients:      make(map[string]*ClientInfo),
		refreshIndex: make(map[string]string),
		cancel:       cancel,
	}

	// Start cleanup goroutine
	go store.cleanup(ctx)

	return store
}

// Close stops the cleanup goroutine and releases resources.
func (s *MemoryTokenStore) Close() error {
	s.cancel()
	return nil
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

	if !info.RefreshExpiresAt.IsZero() && time.Now().After(info.RefreshExpiresAt) {
		return nil, ErrTokenExpired
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
	if googleToken == nil {
		return errors.New("googleToken cannot be nil")
	}

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

// ConsumeState atomically gets and deletes a state, making auth codes single-use.
func (s *MemoryTokenStore) ConsumeState(stateValue string) (*AuthState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.states[stateValue]
	if !ok {
		return nil, ErrInvalidState
	}

	delete(s.states, stateValue)

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

// StoreClient stores a registered OAuth client.
func (s *MemoryTokenStore) StoreClient(client *ClientInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[client.ClientID] = client
	return nil
}

// GetClient retrieves a registered OAuth client by client_id.
func (s *MemoryTokenStore) GetClient(clientID string) (*ClientInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[clientID]
	if !ok {
		return nil, ErrClientNotFound
	}

	return client, nil
}

// DeleteClient removes a registered OAuth client.
func (s *MemoryTokenStore) DeleteClient(clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, clientID)
	return nil
}

// cleanup periodically removes expired tokens, states, and stale clients.
func (s *MemoryTokenStore) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	const maxClients = 1000

	for {
		select {
		case <-ticker.C:
			now := time.Now()

			// Delete directly under write lock to avoid race conditions
			// between collecting expired keys and deleting them.
			s.mu.Lock()
			for accessToken, info := range s.tokens {
				accessExpired := now.After(info.ExpiresAt.Add(1 * time.Hour))
				refreshExpired := !info.RefreshExpiresAt.IsZero() && now.After(info.RefreshExpiresAt)
				if accessExpired || refreshExpired {
					delete(s.refreshIndex, info.RefreshToken)
					delete(s.tokens, accessToken)
				}
			}
			for stateValue, state := range s.states {
				if now.Sub(state.CreatedAt) > 10*time.Minute {
					delete(s.states, stateValue)
				}
			}
			// Evict oldest clients if over limit
			if len(s.clients) > maxClients {
				var oldest string
				var oldestTime time.Time
				for id, client := range s.clients {
					if oldest == "" || client.CreatedAt.Before(oldestTime) {
						oldest = id
						oldestTime = client.CreatedAt
					}
				}
				if oldest != "" {
					delete(s.clients, oldest)
				}
			}
			s.mu.Unlock()

		case <-ctx.Done():
			return
		}
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
