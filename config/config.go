package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the GTM MCP Server.
type Config struct {
	// Server configuration
	Port    int
	BaseURL string
	// DashboardEnabled exposes a public setup page with configuration status only.
	DashboardEnabled bool

	// Transport selects how MCP is served: "http" (Streamable HTTP, the
	// default) or "stdio" (one local client over stdin/stdout, authenticated
	// with service account credentials).
	Transport string
	// StatelessHTTP disables in-memory MCP HTTP sessions (the default). Required
	// for MCP 2026-07-28 and for load-balanced deployments.
	StatelessHTTP bool

	// Google OAuth configuration. The redirect URI is always
	// BaseURL + "/oauth/callback"; there is no separate setting for it.
	GoogleClientID     string
	GoogleClientSecret string

	// Logging
	LogLevel string
	// ToolGroups limits the GTM operation families advertised through MCP.
	// Empty selects the backward-compatible default set.
	ToolGroups []string

	// Token configuration
	AccessTokenTTL time.Duration

	// TokenStorePath, when set, persists issued tokens to this JSON file so
	// sessions survive restarts. Empty means in-memory only (the default).
	TokenStorePath string

	// AllowedHosts lists additional trusted hostnames for dynamic base URL resolution.
	// Enables Docker-to-Docker contexts where the server is reached via internal aliases.
	AllowedHosts []string

	// Service account authentication (S2S mode)
	ServiceAccountAPIKey  string // SERVICE_ACCOUNT_API_KEY
	ServiceAccountKeyJSON string // GOOGLE_SERVICE_ACCOUNT_KEY_JSON

	// TrustProxy enables trusting X-Forwarded-For for rate limiting.
	// Set to true when behind a reverse proxy (e.g. Caddy).
	TrustProxy bool

	// AutoRefreshMaxAge bounds the silent renewal chain. tryAutoRefresh extends a
	// bearer in place without rotating it, so its expiry bounds nothing on its
	// own; past this age since the token was issued the server stops renewing and
	// answers 401, forcing the client through the refresh grant, which does
	// rotate. See issue #79.
	AutoRefreshMaxAge time.Duration
}

// Load reads configuration from environment variables.
// It first attempts to load from .env file if present, then .env.local for overrides.
//
// A value that is set but cannot be parsed is an error rather than a silent
// fallback to the default: a typo such as TRUST_PROXY=yes or
// ACCESS_TOKEN_TTL=7d would otherwise change behaviour without any signal.
// All such errors are reported together so one start-up shows every problem.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()
	// Load .env.local for local development overrides (takes precedence)
	_ = godotenv.Overload(".env.local")

	var p envParser
	cfg := &Config{
		DashboardEnabled:      p.boolean("DASHBOARD_ENABLED", false),
		Port:                  p.integer("PORT", 8080),
		BaseURL:               normalizeBaseURL(getEnv("BASE_URL", "http://localhost:8080")),
		Transport:             strings.ToLower(getEnv("MCP_TRANSPORT", "http")),
		StatelessHTTP:         p.boolean("MCP_STATELESS_HTTP", true),
		GoogleClientID:        getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:    getEnv("GOOGLE_CLIENT_SECRET", ""),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		ToolGroups:            getEnvList("GTM_TOOL_GROUPS"),
		AccessTokenTTL:        p.duration("ACCESS_TOKEN_TTL", 8*time.Hour),
		TokenStorePath:        getEnv("TOKEN_STORE_PATH", ""),
		AllowedHosts:          getEnvList("ALLOWED_HOSTS"),
		ServiceAccountAPIKey:  getEnv("SERVICE_ACCOUNT_API_KEY", ""),
		ServiceAccountKeyJSON: getEnv("GOOGLE_SERVICE_ACCOUNT_KEY_JSON", ""),
		TrustProxy:            p.boolean("TRUST_PROXY", false),
		AutoRefreshMaxAge:     p.duration("AUTH_AUTO_REFRESH_MAX_AGE", 7*24*time.Hour),
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		p.fail("PORT=%d is out of range (1-65535)", cfg.Port)
	}
	if cfg.AccessTokenTTL <= 0 {
		p.fail("ACCESS_TOKEN_TTL=%s must be positive", cfg.AccessTokenTTL)
	}
	if err := validateBaseURL(cfg.BaseURL); err != nil {
		p.errs = append(p.errs, err)
	}

	// OAuth credentials are validated later, when auth is actually needed.
	// This allows the server to start and respond to initialize/ping
	// even without OAuth credentials configured.

	if err := errors.Join(p.errs...); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

// normalizeBaseURL removes trailing slashes. Every OAuth URL the server
// publishes (issuer, endpoints, redirect URI, WWW-Authenticate) is built as
// BaseURL + "/path", so a trailing slash would produce "//path" and a
// redirect URI that does not match the one registered with Google.
func normalizeBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

// validateBaseURL rejects values that cannot produce working OAuth metadata,
// such as a bare hostname without a scheme.
func validateBaseURL(baseURL string) error {
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		// Not echoed, like the case below: a malformed value may still
		// contain credentials.
		return errors.New("BASE_URL must be an absolute http:// or https:// URL such as https://gtm-mcp.example.com")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		// The value is not echoed: it may contain a password.
		return errors.New("BASE_URL must not contain credentials, a query, or a fragment")
	}
	return nil
}

// ValidateAuth checks if OAuth credentials are configured.
func (c *Config) ValidateAuth() error {
	if c.GoogleClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is required for authentication")
	}
	if c.GoogleClientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET is required for authentication")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvList(key string) []string {
	if value := os.Getenv(key); value != "" {
		var hosts []string
		for _, h := range strings.Split(value, ",") {
			if h = strings.TrimSpace(h); h != "" {
				hosts = append(hosts, h)
			}
		}
		return hosts
	}
	return nil
}

// envParser reads typed environment variables and collects parse errors
// instead of discarding them. An unset or empty variable takes its default.
type envParser struct {
	errs []error
}

func (p *envParser) fail(format string, args ...any) {
	p.errs = append(p.errs, fmt.Errorf(format, args...))
}

func (p *envParser) duration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		p.fail("%s=%q is not a valid duration; use Go syntax such as 8h, 90m, or 168h", key, value)
		return defaultValue
	}
	return d
}

func (p *envParser) boolean(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		p.fail("%s=%q is not a valid boolean; use true or false", key, value)
		return defaultValue
	}
	return b
}

func (p *envParser) integer(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		p.fail("%s=%q is not a valid integer", key, value)
		return defaultValue
	}
	return n
}
