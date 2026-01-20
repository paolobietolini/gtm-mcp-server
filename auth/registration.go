package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientRegistrationRequest per RFC 7591
type ClientRegistrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
}

// ClientRegistrationResponse per RFC 7591
type ClientRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at"`
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name,omitempty"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// RegistrationHandler handles POST /register
func (s *Server) RegistrationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ClientRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.registrationError(w, "invalid_request", "Invalid JSON")
		return
	}

	// Validate redirect URIs per RFC 7591
	// DCR accepts any valid HTTPS URI (or localhost for development)
	if len(req.RedirectURIs) == 0 {
		s.registrationError(w, "invalid_redirect_uri", "At least one redirect_uri required")
		return
	}

	for _, uri := range req.RedirectURIs {
		if !isValidDCRRedirectURI(uri) {
			s.registrationError(w, "invalid_redirect_uri", "Invalid redirect_uri: "+uri)
			return
		}
	}

	// Genera client_id
	clientID, err := GenerateToken(16)
	if err != nil {
		s.logger.Error("failed to generate client_id", "error", err)
		s.registrationError(w, "server_error", "Internal server error")
		return
	}

	// Per public clients (MCP), non generiamo client_secret
	resp := ClientRegistrationResponse{
		ClientID:                clientID,
		ClientSecretExpiresAt:   0, // Non scade
		RedirectURIs:            req.RedirectURIs,
		ClientName:              req.ClientName,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none", // Public client
	}

	// Store the registered client
	clientInfo := &ClientInfo{
		ClientID:                clientID,
		RedirectURIs:            req.RedirectURIs,
		ClientName:              req.ClientName,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
		CreatedAt:               time.Now(),
	}

	if err := s.store.StoreClient(clientInfo); err != nil {
		s.logger.Error("failed to store registered client", "error", err)
		s.registrationError(w, "server_error", "Internal server error")
		return
	}

	s.logger.Info("client registered", "client_id", clientID, "client_name", req.ClientName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) registrationError(w http.ResponseWriter, errCode, errDesc string) {
	resp := map[string]string{
		"error":             errCode,
		"error_description": errDesc,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(resp)
}

// isValidDCRRedirectURI validates redirect URIs for Dynamic Client Registration.
// Per RFC 7591, we accept any valid HTTPS URI, plus localhost for development.
// This is more permissive than the hardcoded list used for non-DCR clients.
func isValidDCRRedirectURI(uri string) bool {
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}

	// Must have a scheme and host
	if parsed.Scheme == "" || parsed.Host == "" {
		return false
	}

	// Allow localhost for development (http or https)
	host := strings.Split(parsed.Host, ":")[0] // Remove port if present
	if host == "localhost" || host == "127.0.0.1" {
		return parsed.Scheme == "http" || parsed.Scheme == "https"
	}

	// For all other hosts, require HTTPS
	return parsed.Scheme == "https"
}
