package media

import (
	"sort"

	"github.com/tamnd/tori/repo"
	"github.com/tamnd/x-cli/x"
)

// Plan walks the profile and a record set and returns every distinct media item
// to localise under the policy, deduped by destination path and ordered
// deterministically (TP5). Video/gif renditions are picked here; each item's
// local Path is computed now so the downloader only fetches. Quoted-tweet media
// and author avatars are included so an embedded card and the author row render
// offline (spec §8).
func Plan(profile *x.User, tweets []*x.Tweet, policy Policy, video VideoPref) []Item {
	if policy == PolicyNone {
		return nil
	}
	byPath := map[string]Item{}
	add := func(it Item) {
		if it.Source == "" || it.Path == "" {
			return
		}
		if _, ok := byPath[it.Path]; !ok {
			byPath[it.Path] = it
		}
	}

	wantVideo := policy == PolicyAll

	var visit func(t *x.Tweet)
	visit = func(t *x.Tweet) {
		if t == nil {
			return
		}
		if t.Author != nil {
			if it, ok := avatarItem(t.Author); ok {
				add(it)
			}
		}
		for _, m := range t.Media {
			switch m.Type {
			case "photo":
				ext := photoExt(m.URL)
				add(Item{Key: m.Key, Type: "photo", Source: m.URL, Path: repo.MediaPath("photo", m.Key, m.URL, ext)})
			case "video":
				if !wantVideo {
					continue
				}
				url, streamOnly := pickVariant(m, video)
				it := Item{Key: m.Key, Type: "video", Source: url, Path: repo.MediaPath("video", m.Key, url, "mp4")}
				if streamOnly {
					it.AltErr = "stream-only"
				}
				add(it)
			case "animated_gif", "gif":
				if !wantVideo {
					continue
				}
				url, streamOnly := pickVariant(m, video)
				it := Item{Key: m.Key, Type: "gif", Source: url, Path: repo.MediaPath("animated_gif", m.Key, url, "mp4")}
				if streamOnly {
					it.AltErr = "stream-only"
				}
				add(it)
			}
		}
		visit(t.Quoted)
		visit(t.Retweeted)
	}

	if profile != nil {
		if it, ok := avatarItem(profile); ok {
			add(it)
		}
		if it, ok := bannerItem(profile); ok {
			add(it)
		}
	}
	for _, t := range tweets {
		visit(t)
	}

	out := make([]Item, 0, len(byPath))
	for _, it := range byPath {
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// avatarItem builds the avatar download item for a user, keyed by handle so an
// avatar shared across a thousand tweets is one file.
func avatarItem(u *x.User) (Item, bool) {
	if u == nil || u.ProfileImage == "" {
		return Item{}, false
	}
	ext := photoExt(u.ProfileImage)
	return Item{
		Key:    "avatar:" + u.Username,
		Type:   "avatar",
		Source: u.ProfileImage,
		Path:   repo.AvatarPath(u.Username, u.ProfileImage, ext),
	}, true
}

// bannerItem builds the banner download item for a profile.
func bannerItem(u *x.User) (Item, bool) {
	if u == nil || u.ProfileBanner == "" {
		return Item{}, false
	}
	ext := photoExt(u.ProfileBanner)
	if ext == "" {
		ext = "jpg"
	}
	return Item{
		Key:    "banner:" + u.Username,
		Type:   "banner",
		Source: u.ProfileBanner,
		Path:   repo.BannerPath(u.Username, ext),
	}, true
}
