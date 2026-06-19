---
title: "Repository layout"
description: "The on-disk shape of a tori archive: the directory tree, what each file is, and the manifest fields."
weight: 20
---

A capture writes one self-contained repository.
Everything it produces, records, media, views, styling, and the manifest, lives under a single root, and every internal reference is a relative path, so the folder is movable and opens with no network.

## Where it lands

The root is `<out>/x/<slug>`, where `<out>` is `-o/--out` (default `$HOME/data/tori`, or `$TORI_OUT`).
The slug encodes the target kind:

| Target | Slug |
|--------|------|
| Profile `karpathy` | `karpathy` |
| Tweet `20` | `status-20` |
| Thread `20` | `thread-20` |
| Search `from:nasa` | `search-from-nasa` |
| Likes of `karpathy` | `likes-karpathy` |
| List `123` | `list-123` |
| Bookmarks | `bookmarks` |

## The tree

A profile capture of `karpathy` looks like this:

```
$HOME/data/tori/x/karpathy/
├── tweets/                  # canonical records, the source of truth
│   └── 1745...json          # tweets/<id>.json, one per tweet
├── html/                    # rendered inert per-tweet pages
│   └── 1745...html
├── threads/                 # reconstructed conversations
│   ├── 1740...html
│   └── 1740...md
├── md/                      # rendered per-tweet Markdown
│   └── 1745...md
├── media/                   # localised media, bucketed by type
│   ├── photo/
│   ├── video/
│   ├── gif/
│   ├── avatar/
│   └── banner/
├── _assets/
│   └── tori.css             # the one stylesheet the HTML views share
├── index.html               # the browsable archive home
├── README.md                # the Markdown index
├── profile.json             # the captured profile
└── manifest.json            # the repository index
```

Key points:

- **JSON is the source of truth.** Each tweet is `tweets/<id>.json`, written the instant it arrives. The id is a snowflake string used verbatim, so the path is a pure function of the id and a re-capture overwrites the same file.
- **Views are derived.** `html/`, `md/`, `threads/`, `index.html`, and `README.md` are all rebuilt from the JSON by the renderer. Delete them and `tori render <repo>` recreates them with no network.
- **Media is localised and deduped.** Files go under `media/<type>/`, named by the media key plus a short hash of the source URL. Two renditions never collide, and one photo shared across many tweets resolves to a single file.
- **A standalone tweet versus a thread.** A tweet with no surrounding conversation is rendered as a single page (`html/<id>.html`, `md/<id>.md`); a multi-tweet conversation is rendered as one page under `threads/`.

## The manifest

`manifest.json` is the first file `tori info`, `tori add`, and `tori render` read.
Its record-bearing fields are sorted so a re-capture of the same content writes a byte-identical manifest; the only wall-clock values live in the capture entries.

| Field | Meaning |
|-------|---------|
| `service` | The source service, always `x` |
| `target` | What the repo archives: `kind`, `ref`, optional `user_id` and `query` |
| `tiers_used` | The access tiers that served records (`syndication`, `guest`, `session`) |
| `tweets` | Total records held |
| `media` | Count of media items localised (`status` == `local`) |
| `threads` | Number of reconstructed conversations |
| `range` | The `oldest` and `newest` captured tweet timestamps |
| `captures` | One entry per run: `at` (the stamp), `added`, and `tier` |
| `media_index` | Every media item with its `key`, `type`, `path`, `source`, and `status` |
| `tori_version` | The tori version that wrote the repo |
| `schema` | The on-disk layout version, for future migration |

Each media item's `status` is one of `local` (on disk), `unavailable` (could not be fetched), `stream-only` (needs an external `--tool` like yt-dlp), or `skipped`.
The index is the archive being honest about exactly what is and is not localised.
