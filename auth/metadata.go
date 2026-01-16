package auth

import (
	"encoding/json"
	"net/http"
)

// OAuthMetadata represents RFC 8414 OAuth 2.0 Authorization Server Metadata.
type OAuthMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                   []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

// NewOAuthMetadata creates metadata for the given base URL.
func NewOAuthMetadata(baseURL string) *OAuthMetadata {
	return &OAuthMetadata{
		Issuer:                baseURL,
		AuthorizationEndpoint: baseURL + "/authorize",
		TokenEndpoint:         baseURL + "/token",
		RegistrationEndpoint:  baseURL + "/register",
		ScopesSupported: []string{
			"https://www.googleapis.com/auth/tagmanager.edit.containers",
			"https://www.googleapis.com/auth/tagmanager.readonly",
		},
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_post", "none"},
		CodeChallengeMethodsSupported:     []string{"S256"},
	}
}

// MetadataHandler returns an HTTP handler for /.well-known/oauth-authorization-server.
func MetadataHandler(baseURL string) http.HandlerFunc {
	metadata := NewOAuthMetadata(baseURL)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "public, max-age=3600")

		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			http.Error(w, "Failed to encode metadata", http.StatusInternalServerError)
			return
		}
	}
}
