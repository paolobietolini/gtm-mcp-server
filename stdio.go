package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"gtm-mcp-server/auth"
	"gtm-mcp-server/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
)

// transportFromArgs lets a "--stdio" (or "stdio") argument override the
// configured transport, so MCP clients can launch the binary without env vars.
func transportFromArgs(args []string, configured string) string {
	for _, arg := range args {
		if arg == "--stdio" || arg == "stdio" {
			return "stdio"
		}
	}
	return configured
}

// runStdio serves MCP over stdin/stdout until the client disconnects or the
// process is signalled. Credentials come from GOOGLE_SERVICE_ACCOUNT_KEY_JSON
// or Application Default Credentials (GOOGLE_APPLICATION_CREDENTIALS).
func runStdio(server *mcp.Server, cfg *config.Config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tokenSource, err := auth.NewServiceAccountTokenSource(ctx, cfg.ServiceAccountKeyJSON)
	if err != nil {
		return fmt.Errorf("stdio mode requires service account credentials: %w", err)
	}
	server.AddReceivingMiddleware(serviceAccountMiddleware(tokenSource))

	logger.Info("starting GTM MCP server", "transport", "stdio")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	logger.Info("server stopped")
	return nil
}

// serviceAccountMiddleware injects the service account token source into every
// request context, mirroring what auth.Middleware does for S2S HTTP requests,
// so getClient() and auth_status work unchanged.
func serviceAccountMiddleware(tokenSource oauth2.TokenSource) mcp.Middleware {
	info := auth.ServiceAccountTokenInfo()
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			ctx = context.WithValue(ctx, auth.SATokenSourceKey, tokenSource)
			ctx = context.WithValue(ctx, auth.TokenInfoKey, info)
			return next(ctx, method, req)
		}
	}
}
