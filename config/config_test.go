package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestAutoRefreshMaxAge_DefaultsToSevenDays pins the bound on the renewal chain.
// Auto-refresh renews a bearer without rotating it, so without an absolute cap a
// leaked bearer stays useful for the whole refresh window (issue #79).
func TestAutoRefreshMaxAge_DefaultsToSevenDays(t *testing.T) {
	os.Unsetenv("AUTH_AUTO_REFRESH_MAX_AGE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if want := 7 * 24 * time.Hour; cfg.AutoRefreshMaxAge != want {
		t.Errorf("AutoRefreshMaxAge = %v, want %v", cfg.AutoRefreshMaxAge, want)
	}
}

func TestAutoRefreshMaxAge_HonoursEnv(t *testing.T) {
	t.Setenv("AUTH_AUTO_REFRESH_MAX_AGE", "48h")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if want := 48 * time.Hour; cfg.AutoRefreshMaxAge != want {
		t.Errorf("AutoRefreshMaxAge = %v, want %v", cfg.AutoRefreshMaxAge, want)
	}
}

func TestDashboardOptIn(t *testing.T) {
	for _, value := range []string{"", "false", "true"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("DASHBOARD_ENABLED", value)
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.DashboardEnabled != (value == "true") {
				t.Fatalf("DASHBOARD_ENABLED=%q: %v", value, cfg.DashboardEnabled)
			}
		})
	}
}

func TestToolGroupsFromEnvironment(t *testing.T) {
	t.Setenv("GTM_TOOL_GROUPS", "accounts, zones, tags")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"accounts", "zones", "tags"}
	if len(cfg.ToolGroups) != len(want) {
		t.Fatalf("ToolGroups=%v", cfg.ToolGroups)
	}
	for i := range want {
		if cfg.ToolGroups[i] != want[i] {
			t.Fatalf("ToolGroups=%v, want %v", cfg.ToolGroups, want)
		}
	}
}

func TestStatelessHTTPFromEnvironment(t *testing.T) {
	for _, value := range []string{"", "false", "0", "true", "1"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("MCP_STATELESS_HTTP", value)
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			want := value != "false" && value != "0"
			if cfg.StatelessHTTP != want {
				t.Fatalf("MCP_STATELESS_HTTP=%q: got %v, want %v", value, cfg.StatelessHTTP, want)
			}
		})
	}
}

// TestInvalidValuesFailLoad pins fail-fast parsing: a set but unparsable value
// must stop start-up instead of silently falling back to the default.
func TestInvalidValuesFailLoad(t *testing.T) {
	for _, tt := range []struct{ key, value string }{
		{"MCP_STATELESS_HTTP", "invalid"},
		{"TRUST_PROXY", "yes"},
		{"DASHBOARD_ENABLED", "on"},
		{"ACCESS_TOKEN_TTL", "7d"},
		{"AUTH_AUTO_REFRESH_MAX_AGE", "one week"},
		{"PORT", "http"},
		{"PORT", "0"},
		{"PORT", "65536"},
		{"ACCESS_TOKEN_TTL", "0s"},
		{"ACCESS_TOKEN_TTL", "-1h"},
		{"BASE_URL", "gtm-mcp.example.com"},
		{"BASE_URL", "ftp://gtm-mcp.example.com"},
		{"BASE_URL", "https://"},
		{"BASE_URL", "https://user:pass@gtm-mcp.example.com"},
		{"BASE_URL", "https://gtm-mcp.example.com?x=1"},
	} {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)
			cfg, err := Load()
			if err == nil {
				t.Fatalf("Load succeeded with %s=%q: %+v", tt.key, tt.value, cfg)
			}
			if !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("error does not name %s: %v", tt.key, err)
			}
		})
	}
}

func TestInvalidValuesAreAllReported(t *testing.T) {
	t.Setenv("TRUST_PROXY", "yes")
	t.Setenv("ACCESS_TOKEN_TTL", "7d")
	_, err := Load()
	if err == nil {
		t.Fatal("Load succeeded with two invalid values")
	}
	for _, key := range []string{"TRUST_PROXY", "ACCESS_TOKEN_TTL"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error does not name %s: %v", key, err)
		}
	}
}

func TestBaseURLTrailingSlashIsRemoved(t *testing.T) {
	for _, value := range []string{
		"https://gtm-mcp.example.com",
		"https://gtm-mcp.example.com/",
		"https://gtm-mcp.example.com//",
		" https://gtm-mcp.example.com/ ",
	} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("BASE_URL", value)
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if want := "https://gtm-mcp.example.com"; cfg.BaseURL != want {
				t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, want)
			}
		})
	}
}

func TestBaseURLDefaultAndLocalhost(t *testing.T) {
	for value, want := range map[string]string{
		"":                       "http://localhost:8080",
		"http://localhost:8080/": "http://localhost:8080",
		"http://127.0.0.1:9000":  "http://127.0.0.1:9000",
	} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("BASE_URL", value)
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.BaseURL != want {
				t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, want)
			}
		})
	}
}

func TestAutoRefreshMaxAgeZeroIsAccepted(t *testing.T) {
	// A non-positive max age disables the renewal cap in auth.tryAutoRefresh;
	// it is a deliberate operator choice, not a parse error.
	t.Setenv("AUTH_AUTO_REFRESH_MAX_AGE", "0s")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AutoRefreshMaxAge != 0 {
		t.Fatalf("AutoRefreshMaxAge = %v, want 0", cfg.AutoRefreshMaxAge)
	}
}
