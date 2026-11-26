---
name: gtm-mcp
description: Explains how to operate the GTM MCP Server, including ID discovery, tool input formats and safety guards. Use when its GTM tools are available or a tool rejects JSON input.
---

# GTM MCP Server

How to drive the tools. For what to do with them, use the matching skill:

| Task | Skill |
|------|-------|
| Review or score a container | gtm-container-audit |
| Add, change, delete, publish, roll back | gtm-safe-edit |
| Tracking plan, data layer spec | gtm-tracking-plan |
| GA4 and consent mode | gtm-ga4-consent |
| Server containers | gtm-server-side |

Without an MCP server, use the `gtm` command-line tool (gtm-cli).

If those skills are not installed, read the server's best-practices resources.

## Discover IDs

Account > Container > Workspace > entities. Never guess an ID:

`list_accounts` > `list_containers` > `list_workspaces`

Changes happen in a workspace. Nothing is live until a version is published.

## Before creating or updating

- Call `get_tag_templates` or `get_trigger_templates` for the parameter format.
- Read [tool-input-formats.md](references/tool-input-formats.md) for the JSON
  string inputs.
- Create variables, then triggers, then tags. `create_tag` takes
  `firingTriggerIds`, an array of trigger ID strings.
- GA4 event tags use type `gaawe`: `measurementId` is an empty `tagReference` and
  the value goes in `measurementIdOverride`.
- For a Google tag use `googtag` with a `tagId` template parameter.
- Custom event triggers take `customEventFilterJson`. Click and form triggers
  take `autoEventFilterJson`; the server moves those conditions to `filter`,
  because the GTM API drops them otherwise.
- `create_trigger` cannot set trigger group members. Create the `triggerGroup`,
  then set them with `update_trigger` and `parameterJson`.

## Safety guards

Deletes, publishing, disabling built-in variables and several admin operations
are rejected without `confirm: true`. The flag is not the user's approval: say
what will happen and get a yes first.

Publishing: `get_workspace_status`, resolve conflicts, `create_version`, then
`publish_version`.

## Tool availability

The operator selects tool groups. Container, folder, workspace and version
administration, reverts, environments, destinations and Google tag settings are
off by default. If a tool is missing, say so instead of working around it.

## Prompts and resources

| Prompt | Purpose |
|--------|---------|
| `audit_container` | Load a workspace for an audit |
| `best_practices_review` | Score a workspace against the rules |
| `plan_safe_edit` | Plan a change |
| `generate_tracking_plan` | Document a workspace |
| `suggest_ga4_setup` | Recommend a GA4 structure from goals |
| `find_gallery_template` | Find and import a Gallery template |
