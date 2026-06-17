---
title: "Introduction"
description: "How tori captures X content through the free x-cli engine, why JSON is the source of truth, and what each access tier can reach."
weight: 10
---

A tweet you see in your browser is not a document, it is the output of a program.
The HTML X sends is a near-empty shell, and the post is assembled in the page by JavaScript fetching data and building the DOM.
That is why "Save As" fails: you keep the shell, not the post, and what you do keep still calls home when you open it.
Worse, the content itself is fragile.
Posts get deleted, accounts go private, and the timeline only exposes a recent window.

tori treats an archive as a capture, a store, and a set of views, in that order.

## 1. Capture through the free engine

tori reuses the x-cli `x` engine to read X for free, with no API key.
It never invents its own scraping: it asks the engine for a profile, a thread, a search, and streams the records back.
The engine picks the cheapest surface that can serve the request, across three tiers:

- **Tier 0, syndication.** No setup, no auth. It reaches public profiles and tweets, but only the recent timeline window.
- **Tier 1, guest token.** Turned on with `--guest`. It pages deeper through the GraphQL surface, enough for `--by-month` full-history walks and larger `--max` budgets.
- **Tier 2, session cookies.** Imported once with `tori auth import --auth-token <...> --ct0 <...>`. It reaches the deepest, including your own bookmarks, and is the tier to use for heavy full-history runs that would otherwise hit the guest-tier rate limit.

You can let the engine choose, or force a tier with `--tier syndication|guest|session`.

## 2. Store JSON as the source of truth

Every tweet is written to `tweets/<id>.json` the moment it arrives.
This is the canonical record, and it is what every other part of the archive is derived from.
Writing each record immediately is deliberate: a run that is interrupted, rate-limited, or hits a session limit still leaves a valid, smaller archive on disk.
The path is a pure function of the tweet id, so a re-capture overwrites the same file and the output stays deterministic.

Media is stored beside the records.
Photos, video, animated GIFs, avatars, and banners are downloaded into `media/`, deduped (a photo referenced by a thousand tweets resolves to one file), and recorded honestly in the manifest as `local`, `unavailable`, or `stream-only`.

## 3. Derive the views

From the stored JSON, tori renders inert views you can read with no network:

- **HTML.** A browsable `index.html`, per-tweet pages, and reconstructed conversation pages under `threads/`. Scripts and handlers are stripped, media points at the local files, and links resolve to the other pages in the archive.
- **Markdown.** A `README.md` index plus per-tweet and per-thread Markdown, for reading in any editor or feeding to other tools.

Because the views are derived, they are disposable.
`tori render <repo>` rebuilds them from the JSON with no network, which is how you replay a renderer improvement over an old archive or add a Markdown view to an HTML-only one.

## The shape of an archive

A capture lands in a self-contained repository at `<out>/x/<slug>`, where `<out>` defaults to `$HOME/data/tori` (or `$TORI_OUT`).
The slug is the handle for a profile, or `status-<id>`, `search-<query>`, `likes-<user>`, and so on for the other target kinds.
Move the folder anywhere and it still opens.
See [repository layout](/reference/repository-layout/) for the full tree.

Next: [install tori](/getting-started/installation/).
