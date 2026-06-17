# Consumed by GoReleaser: it copies the already cross-compiled binary out of the
# build context rather than compiling, so the image build is fast and uses the
# same static binary every other artifact ships.
#
# tori is a pure-Go CLI with no browser and no runtime dependencies beyond CA
# certificates, so the image is tiny: just the static binary on a minimal base.
#
# GoReleaser builds one multi-platform image with buildx and stages each
# platform's binary under a $TARGETPLATFORM directory (e.g. linux/amd64/) in the
# build context, so the COPY line selects the right one through the automatic
# TARGETPLATFORM build arg.
FROM alpine:3.21

ARG TARGETPLATFORM

# ca-certificates for HTTPS to x.com; tzdata for sane timestamps.
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 tori \
 && mkdir -p /out \
 && chown tori:tori /out

COPY $TARGETPLATFORM/tori /usr/bin/tori

USER tori
WORKDIR /out

# Archives are written under /out by default:
#
#   docker run --rm -v "$PWD/out:/out" ghcr.io/tamnd/tori archive karpathy --guest
#
# The tori user has no home directory of its own, so HOME points at the mounted
# /out volume. That keeps tori's default output and resume state writable (it
# lands under $HOME/data/tori), and lets the imported session file live there.
ENV TORI_OUT=/out \
    HOME=/out

VOLUME ["/out"]

ENTRYPOINT ["/usr/bin/tori"]
