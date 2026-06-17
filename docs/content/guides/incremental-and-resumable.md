---
title: "Incremental and resumable captures"
description: "Re-run a capture to fetch only what is new, resume an interrupted run, force a clean recapture, and pin the stamp for reproducible output."
weight: 40
---

A tori archive is meant to be kept, not captured once and forgotten.
Re-running is cheap because tori knows what it already holds.

## Fetching only what is new

`tori add` (alias `tori update`) re-captures an existing target but fetches only what is newer than the newest tweet already on disk, then re-renders the views:

```bash
tori add karpathy
```

Under the hood, an add reads the newest tweet id in the repository and uses it as the floor for the new fetch, so it pulls the gap since your last capture and nothing more.
The capture summary shows the delta as `(+N new)`.
`tori archive` on an existing repo does the same incremental fetch; `add` is just the clearer name for a re-run.

You can set the floor by hand with `--since-id`, or cap the top with `--until-id`, when you want a specific id range.

## Resuming an interrupted run

Every record is written to disk the instant it arrives, not buffered for the end.
That means an interrupted run is never wasted:

- Press Ctrl-C and tori stops, keeping every record it already wrote. It exits with code 130.
- Hit the search rate limit on a long `--by-month` walk and tori exits 5 with the partial archive intact.

Either way, run the same command again (or `tori add`) and it continues from what is on disk, fetching only the rest.

## Forcing a clean recapture

To ignore the held state and recapture from scratch (for example, to pull in edits to tweets you already have), use `--force`:

```bash
tori add karpathy --force
```

This refetches the full window rather than just the new tail, overwriting the records it finds again.

## Reproducible output

tori's output is deterministic by design: record paths and media filenames are pure functions of their content, and the manifest's record-bearing fields are sorted.
The one wall-clock value is the capture stamp.
Pin it with `--date` to make a run byte-for-byte reproducible:

```bash
tori archive karpathy --guest --date 2025-06-17T00:00:00Z
```

The same `--date` is available on `tori render` to fix the footer stamp when you rebuild the views.

## Previewing without fetching

`--dry-run` prints what a capture would target without touching the network:

```bash
tori archive karpathy --by-month --dry-run
```

## A self-contained, movable archive

The repository is fully self-contained.
Records, media, views, CSS, and the manifest all live under the one root directory, with every internal reference written as a relative path.
Move the folder to another disk or machine, open `index.html`, and it still works with the network unplugged.
To browse it over HTTP, point `tori serve` at it:

```bash
tori serve $HOME/data/tori/x/karpathy
```

## Next

- The [CLI reference](/reference/cli/) lists every flag.
- [Repository layout](/reference/repository-layout/) maps what lands on disk.
