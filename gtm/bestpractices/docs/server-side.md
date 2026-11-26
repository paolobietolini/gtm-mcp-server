# Server-Side Container Practices

Applies when the container usage context is "server". Server containers have
clients; web containers do not (`list_clients`).

## Clients

- One client per incoming protocol: typically the GA4 client claims `/g/collect` requests. Don't create overlapping clients that claim the same request paths — first match wins and ordering bugs are silent.
- Keep the default GA4 client unless there is a stated reason to replace it.

## Tags

- Server tags fire from client-parsed events, not page triggers. Trigger conditions use event data (e.g. `Client Name equals GA4`, `Event Name equals purchase`).
- Forward only what each vendor needs; server-side exists to control data sharing — do not blanket-forward all event data to every vendor tag.

## Transformations for PII

- Use exclude/redact transformations to strip PII (email, phone, precise location) from event data before vendor tags, unless the user explicitly enables enhanced conversions or similar features that require hashed PII.
- Augment transformations for enrichments (adding a hashed user ID, normalizing currency) — keep one concern per transformation, named after what it does.

## First-party domain

- The tagging server should run on a subdomain of the site (`gtm.example.com`), not the default `*.run.app`/`*.appspot.com` URL — otherwise ad-blockers and ITP treat it as third-party and most of the benefit is lost.
- The web container's GA4 config tag must set `server_container_url` to that first-party domain.

## When server-side is worth it

Recommend server-side only when the user needs: data filtering/redaction before vendors, extended cookie lifetime, reduced client-side JavaScript, or vendor API integrations. It adds hosting cost and operational complexity — do not suggest it by default.
