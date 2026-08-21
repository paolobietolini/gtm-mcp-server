# Plan: close the GTM API coverage gap

Status: **not implemented**. This document is the TODO list for it.

Gap measured against `stape-io/google-tag-manager-mcp-server` on 21 August
2026, by reading its `packages/core/src/tools/*.ts` action enums directly.

## Where the two servers stand

This server exposes **50 tools**, one for each operation. stape-io exposes
**18 tools**, one for each resource, each with an `action` enum. The tool count
is not the coverage: 18 multiplexed tools cover more GTM API surface than 50
single-purpose ones.

## Decision: keep one tool per operation

Confirmed 21 August 2026. The new tools follow the existing convention
(`create_zone`, `list_zones`, …), not stape-io's action-enum shape.

The cost is real and should be stated. This plan adds roughly 30 tools, taking
the server past 80. That consumes more of the model's tool budget, and some
clients cap the combined length of the server name plus the tool name — Cursor
at 60 characters, which stape-io documents in its own README. The benefit is
that each schema describes one operation, so parameters that an operation
requires can be mandatory rather than optional-and-checked-at-runtime, and the
server stays internally consistent.

If the tool count becomes a practical problem, the fallback is to gate whole
resource groups behind configuration rather than to multiplex tools, so the
schemas stay honest.

## Missing resources

Each needs a `gtm/tool_<name>.go`, registration in `gtm/tools.go`, tests, and
a README table entry.

- [ ] **Zones** — `create`, `get`, `list`, `update`, `delete`, `revert`.
      Zones subdivide a container for delegated access. Independent of the
      other items here; a reasonable first one to build.
- [ ] **Environments** — `create`, `get`, `list`, `update`, `delete`,
      `reauthorize`. `reauthorize` rotates an environment's auth token and has
      no counterpart in the existing tools; treat it as a destructive
      operation and require confirmation.
- [ ] **Google tag configurations** (`gtag_config`) — `create`, `get`,
      `list`, `update`, `delete`.
- [ ] **Destinations** — `get`, `list`, `link`, `unlink`. `link` and `unlink`
      change what a container publishes to; require confirmation.
- [ ] **User permissions** — `create`, `get`, `list`, `update`, `delete`.
      **Read the security note below before building this one.**
- [ ] **Version headers** — `list`, `latest`. Cheap, and useful for finding a
      version to publish without listing full versions.

## Missing operations on resources already supported

- [ ] `container.combine`, `container.lookup`, `container.snippet`
- [ ] `version.live`, `version.undelete`
- [ ] `workspace.sync`, `workspace.quick_preview`, `workspace.resolve_conflict`
- [ ] `revert` on tags, triggers, variables, folders, clients,
      transformations, templates, and built-in variables

`revert` is the largest single item: eight near-identical tools. Build one,
settle the shape, then repeat.

## Security note on user permissions

`user_permission` is different in kind from everything else in this server. It
grants and removes access to GTM accounts and containers. An AI assistant that
can call `create_user_permission` can grant a stranger administrator access to
a container, and `delete_user_permission` can lock the owner out.

Before building it, decide:

- [ ] Whether write operations on permissions are exposed at all, or only
      `get` and `list`.
- [ ] Whether they are gated behind configuration, off by default, so an
      operator opts in rather than inheriting the capability silently.
- [ ] What confirmation looks like. The existing confirmation pattern was
      designed for "you will lose a tag", not "you will give someone else
      administrator access".

The safe default is read-only first, writes behind an explicit opt-in. This
question should be settled before the code is written, not after.

## Sequence

1. Version headers and zones — small, self-contained, establish the pattern for
   a new resource file.
2. Environments and gtag configs — the same pattern, with one unusual
   operation each.
3. The missing operations on existing resources, `revert` last since it is
   repetitive.
4. Destinations.
5. User permissions, read-only, only after the questions above are answered.

## TODO — per resource

For each resource, in this order:

- [ ] Failing tests first, covering the happy path, a not-found error, and the
      confirmation requirement on destructive operations.
- [ ] `gtm/tool_<name>.go` following the existing file shape.
- [ ] Registration in `gtm/tools.go`.
- [ ] MCP resource URIs under `gtm://` if the resource is readable as a list.
- [ ] README table entry.
- [ ] `llms.txt` entry, so agents that read it learn the new tools.

## Not in scope

- Changing the existing 50 tools.
- Multiplexing tools behind an `action` parameter.
- The stdio transport — see `planning/stdio-transport.md`.
