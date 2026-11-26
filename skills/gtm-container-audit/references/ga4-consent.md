# GA4 and Consent Configuration

## GA4 structure

- One Google tag for GA4 (type `googtag`; `gaawc` is the legacy GA4 Configuration type) per container, firing on Initialization or All Pages, with the measurement ID sourced from a lookup table variable (`LT - GA4 Measurement ID`) keyed on hostname or environment. One tag then serves dev/staging/prod and staging traffic never hits the production property.
  - Simple single-environment containers may use a `CONST - GA4 Measurement ID` variable instead — but never a literal ID typed into the tag.
  - More than one config tag is only justified when intentionally sending to multiple GA4 properties simultaneously (e.g. a rollup property). Two config tags resolving to the same measurement ID = duplicate measurement; flag it.
- Event tags (`gaawe`) must reference the config via measurement ID; put the variable reference in `measurementIdOverride`, keep `measurementId` as an empty `tagReference`.
- Event names follow GA4 recommended events where one exists (`purchase`, `add_to_cart`, `generate_lead`, `sign_up`) before inventing custom names. Custom names: `snake_case`, verb-first.
- Event parameters come from data layer variables (`DLV - ...`), not hardcoded values, whenever the value varies per event.
- Ecommerce events should read the `ecommerce` object from the data layer (`Send Ecommerce data` option) rather than mapping items manually.

## Consent mode v2 checklist

For containers serving EEA/UK traffic:

1. A consent management platform (CMP) tag fires before all marketing tags — Consent Initialization trigger, not All Pages.
2. Default consent state is set (denied for `ad_storage`, `ad_user_data`, `ad_personalization`, `analytics_storage` unless the user's policy says otherwise) before any Google tag fires.
3. Every tag has appropriate consent handling. Google tags have built-in consent checks: in advanced consent mode add no requirement to them; in basic mode check `consentSettings` for `ad_storage` on advertising tags and `analytics_storage` on analytics tags. Other marketing tags always need a requirement.
4. No marketing pixel fires on a plain Page View trigger without consent checks in a consent-mode container.

Flag as **fail**: non-Google marketing tags with `consentStatus: notSet` in a container that has a CMP tag.

## Data layer conventions

- Event names and parameter keys: `snake_case`.
- One `dataLayer.push` per user action; include all parameters in the same push as the `event` key.
- Document required pushes for the site developer whenever a custom event trigger is created.

## Duplicate measurement pitfalls

- gtag.js snippet on-page AND a GA4 config tag in GTM for the same ID → double counting. When auditing, remind the user to check the site source.
- Two triggers with identical conditions attached to the same tag → double firing.
- Per-environment duplicate config tags (`GA4 - Config - Prod`, `GA4 - Config - Dev`) with firing exceptions → fragile; replace with one config tag + `LT - GA4 Measurement ID` lookup table.
