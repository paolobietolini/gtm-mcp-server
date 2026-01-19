package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Server handles OAuth2 authorization endpoints.
type Server struct {
	baseURL        string
	google         *GoogleProvider
	store          TokenStore
	logger         *slog.Logger
	accessTokenTTL time.Duration
}

// NewServer creates a new OAuth server.
func NewServer(baseURL string, google *GoogleProvider, store TokenStore, logger *slog.Logger) *Server {
	return &Server{
		baseURL:        baseURL,
		google:         google,
		store:          store,
		logger:         logger,
		accessTokenTTL: 1 * time.Hour,
	}
}

// AuthorizeHandler handles GET /authorize - redirects to Google OAuth.
func (s *Server) AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse OAuth parameters from Claude
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	responseType := r.URL.Query().Get("response_type")
	state := r.URL.Query().Get("state")
	codeChallenge := r.URL.Query().Get("code_challenge")
	codeChallengeMethod := r.URL.Query().Get("code_challenge_method")

	// Validate required parameters
	if responseType != "code" {
		s.errorResponse(w, "unsupported_response_type", "Only 'code' response type is supported")
		return
	}

	if state == "" {
		s.errorResponse(w, "invalid_request", "State parameter is required")
		return
	}

	// Validate redirect URI
	// If client is registered via DCR, validate against their registered URIs
	// Otherwise fall back to known safe patterns
	if clientID != "" {
		if client, err := s.store.GetClient(clientID); err == nil {
			// Client is registered, validate against registered redirect_uris
			validRedirect := false
			for _, uri := range client.RedirectURIs {
				if uri == redirectURI {
					validRedirect = true
					break
				}
			}
			if !validRedirect {
				s.errorResponse(w, "invalid_request", "redirect_uri does not match registered URIs")
				return
			}
		} else {
			// Client not registered via DCR, fall back to default validation
			if !isValidRedirectURI(redirectURI) {
				s.errorResponse(w, "invalid_request", "Invalid redirect_uri")
				return
			}
		}
	} else if !isValidRedirectURI(redirectURI) {
		s.errorResponse(w, "invalid_request", "Invalid redirect_uri")
		return
	}

	// PKCE is required
	if codeChallenge == "" || codeChallengeMethod != "S256" {
		s.errorResponse(w, "invalid_request", "PKCE with S256 is required")
		return
	}

	// Generate our own state for Google OAuth
	googleState, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate state", "error", err)
		s.errorResponse(w, "server_error", "Internal server error")
		return
	}

	// Store the auth state for later verification
	authState := &AuthState{
		State:        googleState,
		CodeVerifier: codeChallenge, // Store the challenge, we'll verify later
		RedirectURI:  redirectURI,
		ClientID:     clientID,
		CreatedAt:    time.Now(),
	}

	// Also store Claude's state so we can pass it back
	authState.State = googleState + "|" + state

	if err := s.store.StoreState(authState); err != nil {
		s.logger.Error("failed to store state", "error", err)
		s.errorResponse(w, "server_error", "Internal server error")
		return
	}

	// Redirect to Google OAuth
	googleAuthURL := s.google.AuthCodeURL(authState.State)

	s.logger.Info("redirecting to Google OAuth",
		"client_id", clientID,
		"redirect_uri", redirectURI,
	)

	http.Redirect(w, r, googleAuthURL, http.StatusFound)
}

// CallbackHandler handles GET /oauth/callback - receives code from Google.
func (s *Server) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for errors from Google
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		errDesc := r.URL.Query().Get("error_description")
		s.logger.Error("Google OAuth error", "error", errCode, "description", errDesc)
		s.errorResponse(w, errCode, errDesc)
		return
	}

	code := r.URL.Query().Get("code")
	combinedState := r.URL.Query().Get("state")

	if code == "" || combinedState == "" {
		s.errorResponse(w, "invalid_request", "Missing code or state")
		return
	}

	// Split the combined state
	parts := strings.SplitN(combinedState, "|", 2)
	if len(parts) != 2 {
		s.errorResponse(w, "invalid_request", "Invalid state format")
		return
	}
	claudeState := parts[1]

	// Retrieve stored auth state
	authState, err := s.store.GetState(combinedState)
	if err != nil {
		s.logger.Error("failed to get state", "error", err)
		s.errorResponse(w, "invalid_request", "Invalid or expired state")
		return
	}

	// Clean up the state
	_ = s.store.DeleteState(combinedState)

	// Exchange code with Google
	googleToken, err := s.google.Exchange(r.Context(), code)
	if err != nil {
		s.logger.Error("failed to exchange code with Google", "error", err)
		s.errorResponse(w, "server_error", "Failed to exchange authorization code")
		return
	}

	// Generate our own authorization code to return to Claude
	ourCode, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate code", "error", err)
		s.errorResponse(w, "server_error", "Internal server error")
		return
	}

	// Store temporarily with the Google token (code is short-lived)
	tempToken := &TokenInfo{
		AccessToken:  ourCode, // Temporary: using code as key
		GoogleToken:  googleToken,
		ClientID:     authState.ClientID,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(5 * time.Minute), // Code expires in 5 min
	}

	// Store code verifier for PKCE verification
	codeState := &AuthState{
		State:        ourCode,
		CodeVerifier: authState.CodeVerifier,
		RedirectURI:  authState.RedirectURI,
		ClientID:     authState.ClientID,
		CreatedAt:    time.Now(),
	}

	if err := s.store.StoreState(codeState); err != nil {
		s.logger.Error("failed to store code state", "error", err)
		s.errorResponse(w, "server_error", "Internal server error")
		return
	}

	if err := s.store.StoreToken(tempToken); err != nil {
		s.logger.Error("failed to store temp token", "error", err)
		s.errorResponse(w, "server_error", "Internal server error")
		return
	}

	// Redirect back to Claude with our code
	redirectURL, _ := url.Parse(authState.RedirectURI)
	q := redirectURL.Query()
	q.Set("code", ourCode)
	q.Set("state", claudeState)
	redirectURL.RawQuery = q.Encode()

	s.logger.Info("OAuth callback successful, redirecting to Claude",
		"redirect_uri", authState.RedirectURI,
	)

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// TokenHandler handles POST /token - exchanges code for tokens.
func (s *Server) TokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.tokenError(w, "invalid_request", "Failed to parse request")
		return
	}

	grantType := r.FormValue("grant_type")

	switch grantType {
	case "authorization_code":
		s.handleAuthorizationCodeGrant(w, r)
	case "refresh_token":
		s.handleRefreshTokenGrant(w, r)
	default:
		s.tokenError(w, "unsupported_grant_type", "Unsupported grant type")
	}
}

