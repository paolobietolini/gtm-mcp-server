# Google Tag Manager MCP Server

An MCP server for Google Tag Manager that simplifies and accelerates common operations such as tag creation, auditing, container management, and publishing.

**Production URL:** `https://mcp.gtmeditor.com`

To provide better context to your LLM, I recommend having it read the documents in this [repository](https://github.com/paolobietolini/gtm-api-for-llms) or adding the [GTM API skill](https://github.com/paolobietolini/gtm-api-for-llms/tree/main/skills/gtm-api).

---

## Quick Start

Add the server to Claude Code:

```bash
claude mcp add -t http gtm https://mcp.gtmeditor.com
```

On first use, you'll be prompted to authenticate with Google OAuth.

## Available Tools

### Read Operations (Phase 3 - Complete)

| Tool | Description |
|------|-------------|
| `list_accounts` | List all GTM accounts accessible to the user |
| `list_containers` | List containers in an account |
| `list_workspaces` | List workspaces in a container |
| `list_tags` | List all tags in a workspace |
| `get_tag` | Get a specific tag by ID |
| `search_tags` | Search tags by name or type |
| `list_triggers` | List all triggers in a workspace |
| `list_variables` | List all variables in a workspace |

### Utility Tools

| Tool | Description |
|------|-------------|
| `ping` | Test connectivity to the server |
| `auth_status` | Check authentication status |

## Roadmap

- [x] **Phase 1:** HTTP Transport & MCP Foundation
- [x] **Phase 2:** OAuth 2.1 Authentication (Google OAuth + PKCE)
- [x] **Phase 3:** GTM API Read Operations
- [ ] **Phase 4:** GTM API Write Operations (create/update/delete)
- [ ] **Phase 5:** Resources & Prompts (audit templates, tracking plans)
- [ ] **Phase 6:** Production Hardening (rate limiting, metrics)

## Architecture

A remote MCP server operating over HTTP/SSE (Server-Sent Events), enabling centralized management without local installation. OAuth 2.1 with PKCE secures access to Google Tag Manager APIs.

## Status

Started January 2026. Currently in active development.

[Watch](https://github.com/paolobietolini/gtm-mcp-server) for updates or open [Pull Requests](https://github.com/paolobietolini/gtm-mcp-server/pulls) to contribute.

## Author

Paolo Bietolini

<mcp at paolobietolini dot com>

## License

[BSD-3](https://github.com/paolobietolini/gtm-mcp-server/blob/main/LICENSE)
