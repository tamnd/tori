---
title: "tori"
description: "tori (鳥, bird) builds offline, browsable archives of X (Twitter) content from one pure-Go binary. Capture a profile, thread, or search into canonical JSON with localised media and inert HTML and Markdown views that open with the network unplugged. No API key."
heroTitle: "An X archive that outlives the post"
heroLead: "tori reads X through the free tiers of the x-cli engine, writes every tweet as canonical JSON, downloads the media beside it, and renders inert HTML and Markdown you can open straight from disk. The headline trick: it walks monthly search windows to capture a profile's full history, not just the recent window X hands out."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

X is a walled garden.
Posts get deleted, accounts go private or vanish, and the timeline only ever shows you a recent slice.
"Save As" on a tweet gives you a dead page: the markup is built by JavaScript at runtime, so you keep a shell that renders blank and still phones home.
tori (鳥, "bird") takes the opposite approach.
It captures the content through the free x-cli engine, stores it as plain JSON, and renders views that run no code.

Say you want to keep Andrej Karpathy's posts on a laptop with no wifi.
One command captures the profile; a second serves it back offline:

```bash
tori archive karpathy --guest
tori serve $HOME/data/tori/x/karpathy
```

## What it does

- **Captures over the free tiers.** tori reuses the x-cli `x` engine to read X with no API key. Tier 0 syndication needs no setup, `--guest` opens the guest-token tier for deeper paging, and `tori auth import` uses your own session for the rest.
- **Keeps JSON as the source of truth.** Every tweet lands as `tweets/<id>.json`. The HTML and Markdown views are derived from it and regenerable offline with [`tori render`](/guides/media-and-views/).
- **Localises the media.** Photos, video, and avatars are downloaded beside the records, deduped, and rewritten to local paths so the archive is self-contained and movable.
- **Walks the full history.** [`--by-month`](/guides/archiving-a-profile/) exhausts a profile through monthly `from:<handle>` search windows, sidestepping the ~3200-tweet timeline cap.
- **Stays incremental and resumable.** Re-run with [`tori add`](/guides/incremental-and-resumable/) to fetch only what is new. Ctrl-C keeps what it already got. The output is deterministic.

## Where to go next

- New here? Start with the [introduction](/getting-started/introduction/), then the [quick start](/getting-started/quick-start/).
- Want to install it? See [installation](/getting-started/installation/).
- Looking for a specific task? The [guides](/guides/) cover archiving a whole profile, capturing threads and searches, media and views, and incremental re-capture.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface, and [repository layout](/reference/repository-layout/) maps what lands on disk.
