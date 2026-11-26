# Tracking plan template

Markdown. Drop a section only if it has no content.

## 1. Overview

Container, source and date, goals, and counts of tags, triggers and variables.

## 2. Events

One row per user action:

| Event | Status | Fires when | Parameters | Destinations | Key event |

Status is `configured`, `needs fix`, `new`, `paused` or `remove`. `configured`
does not mean published or verified on the site. "Fires when" describes the user
or system action, precisely enough to test.

## 3. Data layer requirements

For developers, usable without GTM access. For each event:

- A complete `dataLayer.push` example.
- Each parameter: name, type, required or optional, allowed values.
- Exactly when to push, how often, and when not to.
- For ecommerce, push `{ ecommerce: null }` first.

No personal data in pushes unless the user asked for it and it has a stated legal
basis.

## 4. GTM configuration

| Tag | Type | Trigger | Variables | Consent | Status |

Then triggers with their conditions and variables with their source. List unused
entities separately. End with the rollout order: which developer releases must
come before which GTM publishes.

## 5. Acceptance tests

| Action | Expected result |

Include a negative case per new event, and consent accepted and rejected.

## 6. Gaps and open questions

Problems found in the current setup, and what must be confirmed on the site or in
vendor accounts.
