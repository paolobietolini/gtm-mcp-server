# Safe-Edit Workflow

All GTM changes must follow this sequence. Never skip steps.

## 1. Isolate

- Create a dedicated workspace (`create_workspace`) named after the change (e.g. `Add GA4 ecommerce events`).
- Only use the Default Workspace for trivial single-entity fixes when the user says so.

## 2. Change

- Make all edits inside that workspace.
- Order of creation: variables → triggers → tags (later entities reference earlier ones).
- Check the parameter format for each tag and trigger type before constructing it (`get_tag_templates` / `get_trigger_templates`).

## 3. Review

- Get the workspace status (`get_workspace_status`) and present the pending changes to the user as a diff summary: what is added, modified, deleted.
- If there are merge conflicts, resolve them before proceeding.
- Do not proceed to versioning until the user has seen the summary.

## 4. Version

- Create a version (`create_version`) with a descriptive name and notes: what changed and why (e.g. `Add purchase + refund GA4 events for checkout revamp`).
- Version names like `v2` or `changes` fail review.

## 5. Publish — only with explicit approval

- Publishing requires explicit user approval in conversation. Ask, wait for a yes, then publish the version (`publish_version`, which also requires `confirm: true`).
- After publishing, tell the user the version ID so they can roll back.

## Rollback

- To roll back, publish the previous known-good version again (`publish_version`) — do not attempt to hand-revert entities in a new workspace.
- Find the previous live version in the version list (`list_versions`).

## Never

- Never publish directly after editing without showing the workspace status.
- Never delete entities as part of an unrelated change; propose deletions separately.
- Never edit two unrelated concerns in one workspace.
