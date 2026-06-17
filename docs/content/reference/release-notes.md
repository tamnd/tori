---
title: "Release notes"
description: "What changed in each tori release."
weight: 40
---

The authoritative, commit-level history lives on the [releases page](https://github.com/tamnd/tori/releases).
This page summarises each version.

## v0.1.0

The first release.
tori captures a corner of X into a self-contained folder you can browse with the network unplugged: canonical JSON, the media beside it, and inert HTML and Markdown views that run no code.

- **`tori archive <target>`** captures a profile, a tweet, a thread, a search, a List, or your own likes and bookmarks into a repository at `<out>/x/<slug>`.
Every tweet is written as `tweets/<id>.json` with its raw payload kept alongside, so the JSON is the source of truth and the views are derived from it.
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
