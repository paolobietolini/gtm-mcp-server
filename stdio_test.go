package main

import (
	"context"
	"testing"

	"gtm-mcp-server/auth"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
)

func TestTransportFromArgs(t *testing.T) {
	tests := []struct {
		args       []string
		configured string
		want       string
	}{
		{nil, "http", "http"},
		{nil, "stdio", "stdio"},
		{[]string{"--stdio"}, "http", "stdio"},
		{[]string{"stdio"}, "http", "stdio"},
		{[]string{"--other"}, "http", "http"},
	}
	for _, tt := range tests {
		if got := transportFromArgs(tt.args, tt.configured); got != tt.want {
			t.Errorf("transportFromArgs(%v, %q) = %q, want %q", tt.args, tt.configured, got, tt.want)
		}
	}
}

func TestServiceAccountMiddlewareInjectsCredentials(t *testing.T) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	var gotTS oauth2.TokenSource
	var gotInfo *auth.TokenInfo
	next := func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		gotTS = auth.GetSATokenSource(ctx)
		gotInfo = auth.GetTokenInfo(ctx)
		return nil, nil
	}

	if _, err := serviceAccountMiddleware(ts)(next)(context.Background(), "tools/call", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTS != ts {
		t.Error("service account token source not injected into context")
	}
	if gotInfo == nil || gotInfo.ClientID != "service-account" {
		t.Errorf("token info = %+v, want service-account", gotInfo)
	}
}
