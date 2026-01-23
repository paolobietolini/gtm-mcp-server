package gtm

import (
	"context"
	"fmt"

	"gtm-mcp-server/auth"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterTools adds all GTM tools to the MCP server.
func RegisterTools(server *mcp.Server) {
	// Read operations
	registerListAccounts(server)
	registerListContainers(server)
	registerListWorkspaces(server)
	registerListTags(server)
	registerGetTag(server)
	registerListTriggers(server)
	registerListVariables(server)
	registerListFolders(server)
	registerGetFolderEntities(server)

	// Write operations
	registerCreateTag(server)
	registerUpdateTag(server)
	registerDeleteTag(server)
	registerCreateTrigger(server)
	registerUpdateTrigger(server)
	registerDeleteTrigger(server)
	registerCreateVariable(server)
	registerDeleteVariable(server)
	registerCreateContainer(server)
	registerCreateWorkspace(server)

	// Version operations
	registerCreateVersion(server)
	registerPublishVersion(server)

	// Templates (help LLMs with correct parameter formats)
	registerGetTagTemplates(server)
	registerGetTriggerTemplates(server)

	// Resources (URI-based read access)
	RegisterResources(server)

	// Prompts (template workflows)
	RegisterPrompts(server)
}

// getClient creates a GTM client from the request context with auto-refreshing tokens.
func getClient(ctx context.Context) (*Client, error) {
	tokenInfo := auth.GetTokenInfo(ctx)
	if tokenInfo == nil || tokenInfo.GoogleToken == nil {
		return nil, fmt.Errorf("not authenticated - please authenticate with Google first")
	}

	store := auth.GetTokenStore(ctx)
	google := auth.GetGoogleProvider(ctx)

	// Create auto-refreshing token source
	var tokenSource = auth.NewAutoRefreshTokenSource(
		store,
		tokenInfo.AccessToken,
		google.Config(),
		tokenInfo.GoogleToken,
	)

	return NewClient(ctx, tokenSource)
}
