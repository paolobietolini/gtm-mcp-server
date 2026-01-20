package gtm

import (
	"context"
	"fmt"

	"gtm-mcp-server/auth"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterTools adds all GTM tools to the MCP server.
func RegisterTools(server *mcp.Server) {
	registerListAccounts(server)
	registerListContainers(server)
	registerListWorkspaces(server)
	registerListTags(server)
	registerGetTag(server)
	registerSearchTags(server)
	registerListTriggers(server)
	registerListVariables(server)
}

// getClient creates a GTM client from the request context.
func getClient(ctx context.Context) (*Client, error) {
	tokenInfo := auth.GetTokenInfo(ctx)
	if tokenInfo == nil || tokenInfo.GoogleToken == nil {
		return nil, fmt.Errorf("not authenticated - please authenticate with Google first")
	}
	return NewClient(ctx, tokenInfo.GoogleToken)
}
