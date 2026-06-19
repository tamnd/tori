---
title: "Installation"
description: "Install tori from Go, Homebrew, Scoop, a release archive, a Linux package, or the container image, and add shell completion."
weight: 20
---

tori is a single binary.
It needs no API key.
The only optional secret is your own X session, imported once for Tier 2 (see [the auth guide](/guides/capturing-threads-and-searches/)).

## Go

```bash
go install github.com/tamnd/tori/cmd/tori@latest
```

## Homebrew (macOS)

```bash
brew install tamnd/tap/tori
```

The cask installs the prebuilt macOS binary. On Linux, use the packages below or
`go install`.

## Scoop (Windows)

```bash
scoop bucket add tamnd https://github.com/tamnd/scoop-bucket
scoop install tori
```

## Release archives and Linux packages

Every [release](https://github.com/tamnd/tori/releases) attaches `tar.gz` archives (and a `.zip` for Windows) for Linux, macOS, Windows, and FreeBSD, plus `.deb`, `.rpm`, and `.apk` packages and a `checksums.txt`.
Download the one for your platform, extract `tori`, and put it on your `PATH`.

```bash
# Debian/Ubuntu
sudo dpkg -i tori_*_amd64.deb

# Fedora/RHEL
sudo rpm -i tori-*.x86_64.rpm
```

## Container

The image carries tori and nothing else.
Mount a directory for the output and point the archive at a host inside the container:

```bash
docker run --rm -v "$PWD/out:/out" ghcr.io/tamnd/tori archive karpathy --guest
```

The archive lands in `./out/x/karpathy/` on your host.
Set the output root with `-o /out` if your mount differs, or with the `TORI_OUT` environment variable.

## Shell completion

tori ships completion scripts for bash, zsh, fish, and PowerShell:

```bash
# zsh, for the current session
source <(tori completion zsh)

# bash, installed system-wide
tori completion bash | sudo tee /etc/bash_completion.d/tori
```

## No API key needed

tori reads X through the free tiers of the x-cli engine.
Tier 0 syndication and the `--guest` tier need no credentials at all.
To reach your bookmarks and the deepest history, import your own session once:

```bash
tori auth import --auth-token <...> --ct0 <...>
```

The cookies are stored locally and never sent anywhere but to X itself.
See [capturing threads and searches](/guides/capturing-threads-and-searches/) for where to find them.

Next: [the quick start](/getting-started/quick-start/).
