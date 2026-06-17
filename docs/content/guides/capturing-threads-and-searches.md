---
title: "Capturing threads and searches"
description: "Archive a single tweet, a whole conversation, a search query, a user's likes, a List, or your own bookmarks."
weight: 20
---

A profile is just one kind of target.
tori captures several others, each into its own self-contained repository.

## A single tweet

Pass a tweet id or URL. No setup is needed for a public tweet:

```bash
tori archive 20
tori archive https://x.com/jack/status/20
```

## A whole thread

`--thread` captures the entire conversation rooted at a tweet, not just the one post.
The reconstructed conversation is rendered as a single page under `threads/`:

```bash
tori archive https://x.com/jack/status/20 --thread
```

## A search query

`--search` archives a search instead of a profile.
Quote the query and pass any X search operators you like:

```bash
tori archive --search "from:nasa #Artemis" --guest
tori archive --search "open source since:2025-01-01" --guest
```

Search needs paging, so use `--guest` (or a session) and bound the result count with `--max`.

## A user's likes

`--likes <user>` captures the tweets a user has liked:

```bash
tori archive --likes karpathy --guest
```

## A List

`--list <id>` captures a List's timeline by its numeric id:

```bash
tori archive --list 1234567890 --guest
```

## Your bookmarks

`--bookmarks` captures your own bookmarks.
Bookmarks are private, so this needs an imported session (Tier 2):

```bash
tori archive --bookmarks
```

## Importing a session

Tier 2 reads what the free tiers cannot: your bookmarks, and the deepest history.
It uses your own browser session, two cookies for `x.com`:

1. Open `x.com` while logged in and look in your browser's cookies for `x.com`.
2. Copy the `auth_token` and `ct0` values.
3. Import them once:

   ```bash
   tori auth import --auth-token <auth_token> --ct0 <ct0>
   ```

The cookies are stored locally and sent only to X itself, never anywhere else.
You can also supply them through the `X_AUTH_TOKEN` and `X_CT0` environment variables when the flags are omitted.
Check and clear the stored session with:

```bash
tori auth status
tori auth logout
```

If a capture needs a session and none is stored, tori exits with code 4 (needs-auth).

## Next

- [Media and views](/guides/media-and-views/): what gets downloaded and rendered.
- [Incremental and resumable captures](/guides/incremental-and-resumable/): re-running without re-fetching.
