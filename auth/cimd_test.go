package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

// A redirect must not be able to walk the fetch off https onto an internal
// plain-http endpoint. Asserts the target was never contacted, not merely that
// Fetch returned some error — an unreachable target would fail either way.
func TestCIMDFetcher_Fetch_DoesNotFollowRedirectToNonHTTPS(t *testing.T) {
	var targetHits int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetHits++
		w.Write([]byte("internal"))
	}))
	defer target.Close()

	fetcher, u, cleanup := newCIMDTestServer(t, "/client.json", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/latest/meta-data/", http.StatusFound)
	})
	defer cleanup()

	_, err := fetcher.Fetch(context.Background(), u)
	if err == nil {
		t.Error("expected an error when the metadata document redirects to http")
	}
	if targetHits != 0 {
		t.Errorf("redirect target was contacted %d times, want 0", targetHits)
	}
}

// The https case: the redirect stays on https but points at a private address.
// Exercised directly because the AllowPrivateHosts escape hatch that lets the
// httptest server (127.0.0.1) be reached at all would also skip this check.
func TestCIMDFetcher_CheckRedirect_RejectsPrivateHost(t *testing.T) {
	f := NewCIMDFetcher()
	req := httptest.NewRequest(http.MethodGet, "https://127.0.0.1/latest/meta-data/", nil)

	if err := f.checkRedirect(req, nil); err == nil {
		t.Error("expected an https redirect to a loopback address to be rejected")
	}
}

func TestCIMDFetcher_CheckRedirect_RejectsTooManyHops(t *testing.T) {
	f := NewCIMDFetcher()
	f.AllowPrivateHosts = true
	req := httptest.NewRequest(http.MethodGet, "https://example.com/next", nil)

	via := make([]*http.Request, cimdMaxRedirects)
	if err := f.checkRedirect(req, via); err == nil {
		t.Errorf("expected rejection after %d hops", cimdMaxRedirects)
	}
	if err := f.checkRedirect(req, via[:1]); err != nil {
		t.Errorf("a single hop to a public https host should be allowed, got %v", err)
	}
}

// The dial-time guard sees the address actually being connected to, so it
// covers both redirect targets and DNS rebinding, which rejectPrivateHost
// (name-based, pre-flight) cannot.
func TestRejectPrivateAddr(t *testing.T) {
	tests := []struct {
		address string
		wantErr bool
	}{
		{"93.184.216.34:443", false}, // public
		{"[2606:2800:220:1:248:1893:25c8:1946]:443", false},
		{"127.0.0.1:80", true},
		{"169.254.169.254:80", true}, // cloud metadata service
		{"10.0.0.5:443", true},
		{"192.168.1.1:443", true},
		{"172.16.0.1:443", true},
		{"[::1]:443", true},
		{"[fd00::1]:443", true}, // IPv6 unique-local
		{"0.0.0.0:80", true},
		{"[::ffff:127.0.0.1]:80", true}, // IPv4-mapped loopback

		// Ranges net.IP's own helpers do not classify. IsPrivate covers only
		// RFC1918 and fc00::/7, so these reach real infrastructure otherwise.
		{"100.64.0.1:80", true},           // RFC 6598 CGNAT (AWS/GCP/k8s NAT)
		{"[64:ff9b::a9fe:a9fe]:80", true}, // NAT64 of 169.254.169.254
		{"192.0.0.1:80", true},            // IETF protocol assignments
		{"198.18.0.1:80", true},           // benchmarking
		{"240.0.0.1:80", true},            // class E
		{"255.255.255.255:80", true},      // broadcast
		{"[2002:7f00:1::]:80", true},      // 6to4 of 127.0.0.1
		{"224.0.0.1:80", true},            // multicast
		{"[ff02::1]:80", true},            // link-local multicast

		// IPv4-compatible IPv6 (::a.b.c.d). To4() normalizes only the
		// ::ffff: form, so no net.IP helper fires on these.
		{"[::7f00:1]:80", true},    // 127.0.0.1
		{"[::a9fe:a9fe]:80", true}, // 169.254.169.254
		{"192.88.99.1:80", true},   // 6to4 relay anycast
		{"[2001::1]:80", true},     // Teredo
		{"[2001:10::1]:80", true},  // ORCHID
	}

	for _, tt := range tests {
		err := rejectPrivateAddr("tcp", tt.address, nil)
		if (err != nil) != tt.wantErr {
			t.Errorf("rejectPrivateAddr(%q) error = %v, wantErr %v", tt.address, err, tt.wantErr)
		}
	}
}

func TestCIMDFetcher_DefaultTransportGuardsDialAddress(t *testing.T) {
	// Dial a listener that is genuinely accepting connections, so an
	// unguarded transport would succeed here. Dialling a closed port would
	// fail with "connection refused" either way and prove nothing.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open listener: %v", err)
	}
	defer ln.Close()

	f := NewCIMDFetcher()
	tr, ok := f.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("default HTTPClient.Transport is %T, want *http.Transport", f.HTTPClient.Transport)
	}

	_, err = tr.DialContext(context.Background(), "tcp", ln.Addr().String())
	if err == nil {
		t.Fatal("default transport dialled a reachable loopback address; dial guard is not wired")
	}
	if !strings.Contains(err.Error(), "non-public address") {
		t.Errorf("dial failed for the wrong reason: %v", err)
	}
}

// An inherited proxy would make the dial guard blind: Control would only ever
// see the proxy's address, never the real target, silently reinstating the
// DNS-rebinding bypass.
func TestCIMDFetcher_DefaultTransportDoesNotUseProxy(t *testing.T) {
	f := NewCIMDFetcher()
	tr := f.HTTPClient.Transport.(*http.Transport)
	if tr.Proxy != nil {
		t.Error("default transport inherits a proxy, which bypasses the dial guard")
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
