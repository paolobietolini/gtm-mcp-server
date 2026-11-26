# Reading a GTM container export

A user creates it in GTM under Admin > Export Container, choosing a version or a
workspace. It is a JSON snapshot of the configuration.

## Layout

`containerVersion` holds `container` plus one array per entity type: `tag`,
`trigger`, `variable`, `folder`, `builtInVariable`, `customTemplate`, and on
server containers `client` and `transformation`. A missing array means there are
none.

- Container type: `containerVersion.container.usageContext` is `WEB`, `SERVER`
  or a mobile value. Server containers may list `taggingServerUrls`.
- Parameters are `{type, key, value}` objects, nested through `list` and `map`.
- Custom and Gallery template code is in `customTemplate[].templateData`. A tag,
  variable or client of type `cvt_...` uses one of these. Read the code before
  saying what such an entity does. If the code is not in the file, say its
  behavior cannot be verified.

## Enum casing

Exports write enums in upper snake case (`PAGEVIEW`, `CUSTOM_EVENT`, `TEMPLATE`,
`EQUALS`, `NOT_SET`). The API and MCP tools use camel case (`pageview`,
`customEvent`, `template`, `equals`, `notSet`). Type codes such as `googtag` are
lowercase in both.

## Built-in triggers

These IDs appear in `firingTriggerId` or `blockingTriggerId` but never in the
`trigger` array. They are not broken references.

| ID | Trigger |
|----|---------|
| `2147479553` | All Pages |
| `2147479572` | Consent Initialization - All Pages |
| `2147479573` | Initialization - All Pages |
| `2147479574` | Server containers: built-in trigger for all events |

For any other unlisted ID in this range, say it is a built-in you could not
identify.

## Type codes

| Code | Entity |
|------|--------|
| `googtag` | Google tag. ID in `tagId`, settings in `configSettingsTable` |
| `gaawc` | Legacy GA4 Configuration tag |
| `gaawe` | GA4 event tag. `eventName`, `measurementIdOverride`, parameters in `eventSettingsTable` (older: `eventParameters`) |
| `html` / `img` | Custom HTML / custom image tag |
| `awct`, `sp`, `gclidw` | Google Ads conversion, remarketing, Conversion Linker |
| `sgtmgaaw`, `gaaw_client` | Server GA4 tag, GA4 client |
| `v`, `c`, `jsm`, `j` | Data layer, constant, custom JavaScript, JavaScript variable |
| `smm`, `remm` | Lookup table, regex table |
| `k`, `u`, `ed` | Cookie, URL, event data (server) variable |

## Finding references

Check every route before calling anything unused:

- **Triggers:** `firingTriggerId`, `blockingTriggerId`, and the `triggerIds`
  list inside a `TRIGGER_GROUP` trigger.
- **Tags:** a tag with no trigger may be another tag's `setupTag` or
  `teardownTag`, which point to it by name.
- **Variables:** search the whole file for `{{Variable Name}}`, including custom
  HTML, JavaScript variables, clients and transformations.

`paused: true` marks a paused tag. Consent requirements are in
`consentSettings`.

## What an export cannot show

Publish history, which version is live, pending workspace changes, site code, the
data layer at runtime, and settings in GA4 or vendor accounts. State these limits
where they affect a conclusion.

Exports can contain access tokens and personal data. Do not repeat secret values
in answers.

Without live access, deliver changes as steps in the GTM interface. Do not claim
anything was changed.
