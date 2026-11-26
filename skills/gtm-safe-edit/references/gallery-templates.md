# Community Template Gallery

## Find the template

A Gallery template is a public GitHub repository, identified by owner and
repository name.

1. Search the Gallery at `https://tagmanager.google.com/gallery/` or search the
   web for the vendor name plus "GTM template github".
2. Take owner and repository from the URL
   `github.com/{owner}/{repository}`. Example:
   `github.com/iubenda/gtm-cookie-solution` gives owner `iubenda`, repository
   `gtm-cookie-solution`.
3. Confirm the repository exists and is the one the Gallery listing links to.
   Prefer the vendor's own account. Never guess an owner or repository.

If you cannot browse, ask the user for the repository URL.

Known repositories:

| Template | galleryOwner | galleryRepository |
|----------|--------------|-------------------|
| iubenda Cookie Solution | iubenda | gtm-cookie-solution |
| Cookiebot CMP | Usercentrics | gtm-templates-cookiebot-cmp |
| Meta Pixel | meta | GoogleTagManager-WebTemplate-For-FacebookPixel |

## Review before use

A template runs on the user's site. Report its publisher and the permissions it
requests (scripts it injects, domains it sends to, data it reads).

## Import

| Route | How |
|-------|-----|
| MCP | `import_gallery_template` with `galleryOwner` and `galleryRepository` |
| GTM interface | Templates > Search Gallery > Add to workspace |

The import goes into the workspace, so the safe-edit workflow applies. Use the
template type the import returns (`cvt_{galleryTemplateId}`) when creating the tag or variable.
Check existing templates first (`list_templates`) to avoid a duplicate.
