---
title: "CLI reference"
description: "Every tori command, flag, and exit code."
weight: 10
---

```
tori [command] [flags]
```

The commands: `archive` captures a target into a repository, `add` (alias `update`) re-captures an existing one incrementally, `render` rebuilds the views from stored JSON, `serve` previews a repository over HTTP, `info` summarises one, `auth` manages your X session, and `completion` prints a shell completion script.
`tori --version` reports the build version and commit.
Run `tori <command> --help` for the canonical, up-to-date list.

## Global flags

These persistent flags apply to every fetching command.
They configure the shared x-cli engine: access tier and politeness.
tori holds no API key; the only secret is your own session, managed with `tori auth`.

| Flag | Default | Meaning |
|------|---------|---------|
| `--guest` | `false` | Use the free guest-token tier for deeper paging |
| `--tier` | engine default | Force a tier: `syndication`, `guest`, or `session` |
| `--rate` | engine default | Minimum delay between requests |
| `--retries` | engine default | Retry attempts on a transient failure |
| `--timeout` | engine default | Per-request timeout |
| `--no-cache` | `false` | Bypass the on-disk response cache |
| `-v, --verbose` | `false` | Log the tier and each record as it is captured |

## tori archive

```
tori archive <target>... [flags]
```

Captures one or more targets into a repository at `<out>/x/<slug>`.
A target is a tweet id or URL, or a profile handle; the selector flags below switch to a thread, search, likes, List, or bookmarks capture.
Records are written as JSON as they arrive, then media is localised and the views rendered.

### Target selectors

| Flag | Default | Meaning |
|------|---------|---------|
| `--thread` | `false` | Capture the whole conversation rooted at a tweet target |
| `--search` | | Capture a search query instead of a profile |
| `--bookmarks` | `false` | Capture your own bookmarks (needs an imported session) |
| `--likes` | | Capture the tweets a user liked |
| `--list` | | Capture a List's timeline by id |

### Record shaping

| Flag | Default | Meaning |
|------|---------|---------|
| `--with-replies` | `false` | Include replies in a profile or timeline capture |
| `--with-retweets` | `false` | Include retweets in a profile or timeline capture |
| `--media-only` | `false` | Capture only tweets that carry media |
| `--by-month` | `false` | Exhaust a profile's full history via monthly search windows (needs `--guest` or a session) |
| `--since` | | Only tweets at or after this time (RFC3339 or `2006-01-02`) |
| `--until` | | Only tweets before this time (RFC3339 or `2006-01-02`) |
| `--since-id` | | Only tweets newer than this id |
| `--until-id` | | Only tweets older than this id |
| `--max` | `0` | Record budget (0 = as many as the tier gives; defaults to 1000 for a profile or search) |

### Media

| Flag | Default | Meaning |
|------|---------|---------|
| `--media` | `all` | Media to localise: `all`, `photos`, or `none` |
| `--video` | `best` | Video rendition: `best` or `worst` |
| `--tool` | | External downloader for stream-only video (e.g. `yt-dlp`) |

### Output and rendering

| Flag | Default | Meaning |
|------|---------|---------|
| `--view` | `html,md` | Views to render: `html`, `md`, or `html,md` (JSON is always written) |
| `-o, --out` | `$HOME/data/tori` | Output root; the repo lands at `<out>/x/<slug>` |
| `--date` | capture time | Fix the capture stamp (RFC3339) for reproducible output |
| `--force` | `false` | Ignore held state and recapture from scratch |
| `--dry-run` | `false` | Print what would be captured without fetching |

The output root also reads the `TORI_OUT` environment variable when `-o/--out` is not given.

## tori add

```
tori add <target>... [flags]
```

Alias: `tori update`.
The same capture machinery as `tori archive`, but it defaults to the incremental path: fetch only what is newer than the newest record already on disk, then re-render.
It takes every flag `tori archive` does.

## tori render

```
tori render <repo> [flags]
```

Re-renders the HTML and Markdown views from the stored JSON with no network.
This adds a view to an archive, or replays a renderer change over an old one.

| Flag | Default | Meaning |
|------|---------|---------|
| `--view` | `html,md` | Views to render: `html`, `md`, or `html,md` |
| `--date` | | Fix the footer stamp (RFC3339) for reproducible output |

## tori info

```
tori info <repo>
```

Prints a manifest summary: the service and target, tweet, thread and media counts, the date range, the tiers used, the capture history, and the on-disk size.
Takes no flags.

## tori serve

```
tori serve <repo> [flags]
```

Runs a local static file server over a repository so links and media resolve as they would on a host.
The archive is already self-contained, so this is a convenience over opening `index.html` directly.

| Flag | Default | Meaning |
|------|---------|---------|
| `--addr` | `127.0.0.1:8080` | Address to listen on |

## tori auth

```
tori auth import|status|logout [flags]
```

Manages the X session (Tier 2) tori shares with the x-cli toolchain.
The session is your own browser cookies, stored locally and sent only to X.

`tori auth import` flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `--auth-token` | `$X_AUTH_TOKEN` | The `auth_token` cookie from x.com |
| `--ct0` | `$X_CT0` | The `ct0` cookie from x.com |
| `--handle` | | Your @handle (optional, for display) |

`tori auth status` reports whether a session is stored; `tori auth logout` removes it.
Both take no flags.

## Exit codes

A script can branch on the outcome of any command:

| Code | Name | Meaning |
|------|------|---------|
| `0` | ok | Captured successfully |
| `1` | usage | Bad flag, malformed target, or other usage error |
| `2` | partial | An archive was written but not every reference could be localised |
| `4` | needs-auth | The target needs an imported session (Tier 2) |
| `5` | blocked | Rate-limited or blocked by X (for example a search 429) |
| `6` | not-found | The target does not exist |
| `130` | interrupted | Cancelled with Ctrl-C; the partial archive is kept |
