# Naming and Organization

## Naming convention

Pattern: `<Platform> - <Type> - <Descriptor>`

- Tags: `GA4 - Event - purchase`, `GA4 - Config - G-ABC123`, `Meta - Pixel - PageView`, `cHTML - Chat Widget`
- Triggers: `<Type> - <Condition>`: `Click - CTA Button`, `CE - purchase` (CE = custom event), `PV - Checkout Pages`, `DOM Ready - All Pages`
- Variables: `<Source> - <Name>`: `DLV - transaction_id` (data layer variable), `JS - Page Type`, `LT - Environment` (lookup table), `CONST - GA4 Measurement ID`

Rules:

- Names must be descriptive without opening the entity. `Tag 14` or `New Trigger` always fails review.
- Use one consistent separator (` - `) across the container.
- Custom event trigger names should include the exact event name they match.
- Constants for every hardcoded ID (measurement IDs, pixel IDs, API endpoints): create a `CONST - ...` variable and reference it. Never inline the same literal ID in two places.

## Folders

- Use folders when a container exceeds ~20 entities. Group by platform or purpose (`GA4`, `Advertising`, `Consent`, `Utilities`).
- Every new entity goes into a folder if the container already uses folders.

## Orphans and dead weight

Flag and propose removal (never auto-delete) of:

- Triggers not referenced by any tag's `firingTriggerId` or `blockingTriggerId`.
- Variables not referenced by any tag, trigger, or other variable (search for `{{Variable Name}}` in serialized entities).
- Paused tags older than the last few published versions with no stated reason.

## Workspace hygiene

- One workspace per logical change, named after the change (`Add GA4 ecommerce`), not `Workspace 3`.
- Keep workspaces short-lived: version or discard; don't accumulate parallel long-running workspaces (GTM's 3-workspace limit on free tier makes stale workspaces expensive).
