# GTM MCP Server - Developer Reference

## Project Overview

A **Model Context Protocol (MCP) Server** that connects AI assistants (Claude, ChatGPT, etc.) with the **Google Tag Manager API**. The server enables AI to read, create, modify, and publish GTM configurations.

- **Production:** `https://mcp.gtmeditor.com`
- **Repository:** `github.com/paolobietolini/gtm-mcp-server`
- **Language:** Go 1.24+
- **Status:** ✅ Complete (Phases 1-5)

---

## Project Status: COMPLETE

| Phase | Description | Status |
|-------|-------------|--------|
| Phase 1 | HTTP Transport & MCP Foundation | ✅ Complete |
| Phase 2 | OAuth 2.1 Authentication | ✅ Complete |
| Phase 3 | GTM Read Operations | ✅ Complete |
| Phase 4 | GTM Write Operations | ✅ Complete |
| Phase 5 | Resources & Prompts | ✅ Complete |
| Phase 6 | Production Hardening | 🔄 Optional |

---

## Development Workflow

After implementing a feature or fixing a bug:

```bash
# 1. Build & verify
go build ./...

# 2. Test locally (Claude has gtm-local MCP configured at localhost:8081)
PORT=8081 go run main.go

# 3. Build Docker image
docker compose build

# 4. Commit (don't push yet)
git add . && git commit -m "Description"

# 5. Deploy to production
bash deploy.sh

# 6. Verify production
curl https://mcp.gtmeditor.com/health

# 7. Push & close issues
git push origin main
gh issue close <issue-number>
```

---

## File Structure

```
gtm-mcp-server/
├── main.go                    # Entry point, HTTP server, MCP + OAuth setup
├── deploy.sh                  # Production deployment script
├── Dockerfile                 # Multi-stage Docker build
├── docker-compose.yml         # Docker + Caddy setup
├── Caddyfile                  # Caddy reverse proxy config
│
├── config/
│   └── config.go              # Environment configuration
│
├── auth/
│   ├── metadata.go            # OAuth2 server metadata (RFC 8414)
│   ├── google.go              # Google OAuth provider
│   ├── token_store.go         # Token storage interface + memory impl
│   ├── token_refresh.go       # Auto-refresh token source
│   ├── handlers.go            # /authorize, /callback, /token handlers
│   ├── dcr.go                 # Dynamic Client Registration (RFC 7591)
│   └── middleware.go          # Bearer token validation middleware
│
├── gtm/
│   ├── client.go              # GTM API client wrapper
│   ├── accounts.go            # Account operations
│   ├── containers.go          # Container operations
│   ├── workspaces.go          # Workspace operations
│   ├── tags.go                # Tag read operations
│   ├── triggers.go            # Trigger read operations
│   ├── variables.go           # Variable operations
│   ├── folders.go             # Folder operations
│   ├── versions.go            # Version management
│   ├── mutations.go           # Create/Update/Delete operations
│   ├── validation.go          # Input validation
│   ├── errors.go              # Error mapping with retry logic
│   ├── types.go               # GTM type definitions
│   ├── templates.go           # Tag/Trigger parameter templates for LLMs
│   ├── tools.go               # Tool registration (calls RegisterResources/Prompts)
│   ├── tool_*.go              # Individual tool implementations
│   ├── resources.go           # MCP Resource handlers (gtm:// URIs)
│   └── prompts.go             # MCP Prompt handlers
│
├── middleware/
│   └── logging.go             # MCP request logging
│
└── go-sdk-main/               # MCP Go SDK reference (for patterns)
```

---

## MCP Features

### Tools (23 total)

**Read Operations:**
- `list_accounts`, `list_containers`, `list_workspaces`
- `list_tags`, `get_tag`, `list_triggers`, `list_variables`
- `list_folders`, `get_folder_entities`
- `auth_status`, `ping`

**Write Operations:**
- `create_tag`, `update_tag`, `delete_tag`
- `create_trigger`, `update_trigger`, `delete_trigger`
- `create_variable`

**Version Operations:**
- `create_version`, `publish_version`

**Templates:**
- `get_tag_templates` - GA4/HTML tag parameter examples
- `get_trigger_templates` - Trigger type examples

### Resources (6 URI patterns)

```
gtm://accounts
gtm://accounts/{accountId}/containers
gtm://accounts/{accountId}/containers/{containerId}/workspaces
gtm://accounts/.../workspaces/{workspaceId}/tags
gtm://accounts/.../workspaces/{workspaceId}/triggers
gtm://accounts/.../workspaces/{workspaceId}/variables
```

### Prompts (3 workflows)

| Prompt | Arguments | Purpose |
|--------|-----------|---------|
| `audit_container` | accountId, containerId, workspaceId | Analyze workspace for issues |
| `generate_tracking_plan` | accountId, containerId, workspaceId | Generate markdown documentation |
| `suggest_ga4_setup` | goals | Recommend GA4 tag structure |

---

## Authentication

OAuth 2.1 flow with Google as identity provider:

```
Client (Claude) → GTM MCP Server → Google OAuth → User Browser
```

**Endpoints:**
- `/.well-known/oauth-authorization-server` - RFC 8414 metadata
- `/.well-known/oauth-protected-resource` - RFC 9728 metadata
- `/authorize` - Redirect to Google
- `/oauth/callback` - Handle Google redirect
- `/token` - Exchange/refresh tokens
- `/register` - Dynamic Client Registration (RFC 7591)

**Environment Variables:**
```bash
GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=xxx
JWT_SECRET=your-secret-key
BASE_URL=https://mcp.gtmeditor.com
PORT=8080  # optional, default 8080
```

---

## GTM API Reference

**Hierarchy:** Account → Container → Workspace → (Tags, Triggers, Variables)

**Key Rules:**
- All mutations happen at Workspace level (never live container)
- Changes must be versioned before publishing
- Delete/Publish operations require `confirm: true`
- Updates auto-handle fingerprint for concurrency

**External Docs:**
- GTM API Reference: `/home/paolo/code/projects/gtm-api-for-llms/`
- GTM API Discovery: `https://tagmanager.googleapis.com/$discovery/rest?version=v2`

---

## Quick Commands

```bash
# Run locally
PORT=8081 go run main.go

# Test tools via local MCP
# (use mcp__gtm-local__* tools in Claude)

# Deploy to production
bash deploy.sh

# Check production health
curl https://mcp.gtmeditor.com/health

# View production logs
ssh debian@83.228.212.237 "docker logs gtm-mcp-server --tail 100"
```

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.24+ |
| MCP SDK | `github.com/modelcontextprotocol/go-sdk/mcp` |
| GTM API | `google.golang.org/api/tagmanager/v2` |
| OAuth | `golang.org/x/oauth2` |
| URI Templates | `github.com/yosida95/uritemplate/v3` |
| Deployment | Docker + Caddy (auto-TLS) |

---

## External References

- [MCP Specification](https://modelcontextprotocol.io/specification/2025-11-25/)
- [MCP Go SDK](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp)
- [GTM API Discovery](https://tagmanager.googleapis.com/$discovery/rest?version=v2)
- [OAuth 2.1 Draft](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12)
- [RFC 8414 - OAuth Metadata](https://datatracker.ietf.org/doc/html/rfc8414)
- [RFC 7591 - Dynamic Client Registration](https://datatracker.ietf.org/doc/html/rfc7591)
- [RFC 9728 - Protected Resource Metadata](https://datatracker.ietf.org/doc/html/rfc9728)
