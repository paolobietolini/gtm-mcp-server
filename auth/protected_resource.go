package auth

import (
	"encoding/json"
	"net/http"
)

// ProtectedResourceMetadata represents RFC 9728 OAuth 2.0 Protected Resource Metadata
type ProtectedResourceMetadata struct {
	Resource                 string   `json:"resource"`
	AuthorizationServers     []string `json:"authorization_servers"`
	ScopesSupported          []string `json:"scopes_supported,omitempty"`
	BearerMethodsSupported   []string `json:"bearer_methods_supported"`
}

// NewProtectedResourceMetadata creates metadata for the protected resource
func NewProtectedResourceMetadata(baseURL, resourceURL string) *ProtectedResourceMetadata {
	return &ProtectedResourceMetadata{
		Resource:               resourceURL,
		AuthorizationServers:   []string{baseURL},
		ScopesSupported:        GoogleScopes,
		BearerMethodsSupported: []string{"header"},
	}
}

// ProtectedResourceMetadataHandler returns HTTP handler for /.well-known/oauth-protected-resource
func ProtectedResourceMetadataHandler(baseURL, resourceURL string) http.HandlerFunc {
	metadata := NewProtectedResourceMetadata(baseURL, resourceURL)

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
