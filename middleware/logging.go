package middleware

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewLoggingMiddleware creates MCP-level logging middleware that logs
// all incoming requests and their results. For tools/call requests,
// it extracts and logs the tool name for audit purposes.
func NewLoggingMiddleware(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			start := time.Now()
			sessionID := ""
			if session := req.GetSession(); session != nil {
				sessionID = session.ID()
			}

			// Extract tool name for tools/call requests
			toolName := ""
			if method == "tools/call" {
				toolName = extractToolName(req)
			}

			attrs := []any{
				"method", method,
				"session_id", sessionID,
			}
			if toolName != "" {
				attrs = append(attrs, "tool", toolName)
			}

			logger.Info("mcp request", attrs...)

			result, err := next(ctx, method, req)

			duration := time.Since(start)
			attrs = append(attrs, "duration_ms", duration.Milliseconds())

			if err != nil {
				// Context cancellation is not an error - don't log when client disconnects
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					logger.Error("mcp request failed",
						append(attrs, "error", err.Error())...,
					)
				}
			} else {
				logger.Info("mcp request completed", attrs...)
			}

			return result, err
		}
	}
}

// extractToolName attempts to extract the tool name from a tools/call request.
func extractToolName(req mcp.Request) string {
	if ctr, ok := req.(*mcp.CallToolRequest); ok {
		return ctr.Params.Name
	}
	return ""
}
