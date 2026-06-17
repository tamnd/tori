---
title: "Media and views"
description: "Choose which media to localise, which video rendition to keep, and which views to render, then re-render from stored JSON with no network."
weight: 30
---

A capture has two derived layers on top of the canonical JSON: the localised media, and the rendered views.
Both are controlled at capture time, and both can be rebuilt later from the JSON.

## Choosing media

`--media` decides what gets downloaded beside the records:

```bash
tori archive karpathy --guest --media all     # photos and video (default)
tori archive karpathy --guest --media photos  # photos only
tori archive karpathy --guest --media none    # records only, no media
```

For video, `--video` picks the rendition:

```bash
tori archive karpathy --guest --video best    # highest quality (default)
tori archive karpathy --guest --video worst   # smallest file
```

## Stream-only video

Some X video is served only as an adaptive stream, not a single downloadable file.
tori cannot fetch those on its own and records them in the manifest as `stream-only`.
Point `--tool` at an external downloader (yt-dlp handles X streams) to capture them:

```bash
tori archive karpathy --guest --tool yt-dlp
```

The capture summary reports stream-only counts so you know when the tool is worth adding.

## How media is localised

Everything downloaded lands under `media/`, bucketed by type (`photo`, `video`, `gif`, `avatar`, `banner`).
Each file's name is the media key plus a short hash of its source URL, which makes two things true: two renditions of one item never collide, and a photo referenced by a thousand tweets resolves to a single file on disk.
The rendered pages rewrite their `src` attributes to these local paths, so the archive opens with no network.

The manifest is honest about what made it: every media item is recorded as `local`, `unavailable`, `stream-only`, or `skipped`.

## Choosing views

`--view` selects which rendered views to write.
JSON is always written regardless:

```bash
tori archive karpathy --guest --view html,md   # both (default)
tori archive karpathy --guest --view html      # HTML only
tori archive karpathy --guest --view md        # Markdown only
```

HTML gives you a browsable `index.html`, per-tweet pages, and conversation pages under `threads/`.
Markdown gives you a `README.md` index plus per-tweet and per-thread Markdown.

## Re-rendering offline

Because the views are derived, you can rebuild them from the stored JSON at any time with no network:

```bash
tori render $HOME/data/tori/x/karpathy
```

This is how you add a Markdown view to an HTML-only archive, or replay a renderer improvement over an old capture:

```bash
# Add Markdown to an archive that only had HTML
tori render $HOME/data/tori/x/karpathy --view md
```

`tori render` reads only `tweets/<id>.json` and `profile.json`.
It never touches the network and never re-downloads media.
Pass `--date` to fix the footer stamp for reproducible output.

## Next

- [Incremental and resumable captures](/guides/incremental-and-resumable/): keeping the archive current.
