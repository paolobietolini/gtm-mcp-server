package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestIsValidRedirectURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		// Valid Claude.ai URIs
		{
			name:     "claude.ai with correct path",
			uri:      "https://claude.ai/api/mcp/auth_callback",
			expected: true,
		},
		{
			name:     "claude.ai with correct path and query params",
			uri:      "https://claude.ai/api/mcp/auth_callback?foo=bar",
			expected: true,
		},
		{
			name:     "claude.ai with wrong path",
			uri:      "https://claude.ai/wrong/path",
			expected: false,
		},
		{
			name:     "claude.ai with http instead of https",
			uri:      "http://claude.ai/api/mcp/auth_callback",
			expected: false,
		},
		// Valid claude.com URIs
		{
			name:     "claude.com with correct path",
			uri:      "https://claude.com/api/mcp/auth_callback",
			expected: true,
		},
		// Valid ChatGPT URIs
		{
			name:     "chatgpt.com with correct path",
			uri:      "https://chatgpt.com/connector_platform_oauth_redirect",
			expected: true,
		},
		{
			name:     "chatgpt.com with wrong path",
			uri:      "https://chatgpt.com/wrong/path",
			expected: false,
		},
		// Valid OpenAI platform URIs
		{
			name:     "platform.openai.com with correct path",
			uri:      "https://platform.openai.com/apps-manage/oauth",
			expected: true,
		},
		{
			name:     "platform.openai.com with wrong path",
			uri:      "https://platform.openai.com/wrong",
			expected: false,
		},
		// Localhost URIs (development)
		{
			name:     "localhost with http",
			uri:      "http://localhost:8080/callback",
			expected: true,
		},
		{
			name:     "localhost with https",
			uri:      "https://localhost:8080/callback",
			expected: true,
		},
		{
			name:     "127.0.0.1 with http",
			uri:      "http://127.0.0.1:8080/callback",
			expected: true,
		},
		{
			name:     "127.0.0.1 with https",
			uri:      "https://127.0.0.1:8080/callback",
			expected: true,
		},
		// Invalid URIs - subdomain attacks
		{
			name:     "subdomain attack - claude.ai.evil.com",
			uri:      "https://claude.ai.evil.com/api/mcp/auth_callback",
			expected: false,
		},
		{
			name:     "subdomain attack - evil.claude.ai",
			uri:      "https://evil.claude.ai/api/mcp/auth_callback",
			expected: false,
		},
		{
			name:     "subdomain attack - localhost.evil.com",
			uri:      "http://localhost.evil.com/callback",
			expected: false,
		},
		// Invalid URIs - malformed
		{
			name:     "empty URI",
			uri:      "",
			expected: false,
		},
		{
			name:     "invalid URL format",
			uri:      "not-a-url",
			expected: false,
		},
		{
			name:     "missing scheme",
			uri:      "//claude.ai/api/mcp/auth_callback",
			expected: false,
		},
		{
			name:     "missing host",
			uri:      "https:///callback",
			expected: false,
		},
		// Invalid URIs - unknown domains
		{
			name:     "unknown domain",
			uri:      "https://evil.com/callback",
			expected: false,
		},
		{
			name:     "data URI",
			uri:      "data:text/html,<script>alert('xss')</script>",
			expected: false,
		},
		{
			name:     "javascript URI",
			uri:      "javascript:alert('xss')",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidRedirectURI(tt.uri)
			if result != tt.expected {
				t.Errorf("isValidRedirectURI(%q) = %v, expected %v", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestServer_TokenError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := &Server{logger: logger}

	tests := []struct {
		name           string
		errCode        string
		errDesc        string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "invalid_grant",
			errCode:        "invalid_grant",
			errDesc:        "Invalid refresh token",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid_grant","error_description":"Invalid refresh token"}`,
		},
		{
			name:           "invalid_request",
			errCode:        "invalid_request",
			errDesc:        "Missing code",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid_request","error_description":"Missing code"}`,
		},
		{
			name:           "unsupported_grant_type",
			errCode:        "unsupported_grant_type",
			errDesc:        "Unsupported grant type",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"unsupported_grant_type","error_description":"Unsupported grant type"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			server.tokenError(w, tt.errCode, tt.errDesc)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			body := strings.TrimSpace(w.Body.String())
			if body != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, body)
			}
		})
	}
}

