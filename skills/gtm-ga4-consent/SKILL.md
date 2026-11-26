---
name: gtm-ga4-consent
description: Sets up and debugs GA4 and consent mode v2 in Google Tag Manager. Use when working on Google tags, GA4 events, cookie banners, consent checks, or duplicate or missing GA4 data.
---

# GA4 and consent in GTM

Read [ga4-consent.md](references/ga4-consent.md) first. To decide which events to
measure, use gtm-tracking-plan. Apply changes with gtm-safe-edit; without it,
work in a new workspace, show the changes and get approval before publishing.

## Inputs

| Source | How |
|--------|-----|
| MCP tools | `list_tags`, `list_triggers`, `list_variables`, `get_tag` |
| `gtm` CLI | `gtm tags list`, `gtm triggers list`, `gtm variables list`, `gtm tags get`; see gtm-cli |
| Export JSON | Read [container-export.md](references/container-export.md) first |
| Neither | For setup, work from the goals. For debugging, ask for an export |

## Set up

Follow [ga4-setup.md](references/ga4-setup.md).

## Debug

| Symptom | Check |
|---------|-------|
| Events or page views counted twice | Two Google tags for one ID (resolve variables); several firing triggers on one tag; gtag.js or a platform integration also on the page; history-change page views on single-page sites |
| Purchases counted twice | The above, plus a `transaction_id` that is empty or differs between hits, which defeats GA4 deduplication |
| Events missing | Trigger event name matches the push exactly; consent state at fire time; blocking triggers; variables resolving to empty |
| Wrong values | Hardcoded parameters; variable path does not match the push; ecommerce data not enabled |
| Consent ignored | No default set; CMP fires too late; tag without a consent requirement; consent requirement on a tag that was meant to send cookieless pings |

## Consent mode

Whether to block Google tags until consent (basic) or let them send cookieless
pings (advanced) is a legal and business decision: explain both and ask.

| Mode | Google tags | Other marketing tags |
|------|-------------|----------------------|
| Advanced | Rely on built-in consent checks; add no consent requirement. The checklist's per-tag requirement does not apply to them | Require consent |
| Basic | Require additional consent | Require consent |

A tag blocked for missing consent is not replayed when the visitor accepts. Also
trigger it on the CMP's consent event, and set it to fire once per page. Set
consent defaults with the CMP's template, not a Custom HTML tag.

## Report

State the cause with the entity names, the fix as a target configuration that
follows [ga4-consent.md](references/ga4-consent.md), how to verify it in Preview
mode and GA4 DebugView, and what only the site or GA4 can confirm.
