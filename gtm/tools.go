package gtm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gtm-mcp-server/auth"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolGroups selects which GTM operation families appear in tools/list. The
// server keeps resources and prompts available because they add no tool-schema
// cost and provide documentation for whichever families are enabled.
type ToolGroups map[string]bool

var defaultToolGroupNames = []string{
	"accounts", "workspaces", "tags", "triggers", "variables",
	"folders", "builtins", "zones", "templates", "server", "guidance",
}

var optionalToolGroupNames = []string{
	"environments", "destinations", "gtag", "container-admin",
	"folder-admin", "workspace-admin", "version-admin", "reverts",
}

// ParseToolGroups validates GTM_TOOL_GROUPS. An empty value preserves the
// complete pre-grouping tool surface. "all" also enables future families.
func ParseToolGroups(names []string) (ToolGroups, error) {
	if len(names) == 0 {
		names = defaultToolGroupNames
	}
	known := make(map[string]bool, len(defaultToolGroupNames)+len(optionalToolGroupNames)+1)
	for _, name := range defaultToolGroupNames {
		known[name] = true
	}
	for _, name := range optionalToolGroupNames {
		known[name] = true
	}
	known["all"] = true
	groups := make(ToolGroups, len(names))
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if !known[name] {
			return nil, fmt.Errorf("unknown GTM tool group %q", name)
		}
		groups[name] = true
	}
	return groups, nil
}

func (groups ToolGroups) enabled(name string) bool { return groups["all"] || groups[name] }

func (groups ToolGroups) Names() []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RegisterTools adds all GTM tools to the MCP server.
func RegisterTools(server *mcp.Server) {
	groups, err := ParseToolGroups(nil)
	if err != nil {
		panic(err)
	}
	RegisterToolsForGroups(server, groups)
}

// RegisterToolsForGroups adds only the selected operation families.
func RegisterToolsForGroups(server *mcp.Server, groups ToolGroups) {
	if groups.enabled("accounts") {
		registerListAccounts(server)
		registerUpdateAccount(server)
		registerListContainers(server)
		registerLookupContainer(server)
		registerGetContainerSnippet(server)
		registerCreateContainer(server)
		registerUpdateContainer(server)
		registerDeleteContainer(server)
	}

	if groups.enabled("workspaces") {
		registerListWorkspaces(server)
		registerGetWorkspace(server)
		registerQuickPreviewWorkspace(server)
		registerCreateWorkspace(server)
		registerUpdateWorkspace(server)
		registerDeleteWorkspace(server)
		registerGetWorkspaceStatus(server)
		registerListVersions(server)
		registerGetLatestVersionHeader(server)
		registerGetVersion(server)
		registerGetLiveVersion(server)
		registerCreateVersion(server)
		registerPublishVersion(server)
	}

	if groups.enabled("tags") {
		registerListTags(server)
		registerGetTag(server)
		registerCreateTag(server)
		registerUpdateTag(server)
		registerDeleteTag(server)
	}

	if groups.enabled("triggers") {
		registerListTriggers(server)
		registerGetTrigger(server)
		registerCreateTrigger(server)
		registerUpdateTrigger(server)
		registerDeleteTrigger(server)
	}

	if groups.enabled("variables") {
		registerListVariables(server)
		registerGetVariable(server)
		registerCreateVariable(server)
		registerUpdateVariable(server)
		registerDeleteVariable(server)
	}

	if groups.enabled("folders") {
		registerListFolders(server)
		registerGetFolderEntities(server)
	}

	if groups.enabled("folder-admin") {
		registerGetFolder(server)
		registerCreateFolder(server)
		registerUpdateFolder(server)
		registerDeleteFolder(server)
		registerMoveEntitiesToFolder(server)
		registerRevertFolder(server)
	}

	if groups.enabled("workspace-admin") {
		registerBulkUpdateWorkspace(server)
		registerResolveWorkspaceConflict(server)
		registerSyncWorkspace(server)
	}

	if groups.enabled("version-admin") {
		registerDeleteVersion(server)
		registerUndeleteVersion(server)
		registerSetLatestVersion(server)
		registerUpdateVersion(server)
	}

	if groups.enabled("reverts") {
		registerRevertWorkspaceEntity(server)
	}

	if groups.enabled("zones") {
		registerListZones(server)
		registerGetZone(server)
		registerCreateZone(server)
		registerUpdateZone(server)
		registerDeleteZone(server)
	}

	if groups.enabled("environments") {
		registerListEnvironments(server)
		registerGetEnvironment(server)
		registerCreateEnvironment(server)
		registerUpdateEnvironment(server)
		registerReauthorizeEnvironment(server)
		registerDeleteEnvironment(server)
	}

	if groups.enabled("destinations") {
		registerListDestinations(server)
		registerGetDestination(server)
		registerLinkDestination(server)
	}

	if groups.enabled("gtag") {
		registerListGoogleTagConfigs(server)
		registerGetGoogleTagConfig(server)
		registerCreateGoogleTagConfig(server)
		registerUpdateGoogleTagConfig(server)
		registerDeleteGoogleTagConfig(server)
	}

	if groups.enabled("container-admin") {
		registerCombineContainers(server)
		registerMoveTagID(server)
	}

	if groups.enabled("builtins") {
		registerListBuiltInVariables(server)
		registerEnableBuiltInVariables(server)
		registerDisableBuiltInVariables(server)
	}

	if groups.enabled("templates") {
		registerListTemplates(server)
		registerGetTemplate(server)
		registerImportGalleryTemplate(server)
		registerCreateTemplate(server)
		registerUpdateTemplate(server)
		registerDeleteTemplate(server)
	}

	if groups.enabled("server") {
		registerListClients(server)
		registerGetClient(server)
		registerCreateClient(server)
		registerUpdateClient(server)
		registerDeleteClient(server)
		registerListTransformations(server)
		registerGetTransformation(server)
		registerCreateTransformation(server)
		registerUpdateTransformation(server)
		registerDeleteTransformation(server)
	}

	if groups.enabled("guidance") {
		registerGetTagTemplates(server)
		registerGetTriggerTemplates(server)
	}

	RegisterResources(server)
	RegisterPrompts(server)
}

// getClient creates a GTM client from the request context.
// In S2S mode the shared service account token source is used directly.
// In OAuth mode an auto-refreshing token source is built from the user's Google token.
func getClient(ctx context.Context) (*Client, error) {
	// S2S mode: shared service account token source injected by middleware
	if saTS := auth.GetSATokenSource(ctx); saTS != nil {
		return NewClient(ctx, saTS)
	}

	// OAuth mode: build auto-refreshing token source from user's Google token
	tokenInfo := auth.GetTokenInfo(ctx)
	if tokenInfo == nil || tokenInfo.GoogleToken == nil {
		return nil, fmt.Errorf("not authenticated - please authenticate with Google first")
	}

	store := auth.GetTokenStore(ctx)
	google := auth.GetGoogleProvider(ctx)

	var tokenSource = auth.NewAutoRefreshTokenSource(
		store,
		tokenInfo.AccessToken,
		google.Config(),
		tokenInfo.GoogleToken,
	)

	return NewClient(ctx, tokenSource)
}