func TestServer_AuthorizeHandler_MethodNotAllowed(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	req := httptest.NewRequest(http.MethodPost, "/authorize", nil)
	w := httptest.NewRecorder()

	server.AuthorizeHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestServer_AuthorizeHandler_InvalidResponseType(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	req := httptest.NewRequest(http.MethodGet, "/authorize?response_type=token&state=test", nil)
	w := httptest.NewRecorder()

	server.AuthorizeHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "unsupported_response_type") {
		t.Error("expected unsupported_response_type error")
	}
}

func TestServer_AuthorizeHandler_MissingState(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	req := httptest.NewRequest(http.MethodGet, "/authorize?response_type=code", nil)
	w := httptest.NewRecorder()

	server.AuthorizeHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "invalid_request") {
		t.Error("expected invalid_request error")
	}
}

func TestServer_AuthorizeHandler_InvalidRedirectURI(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	tests := []struct {
		name        string
		redirectURI string
	}{
		{
			name:        "evil domain",
			redirectURI: "https://evil.com/callback",
		},
		{
			name:        "subdomain attack",
			redirectURI: "https://claude.ai.evil.com/api/mcp/auth_callback",
		},
		{
			name:        "wrong scheme for claude.ai",
			redirectURI: "http://claude.ai/api/mcp/auth_callback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("response_type", "code")
			params.Set("state", "test-state")
			params.Set("redirect_uri", tt.redirectURI)
			params.Set("code_challenge", "test-challenge")
			params.Set("code_challenge_method", "S256")

			req := httptest.NewRequest(http.MethodGet, "/authorize?"+params.Encode(), nil)
			w := httptest.NewRecorder()

			server.AuthorizeHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			if !strings.Contains(w.Body.String(), "Invalid redirect_uri") {
				t.Error("expected Invalid redirect_uri error")
			}
		})
	}
}

func TestServer_AuthorizeHandler_MissingPKCE(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	tests := []struct {
		name                string
		codeChallenge       string
		codeChallengeMethod string
	}{
		{
			name:                "missing both",
			codeChallenge:       "",
			codeChallengeMethod: "",
		},
		{
			name:                "missing challenge",
			codeChallenge:       "",
			codeChallengeMethod: "S256",
		},
		{
			name:                "wrong method",
			codeChallenge:       "test",
			codeChallengeMethod: "plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("response_type", "code")
			params.Set("state", "test-state")
			params.Set("redirect_uri", "http://localhost:8080/callback")
			if tt.codeChallenge != "" {
				params.Set("code_challenge", tt.codeChallenge)
			}
			if tt.codeChallengeMethod != "" {
				params.Set("code_challenge_method", tt.codeChallengeMethod)
			}

			req := httptest.NewRequest(http.MethodGet, "/authorize?"+params.Encode(), nil)
			w := httptest.NewRecorder()

			server.AuthorizeHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			if !strings.Contains(w.Body.String(), "PKCE") {
				t.Error("expected PKCE error")
			}
		})
	}
}

func TestServer_AuthorizeHandler_RegisteredClientValidation(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	// Register a client with specific redirect URIs
	clientInfo := &ClientInfo{
		ClientID:     "registered-client",
		RedirectURIs: []string{"https://example.com/callback"},
		CreatedAt:    time.Now(),
	}
	store.StoreClient(clientInfo)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	// Test that unregistered redirect URI is rejected for a registered client
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("state", "test-state")
	params.Set("client_id", "registered-client")
	params.Set("redirect_uri", "https://evil.com/callback")
	params.Set("code_challenge", "test-challenge")
	params.Set("code_challenge_method", "S256")

	req := httptest.NewRequest(http.MethodGet, "/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	server.AuthorizeHandler(w, req)

	if !strings.Contains(w.Body.String(), "redirect_uri does not match") {
		t.Error("expected redirect_uri validation error for unregistered URI")
	}
}

func TestPKCEVerification(t *testing.T) {
	// Test PKCE challenge/verifier validation logic
	tests := []struct {
		name         string
		verifier     string
		challenge    string
		shouldMatch  bool
	}{
		{
			name:         "valid match",
			verifier:     "test-verifier-123",
			challenge:    "", // Will be calculated
			shouldMatch:  true,
		},
		{
			name:         "invalid match",
			verifier:     "test-verifier-123",
			challenge:    "wrong-challenge",
			shouldMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate the correct challenge
			h := sha256.Sum256([]byte(tt.verifier))
			correctChallenge := base64.RawURLEncoding.EncodeToString(h[:])

			challenge := tt.challenge
			if tt.shouldMatch {
				challenge = correctChallenge
			}

			// Verify
			h2 := sha256.Sum256([]byte(tt.verifier))
			calculatedChallenge := base64.RawURLEncoding.EncodeToString(h2[:])

			matched := calculatedChallenge == challenge
			if matched != tt.shouldMatch {
				t.Errorf("expected match=%v, got match=%v", tt.shouldMatch, matched)
			}
		})
	}
}

