package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestIsClientIDMetadataURL(t *testing.T) {
	tests := []struct {
		clientID string
		want     bool
	}{
		{"https://app.example.com/oauth/client-metadata.json", true},
		{"https://claude.ai/.well-known/client-metadata", true},
		{"http://app.example.com/client.json", false},     // not https
		{"https://app.example.com", false},                // no path component
		{"https://app.example.com/", false},               // empty path
		{"https://app.example.com/meta.json#frag", false}, // fragment
		{"client-abc123", false},                          // opaque DCR-style id
		{"", false},
	}

	for _, tt := range tests {
		if got := IsClientIDMetadataURL(tt.clientID); got != tt.want {
			t.Errorf("IsClientIDMetadataURL(%q) = %v, want %v", tt.clientID, got, tt.want)
		}
	}
}

// newCIMDTestServer serves a metadata document over TLS and returns a fetcher
// wired to trust it, plus the metadata URL.
func newCIMDTestServer(t *testing.T, path string, handler http.HandlerFunc) (*CIMDFetcher, string, func()) {
	t.Helper()
	ts := httptest.NewTLSServer(handler)
	fetcher := NewCIMDFetcher()
	fetcher.HTTPClient = ts.Client()
	fetcher.AllowPrivateHosts = true // httptest listens on 127.0.0.1
	return fetcher, ts.URL + path, ts.Close
}

func TestCIMDFetcher_Fetch_Valid(t *testing.T) {
	var metadataURL string
	fetcher, u, cleanup := newCIMDTestServer(t, "/client.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"client_id": %q,
			"client_name": "Test MCP Client",
			"redirect_uris": ["http://localhost:3000/callback"]
		}`, metadataURL)
	})
	defer cleanup()
	metadataURL = u

	client, err := fetcher.Fetch(context.Background(), metadataURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.ClientID != metadataURL {
		t.Errorf("ClientID = %q, want %q", client.ClientID, metadataURL)
	}
	if client.ClientName != "Test MCP Client" {
		t.Errorf("ClientName = %q", client.ClientName)
	}
	if len(client.RedirectURIs) != 1 || client.RedirectURIs[0] != "http://localhost:3000/callback" {
		t.Errorf("RedirectURIs = %v", client.RedirectURIs)
	}
}

func TestCIMDFetcher_Fetch_ClientIDMismatch(t *testing.T) {
	fetcher, u, cleanup := newCIMDTestServer(t, "/client.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"client_id":"https://evil.example.com/other.json","client_name":"x","redirect_uris":["http://localhost:3000/cb"]}`))
	})
	defer cleanup()

	if _, err := fetcher.Fetch(context.Background(), u); err == nil {
		t.Error("expected error when document client_id does not match URL")
	}
}

func TestCIMDFetcher_Fetch_MissingRequiredFields(t *testing.T) {
	var metadataURL string
	fetcher, u, cleanup := newCIMDTestServer(t, "/client.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"client_id": %q, "client_name": "x"}`, metadataURL) // no redirect_uris
	})
	defer cleanup()
	metadataURL = u

	if _, err := fetcher.Fetch(context.Background(), metadataURL); err == nil {
		t.Error("expected error when redirect_uris missing")
	}
}

func TestCIMDFetcher_Fetch_RejectsPrivateHostsByDefault(t *testing.T) {
	fetcher := NewCIMDFetcher()
	for _, u := range []string{
		"https://127.0.0.1/client.json",
		"https://localhost/client.json",
		"https://10.0.0.5/client.json",
		"https://192.168.1.1/client.json",
		"https://[::1]/client.json",
	} {
		if _, err := fetcher.Fetch(context.Background(), u); err == nil {
			t.Errorf("expected SSRF rejection for %s", u)
		}
	}
}

func TestCIMDFetcher_Fetch_Caches(t *testing.T) {
	hits := 0
	var metadataURL string
	fetcher, u, cleanup := newCIMDTestServer(t, "/client.json", func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"client_id": %q, "client_name": "x", "redirect_uris": ["http://localhost:3000/cb"]}`, metadataURL)
	})
	defer cleanup()
	metadataURL = u

	for i := 0; i < 3; i++ {
		if _, err := fetcher.Fetch(context.Background(), metadataURL); err != nil {
			t.Fatalf("fetch %d failed: %v", i, err)
		}
	}
	if hits != 1 {
		t.Errorf("expected 1 upstream hit due to caching, got %d", hits)
	}
}

const testTTL = 1 * time.Hour

// doAuthorize performs a GET /authorize with valid PKCE params.
func doAuthorize(t *testing.T, server *Server, clientID, redirectURI string) *httptest.ResponseRecorder {
	t.Helper()
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("state", "client-state")
	h := sha256.Sum256([]byte("verifier"))
	params.Set("code_challenge", base64.RawURLEncoding.EncodeToString(h[:]))
	params.Set("code_challenge_method", "S256")

	req := httptest.NewRequest(http.MethodGet, "/authorize?"+params.Encode(), nil)
	w := httptest.NewRecorder()
	server.AuthorizeHandler(w, req)
	return w
}

func TestServer_AuthorizeHandler_CIMDClient(t *testing.T) {
	var metadataURL string
	fetcher, u, cleanup := newCIMDTestServer(t, "/client.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"client_id": %q, "client_name": "CIMD Client", "redirect_uris": ["https://client.example.com/callback"]}`, metadataURL)
	})
	defer cleanup()
	metadataURL = u

	store := NewMemoryTokenStore()
	defer store.Close()
	google, gcleanup := newFakeGoogleProvider(t)
	defer gcleanup()
	logger := testLogger()
	server := NewServer("http://localhost:8080", google, store, logger, testTTL)
	server.SetCIMDFetcher(fetcher)

	// Registered redirect URI → accepted (302 to Google)
	w := doAuthorize(t, server, metadataURL, "https://client.example.com/callback")
	if w.Code != http.StatusFound {
		t.Errorf("valid CIMD redirect: expected 302, got %d: %s", w.Code, w.Body.String())
	}

	// Unregistered redirect URI → rejected
	w = doAuthorize(t, server, metadataURL, "https://attacker.example.com/callback")
	if w.Code == http.StatusFound {
		t.Error("unregistered CIMD redirect: expected rejection, got 302")
	}
}

func TestServer_AuthorizeHandler_CIMDFetchFailure(t *testing.T) {
	fetcher, u, cleanup := newCIMDTestServer(t, "/client.json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	store := NewMemoryTokenStore()
	defer store.Close()
	server := NewServer("http://localhost:8080", nil, store, testLogger(), testTTL)
	server.SetCIMDFetcher(fetcher)

	w := doAuthorize(t, server, u, "https://client.example.com/callback")
	if w.Code == http.StatusFound {
		t.Error("expected rejection when metadata document cannot be fetched")
	}
}