func (s *Server) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	codeVerifier := r.FormValue("code_verifier")
	// clientID := r.FormValue("client_id")
	// redirectURI := r.FormValue("redirect_uri")

	if code == "" {
		s.tokenError(w, "invalid_request", "Missing code")
		return
	}

	// Get the stored code state
	codeState, err := s.store.GetState(code)
	if err != nil {
		s.logger.Error("failed to get code state", "error", err)
		s.tokenError(w, "invalid_grant", "Invalid or expired code")
		return
	}

	// Verify PKCE
	if codeVerifier == "" {
		s.tokenError(w, "invalid_request", "Missing code_verifier")
		return
	}

	// Verify: SHA256(code_verifier) == code_challenge
	h := sha256.Sum256([]byte(codeVerifier))
	calculatedChallenge := base64.RawURLEncoding.EncodeToString(h[:])

	if calculatedChallenge != codeState.CodeVerifier {
		s.logger.Error("PKCE verification failed",
			"expected", codeState.CodeVerifier,
			"got", calculatedChallenge,
		)
		s.tokenError(w, "invalid_grant", "PKCE verification failed")
		return
	}

	// Get the temporary token with Google credentials
	tempToken, err := s.store.GetTokenByAccess(code)
	if err != nil {
		s.logger.Error("failed to get temp token", "error", err)
		s.tokenError(w, "invalid_grant", "Invalid or expired code")
		return
	}

	// Clean up temporary storage
	_ = s.store.DeleteState(code)
	_ = s.store.DeleteToken(code)

	// Generate real tokens
	accessToken, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate access token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	refreshToken, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate refresh token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	// Create and store the real token
	tokenInfo := &TokenInfo{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.accessTokenTTL),
		GoogleToken:  tempToken.GoogleToken,
		ClientID:     codeState.ClientID,
		CreatedAt:    time.Now(),
	}

	if err := s.store.StoreToken(tokenInfo); err != nil {
		s.logger.Error("failed to store token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	s.logger.Info("issued access token", "client_id", codeState.ClientID)

	// Return token response
	s.tokenResponse(w, accessToken, refreshToken, int(s.accessTokenTTL.Seconds()))
}

func (s *Server) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.FormValue("refresh_token")

	if refreshToken == "" {
		s.tokenError(w, "invalid_request", "Missing refresh_token")
		return
	}

	// Get existing token info
	tokenInfo, err := s.store.GetTokenByRefresh(refreshToken)
	if err != nil {
		s.logger.Error("failed to get token by refresh", "error", err)
		s.tokenError(w, "invalid_grant", "Invalid refresh token")
		return
	}

	// Refresh the Google token if needed
	if tokenInfo.GoogleToken.Expiry.Before(time.Now()) {
		newGoogleToken, err := s.google.RefreshToken(r.Context(), tokenInfo.GoogleToken.RefreshToken)
		if err != nil {
			s.logger.Error("failed to refresh Google token", "error", err)
			s.tokenError(w, "invalid_grant", "Failed to refresh upstream token")
			return
		}
		tokenInfo.GoogleToken = newGoogleToken
	}

	// Generate new access token
	newAccessToken, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate access token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	// Delete old token
	_ = s.store.DeleteToken(tokenInfo.AccessToken)

	// Store new token (keep same refresh token)
	newTokenInfo := &TokenInfo{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.accessTokenTTL),
		GoogleToken:  tokenInfo.GoogleToken,
		ClientID:     tokenInfo.ClientID,
		CreatedAt:    time.Now(),
	}

	if err := s.store.StoreToken(newTokenInfo); err != nil {
		s.logger.Error("failed to store new token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	s.logger.Info("refreshed access token", "client_id", tokenInfo.ClientID)

	// Return token response (same refresh token)
	s.tokenResponse(w, newAccessToken, refreshToken, int(s.accessTokenTTL.Seconds()))
}

func (s *Server) tokenResponse(w http.ResponseWriter, accessToken, refreshToken string, expiresIn int) {
	resp := map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    expiresIn,
		"refresh_token": refreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) tokenError(w http.ResponseWriter, errCode, errDesc string) {
	resp := map[string]string{
		"error":             errCode,
		"error_description": errDesc,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) errorResponse(w http.ResponseWriter, errCode, errDesc string) {
	http.Error(w, fmt.Sprintf("%s: %s", errCode, errDesc), http.StatusBadRequest)
}

// isValidRedirectURI checks if the redirect URI is Claude's callback.
func isValidRedirectURI(uri string) bool {
	validURIs := []string{
		"https://claude.ai/api/mcp/auth_callback",
		"https://claude.com/api/mcp/auth_callback",
		// For local development
		"http://localhost",
		"http://127.0.0.1",
	}

	for _, valid := range validURIs {
		if strings.HasPrefix(uri, valid) {
			return true
		}
	}

	return false
}
