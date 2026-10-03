# GTM Configuration Best Practices

Opinionated rules for building maintainable Google Tag Manager containers.
Read the relevant topic before creating or editing entities, and apply the
rules unless the user's container already follows a different documented
convention.

## Topics

| Topic | Resource URI | Covers |
|-------|--------------|--------|
| naming-organization | gtm://best-practices/naming-organization | Naming conventions, folders, orphan cleanup, workspace hygiene |
| safe-edit-workflow | gtm://best-practices/safe-edit-workflow | How to make changes safely: workspace → diff → version → publish |
| ga4-consent | gtm://best-practices/ga4-consent | GA4 tag patterns, consent mode v2, duplicate measurement |
| server-side | gtm://best-practices/server-side | Server containers: clients, transformations, PII, first-party domains |
| tool-input-formats | gtm://best-practices/tool-input-formats | JSON parameter, sequencing, trigger, transformation, and template input shapes |

## How to apply

1. Before any edit: read safe-edit-workflow and follow it.
2. Before creating entities: read naming-organization and name accordingly.
3. When working with GA4 or consent tools: read ga4-consent.
4. When the container usage context is "server": read server-side.
5. Before passing JSON strings to mutation tools: read tool-input-formats.
6. When auditing: score the container against every topic (pass / warn / fail
   per rule) and propose concrete fixes.

Existing conventions in a container take precedence over these rules — match
what is there, and flag inconsistency instead of silently introducing a
second convention.
