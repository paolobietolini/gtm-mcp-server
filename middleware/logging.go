package middleware

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewLoggingMiddleware creates MCP-level logging middleware that logs
// all incoming requests and their results.
func NewLoggingMiddleware(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			start := time.Now()
			sessionID := ""
			if session := req.GetSession(); session != nil {
				sessionID = session.ID()
			}

			logger.Info("mcp request",
				"method", method,
				"session_id", sessionID,
			)

			result, err := next(ctx, method, req)

			duration := time.Since(start)

			if err != nil {
				// Context cancellation is not an error - don't log when client disconnects
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					logger.Error("mcp request failed",
						"method", method,
						"session_id", sessionID,
						"duration_ms", duration.Milliseconds(),
						"error", err.Error(),
					)
				}
			} else {
				logger.Debug("mcp request completed",
					"method", method,
					"session_id", sessionID,
					"duration_ms", duration.Milliseconds(),
				)
			}

			return result, err
		}
	}
}
