package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gtm-mcp-server/auth"
	"gtm-mcp-server/config"
	"gtm-mcp-server/middleware"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName    = "gtm-mcp-server"
	serverVersion = "0.1.0"
)

func main() {
	// Set up structured logging to stderr (stdout is reserved for MCP in stdio mode)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Adjust log level
	if cfg.LogLevel == "debug" {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
	}

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)

	// Add logging middleware
	server.AddReceivingMiddleware(middleware.NewLoggingMiddleware(logger))

	// Register tools
	registerTools(server)

	// Create HTTP handler for MCP
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Health check endpoint (no auth required)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": serverName,
			"version": serverVersion,
		})
	})

	// OAuth metadata endpoints (always served, no auth required)
	// RFC 9728: Protected Resource Metadata - tells clients where to find the authorization server
	mux.HandleFunc("GET /.well-known/oauth-protected-resource",
		auth.ProtectedResourceMetadataHandler(cfg.BaseURL, cfg.BaseURL))

	// RFC 8414: Authorization Server Metadata - tells clients about OAuth endpoints
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", auth.MetadataHandler(cfg.BaseURL))

	// Check if OAuth is configured
	var authServer *auth.Server
	var tokenStore auth.TokenStore

	if err := cfg.ValidateAuth(); err != nil {
		logger.Warn("OAuth not configured, running without authentication", "error", err)
		// MCP endpoint without auth
		mux.Handle("/", mcpHandler)
	} else {
		// Set up OAuth
		tokenStore = auth.NewMemoryTokenStore()
		googleProvider := auth.NewGoogleProvider(
			cfg.GoogleClientID,
			cfg.GoogleClientSecret,
			cfg.BaseURL+"/oauth/callback",
		)
		authServer = auth.NewServer(cfg.BaseURL, googleProvider, tokenStore, logger)

		// OAuth endpoints (no auth required)
		mux.HandleFunc("GET /authorize", authServer.AuthorizeHandler)
		mux.HandleFunc("GET /oauth/callback", authServer.CallbackHandler)
		mux.HandleFunc("POST /token", authServer.TokenHandler)
		mux.HandleFunc("POST /register", authServer.RegistrationHandler)

		// MCP endpoint with REQUIRED auth middleware
		// Returns 401 if no valid Bearer token - triggers Claude's OAuth flow
		authMiddleware := auth.Middleware(tokenStore, logger, cfg.BaseURL)
		mux.Handle("/", authMiddleware(mcpHandler))

		logger.Info("OAuth configured",
			"authorize_endpoint", cfg.BaseURL+"/authorize",
			"token_endpoint", cfg.BaseURL+"/token",
			"callback_endpoint", cfg.BaseURL+"/oauth/callback",
			"register_endpoint", cfg.BaseURL+"/register",
			"protected_resource_metadata", cfg.BaseURL+"/.well-known/oauth-protected-resource",
			"authorization_server_metadata", cfg.BaseURL+"/.well-known/oauth-authorization-server",
		)
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%d", cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // Disabled for SSE streams
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start server
	go func() {
		logger.Info("starting GTM MCP server",
			"port", cfg.Port,
			"base_url", cfg.BaseURL,
		)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("shutting down server")

	// Give outstanding requests 10 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("server stopped")
}

// registerTools adds MCP tools to the server.
func registerTools(server *mcp.Server) {
	// Placeholder ping tool for testing connectivity
	type PingInput struct {
		Message string `json:"message,omitempty" jsonschema:"Optional message to echo back"`
	}

	type PingOutput struct {
		Reply     string `json:"reply"`
		Timestamp string `json:"timestamp"`
	}

	pingHandler := func(ctx context.Context, req *mcp.CallToolRequest, input PingInput) (*mcp.CallToolResult, PingOutput, error) {
		reply := "pong"
		if input.Message != "" {
			reply = fmt.Sprintf("pong: %s", input.Message)
		}

		output := PingOutput{
			Reply:     reply,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		return nil, output, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ping",
		Description: "Test connectivity to the GTM MCP server",
	}, pingHandler)

	// Auth status tool - tells user if they're authenticated
	type AuthStatusInput struct{}
	type AuthStatusOutput struct {
		Authenticated bool   `json:"authenticated"`
		Message       string `json:"message"`
	}

	authStatusHandler := func(ctx context.Context, req *mcp.CallToolRequest, input AuthStatusInput) (*mcp.CallToolResult, AuthStatusOutput, error) {
		tokenInfo := auth.GetTokenInfo(ctx)

		output := AuthStatusOutput{
			Authenticated: tokenInfo != nil,
		}

		if tokenInfo != nil {
			output.Message = "You are authenticated and can access GTM data"
		} else {
			output.Message = "Not authenticated. GTM tools will require authentication."
		}

		return nil, output, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "auth_status",
		Description: "Check authentication status with Google Tag Manager",
	}, authStatusHandler)
}
