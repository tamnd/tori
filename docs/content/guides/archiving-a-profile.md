---
title: "Archiving a profile"
description: "Capture a profile's recent timeline or its full history with monthly search windows, and stay under the free-tier rate limit."
weight: 10
---

Archiving a profile is tori's headline job.
There are two depths: the recent timeline X hands out for free, and the full history you have to walk for.

## The recent timeline

```bash
tori archive karpathy --guest
```

This streams the profile's timeline.
On Tier 0 (no `--guest`) X returns only a recent window.
The `--guest` tier pages deeper through GraphQL.
Either way the capture is bounded by `--max`, which defaults to 1000 for a profile so a bare `tori archive <handle>` does not try to pull a whole history by accident.
Raise it to page further:

```bash
tori archive karpathy --guest --max 2000
```

Set `--max 0` for as many as the tier will give.

## The full history, by month

A profile timeline caps out around 3200 tweets no matter how deep you page.
To get past that, `--by-month` walks monthly `from:<handle> since:<date> until:<date>` search windows, where each window is a separate search the cap never applies to:

```bash
tori archive karpathy --by-month --with-replies
```

It does this in two passes.
First it streams the timeline for the recent window, which reads off a different rate limit than search does, then it walks search windows only for the older history the timeline could not reach.
The two passes share one dedupe set, so the overlap costs nothing, and an account small enough to fit under the cap is captured entirely from the timeline with no search windows at all.
A bad window (one that errors) is logged and skipped rather than aborting the whole run.

Two flags make the difference between a partial and a faithful full history.
Use a session, not `--guest`, so the walk does not stall on the search rate limit (see below).
Add `--with-replies` so the author's self-threads survive, since X stores each follow-on post in a thread as a reply (see [Replies and retweets](#replies-and-retweets)).

## Bounding the range

`--since` and `--until` bound which windows the by-month walk covers.
`--since` raises the floor, so the walk only visits the months it needs:

```bash
# Just 2025 onward
tori archive karpathy --guest --by-month --since 2025-01-01

# A fixed span
tori archive karpathy --guest --by-month --since 2024-01-01 --until 2025-01-01
```

Both accept a bare calendar date (`2006-01-02`, read as UTC midnight) or a full RFC3339 timestamp.
Outside `--by-month`, `--since` and `--until` still filter the captured records by time.

## The free-tier rate limit

This is the one thing worth planning for.
X rate-limits search hard.
On the free guest tier, a long `--by-month` run that walks many months will eventually hit a 429 with a multi-minute reset, sometimes around 15 minutes.
tori writes each record as it goes, so nothing is lost, but the run stalls.

Two ways to handle it:

- **Bound the range.** A `--since 2025-01-01` run walks only a handful of windows and completes fine on the guest tier.
- **Use a session for heavy runs.** Import your own X session once and the session tier (Tier 2) has far more search headroom, which is what you want for a full back-to-creation history walk:

  ```bash
  tori auth import --auth-token <...> --ct0 <...>
  tori archive karpathy --by-month --with-replies
  ```

If a run does hit the limit, tori exits with code 5 (blocked or rate-limited) and the partial archive is on disk.
Re-run later, or with `tori add`, to pick up the rest.

## Replies and retweets

By default a profile capture keeps original posts and drops replies and retweets.
Add them back when you want the full conversation footprint:

```bash
tori archive karpathy --with-replies --with-retweets
```

`--with-replies` is worth a second look for a full archive.
A self-thread, where the author replies to their own post to continue a longer point, is stored by X as a reply, so the default capture keeps only the opening post and drops the continuation.
Pass `--with-replies` to keep whole threads intact.

To keep only posts that carry media, add `--media-only`.

## Next

- Other target kinds: [capturing threads and searches](/guides/capturing-threads-and-searches/).
- Choosing what media and views to keep: [media and views](/guides/media-and-views/).
- Keeping the archive current: [incremental and resumable captures](/guides/incremental-and-resumable/).
