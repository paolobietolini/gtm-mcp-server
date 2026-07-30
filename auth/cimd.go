package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// CIMD (Client ID Metadata Documents) support per
// draft-ietf-oauth-client-id-metadata-document-00, required by the MCP
// 2026-07-28 authorization spec. A client_id may be an HTTPS URL pointing
// to a JSON document describing the client.

const (
	cimdMaxBodyBytes = 64 << 10 // 64 KiB is ample for a metadata document
	cimdFetchTimeout = 10 * time.Second
	cimdDefaultTTL   = 5 * time.Minute
	cimdMaxTTL       = 24 * time.Hour
)

// IsClientIDMetadataURL reports whether clientID is a valid CIMD identifier:
// an https URL with a non-empty path and no fragment.
func IsClientIDMetadataURL(clientID string) bool {
	u, err := url.Parse(clientID)
	if err != nil {
		return false
	}
	return u.Scheme == "https" &&
		u.Host != "" &&
		u.Path != "" && u.Path != "/" &&
		u.Fragment == "" && u.RawFragment == ""
}

// clientIDMetadataDocument is the wire format of a CIMD document.
type clientIDMetadataDocument struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
}

type cimdCacheEntry struct {
	client    *ClientInfo
	expiresAt time.Time
}

// CIMDFetcher fetches, validates, and caches Client ID Metadata Documents.
type CIMDFetcher struct {
	// HTTPClient may be overridden in tests. Defaults to a client with a
	// conservative timeout.
	HTTPClient *http.Client
	// AllowPrivateHosts disables the SSRF guard (loopback/private/link-local
	// address rejection). Only for tests.
	AllowPrivateHosts bool

	mu    sync.Mutex
	cache map[string]cimdCacheEntry
}

// NewCIMDFetcher creates a fetcher with safe defaults.
func NewCIMDFetcher() *CIMDFetcher {
	return &CIMDFetcher{
		HTTPClient: &http.Client{Timeout: cimdFetchTimeout},
		cache:      make(map[string]cimdCacheEntry),
	}
}

// Fetch retrieves and validates the metadata document at clientID.
// Results are cached honoring Cache-Control max-age (bounded, with a default).
func (f *CIMDFetcher) Fetch(ctx context.Context, clientID string) (*ClientInfo, error) {
	if !IsClientIDMetadataURL(clientID) {
		return nil, fmt.Errorf("client_id is not a valid metadata document URL")
	}

	f.mu.Lock()
	if entry, ok := f.cache[clientID]; ok && time.Now().Before(entry.expiresAt) {
		f.mu.Unlock()
		return entry.client, nil
	}
	f.mu.Unlock()

	u, _ := url.Parse(clientID)
	if !f.AllowPrivateHosts {
		if err := rejectPrivateHost(ctx, u.Hostname()); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata URL: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch client metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("client metadata fetch returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, cimdMaxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read client metadata: %w", err)
	}
	if len(body) > cimdMaxBodyBytes {
		return nil, fmt.Errorf("client metadata document too large")
	}

	var doc clientIDMetadataDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("client metadata is not valid JSON: %w", err)
	}

	// The document's client_id MUST match the URL it was fetched from exactly.
	if doc.ClientID != clientID {
		return nil, fmt.Errorf("client metadata client_id %q does not match document URL", doc.ClientID)
	}
	if doc.ClientName == "" {
		return nil, fmt.Errorf("client metadata missing required field client_name")
	}
	if len(doc.RedirectURIs) == 0 {
		return nil, fmt.Errorf("client metadata missing required field redirect_uris")
	}

	client := &ClientInfo{
		ClientID:     doc.ClientID,
		ClientName:   doc.ClientName,
		RedirectURIs: doc.RedirectURIs,
		CreatedAt:    time.Now(),
	}

	ttl := cacheTTLFromResponse(resp)
	f.mu.Lock()
	f.cache[clientID] = cimdCacheEntry{client: client, expiresAt: time.Now().Add(ttl)}
	f.mu.Unlock()

	return client, nil
}

// cacheTTLFromResponse derives a bounded cache TTL from Cache-Control max-age.
func cacheTTLFromResponse(resp *http.Response) time.Duration {
	cc := resp.Header.Get("Cache-Control")
	for _, part := range strings.Split(cc, ",") {
		part = strings.TrimSpace(part)
		if v, ok := strings.CutPrefix(part, "max-age="); ok {
			var secs int
			if _, err := fmt.Sscanf(v, "%d", &secs); err == nil && secs > 0 {
				ttl := time.Duration(secs) * time.Second
				if ttl > cimdMaxTTL {
					return cimdMaxTTL
				}
				return ttl
			}
		}
	}
	return cimdDefaultTTL
}

// rejectPrivateHost resolves host and fails if any address is loopback,
// private, link-local, or otherwise non-global (SSRF guard).
func rejectPrivateHost(ctx context.Context, host string) error {
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("failed to resolve client metadata host: %w", err)
	}
	for _, ip := range ips {
		if ip.IP.IsLoopback() || ip.IP.IsPrivate() || ip.IP.IsLinkLocalUnicast() ||
			ip.IP.IsLinkLocalMulticast() || ip.IP.IsUnspecified() {
			return fmt.Errorf("client metadata host resolves to a non-public address")
		}
	}
	return nil
}
