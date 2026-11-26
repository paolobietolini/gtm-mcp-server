---
name: gtm-container-audit
description: Audits a Google Tag Manager container and scores it against best practices. Use when asked to review, audit, clean up or health-check a GTM setup or a container export JSON.
---

# GTM container audit

Read-only. Report findings and fixes; change nothing unless asked. Then use
gtm-safe-edit; without it, work in a new workspace, show the changes and get
approval before publishing.

## Get the configuration

- **Live access:** read tags, triggers, variables and, on server containers,
  clients and transformations, through MCP tools or the `gtm-cli` command-line tool
  (see the gtm-cli skill).
- **Export JSON:** read [container-export.md](references/container-export.md)
  first.
- **Neither:** ask for an export (GTM > Admin > Export Container).

| Operation | MCP tool | `gtm-cli` |
|-----------|----------|-----------|
| Find the workspace | `list_accounts`, `list_containers`, `list_workspaces` | `gtm-cli accounts list`, `gtm-cli containers list`, `gtm-cli workspaces list` |
| Read entities | `list_tags`, `list_triggers`, `list_variables`, `list_folders` | `gtm-cli tags list`, `gtm-cli triggers list`, `gtm-cli variables list`; no folders |
| Read server entities | `list_clients`, `list_transformations` | not available: ask for an export |
| Read one entity in full | `get_tag`, `get_trigger`, `get_variable`, `get_template` | `gtm-cli tags get`, `gtm-cli triggers get`, `gtm-cli variables get`; no templates |

## Check

Run every check in [audit-checks.md](references/audit-checks.md) against these
rules:

- [naming-organization.md](references/naming-organization.md)
- [ga4-consent.md](references/ga4-consent.md)
- [server-side.md](references/server-side.md), server containers only

A convention the container already follows consistently counts as passing; note
how it differs. Mixed or absent conventions do not.

## Report

Use these parts in this order, even if the user did not ask for scores:

1. **Source:** container, type, live workspace or export and its date.
2. **Scorecard:** a table with one row each for Naming and organization,
   Duplicates, Unused entities, GA4, Consent, Security and privacy, Behavior,
   and Server-side. Score each `pass`, `warn`, `fail` or `not applicable`.
3. **Findings:** for each warn and fail, the exact entity names, why it matters,
   and the concrete fix.
4. **Looks unused but is not:** entities a cleanup must keep, with what uses them.
5. **Fix order:** numbered by risk to data, privacy and production.
6. **Not verifiable from this source.**
