# GTM MCP Server

[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![MCP](https://img.shields.io/badge/MCP-Streamable_HTTP-8A2BE2)](https://modelcontextprotocol.io/)
[![Security Checks](https://github.com/paolobietolini/gtm-mcp-server/actions/workflows/security.yml/badge.svg)](https://github.com/paolobietolini/gtm-mcp-server/actions/workflows/security.yml)
[![GitHub release](https://img.shields.io/github/v/release/paolobietolini/gtm-mcp-server)](https://github.com/paolobietolini/gtm-mcp-server/releases)

---
> [!IMPORTANT]
> The hosted instance at `mcp.gtmeditor.com` was shut down during the night of
> 20–21 November 2026. There is no longer a public endpoint: to keep using the
> server, [self-host it](#self-hosting). Remove `https://mcp.gtmeditor.com`
> from every MCP client configuration. The domain is not being renewed, so a
> future owner could serve something else at that address. See
> [#139](https://github.com/paolobietolini/gtm-mcp-server/issues/139) for
> background. Thanks to everyone who used it!

---
GTM MCP Server connects MCP clients to the Google Tag Manager API. It can
inspect containers, create and update workspace entities, create versions, and
publish a selected version after explicit confirmation.

It authenticates either with browser-based Google OAuth (each user acts with
their own Google identity and GTM permissions) or with a Google service account
(one shared identity, for automation or a single local user).

## Contents

- [Project status](#project-status)
- [Choose a deployment mode](#choose-a-deployment-mode)
- [Self-hosting](#self-hosting)
- [Connect an MCP client](#connect-an-mcp-client)
- [Typical workflow](#typical-workflow)
- [Tools](#tools)
- [Safety model](#safety-model)
- [Resources](#resources) and [Prompts](#prompts)
- [Skills](#skills)
- [Configuration reference](#configuration-reference)
- [Releases](#releases)
- [Development](#development)

## Project status

| Item | Current state |
|---|---|
| Version in `server.json` | `1.13.0` |
| Transport | MCP Streamable HTTP, or stdio for local service-account use |
| Runtime tools | 64 GTM tools by default; 94 with `GTM_TOOL_GROUPS=all`; plus 2 utility tools in both cases |
| MCP resources | 8 resource definitions (2 resources, 6 URI templates) |
| MCP prompts | 6 prompts |
| Official GTM API coverage | 101 of 106 methods |
| Hosted endpoint | None. `mcp.gtmeditor.com` was shut down on 20–21 November 2026 |

The project implements 101 methods from Google's 106-method GTM v2 discovery
surface. The five `accounts.user_permissions` methods are intentionally
excluded: granting and revoking GTM access is security-sensitive and should not
be managed by an LLM.

## Choose a deployment mode

| Mode | Who it is for | Google identity used | How the client authenticates |
|---|---|---|---|
| **stdio** | One person, on their own machine | A service account | None: the MCP client starts the binary as a local process |
| **HTTP + OAuth** | Teams and agencies; every user keeps their own GTM permissions | Each user's own Google account | Browser sign-in with Google, handled by the MCP client |
| **HTTP + service account** | Automation, CI, agents, or a team that deliberately shares one identity | One service account for every caller | Static header `Authorization: Bearer <SERVICE_ACCOUNT_API_KEY>` |
| **HTTP + OAuth + service account** | Both of the above on one server | Per request: the API key selects the service account, any other bearer token is an OAuth user | Either of the two above |

Where the client connects from matters for remote (HTTP) deployments:

- **Claude Code, Cursor, VS Code, Gemini CLI, Codex CLI** and other local
  clients connect from the user's machine. A hostname reachable only on your
  corporate network or VPN works.
- **Web connectors** (Claude on the web, ChatGPT) connect from the provider's
  cloud. The server must be reachable from the public internet over HTTPS, and
  these connectors use the OAuth flow, so run OAuth mode for them.

## Self-hosting

The steps below describe a production deployment on your own server (VPS,
virtual machine, or on-premises host). Every value written as
`gtm-mcp.example.com`, `PROJECT_ID`, or `<...>` is a placeholder to replace.

### Requirements

- One of the following:
  - Docker with Docker Compose v2 (recommended), or
  - a prebuilt binary from the
    [release page](https://github.com/paolobietolini/gtm-mcp-server/releases)
    (Linux, macOS, and Windows; amd64 and arm64), or
  - Go 1.26 or later to build from source (`go.mod` declares `go 1.26.0`).
- A Google Cloud project in which you can enable APIs and create credentials.
- For OAuth mode: a dedicated hostname (for example `gtm-mcp.example.com`)
  with HTTPS. The server listens on plain HTTP only, so TLS is terminated by a
  reverse proxy in front of it.
- For service-account mode: the right to add a user to the GTM accounts or
  containers the server should manage.

Constraints that follow from how the server works:

- **Serve it at the root of its own hostname.** All routes (`/`,
  `/authorize`, `/oauth/callback`, `/token`, `/register`, `/health`, and
  `/.well-known/...`) are registered at the root. A path prefix such as
  `https://example.com/gtm` is not supported.
- **Run a single instance in OAuth mode.** MCP HTTP requests are stateless,
  but OAuth state, authorization codes, registered clients, and issued tokens
  live in the memory of one process (tokens can additionally be persisted to
  one local file). A second replica does not know the tokens issued by the
  first one, and users get random `401` responses. Service-account-only
  deployments have no such state and can run several replicas.
- **Persist the token store.** Without `TOKEN_STORE_PATH`, every restart or
  redeploy signs all OAuth users out. Platforms with an ephemeral filesystem
  that scale instances up and down (for example Cloud Run) are therefore a poor
  fit for OAuth mode; they work for service-account mode.

### Step 1: Enable the Tag Manager API

In the Google Cloud console, open **APIs & Services → Library**, search for
**Tag Manager API**, and click **Enable**. With the gcloud CLI:

```bash
gcloud services enable tagmanager.googleapis.com --project=PROJECT_ID
```

No Google Cloud IAM role is needed to call the Tag Manager API. What a
credential can do in GTM is decided only by GTM's own user management.

### Step 2a: Create the OAuth client (OAuth mode)

Skip this step if you use only service-account mode.

1. In the Google Cloud console, open **Google Auth Platform** (formerly
   "OAuth consent screen").
2. **Branding**: set the app name users will see on the consent screen, a
   user support email, and a developer contact email.
3. **Audience**: choose who may sign in. This is the server's only sign-in
   restriction: the server itself has no allowlist of users or domains.
   - **Internal** (available only when the project belongs to a Google
     Workspace or Cloud Identity organization): only members of your
     organization can authorize. This is the recommended choice for a company.
     If your Workspace administrator restricts third-party API access, they
     may need to allow this app under **Security → API controls** in the
     Workspace Admin console.
   - **External, publishing status "Testing"**: only the test users you list
     (at most 100) can authorize, and Google expires their authorization
     seven days after consent. Users then have to sign in again each week.
     Suitable only for a trial.
   - **External, publishing status "In production"**: any Google account can
     authorize. Until Google verifies the app, users may see an
     "unverified app" warning and Google may cap the number of users. Check
     the **Verification Center** page for your project's requirements.
4. **Data Access** (recommended): declare the scopes the server requests, so
   the consent screen and any verification request match what users see:

   ```text
   https://www.googleapis.com/auth/tagmanager.delete.containers
   https://www.googleapis.com/auth/tagmanager.edit.containers
   https://www.googleapis.com/auth/tagmanager.edit.containerversions
   https://www.googleapis.com/auth/tagmanager.manage.accounts
   https://www.googleapis.com/auth/tagmanager.publish
   ```

   These five scopes are hard-coded in the server and cannot be reduced by
   configuration. What a user can actually do is still limited by that user's
   GTM permissions.
5. **Clients → Create client**:
   - **Application type**: Web application.
   - **Authorized JavaScript origins**: leave empty.
   - **Authorized redirect URIs**: add exactly one URI, `BASE_URL` followed by
     `/oauth/callback`:

     ```text
     https://gtm-mcp.example.com/oauth/callback
     ```

     For a server running on your own computer, add
     `http://localhost:8080/oauth/callback` instead (or as well).
6. Click **Create** and copy the **client ID** and **client secret**. Google
   shows the secret only at creation time. Store both in your secret manager.

Google's rules for the redirect URI, which explain most
`redirect_uri_mismatch` errors:

- It must match `BASE_URL + /oauth/callback` character for character: scheme,
  host, port, and path. The server always builds it this way; there is no
  separate setting for it.
- It must use `https://`. Only `localhost` is allowed over `http://`.
- The host cannot be a raw IP address, and its top-level domain must be on the
  public suffix list. A name such as `gtm-mcp.example.com` that resolves only
  inside your network is accepted; `gtm-mcp.corp.local` is not.
- Changes in the console can take from a few minutes to a few hours to apply.

### Step 2b: Create the service account (service-account mode and stdio)

Skip this step if you use only OAuth mode.

1. Create the service account:

   ```bash
   gcloud iam service-accounts create gtm-mcp \
     --display-name="GTM MCP Server" \
     --project=PROJECT_ID
   ```

   Its email address is `gtm-mcp@PROJECT_ID.iam.gserviceaccount.com`.
2. Give it access in GTM: in Tag Manager open **Admin → User Management**
   (account level, or container level for a single container), click **Add**,
   enter the service account email, and grant the lowest permission that
   covers your use: container **Read** for audits, **Edit** to change
   workspaces, **Approve** to create versions, **Publish** to publish. Grant
   account **Administrator** only if the server must create or delete
   containers.
3. Choose how the server obtains the service account credentials:
   - **On Google Cloud** (Compute Engine, GKE, Cloud Run): attach the service
     account to the workload. The server uses Application Default
     Credentials, and no key file is needed.
   - **Anywhere else**: create a JSON key:

     ```bash
     gcloud iam service-accounts keys create sa.json \
       --iam-account=gtm-mcp@PROJECT_ID.iam.gserviceaccount.com
     ```

     Many organizations block key creation with the organization policy
     `iam.disableServiceAccountKeyCreation` (enforced by default in newer
     organizations). If the command is refused, ask your Google Cloud
     administrator for an exception or run the server on Google Cloud with an
     attached service account. Treat `sa.json` as a password: anyone holding
     it has the service account's GTM permissions.
4. For HTTP mode, generate the API key that clients will send:

   ```bash
   openssl rand -hex 32
   ```

   Anyone who has this key acts as the service account in GTM. Share it only
   through a secret manager and rotate it by changing the value and
   restarting the server.

### Step 3: Write the configuration

The server is configured only through environment variables. They can also be
placed in a `.env` file; the full list is in the
[configuration reference](#configuration-reference).

OAuth mode for a team (`.env`):

```dotenv
BASE_URL=https://gtm-mcp.example.com
GOOGLE_CLIENT_ID=<client-id>.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=<client-secret>
TOKEN_STORE_PATH=/data/tokens.json
TRUST_PROXY=true
```

Service-account mode (`.env`):

```dotenv
BASE_URL=https://gtm-mcp.example.com
SERVICE_ACCOUNT_API_KEY=<output of openssl rand -hex 32>
GOOGLE_APPLICATION_CREDENTIALS=/secrets/sa.json
TRUST_PROXY=true
```

For both modes on one server, combine the two files. Omit
`GOOGLE_APPLICATION_CREDENTIALS` when the server runs on Google Cloud with an
attached service account.

Rules that avoid the most common mistakes:

- `BASE_URL` is the public HTTPS URL users and clients reach, with **no
  trailing slash** and no path. A trailing slash produces URLs such as
  `https://gtm-mcp.example.com//oauth/callback`, which do not match the
  redirect URI registered with Google.
- Do not put quotes around values. `docker run --env-file` passes quotes
  through as part of the value.
- Values that cannot be parsed are **silently replaced by their defaults**:
  durations (`ACCESS_TOKEN_TTL`, `AUTH_AUTO_REFRESH_MAX_AGE`) must use Go
  duration syntax such as `8h`, `90m`, or `168h` (`7d` is not valid);
  booleans accept `true`/`false` or `1`/`0`. An unknown name in
  `GTM_TOOL_GROUPS` or `MCP_TRANSPORT` instead stops the server at startup.
- `TRUST_PROXY=true` is correct only with **exactly one** reverse proxy in
  front of the server that sets `X-Forwarded-For` (the setups below). Without
  a proxy, leave it `false`.
- When neither OAuth nor an API key is configured, the server starts in
  **open mode**: anyone who can reach it can call the MCP endpoint, and the
  log shows `No authentication configured`. Never expose open mode beyond your
  own machine.

### Step 4: Run the server

Pick one of the three options.

#### Option A: Docker Compose with Caddy (recommended)

Caddy terminates HTTPS and obtains certificates automatically. On the server:

```bash
git clone https://github.com/paolobietolini/gtm-mcp-server.git
cd gtm-mcp-server
git checkout v1.13.0   # use the latest tag from the release page
```

Create `.env` as in step 3 (with `TOKEN_STORE_PATH=/data/tokens.json` for
OAuth mode). The repository's `.gitignore` already excludes `.env`,
`docker-compose.yml`, and `Caddyfile`.

Create `docker-compose.yml`:

```yaml
services:
  gtm-mcp:
    build: .
    image: gtm-mcp-server:local
    restart: unless-stopped
    env_file: .env
    volumes:
      - gtm-mcp-data:/data
      # Service-account mode with a key file only:
      # - ./secrets/sa.json:/secrets/sa.json:ro

  caddy:
    image: caddy:2
    restart: unless-stopped
    depends_on:
      - gtm-mcp
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
      - caddy-config:/config

volumes:
  gtm-mcp-data:
  caddy-data:
  caddy-config:
```

Create `Caddyfile`:

```caddyfile
gtm-mcp.example.com {
	reverse_proxy gtm-mcp:8080
}
```

Start it:

```bash
docker compose up -d --build
docker compose logs -f gtm-mcp
```

Notes for this setup:

- The `gtm-mcp` service publishes no port; only Caddy is reachable from
  outside.
- Caddy can obtain a public certificate only when `gtm-mcp.example.com`
  resolves publicly to this machine and ports 80 and 443 are reachable from the
  internet. For a host reachable only internally, replace the site block's
  automatic certificate with your organization's certificate
  (`tls /path/to/cert.pem /path/to/key.pem`, mounted into the Caddy container)
  or use a Caddy DNS-challenge plugin.
- Keep `PORT` unset (8080) inside the container: the image's `HEALTHCHECK`
  always probes `http://localhost:8080/health`.
- The image runs as the non-root user `appuser` and creates `/data` owned by
  that user with mode `700`. A named volume, as above, inherits that ownership
  on first use. If you bind-mount a host directory to `/data` instead, make it
  owned by the container user's UID.
- A mounted `sa.json` must be readable by the container user. Find its UID
  with `docker compose run --rm --no-deps --entrypoint id gtm-mcp`, then run
  `sudo chown <uid> secrets/sa.json && chmod 400 secrets/sa.json`.
- To upgrade: `git fetch --tags && git checkout <new tag> && docker compose up -d --build`.
  Tokens survive in the `gtm-mcp-data` volume.

If you already run a reverse proxy on the host, skip Caddy and publish the
server on the loopback interface only:

```bash
docker build -t gtm-mcp-server:local .

docker run -d \
  --name gtm-mcp-server \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  --env-file .env \
  -v gtm-mcp-data:/data \
  gtm-mcp-server:local
```

#### Option B: Prebuilt binary with systemd

```bash
VERSION=1.13.0   # latest version from the release page
ARCH=amd64       # or arm64
curl -fsSLO "https://github.com/paolobietolini/gtm-mcp-server/releases/download/v${VERSION}/gtm-mcp-server_${VERSION}_linux_${ARCH}.tar.gz"
curl -fsSLO "https://github.com/paolobietolini/gtm-mcp-server/releases/download/v${VERSION}/gtm-mcp-server_${VERSION}_checksums.txt"
sha256sum --check --ignore-missing "gtm-mcp-server_${VERSION}_checksums.txt"
tar -xzf "gtm-mcp-server_${VERSION}_linux_${ARCH}.tar.gz"
sudo install -m 0755 gtm-mcp-server /usr/local/bin/gtm-mcp-server
```

Create a system user, a data directory, and the configuration file:

```bash
sudo useradd --system --home-dir /var/lib/gtm-mcp --shell /usr/sbin/nologin gtm-mcp
sudo install -d -o gtm-mcp -g gtm-mcp -m 0700 /var/lib/gtm-mcp
sudo install -d -o root -g gtm-mcp -m 0750 /etc/gtm-mcp
sudoedit /etc/gtm-mcp/gtm-mcp.env
sudo chown root:gtm-mcp /etc/gtm-mcp/gtm-mcp.env
sudo chmod 0640 /etc/gtm-mcp/gtm-mcp.env
```

Use the content from step 3 in `/etc/gtm-mcp/gtm-mcp.env`, with
`TOKEN_STORE_PATH=/var/lib/gtm-mcp/tokens.json` and, for a key file,
`GOOGLE_APPLICATION_CREDENTIALS=/etc/gtm-mcp/sa.json` (owner `root:gtm-mcp`,
mode `0640`).

Create `/etc/systemd/system/gtm-mcp.service`:

```ini
[Unit]
Description=GTM MCP Server
After=network-online.target
Wants=network-online.target

[Service]
User=gtm-mcp
Group=gtm-mcp
WorkingDirectory=/var/lib/gtm-mcp
EnvironmentFile=/etc/gtm-mcp/gtm-mcp.env
ExecStart=/usr/local/bin/gtm-mcp-server
Restart=on-failure
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/gtm-mcp

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gtm-mcp
journalctl -u gtm-mcp -f
```

The binary listens on `PORT` (default 8080) on all interfaces. Put a reverse
proxy in front of it (see below) and block direct access to that port with
your firewall.

#### Option C: From source

Requires Go 1.26 or later.

```bash
git clone https://github.com/paolobietolini/gtm-mcp-server.git
cd gtm-mcp-server
git checkout v1.13.0   # or stay on main for unreleased changes
go build -o gtm-mcp-server .
./gtm-mcp-server
```

The server reads `.env` from the current working directory, then
`.env.local`. Variables already set in the process environment take
precedence over `.env`, and `.env.local` overrides both. Both files are ignored
by Git. For quick local testing, `go run .` with this `.env` is enough:

```dotenv
BASE_URL=http://localhost:8080
GOOGLE_CLIENT_ID=<client-id>.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=<client-secret>
TOKEN_STORE_PATH=./data/tokens.json
```

#### Reverse proxy requirements

Any reverse proxy works if it:

- forwards every path of the hostname to the server unchanged;
- passes the `Authorization` header through;
- does not buffer responses: the MCP endpoint can answer with a
  `text/event-stream` (Server-Sent Events) response;
- allows request bodies up to 5 MiB, the server's own limit;
- sets `X-Forwarded-For` (required for per-client rate limits with
  `TRUST_PROXY=true`).

Caddy does all of this with the two-line `reverse_proxy` configuration above.
For nginx:

```nginx
server {
    listen 443 ssl;
    server_name gtm-mcp.example.com;

    ssl_certificate     /etc/ssl/gtm-mcp/fullchain.pem;
    ssl_certificate_key /etc/ssl/gtm-mcp/privkey.pem;

    client_max_body_size 5m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 1h;
    }
}
```

`X-Forwarded-For $remote_addr` overwrites any value sent by the client, which
is what the server expects with one proxy hop. If a load balancer or CDN sits
in front of this proxy, the rightmost `X-Forwarded-For` entry is that
intermediary's address, so all users share one rate-limit bucket: configure
your proxy to restore the real client address first.

### Step 5: Verify the deployment

1. Health check, which needs no authentication:

   ```bash
   curl -sS https://gtm-mcp.example.com/health
   ```

   Expected: `{"service":"gtm-mcp-server","status":"healthy","version":"1.13.0"}`.
2. OAuth metadata (OAuth mode):

   ```bash
   curl -sS https://gtm-mcp.example.com/.well-known/oauth-protected-resource
   ```

   `resource` must be `https://gtm-mcp.example.com/` and
   `authorization_servers` must contain `https://gtm-mcp.example.com`. If they
   show `http://` or another host, `BASE_URL` is wrong.
3. Unauthenticated MCP request:

   ```bash
   curl -sS -i -X POST https://gtm-mcp.example.com/
   ```

   Expected: `HTTP/... 401` with the header
   `WWW-Authenticate: Bearer resource_metadata="https://gtm-mcp.example.com/.well-known/oauth-protected-resource"`.
   Any other status means the server is running in open mode.
4. API key (service-account mode):

   ```bash
   curl -sS https://gtm-mcp.example.com/ \
     -H "Authorization: Bearer $SERVICE_ACCOUNT_API_KEY" \
     -H "Content-Type: application/json" \
     -H "Accept: application/json, text/event-stream" \
     -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"curl","version":"1.0"}}}'
   ```

   Expected: a JSON-RPC result containing `"serverInfo":{"name":"gtm-mcp-server",...}`,
   possibly wrapped in an SSE `data:` line. This checks the API key only, not
   the Google credentials.
5. Startup log lines to look for (JSON on stderr):
   - `OAuth configured` and `token persistence enabled` in OAuth mode;
   - `s2s_mode_enabled` with `credential_source` `key_json` or
     `workload_identity` (the latter also covers `GOOGLE_APPLICATION_CREDENTIALS`)
     in service-account mode. If the credentials cannot be loaded, the server
     logs `s2s_mode_failed` and exits.
6. End-to-end: connect a client (next section) and ask it to *"List my GTM
   accounts. Do not make changes."* An empty list means the authenticated
   Google identity has no GTM access yet (step 2b, item 2, for a service
   account). An error saying the Tag Manager API is disabled or has not been
   used in the project means step 1 was skipped for the project that owns the
   OAuth client or service account.

### Operating notes

- **Token file.** `TOKEN_STORE_PATH` holds every user's MCP tokens and Google
  OAuth tokens, including Google refresh tokens, as plain JSON (file mode
  `0600`, written atomically). Treat it like a password database: restrict
  access to the host, and exclude it from unencrypted backups. Deleting it
  signs everyone out.
- **Revoking a user.** A user can revoke the app's access at
  <https://myaccount.google.com/permissions>. The server can then no longer
  refresh that user's Google token, so their GTM calls fail and they must sign
  in again. Removing a user from GTM's User Management removes their GTM
  access regardless of what the server holds.
- **Session lifetime.** MCP access tokens last `ACCESS_TOKEN_TTL` (8 hours by
  default). After expiry the server renews them silently while the Google
  refresh token is valid, for at most `AUTH_AUTO_REFRESH_MAX_AGE` (7 days)
  after issue; then the client must complete a refresh, or a new sign-in.
- **Rate limits.** `/authorize`, `/oauth/callback`, and `/token` allow 10
  requests per second per client IP (burst 20); `/register` allows 2 per
  second (burst 5).
- **Logs.** Set `LOG_LEVEL=debug` for per-request authentication details.
  Authentication logs identify tokens by a fingerprint, not by their value.

### Stdio mode (single local user)

For one person on their own machine, the MCP client can start the server as a
local process over stdin/stdout. No HTTP server, OAuth client, reverse proxy,
or API key is involved. Stdio mode always uses service-account credentials
(step 2b): `GOOGLE_SERVICE_ACCOUNT_KEY_JSON`, or a key file named by
`GOOGLE_APPLICATION_CREDENTIALS`, or Application Default Credentials.

Install the binary (Option B above, first block only), then test it:

```sh
GOOGLE_APPLICATION_CREDENTIALS=/path/to/sa.json gtm-mcp-server --stdio
```

The process waits for MCP messages on stdin; stop it with `Ctrl+C`.

Claude Code example (`.mcp.json`):

```json
{
  "mcpServers": {
    "gtm": {
      "command": "/usr/local/bin/gtm-mcp-server",
      "args": ["--stdio"],
      "env": {
        "GOOGLE_APPLICATION_CREDENTIALS": "/path/to/sa.json",
        "GTM_TOOL_GROUPS": "all"
      }
    }
  }
}
```

`MCP_TRANSPORT=stdio` is equivalent to the `--stdio` argument. Logs go to
stderr; stdout carries only MCP messages. The server also reads `.env` from the
directory the client starts it in.

### Optional setup dashboard

Set `DASHBOARD_ENABLED=true` and restart the server to expose `/dashboard/`.
The page shows the canonical MCP endpoint, server version, authentication setup
status, and connection instructions. It is disabled by default.

The dashboard is public when enabled. It deliberately excludes tokens,
sessions, account names, container data, and other GTM content. It reports
server configuration only and does not verify the visitor's Google session.

## Connect an MCP client

The MCP endpoint is your `BASE_URL` itself, for example
`https://gtm-mcp.example.com`. In OAuth mode the client discovers the OAuth
metadata, opens Google sign-in in the browser, and reconnects with the issued
bearer token. In service-account mode the client sends the API key in the
`Authorization` header on every request.

### Claude Code

```bash
# OAuth mode
claude mcp add --transport http gtm https://gtm-mcp.example.com

# Service-account mode
claude mcp add --transport http gtm https://gtm-mcp.example.com \
  --header "Authorization: Bearer <SERVICE_ACCOUNT_API_KEY>"
```

### Gemini CLI

```bash
gemini mcp add --transport http gtm https://gtm-mcp.example.com
```

Or in `settings.json` (remove `headers` for OAuth mode):

```json
{
  "mcpServers": {
    "gtm": {
      "httpUrl": "https://gtm-mcp.example.com",
      "headers": {
        "Authorization": "Bearer <SERVICE_ACCOUNT_API_KEY>"
      }
    }
  }
}
```

### Cursor

Add the following to `.cursor/mcp.json` (remove `headers` for OAuth mode):

```json
{
  "mcpServers": {
    "gtm": {
      "url": "https://gtm-mcp.example.com",
      "headers": {
        "Authorization": "Bearer <SERVICE_ACCOUNT_API_KEY>"
      }
    }
  }
}
```

### VS Code

Add the following to `.vscode/mcp.json` (OAuth mode):

```json
{
  "servers": {
    "gtm": {
      "type": "http",
      "url": "https://gtm-mcp.example.com"
    }
  }
}
```

### ChatGPT, Codex, and Claude on the web

Add a custom or remote MCP connection in the client and use your `BASE_URL` as
the server URL. Web connectors require a server reachable from the public
internet and OAuth mode. Product menus change more often than this server, so
refer to the client's current MCP connection instructions if the label
differs.

After connecting, try:

```text
List my GTM accounts, containers, and workspaces. Do not make changes.
```

The Google identity used (the account chosen at sign-in, or the service
account) determines which GTM accounts the server can access.

## Typical workflow

1. Call `list_accounts`, `list_containers`, and `list_workspaces` to discover
   IDs. Do not guess IDs.
2. Inspect the workspace with list/get tools or the `audit_container` prompt.
3. Create or update tags, triggers, variables, templates, clients, or
   transformations in the selected workspace.
4. Call `get_workspace_status` and resolve conflicts before versioning.
5. Call `create_version` to snapshot the workspace.
6. Inspect the saved version with `get_version`.
7. Call `publish_version` with `confirm: true` only when the selected version
   is ready to go live.
8. Verify the published state with `get_live_version`.

Example requests:

- "Audit this workspace for duplicate tags and unused triggers."
- "Create a GA4 purchase event tag, but do not publish it."
- "Show the difference between the latest saved version and the live version."
- "Generate a Markdown tracking plan from this workspace."
- "Import the iubenda template from the Community Template Gallery."

## Tools

All GTM tools use the authenticated Google identity from the current MCP
request. Inputs and outputs are structured JSON.

Tools marked *opt-in* belong to a tool group that is not enabled by default.
Enable it through `GTM_TOOL_GROUPS` (see [Tool groups](#tool-groups)).

### Utility

Always available, regardless of `GTM_TOOL_GROUPS`.

| Tool | Purpose |
|---|---|
| `ping` | Test MCP connectivity |
| `auth_status` | Check whether the current request is authenticated |

### Accounts and containers

| Tool | Purpose |
|---|---|
| `list_accounts` | List accessible GTM accounts |
| `update_account` | Rename an account |
| `list_containers` | List containers and their public IDs and settings |
| `lookup_container` | Find a container by destination ID or GTM public tag ID |
| `get_container_snippet` | Get a web install snippet or server-container configuration |
| `create_container` | Create a web, app, AMP, or server container |
| `update_container` | Rename a container while preserving its other settings |
| `delete_container` | Permanently delete a container; requires `confirm: true` |
| `combine_containers` | Merge a source container into a target; requires `confirm: true`. *Opt-in: `container-admin`* |
| `move_tag_id` | Move a tag ID into a new container; requires confirmation and terms acceptance. *Opt-in: `container-admin`* |

### Workspaces

| Tool | Purpose |
|---|---|
| `list_workspaces` | List workspaces in a container |
| `create_workspace` | Create a workspace |
| `get_workspace` | Get workspace metadata and its current fingerprint |
| `update_workspace` | Update selected workspace fields |
| `delete_workspace` | Delete a workspace; requires `confirm: true` |
| `quick_preview_workspace` | Compile an ephemeral preview without saving or publishing |
| `get_workspace_status` | Show pending changes and merge conflicts |
| `bulk_update_workspace` | Apply multiple entity changes; requires `confirm: true`. *Opt-in: `workspace-admin`* |
| `resolve_workspace_conflict` | Replace a conflict with a resolved entity; requires `confirm: true`. *Opt-in: `workspace-admin`* |
| `sync_workspace` | Synchronize with the latest container version; requires `confirm: true`. *Opt-in: `workspace-admin`* |

Bulk update and conflict resolution accept raw GTM Entity JSON so every entity
type supported by the official API remains available.

### Tags

| Tool | Purpose |
|---|---|
| `list_tags` | List workspace tags |
| `get_tag` | Get complete tag details |
| `create_tag` | Create a tag |
| `update_tag` | Update a tag with fingerprint-based concurrency control |
| `delete_tag` | Delete a tag; requires `confirm: true` |

### Triggers

| Tool | Purpose |
|---|---|
| `list_triggers` | List workspace triggers |
| `get_trigger` | Get complete trigger details |
| `create_trigger` | Create a trigger |
| `update_trigger` | Update a trigger with fingerprint-based concurrency control |
| `delete_trigger` | Delete a trigger; requires `confirm: true` |

For `update_trigger`, omit `filterJson`, `customEventFilterJson`,
`autoEventFilterJson`, or `parameterJson` to preserve the current field. Pass
the JSON string `"[]"` to clear a field, or a non-empty JSON array to replace
it. For click, link-click, and form-submission triggers, use `filterJson` because
GTM drops `autoEventFilter` for those trigger types.

### Variables

| Tool | Purpose |
|---|---|
| `list_variables` | List workspace variables |
| `get_variable` | Get complete variable details |
| `create_variable` | Create a variable |
| `update_variable` | Update a variable with fingerprint-based concurrency control |
| `delete_variable` | Delete a variable; requires `confirm: true` |

### Folders and built-in variables

| Tool | Purpose |
|---|---|
| `list_folders` | List workspace folders |
| `get_folder_entities` | List tags, triggers, and variables assigned to a folder |
| `get_folder` | Get complete folder metadata. *Opt-in: `folder-admin`* |
| `create_folder` | Create a folder. *Opt-in: `folder-admin`* |
| `update_folder` | Update a folder with fingerprint protection. *Opt-in: `folder-admin`* |
| `delete_folder` | Delete a folder; requires `confirm: true`. *Opt-in: `folder-admin`* |
| `move_entities_to_folder` | Move tags, triggers, and variables; requires `confirm: true`. *Opt-in: `folder-admin`* |
| `revert_folder` | Discard workspace folder changes; requires `confirm: true`. *Opt-in: `folder-admin`* |
| `list_built_in_variables` | List enabled built-in variables |
| `enable_built_in_variables` | Enable built-in variable types |
| `disable_built_in_variables` | Disable built-in variable types; requires `confirm: true` |

Use `revert_workspace_entity` (opt-in group `reverts`) to discard changes to a
built-in variable.

### Zones

| Tool | Purpose |
|---|---|
| `list_zones` | List all zones across every result page |
| `get_zone` | Get boundary, child-container, and type-restriction configuration |
| `create_zone` | Create a workspace zone |
| `update_zone` | Update selected fields with fingerprint concurrency control |
| `delete_zone` | Delete a zone; requires `confirm: true` |

Use `revert_workspace_entity` (opt-in group `reverts`) to discard changes to a
zone.

### Environments

All tools in this section are *opt-in* (group `environments`).

| Tool | Purpose |
|---|---|
| `list_environments` | List all container environments across every result page |
| `get_environment` | Get environment configuration and authorization metadata |
| `create_environment` | Create a user environment |
| `update_environment` | Update selected fields with fingerprint concurrency control |
| `reauthorize_environment` | Rotate the authorization code; requires `confirm: true` |
| `delete_environment` | Delete a user environment; requires `confirm: true` |

The Live and Latest environments are managed by GTM. Creation and deletion
apply to user environments; GTM permits URL and debug updates on other types.

### Destinations and Google tag configurations

All tools in this section are *opt-in*: the destination tools are in group
`destinations`, the Google tag configuration tools in group `gtag`.

| Tool | Purpose |
|---|---|
| `list_destinations` | List Google tag destinations linked to a container |
| `get_destination` | Get a destination by its link ID |
| `link_destination` | Move a destination to a container; requires `confirm: true` |
| `list_google_tag_configs` | List Google tag configurations in a workspace |
| `get_google_tag_config` | Get a Google tag configuration |
| `create_google_tag_config` | Create a Google tag configuration |
| `update_google_tag_config` | Update a configuration with fingerprint protection |
| `delete_google_tag_config` | Delete a configuration; requires `confirm: true` |

Container combine, tag-ID move, and destination link operations do not copy or
enable user permissions. Account permission management remains outside the
current parity target.

### Custom templates

| Tool | Purpose |
|---|---|
| `list_templates` | List custom templates |
| `get_template` | Get template metadata and template code |
| `create_template` | Create a custom template from `.tpl` code |
| `update_template` | Update template code |
| `delete_template` | Delete an unused template; requires `confirm: true` |
| `import_gallery_template` | Import a Community Template Gallery template |
| `get_tag_templates` | Return compact GA4 and Custom HTML input examples |
| `get_trigger_templates` | Return compact trigger input examples |

The full JSON examples live in the
`gtm://best-practices/tool-input-formats` resource so every tool listing does
not repeat them.

### Server-side containers

| Tool | Purpose |
|---|---|
| `list_clients` | List server-container clients |
| `get_client` | Get a client |
| `create_client` | Create a client |
| `update_client` | Update a client |
| `delete_client` | Delete a client; requires `confirm: true` |
| `list_transformations` | List transformations |
| `get_transformation` | Get a transformation |
| `create_transformation` | Create a transformation |
| `update_transformation` | Update a transformation |
| `delete_transformation` | Delete a transformation; requires `confirm: true` |

Use `revert_workspace_entity` (opt-in group `reverts`) to discard changes to a
client or transformation.

### Versions and publication

| Tool | Purpose |
|---|---|
| `list_versions` | List all saved version headers across every result page |
| `get_latest_version_header` | Get the latest saved header, which may differ from live |
| `get_version` | Get a saved version and its complete entity collections |
| `get_live_version` | Get the currently published version and its entities |
| `create_version` | Create a version from a conflict-free workspace |
| `publish_version` | Publish a selected version; requires `confirm: true` |
| `update_version` | Update a saved version's name or description. *Opt-in: `version-admin`* |
| `delete_version` | Soft-delete a version; requires `confirm: true`. *Opt-in: `version-admin`* |
| `undelete_version` | Restore a soft-deleted version; requires `confirm: true`. *Opt-in: `version-admin`* |
| `set_latest_version` | Make a version Latest without publishing; requires `confirm: true`. *Opt-in: `version-admin`* |

### Workspace reverts

*Opt-in* (group `reverts`).

| Tool | Purpose |
|---|---|
| `revert_workspace_entity` | Discard workspace changes to a built-in variable, client, tag, template, transformation, trigger, variable, or zone; requires `confirm: true` |

The tool fetches the current entity fingerprint before calling the matching
official revert method. A successful result can have `existsAfterRevert: false`
when the entity does not exist in the latest container version.

## Safety model

- Delete operations, publication, and disabling built-in variables require
  `confirm: true`.
- Update operations fetch the current resource fingerprint and use Google's
  optimistic concurrency checks.
- `create_version` first checks that the workspace has changes and no merge
  conflicts.
- Read tools distinguish latest saved state from published live state.
- Google API errors are mapped to clearer not-found, permission, conflict, and
  rate-limit failures. Retryable API failures use bounded backoff.
- MCP request bodies are limited to 5 MiB. `/token` and `/register` accept at
  most 1 MiB and, with the other OAuth endpoints, are rate limited per client
  IP.

Most entity edits affect a workspace and are not live until a version is
published. Account and container operations act directly on those resources.
Always review the target IDs and the generated version before publication.

## Resources

The server exposes two concrete resources and six URI templates. Resources and
prompts are always available, regardless of `GTM_TOOL_GROUPS`.

| URI | Content |
|---|---|
| `gtm://accounts` | Accessible accounts |
| `gtm://accounts/{accountId}/containers` | Containers |
| `gtm://accounts/{accountId}/containers/{containerId}/workspaces` | Workspaces |
| `gtm://accounts/{accountId}/containers/{containerId}/workspaces/{workspaceId}/tags` | Tags |
| `gtm://accounts/{accountId}/containers/{containerId}/workspaces/{workspaceId}/triggers` | Triggers |
| `gtm://accounts/{accountId}/containers/{containerId}/workspaces/{workspaceId}/variables` | Variables |
| `gtm://best-practices` | Best-practice topic index |
| `gtm://best-practices/{topic}` | One embedded guidance document |

Best-practice topics include naming and organization, safe edits, GA4 and
consent, server-side containers, and detailed tool input formats.

## Prompts

| Prompt | Purpose |
|---|---|
| `audit_container` | Review tags, triggers, and variables for quality problems |
| `generate_tracking_plan` | Build a Markdown tracking plan from a workspace |
| `suggest_ga4_setup` | Recommend a GA4 structure from stated goals |
| `find_gallery_template` | Guide Community Gallery discovery and import |
| `best_practices_review` | Score a workspace against embedded guidance |
| `plan_safe_edit` | Produce a staged edit/version/publish plan |

Prompts prepare context and instructions for the model. They do not bypass tool
authentication or mutation safeguards.

## Skills

[`skills/`](skills) holds [Agent Skills](https://agentskills.io) with the GTM
know-how, usable without this server. Each one works with the MCP tools or with
a container export (GTM > Admin > Export Container) pasted or attached in a chat.

| Skill | Use it for |
|---|---|
| [`gtm-container-audit`](skills/gtm-container-audit/SKILL.md) | Audit and scored best-practices review |
| [`gtm-safe-edit`](skills/gtm-safe-edit/SKILL.md) | Planning, applying, publishing and rolling back changes |
| [`gtm-tracking-plan`](skills/gtm-tracking-plan/SKILL.md) | Tracking plans and data layer specs |
| [`gtm-ga4-consent`](skills/gtm-ga4-consent/SKILL.md) | GA4 setup and debugging, consent mode v2 |
| [`gtm-server-side`](skills/gtm-server-side/SKILL.md) | Server containers, data flow, transformations |
| [`gtm-mcp`](skills/gtm-mcp/SKILL.md) | Operating this server's tools; only useful with the server |

Each skill folder is self-contained: install all of them or only some.

### Claude Code

Install them as a plugin from this repository:

```text
/plugin marketplace add paolobietolini/gtm-mcp-server
/plugin install gtm-skills@gtm-mcp-server
```

Plugin skills are namespaced, for example `/gtm-skills:gtm-safe-edit`. The
plugin install copies the whole repository into Claude Code's plugin cache.

Alternatively, from a clone, copy the folders into your personal skills
directory (or into `.claude/skills/` of a project):

```bash
mkdir -p ~/.claude/skills
cp -r skills/gtm-* ~/.claude/skills/
```

### claude.ai

Code execution and file creation must be enabled (Settings > Capabilities; on
Team and Enterprise plans an owner controls this and can turn off custom
skills). Build one ZIP per skill, with the skill folder at the ZIP root:

```bash
mkdir -p dist
(cd skills && for s in gtm-*; do zip -qr "../dist/$s.zip" "$s"; done)
```

Then open **Customize > Skills**, click **+**, choose **Create skill** >
**Upload a skill**, upload a ZIP, and turn the skill on.

### Other clients

Copy the `skills/gtm-*` folders into the directory your client reads:

| Client | Project | Personal |
|---|---|---|
| [Codex](https://learn.chatgpt.com/docs/build-skills) | `.agents/skills/` | `~/.agents/skills/` |
| [VS Code / GitHub Copilot](https://code.visualstudio.com/docs/copilot/customization/agent-skills) | `.github/skills/` | `~/.copilot/skills/` |

For other clients that support the standard, see their documentation and the
[Agent Skills site](https://agentskills.io).

## Configuration reference

All settings are environment variables, optionally loaded from `.env` and
`.env.local` (see [Option C](#option-c-from-source) for precedence).

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port (all interfaces). Leave at `8080` in Docker |
| `BASE_URL` | `http://localhost:8080` | Public URL of the server, no trailing slash. Used for OAuth metadata and as the base of the redirect URI `BASE_URL/oauth/callback` |
| `MCP_TRANSPORT` | `http` | `http` (Streamable HTTP) or `stdio`; the `--stdio` argument overrides it |
| `MCP_STATELESS_HTTP` | `true` | Disable in-memory MCP HTTP sessions; required for MCP `2026-07-28` |
| `DASHBOARD_ENABLED` | `false` | Expose the public connection setup page at `/dashboard/` |
| `GOOGLE_CLIENT_ID` | empty | Google OAuth web-client ID. With `GOOGLE_CLIENT_SECRET`, enables OAuth mode |
| `GOOGLE_CLIENT_SECRET` | empty | Google OAuth web-client secret |
| `ACCESS_TOKEN_TTL` | `8h` | Lifetime of MCP access tokens (Go duration) |
| `AUTH_AUTO_REFRESH_MAX_AGE` | `168h` | Maximum age of a bearer token that is still renewed silently (Go duration) |
| `TOKEN_STORE_PATH` | empty | File that persists issued tokens across restarts; empty keeps them in memory only. The directory is created with mode `0700` if missing |
| `SERVICE_ACCOUNT_API_KEY` | empty | Bearer API key that enables service-account mode over HTTP |
| `GOOGLE_SERVICE_ACCOUNT_KEY_JSON` | empty | Service-account key as a JSON string. Takes precedence over Application Default Credentials |
| `GOOGLE_APPLICATION_CREDENTIALS` | empty | Standard Google variable: path to a service-account key file, used through Application Default Credentials |
| `ALLOWED_HOSTS` | empty | Comma-separated extra `host[:port]` values trusted for URL resolution (Docker-internal names) |
| `TRUST_PROXY` | `false` | Use the rightmost `X-Forwarded-For` entry as the client IP for rate limiting |
| `LOG_LEVEL` | `info` | `debug` enables debug logs; any other value means `info` |
| `GTM_TOOL_GROUPS` | 11 default groups | Comma-separated tool groups advertised through MCP, or `all` |

`GOOGLE_REDIRECT_URI` and `JWT_SECRET` are read for backward compatibility but
have no effect.

Service-account credentials are looked up in this order:
`GOOGLE_SERVICE_ACCOUNT_KEY_JSON`, then Application Default Credentials (the
file in `GOOGLE_APPLICATION_CREDENTIALS`, then the file created by
`gcloud auth application-default login`, then the service account attached on
Google Cloud). In HTTP mode they are loaded
only when `SERVICE_ACCOUNT_API_KEY` is set; in stdio mode they are always
required.

### HTTP sessions

Streamable HTTP is stateless by default: the server keeps no in-memory MCP
session and ignores `Mcp-Session-Id`, so MCP requests survive a restart. This
is required for MCP `2026-07-28` tool requests; legacy MCP `2025-11-25`
clients remain supported. Stateless mode does not disable authentication:
clients send credentials on every MCP request. OAuth tokens are still held by
a single process (see [Self-hosting](#self-hosting)). Set
`MCP_STATELESS_HTTP=false` only if a client depends on HTTP sessions or the
GET event stream; this server makes no server-to-client requests.

### Proxies and hosts

`TRUST_PROXY=true` is appropriate only when a trusted reverse proxy overwrites
or appends `X-Forwarded-For`. The implementation uses the rightmost value,
which assumes exactly one proxy hop. With several hops, configure and test the
trust boundary before relying on per-client rate limits.

`ALLOWED_HOSTS` is a comma-separated allowlist used when the same server is
reached through trusted internal Docker hostnames (for example
`gtm-mcp:8080`). For a request whose `Host` header is on the list, OAuth
metadata is built from that host instead of `BASE_URL`. Do not add arbitrary
public hosts.

### Tool groups

`GTM_TOOL_GROUPS` controls schema size for clients that need only part of the
API. Names are case-insensitive; an unknown name stops the server at startup.

| Group | Enabled by default | GTM tools |
|---|---|---|
| `accounts` | yes | 8 |
| `workspaces` | yes | 13 (workspaces plus `list_versions`, `get_latest_version_header`, `get_version`, `get_live_version`, `create_version`, `publish_version`) |
| `tags` | yes | 5 |
| `triggers` | yes | 5 |
| `variables` | yes | 5 |
| `folders` | yes | 2 |
| `builtins` | yes | 3 |
| `zones` | yes | 5 |
| `templates` | yes | 6 |
| `server` | yes | 10 |
| `guidance` | yes | 2 |
| `environments` | no | 6 |
| `destinations` | no | 3 |
| `gtag` | no | 5 |
| `container-admin` | no | 2 |
| `folder-admin` | no | 6 |
| `workspace-admin` | no | 3 |
| `version-admin` | no | 4 |
| `reverts` | no | 1 |

Leaving the variable unset enables the 11 default groups (64 tools). An
explicit list **replaces** the defaults instead of adding to them, so to keep
the defaults and add one opt-in group, list them all:

```text
GTM_TOOL_GROUPS=accounts,workspaces,tags,triggers,variables,folders,builtins,zones,templates,server,guidance,environments
```

Use `all` to enable every group (94 tools), including groups added in future
versions. A subset is also valid:

```text
GTM_TOOL_GROUPS=accounts,workspaces,tags,triggers,variables
```

The two utility tools, the resources, and the prompts are always available.

## Releases

`server.json` is the source of truth for the runtime version. It is embedded in
the Go binary and returned by `/health`; there is no second version constant in
Go code.

To publish a release:

1. Update `server.json`.
2. Commit the release changes.
3. Push a matching tag such as `v1.13.0`.

The release workflow rejects a tag that does not match `server.json`, then
creates cross-platform archives and a checksums file with GoReleaser and
attaches them to the GitHub release. It does not deploy anything.

Recent commits on `main` can be newer than the latest tagged release. Check the
[release page](https://github.com/paolobietolini/gtm-mcp-server/releases) when
you need a reproducible published artifact.

## Development

```bash
go test ./... -count=1
go vet ./...
staticcheck ./...
go run ./cmd/tool-schema-report
go run ./cmd/tool-schema-report -groups tags,zones
```

The schema report prints the GTM tool count, serialized `tools/list` size, token
estimate, and largest definitions. The default GTM tool surface serializes to
78,734 bytes for 64 tools. A regression test enforces an 80,000-byte ceiling so
new parity work does not silently consume unlimited model context. The `all`
group exposes 94 GTM tools and serializes to 115,748 bytes.

### Editing skills and best-practice documents

`skills/` is the only source: the server embeds these folders and reads its
best-practice resources and prompt guidance from them. A file used by several
skills is edited in one of them, listed in `copies` in
[`skills/skills_test.go`](skills/skills_test.go), then copied to the others:

```bash
go test ./skills -update
```

`go test ./skills` fails if a copy differs, if skill frontmatter or links are
invalid, if a skill refers to anything outside its folder, or if it names a tool
the server does not register. Rebuild the server to pick up changes.

Pull requests run `go test`, `govulncheck`, `gosec`, Gitleaks, Trivy, `staticcheck`, and
CodeQL. Request-level GTM tests use local fake Google endpoints. Mutating live
tests must use a disposable container and clean up their entities.

See [ARCHITECTURE.md](ARCHITECTURE.md) for package boundaries, request flow,
authentication internals, token persistence, and security invariants.

For a deep dive into the Google Tag Manager MCP server, read the
[Deep Wiki](https://deepwiki.com/paolobietolini/gtm-mcp-server).

## Additional context

- [`llms.txt`](llms.txt) is embedded in the binary and served by every
  instance at `<BASE_URL>/llms.txt`.
- [`skills/`](skills) contains the [Agent Skills](#skills).
- [`examples/gtm_agent.py`](examples/gtm_agent.py) demonstrates a programmatic
  MCP client.
- [Google Tag Manager API v2 reference](https://developers.google.com/tag-platform/tag-manager/api/reference/rest)
- [Model Context Protocol](https://modelcontextprotocol.io/)

## License

BSD 3-Clause. See [LICENSE](LICENSE).

Maintained by [Paolo Bietolini](https://github.com/paolobietolini).
