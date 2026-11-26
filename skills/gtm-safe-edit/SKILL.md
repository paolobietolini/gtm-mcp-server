---
name: gtm-safe-edit
description: Plans and applies Google Tag Manager changes without breaking production. Use when adding, editing, deleting, publishing or rolling back GTM tags, triggers, variables or templates.
---

# GTM safe edit

Read [safe-edit-workflow.md](references/safe-edit-workflow.md) and
[naming-organization.md](references/naming-organization.md) before planning. A
deadline does not shorten the workflow. To save time, reduce the scope: ship the
requested change and move cleanup to a later version.

## Inputs

- **Live access:** through MCP tools or the `gtm` command-line tool (see
  gtm-cli). Read the current entities before proposing anything.
- **Export JSON:** read [container-export.md](references/container-export.md),
  then give steps for the GTM interface.
- **Neither:** ask for an export (GTM > Admin > Export Container).

For a Community Template Gallery template, read
[gallery-templates.md](references/gallery-templates.md).

## Plan

Show the plan before changing anything. It contains, in order:

1. **Scope:** what changes, and what is deferred.
2. **Rollback target:** the version that is live now. Read it, or tell the user to
   note its number on the Versions page before starting. If the latest version is
   newer than the live one (`get_latest_version_header`), stop and ask: a new
   workspace would carry those unpublished changes.
3. **Dependencies:** everything that references an entity being edited or deleted.
4. **Changes:** a table of action, entity, name, type, key settings and triggers,
   in creation order.
5. **Site work:** data layer pushes developers must add, if any.
6. **Test:** what to check in Preview mode, with the expected result.
7. **Version:** the exact name and notes to use.
8. **Publish and rollback.**
9. **Not in this change:** deferred cleanup and problems found while reading the
   container, briefly.

## Apply

| Step | Operation | MCP tool | `gtm` CLI | GTM interface |
|------|-----------|----------|-----------|---------------|
| Rollback target | Get the live version | `get_live_version` | not available | Versions |
| Isolate | Create a workspace | `create_workspace` | not available | Workspace menu |
| Formats | Get parameter examples | `get_tag_templates`, `get_trigger_templates` | `gtm tags get`, `gtm triggers get` on a similar entity | not needed |
| Change | Create or update | `create_variable`, `create_trigger`, `create_tag`, `update_tag` | `gtm variables create`, `gtm triggers create`, `gtm tags create`, `gtm tags update` | Variables, Triggers, Tags |
| Compile check | Preview the workspace | `quick_preview_workspace` | not available | Preview |
| Review | Get workspace status | `get_workspace_status` | `gtm workspaces status` | Workspace Changes |
| Version | Create a version | `create_version` | `gtm versions create` | Submit > Create Version |
| Publish | Publish the version | `publish_version` | `gtm versions publish` | Publish |

Where the CLI has no command, the user does that step in the GTM interface. With
the CLI, add `--dry-run` to preview each change.

Two gates need an explicit yes from the user: after showing the workspace
changes, before creating the version; and before publishing. A request to "get it
live" made before they saw the changes is not that yes. When the user applies the
steps themselves, write both as checkpoints before Create Version and Publish.

`quick_preview_workspace` only checks that the workspace compiles. Behavior on
the site needs Preview mode in the GTM interface.

After publishing, report the new version ID and the rollback version ID.

## Deletions

Delete only what the user asked for or approved by name, in a version separate
from feature work. Check every reference first (`delete_tag`, `delete_trigger`,
`delete_variable`, or `gtm <nouns> delete`).
