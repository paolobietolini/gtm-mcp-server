---
name: gtm-server-side
description: Reviews and configures server-side Google Tag Manager containers. Use when working on sGTM clients, server tags, transformations, PII redaction or first-party tagging domains.
---

# Server-side GTM

Read [server-side.md](references/server-side.md) first. Apply changes with
gtm-safe-edit; without it, work in a new workspace, show the changes and get
approval before publishing.

## Inputs

| Source | How |
|--------|-----|
| MCP tools | `list_clients`, `list_transformations`, `list_tags`, `list_triggers`, `list_variables`, `get_template` |
| `gtm-cli` | Tags, triggers and variables only (see the gtm-cli skill). Clients, transformations and templates need MCP tools or an export |
| Export JSON | Read [container-export.md](references/container-export.md) first |
| Neither | Ask for an export of the server container |

The web container decides what reaches the server. Ask for its export too when
the question involves consent, event data or the server URL.

## Review

Cover all of these, even if the question is about one vendor:

1. **Clients:** which paths each claims, overlaps, and priority (higher runs
   first).
2. **Data flow:** one row per vendor tag with the trigger, fields mapped in the
   tag, fields the template reads by itself (check its code), consent condition,
   and transformations applied.
3. **Transformations:** for each personal data field a vendor does not need,
   name the exclude or allow transformation to add and the tags it affects.
4. **Domain:** the tagging server URL, and `server_container_url` on the web
   container's Google tag.
5. **Secrets:** tokens stored in plain text.

Consent on a server container: it arrives with the event (GA4 requests carry
`x-ga-gcs` and `x-ga-gcd`, or a parameter the web side adds). For each vendor tag,
find what enforces it: a trigger condition, a template setting or the tag's
`consentSettings`. A transformation changes the event data tags receive; check in
Preview whether a template also reads the request directly.

## Change

| Operation | MCP tool |
|-----------|----------|
| Client | `create_client`, `update_client` |
| Transformation | `create_transformation`, `update_transformation` |
| Tag and trigger | `create_tag`, `create_trigger` |

A transformation changes what every affected tag receives. List those tags and
test each one in Preview mode.

If the user is only considering server-side tagging, say what need would justify
its hosting cost before proposing a setup.
