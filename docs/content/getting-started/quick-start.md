---
title: "Quick start"
description: "From an empty terminal to a self-contained offline X archive you can click through."
weight: 30
---

This walks the core loop: capture a profile, look at what landed on disk, serve it back, then deepen the capture and keep it up to date.

## 1. Capture a profile

```bash
tori archive karpathy --guest
```

tori resolves the profile, streams the timeline through the free guest tier, writes each tweet as JSON, downloads the media beside it, and renders the HTML and Markdown views.
The summary tells you where the archive landed:

```
@karpathy
  repo:    /home/you/data/tori/x/karpathy
  tweets:  1000 total (+1000 new)
  threads: 240
  range:   2024-01-03T... … 2025-06-17T...
  media:   612 local
  tiers:   guest
```

Without `--guest` the capture still works on Tier 0, but X only hands out the recent timeline window.
The `--guest` tier pages deeper.

## 2. Look at what landed

```bash
ls $HOME/data/tori/x/karpathy
```

```
tweets/        # tweets/<id>.json, the source of truth
html/          # per-tweet inert pages
threads/       # reconstructed conversations
md/            # per-tweet Markdown
media/         # localised photos, video, avatars
_assets/       # tori.css
index.html     # the browsable archive home
README.md      # the Markdown index
profile.json   # the captured profile
manifest.json  # counts, range, tiers, capture history
```

Open `index.html` directly in a browser and it renders offline, with no network.

## 3. Serve it back

`tori serve` runs a local static server so links and media resolve exactly as they would on a real host:

```bash
tori serve $HOME/data/tori/x/karpathy
# open http://127.0.0.1:8080
```

## 4. Go deeper, then keep it fresh

The recent timeline is only a slice.
To capture a profile's fuller history, walk monthly search windows.
Bound the range with `--since` so a free-tier run stays under X's search rate limit:

```bash
tori archive karpathy --guest --by-month --since 2025-01-01
```

Later, re-run with `tori add` to fetch only what is new since the last capture and re-render the views:

```bash
tori add karpathy
```

## Where to go next

- The [guides](/guides/) cover full-history archiving, threads and searches, media and views, and incremental re-capture in depth.
- The [CLI reference](/reference/cli/) lists every command and flag.
