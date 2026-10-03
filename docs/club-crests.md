# Club crests

Twenty original files are bundled in `frontend/public/crests`. They are served by Vite or the production frontend host, so no image API, credential or runtime third-party request is needed.

## Provenance and licenses

See [manifest.json](../frontend/public/crests/manifest.json) for each source page, original download URL, author, license, retrieval date and SHA-256. The browser footer links to [credits.html](../frontend/public/crests/credits.html), which is also shipped in the production build. Sixteen files carry Commons public-domain classifications. The Corinthians image by Fratino.koko is CC BY-SA 4.0. Original bytes are preserved; assets retain their individual licenses rather than the code's MIT license. Club marks identify clubs and do not imply endorsement.

## Mapping and fallback

`backend/internal/football/crests.go` maps exact normalized OpenFootball names in `br.1` to local paths. Other leagues cannot accidentally inherit a Brazilian club's image. `TeamBadge.vue` renders an ordinary image, preserves its aspect ratio, supplies alternative text, and shows initials if no mapping exists or loading fails. It never injects SVG markup into the DOM.

Mirassol was added on 2026-10-02 from the public-domain Commons file sourced from the club's official site. The current RB Bragantino crest was added on 2026-10-02 at the maintainer's request from the club's official site (`bragantino.png`). Unlike the other files it is a Red Bull trademark with no free license; the manifest and credits page say so, and it should be removed if the rights holder objects. The current Vasco crest (`vasco.svg`) was added the same day, also at the maintainer's request, from Portuguese Wikipedia, where it is marked as restricted (non-free) content originating from the club; the same removal caveat applies.

## Adding a crest

1. Verify the club identity, artwork version, original source and redistribution license. Avoid historical badges presented as current or uploads with unclear authorship.
2. Download the original to `frontend/public/crests`; inspect SVG files for scripts, external resources and embedded active content. All bundled assets were checked for active elements, event handlers and external href references.
3. Record provenance, author, license, modifications and checksum in the manifest; add matching credits in the HTML page.
4. Add the exact league/name mapping in `crests.go`. Do not fetch user-supplied image URLs.
5. Build the frontend and check the profile, scorecards, mobile layout and broken-image fallback.
