package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestIntegration_MetadataAndMiddleware401_Consistency verifies that the
// authorization_endpoint and token_endpoint returned in 401 responses match
// those in the metadata endpoint response.
func TestIntegration_MetadataAndMiddleware401_Consistency(t *testing.T) {
	baseURL := "https://mcp.gtmeditor.com"
	store := newMockTokenStore()
	logger := testLogger()

	// Both use nil resolver — static URLs
	metaHandler := MetadataHandler(baseURL, nil)
	mw := Middleware(store, nil, logger, baseURL, 1*time.Hour, nil, nil, "", 7*24*time.Hour)
	protectedHandler := mw(dummyHandler)

	// Fetch metadata
	metaReq := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	metaW := httptest.NewRecorder()
	metaHandler.ServeHTTP(metaW, metaReq)

	var meta OAuthMetadata
	json.NewDecoder(metaW.Body).Decode(&meta)

	// Make unauthenticated request to get 401
	authReq := httptest.NewRequest(http.MethodGet, "/", nil)
	authW := httptest.NewRecorder()
	protectedHandler.ServeHTTP(authW, authReq)

	if authW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", authW.Code)
	}

	var errResp map[string]string
	json.NewDecoder(authW.Body).Decode(&errResp)

	// The 401 response endpoints must match the metadata
	if errResp["authorization_endpoint"] != meta.AuthorizationEndpoint {
		t.Errorf("401 authorization_endpoint %q != metadata %q",
			errResp["authorization_endpoint"], meta.AuthorizationEndpoint)
	}
	if errResp["token_endpoint"] != meta.TokenEndpoint {
		t.Errorf("401 token_endpoint %q != metadata %q",
			errResp["token_endpoint"], meta.TokenEndpoint)
	}
}

// TestIntegration_MetadataAndMiddleware401_ConsistencyWithResolver verifies
// consistency when using a URLResolver with an allowed host.
func TestIntegration_MetadataAndMiddleware401_ConsistencyWithResolver(t *testing.T) {
	baseURL := "https://mcp.gtmeditor.com"
	resolver := NewURLResolver(baseURL, []string{"gtm-mcp:8080"})
	store := newMockTokenStore()
	logger := testLogger()

	metaHandler := MetadataHandler(baseURL, resolver)
	resHandler := ProtectedResourceMetadataHandler(baseURL, baseURL, resolver)
	mw := Middleware(store, nil, logger, baseURL, 1*time.Hour, resolver, nil, "", 7*24*time.Hour)
	protectedHandler := mw(dummyHandler)

	// Test with the allowed Docker host
	host := "gtm-mcp:8080"

	// Fetch auth server metadata
	metaReq := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	metaReq.Host = host
	metaW := httptest.NewRecorder()
	metaHandler.ServeHTTP(metaW, metaReq)

	var meta OAuthMetadata
	json.NewDecoder(metaW.Body).Decode(&meta)

	if meta.Issuer != "http://gtm-mcp:8080" {
		t.Errorf("issuer = %q, want http://gtm-mcp:8080", meta.Issuer)
	}

	// Fetch protected resource metadata
	resReq := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	resReq.Host = host
	resW := httptest.NewRecorder()
	resHandler.ServeHTTP(resW, resReq)

	var resMeta ProtectedResourceMetadata
	json.NewDecoder(resW.Body).Decode(&resMeta)

	// Resource should have trailing slash, auth server should not
	if resMeta.Resource != "http://gtm-mcp:8080/" {
		t.Errorf("resource = %q, want http://gtm-mcp:8080/", resMeta.Resource)
	}
	if len(resMeta.AuthorizationServers) != 1 || resMeta.AuthorizationServers[0] != "http://gtm-mcp:8080" {
		t.Errorf("authorization_servers = %v, want [http://gtm-mcp:8080]", resMeta.AuthorizationServers)
	}

	// Issuer must match authorization_servers[0]
	if meta.Issuer != resMeta.AuthorizationServers[0] {
		t.Errorf("issuer %q != authorization_servers[0] %q", meta.Issuer, resMeta.AuthorizationServers[0])
	}

	// Make unauthenticated request with same host to get 401
	authReq := httptest.NewRequest(http.MethodGet, "/", nil)
	authReq.Host = host
	authW := httptest.NewRecorder()
	protectedHandler.ServeHTTP(authW, authReq)

	var errResp map[string]string
	json.NewDecoder(authW.Body).Decode(&errResp)

	// 401 endpoints must match metadata endpoints
	if errResp["authorization_endpoint"] != meta.AuthorizationEndpoint {
		t.Errorf("401 authorization_endpoint %q != metadata %q",
			errResp["authorization_endpoint"], meta.AuthorizationEndpoint)
	}
	if errResp["token_endpoint"] != meta.TokenEndpoint {
		t.Errorf("401 token_endpoint %q != metadata %q",
			errResp["token_endpoint"], meta.TokenEndpoint)
	}

	// WWW-Authenticate resource_metadata URL should also use the resolved host
	wwwAuth := authW.Header().Get("WWW-Authenticate")
	expectedResourceMeta := `resource_metadata="http://gtm-mcp:8080/.well-known/oauth-protected-resource"`
	if !containsSubstr(wwwAuth, expectedResourceMeta) {
		t.Errorf("WWW-Authenticate = %q, want it to contain %q", wwwAuth, expectedResourceMeta)
	}
}

