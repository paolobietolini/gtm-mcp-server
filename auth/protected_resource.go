package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
)

// ProtectedResourceMetadata represents RFC 9728 OAuth 2.0 Protected Resource Metadata
type ProtectedResourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	ScopesSupported        []string `json:"scopes_supported,omitempty"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
}

// NewProtectedResourceMetadata creates metadata for the protected resource.
// The resource identifier is normalized per RFC 9728: root URLs get a trailing
// slash to match what clients (e.g. Gemini CLI) compute via new URL(serverUrl).pathname.
func NewProtectedResourceMetadata(baseURL, resourceURL string) *ProtectedResourceMetadata {
	return &ProtectedResourceMetadata{
		Resource:               normalizeResourceURL(resourceURL),
		AuthorizationServers:   []string{baseURL},
		ScopesSupported:        GoogleScopes,
		BearerMethodsSupported: []string{"header"},
	}
}

// normalizeResourceURL ensures the resource identifier has a path component.
// Per RFC 9728, clients like Gemini CLI construct the expected resource as
// scheme + "://" + host + pathname, where pathname is "/" for root URLs.
// Without this, "https://example.com" != "https://example.com/" causes a mismatch.
func normalizeResourceURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String()
}

// ProtectedResourceMetadataHandler returns HTTP handler for /.well-known/oauth-protected-resource.
// If resolver is non-nil, the base URL is resolved dynamically per-request
// (validated against allowed hosts). Otherwise, baseURL/resourceURL are used statically.
func ProtectedResourceMetadataHandler(baseURL, resourceURL string, resolver *URLResolver) http.HandlerFunc {
	// Pre-compute for the static case
	staticMetadata := NewProtectedResourceMetadata(baseURL, resourceURL)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metadata := staticMetadata
		if resolver != nil {
			if resolved := resolver.Resolve(r); resolved != baseURL {
				metadata = NewProtectedResourceMetadata(resolved, resolved)
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
