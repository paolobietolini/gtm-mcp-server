package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gtm-mcp-server/auth"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	protocolVersion20250618 = "2025-06-18"
	protocolVersion20251125 = "2025-11-25"
	protocolVersion20260728 = "2026-07-28"
)

func TestProtocol20260728DiscoverThenTools(t *testing.T) {
	srv := httptest.NewServer(newMCPHTTPHandler(testProtocolServer()))
	t.Cleanup(srv.Close)

	discoverStatus, discoverBody := postMCP(t, srv.URL, nil, map[string]string{
		"Mcp-Protocol-Version": protocolVersion20260728,
		"Mcp-Method":           "server/discover",
	}, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "server/discover",
		"params": map[string]any{
			"_meta": newProtocolMeta(protocolVersion20260728),
		},
	})
	assertMCPOK(t, discoverStatus, discoverBody, "supportedVersions", protocolVersion20260728)

	listStatus, listBody := postMCP(t, srv.URL, nil, map[string]string{
		"Mcp-Protocol-Version": protocolVersion20260728,
		"Mcp-Method":           "tools/list",
	}, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params": map[string]any{
			"_meta": newProtocolMeta(protocolVersion20260728),
		},
	})
	assertMCPOK(t, listStatus, listBody, `"name":"ping"`)

	callStatus, callBody := postMCP(t, srv.URL, nil, map[string]string{
		"Mcp-Protocol-Version": protocolVersion20260728,
		"Mcp-Method":           "tools/call",
		"Mcp-Name":             "ping",
	}, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"_meta":     newProtocolMeta(protocolVersion20260728),
			"name":      "ping",
			"arguments": map[string]any{"message": "chatgpt"},
		},
	})
	assertMCPOK(t, callStatus, callBody, "pong: chatgpt")
}

func TestLegacyInitializeThenToolsList(t *testing.T) {
	for _, version := range []string{protocolVersion20250618, protocolVersion20251125} {
		t.Run(version, func(t *testing.T) {
			srv := httptest.NewServer(newMCPHTTPHandler(testProtocolServer()))
			t.Cleanup(srv.Close)

			initStatus, initBody := postMCP(t, srv.URL, nil, nil, map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "initialize",
				"params": map[string]any{
					"protocolVersion": version,
					"capabilities":    map[string]any{},
					"clientInfo":      map[string]any{"name": "legacy-client", "version": "1.0"},
				},
			})
			assertMCPOK(t, initStatus, initBody, `"protocolVersion":"`+version+`"`)

			listStatus, listBody := postMCP(t, srv.URL, nil, map[string]string{
				"Mcp-Protocol-Version": version,
			}, map[string]any{
				"jsonrpc": "2.0",
				"id":      2,
				"method":  "tools/list",
				"params":  map[string]any{},
			})
			assertMCPOK(t, listStatus, listBody, `"name":"ping"`)
		})
	}
}

func TestUnauthenticatedMCPRequestIs401(t *testing.T) {
	store := auth.NewMemoryTokenStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	protected := auth.Middleware(
		store, nil, logger, "http://localhost:8080", time.Hour, nil, nil, "", 7*24*time.Hour,
	)(newMCPHTTPHandler(testProtocolServer()))
	srv := httptest.NewServer(protected)
	t.Cleanup(srv.Close)

	status, body := postMCP(t, srv.URL, nil, map[string]string{
		"Mcp-Protocol-Version": protocolVersion20260728,
		"Mcp-Method":           "tools/list",
	}, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params": map[string]any{
			"_meta": newProtocolMeta(protocolVersion20260728),
		},
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %s", status, body)
	}
	if !strings.Contains(string(body), `"error":"unauthorized"`) {
		t.Fatalf("body = %s, want unauthorized", body)
	}
}

func TestBearerTokenReachesToolContext(t *testing.T) {
	store := auth.NewMemoryTokenStore()
	if err := store.StoreToken(&auth.TokenInfo{
		AccessToken: "test-access",
		ClientID:    "oauth-client",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	protected := auth.Middleware(
		store, nil, logger, "http://localhost:8080", time.Hour, nil, nil, "", 7*24*time.Hour,
	)(newMCPHTTPHandler(testProtocolServer()))
	srv := httptest.NewServer(protected)
	t.Cleanup(srv.Close)

	status, body := postMCP(t, srv.URL, http.Header{
		"Authorization": {"Bearer test-access"},
	}, map[string]string{
		"Mcp-Protocol-Version": protocolVersion20260728,
		"Mcp-Method":           "tools/call",
		"Mcp-Name":             "whoami",
	}, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"_meta":     newProtocolMeta(protocolVersion20260728),
			"name":      "whoami",
			"arguments": map[string]any{},
		},
	})
	assertMCPOK(t, status, body, `"clientId":"oauth-client"`)
}

func testProtocolServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: "test"}, nil)
	registerUtilityTools(server)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whoami",
		Description: "Return the bearer client id from the request context",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]string, error) {
		info := auth.GetTokenInfo(ctx)
		clientID := ""
		if info != nil {
			clientID = info.ClientID
		}
		return nil, map[string]string{"clientId": clientID}, nil
	})
	return server
}

func newProtocolMeta(version string) map[string]any {
	return map[string]any{
		mcp.MetaKeyProtocolVersion:    version,
		mcp.MetaKeyClientInfo:         map[string]any{"name": "chatgpt", "version": "1"},
		mcp.MetaKeyClientCapabilities: map[string]any{},
	}
}

func postMCP(t *testing.T, url string, extra http.Header, headers map[string]string, body any) (int, []byte) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	for key, values := range extra {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, respBody
}

func assertMCPOK(t *testing.T, status int, body []byte, needles ...string) {
	t.Helper()
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", status, body)
	}
	text := string(body)
	if strings.Contains(text, `"error"`) {
		t.Fatalf("json-rpc error: %s", text)
	}
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			t.Fatalf("body missing %q: %s", needle, text)
		}
	}
}
