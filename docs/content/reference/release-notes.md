---
title: "Release notes"
description: "What changed in each tori release."
weight: 40
---

The authoritative, commit-level history lives on the [releases page](https://github.com/tamnd/tori/releases).
This page summarises each version.

## v0.2.2

Documentation accuracy.

- **The repository layout now matches what lands on disk.** A pass over every command and flag found the docs and README describing a `tweets/<id>.raw.json` sidecar beside each record, but the engine-backed capture writes only `tweets/<id>.json`, so the raw file was documented and never produced. The tree and the layout notes drop it.
- **No code changes.** The binary is identical to v0.2.1; this release ships the corrected docs.

## v0.2.1

Documentation polish.

- **A terminal demo on the docs.** The recorded walk-through of the capture, inspect, and serve loop now runs on the landing page and the [quick start](/getting-started/quick-start/), not just the README.
- **No code changes.** The binary is identical to v0.2.0; this release exists only to ship the demo and the docs review into a tagged version.

## v0.2.0

A faster, lighter full-history walk.

- **`--by-month` is now a hybrid two-pass capture.** It streams the profile timeline first, which reads off a different rate limit than search, then walks monthly search windows only for the older history the timeline could not reach.
The recent window comes off the timeline quota and the older gap comes off search, so a long run leans less on the heavily throttled search surface.
- **Accounts under the ~3200-post cap cost no search at all.** When the timeline pass reaches the account's creation, the search pass walks zero windows, so a smaller profile is captured entirely from the timeline.
- **Measured on a real archive.** For [@karpathy](/guides/archiving-a-profile/) (10,118 posts, 9,200 captured back to 2009), the hybrid cuts search-quota page requests by 34% (578 down to 381) and drops 71 of 207 month-windows off the search walk, moving the recent ~3200 posts onto the separate timeline quota.
- **Honest about the ceiling.** Both surfaces are throttled per account, so for a very prolific account the two passes run in sequence on similar quotas and the single-run wall-clock gain is modest; the win is the search-quota relief and the zero-search capture of anything under the cap.
- **Tier 0 stays clean.** With no search available, a `--by-month` run keeps the recent timeline window and says the full history needs `--guest` or a session, rather than walking windows it cannot run.

## v0.1.0

The first release.
tori captures a corner of X into a self-contained folder you can browse with the network unplugged: canonical JSON, the media beside it, and inert HTML and Markdown views that run no code.

- **`tori archive <target>`** captures a profile, a tweet, a thread, a search, a List, or your own likes and bookmarks into a repository at `<out>/x/<slug>`.
Every tweet is written as `tweets/<id>.json`, the source of truth the HTML and Markdown views are derived from.
- **The full history, not just the recent window.** [`--by-month`](/guides/archiving-a-profile/) walks a profile through monthly `from:<handle>` search windows newest-first, sidestepping the roughly 3200-tweet cap X puts on a plain timeline.
The acceptance run captured Andrej Karpathy's profile this way.
- **Three free tiers, no API key.** tori reads X through the x-cli engine: Tier 0 syndication needs no setup, `--guest` opens the guest-token tier for deeper paging, and `tori auth import` uses your own browser session for the rest.
The only secret is your own session, stored locally and sent only to X.
- **Media is localised and deduped.** Photos, video, and avatars are downloaded beside the records, stored once by content key, and rewritten to local paths, so the archive is self-contained and moves as one folder.
- **Inert HTML and Markdown views.** [`tori render`](/guides/media-and-views/) rebuilds both from the stored JSON with no network, so a renderer change replays over an old capture and a view can be added later.
The HTML runs no JavaScript and phones nowhere home.
- **Incremental, resumable, deterministic.** [`tori add`](/guides/incremental-and-resumable/) fetches only what is newer than the newest record on disk, Ctrl-C keeps what it already got, and the output is byte-stable so re-running changes nothing it does not need to.
- **`tori serve` and `tori info`.** `serve` previews a repository over a local file server so links and media resolve as they would on a host; `info` prints a manifest summary with the tweet, thread, and media counts, the date range, and the on-disk size.
- **Packaged everywhere.** Archives, `.deb`/`.rpm`/`.apk`, a multi-arch GHCR image, checksums, SBOMs, and a cosign signature, all from one tag.
- **AGPL-3.0-only.** tori links the x-cli engine, which is AGPL because it derives from nitter, so tori carries the same license.
