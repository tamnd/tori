package media

import (
	"context"
	"time"

	"github.com/tamnd/tori/repo"
	"github.com/tamnd/x-cli/x"
)

// mediaTTL is how long a media fetch stays valid in the shared cache. Media at a
// pbs/video.twimg URL is immutable (the URL embeds a content hash), so a long
// TTL means a re-run never re-downloads what it already pulled.
const mediaTTL = 720 * time.Hour

// Result summarises a localisation pass for the manifest and the progress line.
type Result struct {
	Assets     []repo.Asset
	Downloaded int // newly fetched this run
	Reused     int // already on disk, skipped
	Failed     int // fetch errored, recorded unavailable
	StreamOnly int // no progressive rendition, recorded with master URL
}

// Logf is the optional progress sink.
type Logf func(format string, args ...any)

// Download localises every planned item through the shared x.Client (TP1), so
// media rides the one rate limiter, retry/backoff, and disk cache the record
// reads use. An item already on disk is skipped (incremental, TP6); a fetch
// failure is recorded as unavailable and never aborts the capture (spec §8).
func Download(ctx context.Context, c *x.Client, st *repo.Store, items []Item, log Logf) Result {
	var res Result
	for _, it := range items {
		if ctx.Err() != nil {
			break
		}
		asset := repo.Asset{Key: it.Key, Type: it.Type, Source: it.Source, Path: it.Path}

		if it.AltErr == "stream-only" {
			asset.Status = "stream-only"
			asset.Path = ""
			res.StreamOnly++
			res.Assets = append(res.Assets, asset)
			continue
		}
		if st.Exists(it.Path) {
			asset.Status = "local"
			res.Reused++
			res.Assets = append(res.Assets, asset)
			continue
		}

		b, err := c.Do(ctx, x.Req{URL: it.Source, Endpoint: "media", CacheTTL: mediaTTL})
		if err != nil {
			if log != nil {
				log("media %s: %v", it.Key, err)
			}
			asset.Status = "unavailable"
			asset.Path = ""
			res.Failed++
			res.Assets = append(res.Assets, asset)
			continue
		}
		if err := st.WriteMedia(it.Path, b); err != nil {
			if log != nil {
				log("write media %s: %v", it.Key, err)
			}
			asset.Status = "unavailable"
			asset.Path = ""
			res.Failed++
			res.Assets = append(res.Assets, asset)
			continue
		}
		asset.Status = "local"
		res.Downloaded++
		res.Assets = append(res.Assets, asset)
	}
	return res
}
