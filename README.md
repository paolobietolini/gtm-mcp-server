# GTM MCP Server

**Let AI manage your Google Tag Manager containers.**

Create tags, audit configurations, generate tracking plans, and publish changes—all through natural conversation with Claude or ChatGPT.

**URL:** `https://mcp.gtmeditor.com`

---

## Table of Contents

- [What Can You Do?](#what-can-you-do)
- [Quick Start](#quick-start)
- [Features](#features)
- [Use Cases](#use-cases)
- [How It Works](#how-it-works)
- [Safety Features](#safety-features)
- [Self-Hosting](#self-hosting)
- [Available Tools](#available-tools)
- [Resources & Prompts](#resources--prompts)
- [Better AI Context](#better-ai-context)
- [Architecture](#architecture)
- [Links](#links)
- [Author](#author)
- [License](#license)

---

## What Can You Do?

Ask your AI assistant to:

- *"List all my GTM containers"*
- *"Create a GA4 event tag for form submissions"*
- *"Audit this container for issues and duplicates"*
- *"Generate a tracking plan document for the marketing team"*
- *"Set up ecommerce tracking for purchases"*
- *"Publish the changes we just made"*

No more clicking through the GTM interface. No more copy-pasting configurations. Just describe what you need.

---

## Quick Start

### Claude (Web & Desktop)

**Claude.ai:**
1. Go to **Settings** → **Connectors** → **Add Custom Connector**
2. Enter: `https://mcp.gtmeditor.com`
3. Click **Add** and sign in with Google

**Claude Code (CLI):**
```bash
claude mcp add -t http gtm https://mcp.gtmeditor.com
```

### ChatGPT

1. Go to [OpenAI Apps Platform](https://platform.openai.com/apps)
2. Add an MCP integration with URL: `https://mcp.gtmeditor.com`
3. Authorize with your Google account

---

## Features

### Tag Management
Create and modify any GTM tag type:
- **GA4 Configuration & Events** — Set up Google Analytics 4 with proper measurement IDs
- **Ecommerce Tracking** — Purchase, add-to-cart, view-item events
- **Custom HTML** — Inject scripts, pixels, and custom code
- **Custom Image** — Tracking pixels with cache busting

### Trigger Management
Build triggers for any scenario:
- Page views (all pages or specific URLs)
- Custom dataLayer events
- Click tracking
- Form submissions
- Timer-based triggers
- Trigger groups for complex conditions

### Container Operations
- Browse accounts, containers, and workspaces
- Create versions from workspace changes
- Publish versions to go live
- Organize with folders

### Community Template Gallery
Import templates from Google's Community Template Gallery:
- *"Import the iubenda cookie consent template"*
- *"Add Cookiebot to my container"*
- *"Set up Facebook Pixel using the gallery template"*

The AI will search for the template, find the GitHub repository, and import it automatically.

### AI-Powered Workflows

**Container Audit**
*"Audit my container for issues"* — Analyzes your workspace for:
- Naming inconsistencies
- Duplicate tags
- Orphaned triggers
- Security concerns
- Best practice violations

**Tracking Plan Generation**
*"Generate a tracking plan"* — Creates markdown documentation of:
- All events and their triggers
- Data layer requirements
- Variable definitions
- Implementation notes

**GA4 Setup Recommendations**
*"Help me set up GA4 for ecommerce"* — Recommends:
- Which tags to create
- Trigger configurations
- Required variables
- Data layer implementation code

---

## Use Cases

### Build Complete Tracking Setups
Ask AI to create a full GA4 ecommerce implementation from scratch:
- *"Set up GA4 ecommerce tracking for my store"*
- Creates 12+ tags (configuration + all ecommerce events)
- Creates matching triggers for each dataLayer event
- Creates data layer variables for items, currency, value, transaction_id
- Follows Google's recommended event naming and parameters

### Implement Consent Management
Integrate privacy tools like OneTrust with your tracking:
- *"Make GA4 fire only when analytics consent is granted"*
- Creates consent-checking variables
- Sets up conditional triggers
- Updates existing tags to respect user choices

### Bulk Operations & Renaming
Manage containers at scale:
- *"Add 'ecom -' prefix to all ecommerce triggers"*
- *"Update all tags to use a measurement ID variable"*
- Rename, update, or organize dozens of items through conversation

### Custom Variables & Logic
Create sophisticated tracking logic:
- *"Create a variable that returns the local timestamp"*
- *"Add a custom parameter to the purchase tag"*
- Custom JavaScript variables, data layer mappings, and more

### For Agencies
- Manage multiple client containers (7+ accounts shown in demo)
- Standardize implementations across clients
- Rapid setup for new projects
- Version and publish changes safely

---

## How It Works

The GTM MCP Server connects AI assistants to the Google Tag Manager API using the [Model Context Protocol](https://modelcontextprotocol.io). When you ask Claude or ChatGPT to manage your GTM, it:

1. **Authenticates** with your Google account (OAuth 2.1)
2. **Reads** your container configurations
3. **Executes** the changes you request
4. **Confirms** before destructive operations

Your credentials are never stored—the server uses token-based authentication that you can revoke anytime from your Google account.

---

## Safety Features

- **Confirmation required** for deletions and publishing
- **Workspace-only changes** — nothing goes live until you publish
- **Version control** — all changes create a version first
- **Audit logging** — track what was changed

---

## Self-Hosting

Want to run your own instance?

### Docker Setup

```bash
git clone https://github.com/paolobietolini/gtm-mcp-server.git
cd gtm-mcp-server

# Create .env file
cat > .env << 'EOF'
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
JWT_SECRET=$(openssl rand -base64 32)
BASE_URL=http://localhost:8080
EOF

# Start the server
docker compose up -d

# Add to Claude
claude mcp add -t http gtm http://localhost:8080
```

### Google Cloud Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Enable the **Tag Manager API**
3. Create **OAuth 2.0 credentials** (Web application)
4. Add redirect URIs:
   ```
   https://claude.ai/api/mcp/auth_callback
   https://claude.com/api/mcp/auth_callback
   https://chatgpt.com/connector_platform_oauth_redirect
   https://your-domain.com/oauth/callback
   ```

---

## Available Tools

### Read Operations
| Tool | Description |
|------|-------------|
| `list_accounts` | List all GTM accounts |
| `list_containers` | List containers in an account |
| `list_workspaces` | List workspaces in a container |
| `list_tags` | List all tags in a workspace |
| `get_tag` | Get tag details by ID |
| `list_triggers` | List all triggers |
| `list_variables` | List all variables |
| `list_folders` | List folders in a workspace |
| `get_folder_entities` | Get tags/triggers/variables in a folder |

### Utility
| Tool | Description |
|------|-------------|
| `ping` | Test server connectivity |
| `auth_status` | Check authentication status |

### Write Operations
| Tool | Description |
|------|-------------|
| `create_container` | Create a new container in an account |
| `delete_container` | Remove a container (requires confirmation) |
| `create_workspace` | Create a new workspace in a container |
| `create_tag` | Create a new tag |
| `update_tag` | Modify an existing tag |
| `delete_tag` | Remove a tag (requires confirmation) |
| `create_trigger` | Create a new trigger |
| `update_trigger` | Modify an existing trigger |
| `delete_trigger` | Remove a trigger (requires confirmation) |
| `create_variable` | Create a new variable |
| `delete_variable` | Remove a variable (requires confirmation) |

### Publishing
| Tool | Description |
|------|-------------|
| `list_versions` | List all container versions with tag/trigger/variable counts |
| `create_version` | Create a version from workspace changes |
| `publish_version` | Publish a version (requires confirmation) |

### Templates
| Tool | Description |
|------|-------------|
| `get_tag_templates` | Get GA4/HTML tag parameter examples |
| `get_trigger_templates` | Get trigger configuration examples |
| `list_templates` | List custom templates in a workspace |
| `get_template` | Get template details including template code |
| `create_template` | Create a custom template from .tpl code |
| `update_template` | Modify an existing template |
| `delete_template` | Remove a template (requires confirmation) |
| `import_gallery_template` | Import a template from the Community Gallery |

---

## Resources & Prompts

### Resources (URI-based access)
Access GTM data via structured URIs:
```
gtm://accounts
gtm://accounts/{id}/containers
gtm://accounts/{id}/containers/{id}/workspaces
gtm://accounts/.../workspaces/{id}/tags
gtm://accounts/.../workspaces/{id}/triggers
gtm://accounts/.../workspaces/{id}/variables
```

### Prompts (Workflow templates)
| Prompt | Description |
|--------|-------------|
| `audit_container` | Comprehensive container analysis |
| `generate_tracking_plan` | Markdown documentation generator |
| `suggest_ga4_setup` | GA4 implementation recommendations |
| `find_gallery_template` | Guide to find and import Community Gallery templates |

---

## Better AI Context

For best results, give your AI assistant more GTM context:

- **GTM API Skill:** Add the [GTM API skill](https://github.com/paolobietolini/gtm-api-for-llms/tree/main/skills/gtm-api) to Claude
- **Documentation:** Have the AI read the [GTM API docs](https://github.com/paolobietolini/gtm-api-for-llms)

---

## Architecture

- **Protocol:** Model Context Protocol (MCP) over HTTP
- **Authentication:** OAuth 2.1 with PKCE
- **Standards:** RFC 8414, RFC 7591, RFC 9728

---

## Links

- [GitHub Repository](https://github.com/paolobietolini/gtm-mcp-server)
- [GTM API Reference](https://github.com/paolobietolini/gtm-api-for-llms)
- [MCP Specification](https://modelcontextprotocol.io)

---

## Author

**Paolo Bietolini**

mcp@paolobietolini.com

---

## License

[BSD-3-Clause](LICENSE)