func TestServer_TokenHandler_MethodNotAllowed(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	w := httptest.NewRecorder()

	server.TokenHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestServer_TokenHandler_UnsupportedGrantType(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	server.TokenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "unsupported_grant_type") {
		t.Error("expected unsupported_grant_type error")
	}
}

func TestServer_HandleAuthorizationCodeGrant_MissingCode(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	// Missing code

	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	server.TokenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Missing code") {
		t.Error("expected Missing code error")
	}
}

func TestServer_HandleAuthorizationCodeGrant_InvalidCode(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", "invalid-code")
	form.Set("code_verifier", "test-verifier")

	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	server.TokenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "invalid_grant") {
		t.Error("expected invalid_grant error")
	}
}

func TestServer_HandleAuthorizationCodeGrant_MissingCodeVerifier(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	// Store a valid code state
	codeState := &AuthState{
		State:        "valid-code",
		CodeVerifier: "test-challenge",
		CreatedAt:    time.Now(),
	}
	store.StoreState(codeState)

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", "valid-code")
	// Missing code_verifier

	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	server.TokenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Missing code_verifier") {
		t.Error("expected Missing code_verifier error")
	}
}

func TestServer_HandleRefreshTokenGrant_MissingRefreshToken(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	// Missing refresh_token

	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	server.TokenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Missing refresh_token") {
		t.Error("expected Missing refresh_token error")
	}
}

func TestServer_HandleRefreshTokenGrant_InvalidRefreshToken(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", "invalid-refresh-token")

	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	server.TokenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Invalid refresh token") {
		t.Error("expected Invalid refresh token error")
	}
}

func TestServer_CallbackHandler_MethodNotAllowed(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	req := httptest.NewRequest(http.MethodPost, "/oauth/callback", nil)
	w := httptest.NewRecorder()

	server.CallbackHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestServer_CallbackHandler_GoogleError(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	params := url.Values{}
	params.Set("error", "access_denied")
	params.Set("error_description", "User denied access")

	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	server.CallbackHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "access_denied") {
		t.Error("expected access_denied error")
	}
}

func TestServer_CallbackHandler_MissingCodeOrState(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	tests := []struct {
		name   string
		params url.Values
	}{
		{
			name:   "missing code",
			params: url.Values{"state": {"test"}},
		},
		{
			name:   "missing state",
			params: url.Values{"code": {"test"}},
		},
		{
			name:   "missing both",
			params: url.Values{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+tt.params.Encode(), nil)
			w := httptest.NewRecorder()

			server.CallbackHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			if !strings.Contains(w.Body.String(), "Missing code or state") {
				t.Error("expected Missing code or state error")
			}
		})
	}
}

func TestServer_CallbackHandler_InvalidStateFormat(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	params := url.Values{}
	params.Set("code", "test-code")
	params.Set("state", "invalid-state-no-pipe")

	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	server.CallbackHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Invalid state format") {
		t.Error("expected Invalid state format error")
	}
}

func TestServer_CallbackHandler_ExpiredState(t *testing.T) {
	store := NewMemoryTokenStore()
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := NewServer("http://localhost:8080", nil, store, logger)

	params := url.Values{}
	params.Set("code", "test-code")
	params.Set("state", "google-state|claude-state")

	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	server.CallbackHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Invalid or expired state") {
		t.Error("expected Invalid or expired state error")
	}
}
