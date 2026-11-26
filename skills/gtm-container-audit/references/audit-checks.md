# Audit checks

## Naming and organization

- Tags, triggers and variables follow one pattern with one separator.
- No default or opaque names (`Tag 14`, `New Trigger`, `Variable 3`).
- Custom event triggers name the event they match.
- IDs live in variables; no literal ID repeated across entities.
- Folders are used consistently once the container is large.

## Duplicates

- Tags with the same type and destination. Resolve variables first: a literal ID
  and a variable holding the same ID are the same destination.
- One tag with several firing triggers that can match the same user action.
- Triggers with identical conditions.
- A vendor snippet that may also be on the page, outside GTM.

## Unused entities

- Triggers with no use, counting blocking use and trigger group membership.
- Variables with no `{{Name}}` reference anywhere, including custom HTML,
  JavaScript variables and templates.
- Tags with no trigger that are not a setup or teardown tag.
- Paused tags with no stated reason.

Propose removal. Never delete during an audit.

## GA4

- One Google tag per destination, with the ID from a variable.
- Event tags: recommended event names where one exists, values from the data
  layer, ecommerce data mapped, nothing hardcoded that varies per event.

## Consent

- A consent default is set before any Google tag fires.
- The CMP fires on Consent Initialization.
- Each other marketing tag has a consent requirement. Google tags have built-in
  checks: an added requirement is needed only if the site must block them until
  consent (basic mode). This refines the checklist in ga4-consent.md.
- A tag blocked on first page load also needs a trigger on the consent event.

## Security and privacy

- Custom HTML: what each loads and from where; any `http://` source.
- Third-party scripts that receive user data.
- Personal data passed to vendors, in tags or URLs.
- Access tokens or secrets stored in plain text. Name the entity, not the value.
- Custom templates with broad permissions or unknown origin.

## Behavior

- Tags that cannot work as configured (a stub function, a missing library, a
  trigger that never matches). Read the code and say what will happen.
- Conversion tags without value or currency where the event has them.
- Google Ads tags with no Conversion Linker or Google tag covering click IDs.