// TestIntegration_UntrustedHostDoesNotLeakIntoResponses ensures that an
// untrusted Host header never appears in any response.
func TestIntegration_UntrustedHostDoesNotLeakIntoResponses(t *testing.T) {
	baseURL := "https://mcp.gtmeditor.com"
	resolver := NewURLResolver(baseURL, nil)
	store := newMockTokenStore()
	logger := testLogger()

	handlers := map[string]http.Handler{
		"metadata":           MetadataHandler(baseURL, resolver),
		"protected_resource": ProtectedResourceMetadataHandler(baseURL, baseURL, resolver),
		"middleware_401":     Middleware(store, nil, logger, baseURL, 1*time.Hour, resolver, nil, "", 7*24*time.Hour)(dummyHandler),
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = "evil.com"
			req.Header.Set("X-Forwarded-Proto", "https")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			body := w.Body.String()
			if containsSubstr(body, "evil.com") {
				t.Errorf("response body contains untrusted host 'evil.com': %s", body)
			}

			// Check all headers too
			for key, values := range w.Header() {
				for _, v := range values {
					if containsSubstr(v, "evil.com") {
						t.Errorf("response header %s contains untrusted host: %s", key, v)
					}
				}
			}
		})
	}
}

// TestIntegration_ResourceMetadata_GeminiCLICompatibility simulates the exact
// validation Gemini CLI performs: new URL(serverUrl).pathname produces "/" for
// root URLs, so the expected resource is scheme + "://" + host + "/".
func TestIntegration_ResourceMetadata_GeminiCLICompatibility(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		wantResource string
	}{
		{
			name:         "root URL without slash",
			baseURL:      "https://mcp.gtmeditor.com",
			wantResource: "https://mcp.gtmeditor.com/",
		},
		{
			name:         "root URL with slash",
			baseURL:      "https://mcp.gtmeditor.com/",
			wantResource: "https://mcp.gtmeditor.com/",
		},
		{
			name:         "localhost with port",
			baseURL:      "http://localhost:8080",
			wantResource: "http://localhost:8080/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ProtectedResourceMetadataHandler(tt.baseURL, tt.baseURL, nil)

			req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var meta ProtectedResourceMetadata
			json.NewDecoder(w.Body).Decode(&meta)

			if meta.Resource != tt.wantResource {
				t.Errorf("resource = %q, want %q", meta.Resource, tt.wantResource)
			}
		})
	}
}
