package config

import (
	"os"
	"testing"
)

// TestAutoRefreshEnabled_DefaultsTrue pins the deploy-safe default: existing
// deployments that never set AUTH_AUTO_REFRESH keep today's auto-refresh
// behaviour. The flag exists to be switched off for a canary window (#79).
func TestAutoRefreshEnabled_DefaultsTrue(t *testing.T) {
	os.Unsetenv("AUTH_AUTO_REFRESH")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !cfg.AutoRefreshEnabled {
		t.Error("AutoRefreshEnabled = false with AUTH_AUTO_REFRESH unset, want true")
	}
}

func TestAutoRefreshEnabled_HonoursEnv(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"false", false},
		{"0", false},
		{"true", true},
		{"1", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("AUTH_AUTO_REFRESH", tt.value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load failed: %v", err)
			}

			if cfg.AutoRefreshEnabled != tt.want {
				t.Errorf("AUTH_AUTO_REFRESH=%q gave AutoRefreshEnabled=%v, want %v", tt.value, cfg.AutoRefreshEnabled, tt.want)
			}
		})
	}
}
