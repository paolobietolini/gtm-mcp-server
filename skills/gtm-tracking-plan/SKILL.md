---
name: gtm-tracking-plan
description: Writes tracking plans and dataLayer specs for Google Tag Manager. Use when asked to document what a container tracks, plan events for measurement goals, or brief developers.
---

# GTM tracking plan

Covers what to measure and what the site must send. For configuring GA4 tags and
consent in detail, use gtm-ga4-consent.

## Inputs

- **Document an existing setup:** read the container.
- **Plan new measurement:** get the goals, the site type, and where the container
  stands today.

| Source | How |
|--------|-----|
| MCP tools | `list_tags`, `list_triggers`, `list_variables`, `get_tag`, `get_trigger` |
| `gtm-cli` | `gtm-cli tags list`, `gtm-cli triggers list`, `gtm-cli variables list`, `gtm-cli tags get`, `gtm-cli triggers get`; see the gtm-cli skill |
| Export JSON | Read [container-export.md](references/container-export.md) first |
| Neither | Plan from the goals, and ask for an export if a container exists |

## Rules

- Event names: use the GA4 recommended event whenever one fits the action, for
  example `sign_up`, `generate_lead`, `add_to_cart`, `purchase`. Add detail with
  parameters. A custom name needs a stated reason naming the recommended event
  you rejected. `sign_up` means account registration; a newsletter or contact
  form is `generate_lead`.
- Take parameter names and required fields from Google's
  [GA4 event reference](https://developers.google.com/analytics/devguides/collection/ga4/reference/events).
  If you cannot open it, tell the user to check them against it.
- Consent: Google tags have built-in checks and need an added requirement only if
  the site must block them until consent (basic mode). Ask which mode applies.
  Other marketing tags require consent, and a blocked tag is not replayed without
  a trigger on the consent event.
- Follow [ga4-consent.md](references/ga4-consent.md) for data layer conventions
  and [naming-organization.md](references/naming-organization.md) for entity
  names. A consistent existing convention wins.
- Keep observed facts, inferences and proposals distinguishable. Never present a
  data layer push as existing because a tag expects it.

## Output

Write the plan using [plan-template.md](references/plan-template.md).
