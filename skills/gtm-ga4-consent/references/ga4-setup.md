# GA4 setup

## Order of work

1. **Consent**, for EEA and UK traffic: the CMP's tag or template on Consent
   Initialization - All Pages, setting defaults and sending updates.
2. **Variables:** the measurement ID variable (lookup table or constant), then a
   data layer variable per parameter.
3. **Google tag** (`googtag`) on Initialization - All Pages.
4. **Triggers:** a custom event trigger per data layer event. Prefer data layer
   events to click selectors.
5. **Event tags** (`gaawe`), one per event, or one shared tag only if the
   container already works that way.
6. **Built-in variables** that triggers or tags need.

| Operation | MCP tool |
|-----------|----------|
| Parameter examples | `get_tag_templates`, `get_trigger_templates` |
| Create | `create_variable`, `create_trigger`, `create_tag` |
| Enable built-in variables | `enable_built_in_variables` |

The MCP tag examples may still show `gaawc`. For a new Google tag use `googtag`
with a `tagId` template parameter.

## What to deliver

- Tags, triggers and variables with names, types, key settings and firing
  triggers.
- The `dataLayer.push` each event needs, with an example.
- Consent requirement for each tag.
- A test list: each scenario, the expected event and parameters, and consent
  accepted and rejected.

## After setup, outside GTM

Remind the user to register custom parameters as custom dimensions, mark key
events in GA4, and remove any duplicate gtag.js on the site.
