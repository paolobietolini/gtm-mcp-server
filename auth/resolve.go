package auth

import (
	"net/http"
	"net/url"
)

// URLResolver resolves the base URL for OAuth metadata responses.
// It validates the request's Host header against a set of allowed hosts,
// falling back to the configured base URL if the host is not recognized.
// This enables Docker-to-Docker contexts where the server is reached via
// internal network aliases, while preventing host header injection.
type URLResolver struct {
	configuredURL    string
	configuredHost   string
	configuredScheme string
	allowedHosts     map[string]bool
}

// NewURLResolver creates a resolver that trusts the configured base URL's host
// plus any additional allowed hosts.
func NewURLResolver(baseURL string, allowedHosts []string) *URLResolver {
	parsed, _ := url.Parse(baseURL)
	configuredHost := ""
	configuredScheme := ""
	if parsed != nil {
		configuredHost = parsed.Host
		configuredScheme = parsed.Scheme
	}

	allowed := make(map[string]bool, len(allowedHosts)+1)
	if configuredHost != "" {
		allowed[configuredHost] = true
	}
	for _, h := range allowedHosts {
		allowed[h] = true
	}

	return &URLResolver{
		configuredURL:    baseURL,
		configuredHost:   configuredHost,
		configuredScheme: configuredScheme,
		allowedHosts:     allowed,
	}
}

// Resolve returns the base URL to use for the given request.
// If the request's Host matches a trusted host, the URL is built dynamically
// (preserving the correct scheme via X-Forwarded-Proto). Otherwise, the
// static configured URL is returned.
func (u *URLResolver) Resolve(r *http.Request) string {
	host := r.Host
	if host == "" {
		return u.configuredURL
	}

	if !u.allowedHosts[host] {
		return u.configuredURL
	}

	// Host is trusted — build URL dynamically. X-Forwarded-Proto is unauthenticated
	// (unlike X-Forwarded-For, no TrustProxy gate applies here), so for the
	// configured host the configured scheme is the floor: a proxy that drops the
	// header, or a request that suppresses it, cannot make an https deployment
	// announce an http issuer.
	scheme := "http"
	switch {
	case r.TLS != nil, r.Header.Get("X-Forwarded-Proto") == "https":
		scheme = "https"
	case host == u.configuredHost && u.configuredScheme == "https":
		scheme = "https"
	}

	return scheme + "://" + host
}
