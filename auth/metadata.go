package auth

import (
	"bytes"
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
	ClientIDMetadataDocumentSupported bool     `json:"client_id_metadata_document_supported"`
}

// NewOAuthMetadata creates metadata for the given base URL.
func NewOAuthMetadata(baseURL string) *OAuthMetadata {
	return &OAuthMetadata{
		Issuer:                            baseURL,
		AuthorizationEndpoint:             baseURL + "/authorize",
		TokenEndpoint:                     baseURL + "/token",
		RegistrationEndpoint:              baseURL + "/register",
		ScopesSupported:                   GoogleScopes,
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_post", "none"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		ClientIDMetadataDocumentSupported: true,
	}
}

// MetadataHandler returns an HTTP handler for /.well-known/oauth-authorization-server.
// If resolver is non-nil, the base URL is resolved dynamically per-request
// (validated against allowed hosts). Otherwise, baseURL is used statically.
func MetadataHandler(baseURL string, resolver *URLResolver) http.HandlerFunc {
	// Pre-compute for the static case
	staticMetadata := NewOAuthMetadata(baseURL)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metadata := staticMetadata
		if resolver != nil {
			if resolved := resolver.Resolve(r); resolved != baseURL {
				metadata = NewOAuthMetadata(resolved)
			}
		}

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(metadata); err != nil {
			http.Error(w, "Failed to encode metadata", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(buf.Bytes())
	}
}
